package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/yourusername/logwatch/internal/auth"
	"github.com/yourusername/logwatch/internal/geoip"
	"github.com/yourusername/logwatch/internal/storage"
)

type Server struct {
	apiRL  *RateLimiter
	authRL *RateLimiter
	totp   *auth.TOTPManager
	geo     *geoip.Resolver
	db      *storage.DB
	port    int
	jwt     *auth.JWTManager
	users   *auth.UserStore
	hub     *Hub
}

func New(db *storage.DB, port int, jwt *auth.JWTManager, users *auth.UserStore) *Server {
	hub := NewHub()
	go hub.Run()
	apiRL  := NewRateLimiter(100, 60*time.Second, 60*time.Second)
	authRL := NewRateLimiter(20, 60*time.Second, 5*60*time.Second)
	return &Server{db: db, port: port, jwt: jwt, users: users, hub: hub, apiRL: apiRL, authRL: authRL}
}

func (s *Server) Start() error {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(securityHeaders)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173", "http://127.0.0.1:5173", "http://localhost:5174", "http://127.0.0.1:5174", "http://autun8nservice.duckdns.org:9090", "https://autun8nservice.duckdns.org:9090"},
		AllowedMethods:   []string{"GET", "POST", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	// Public routes
	r.Get("/health", s.handleHealth)
	r.Get("/api/health", s.handleHealth)
	r.With(s.authRL.AuthMiddleware).Post("/api/auth/login", s.handleLogin)
	r.Post("/api/auth/setup", s.handleSetup) // First-run admin setup
	r.Get("/api/v1/ws", s.handleWebSocket) // WebSocket — JWT check inside handler

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(s.jwt.Middleware)

		r.Get("/api/auth/me", s.handleMe)
		r.Get("/api/auth/totp/status", s.handleTOTPStatus)
		r.Post("/api/auth/totp/setup", s.handleTOTPSetup)
		r.Post("/api/auth/totp/enable", s.handleTOTPEnable)
		r.Post("/api/auth/totp/disable", s.handleTOTPDisable)
		r.Post("/api/auth/logout", s.handleLogout)

		r.With(s.apiRL.Middleware).Route("/api/v1", func(r chi.Router) {
			// Viewer + Admin
			r.Get("/stats", s.handleStats)
			r.Get("/top/ips", s.handleTopIPs)
			r.Get("/top/paths", s.handleTopPaths)
			r.Get("/status-codes", s.handleStatusCodes)
			r.Get("/security/stats", s.handleSecurityStats)
			r.Get("/security/threats", s.handleRecentThreats)
			r.Get("/security/attackers", s.handleTopAttackers)
			r.Get("/geo/map", s.handleGeoMap)
			r.Get("/services", s.handleServices)
			r.Get("/collectors/fail2ban", s.handleFail2banStats)
			r.Get("/collectors/ufw", s.handleUFWStats)
			r.Get("/export/json", s.handleExportJSON)
			r.Get("/export/csv", s.handleExportCSV)
			r.Get("/export/pdf", s.handleExportPDF)

			// Admin only
			r.Group(func(r chi.Router) {
				r.Use(auth.AdminOnly)
				r.Get("/users", s.handleListUsers)
				r.Post("/users", s.handleCreateUser)
				r.Delete("/users/{id}", s.handleDeleteUser)
			})
		})
	})

	r.Handle("/*", staticHandler())

	addr := fmt.Sprintf("127.0.0.1:%d", s.port)
	log.Printf("[api] listening on http://%s", addr)
	return http.ListenAndServe(addr, r)
}

// Auth handlers
func (s *Server) handleSetup(w http.ResponseWriter, r *http.Request) {
	count, err := s.users.Count()
	if err != nil || count > 0 {
		http.Error(w, `{"error":"setup already done"}`, http.StatusForbidden)
		return
	}
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Username == "" || req.Password == "" {
		http.Error(w, `{"error":"username and password required"}`, http.StatusBadRequest)
		return
	}
	if len(req.Password) < 8 {
		http.Error(w, `{"error":"password must be at least 8 characters"}`, http.StatusBadRequest)
		return
	}
	user, err := s.users.Create(req.Username, req.Password, auth.RoleAdmin)
	if err != nil {
		http.Error(w, `{"error":"failed to create user"}`, http.StatusInternalServerError)
		return
	}
	token, _ := s.jwt.Generate(user)
	writeJSON(w, map[string]any{"token": token, "user": map[string]any{
		"id": user.ID, "username": user.Username, "role": user.Role,
	}})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Code     string `json:"code,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}
	user, err := s.users.Verify(req.Username, req.Password)
	if err != nil {
		http.Error(w, `{"error":"invalid credentials"}`, http.StatusUnauthorized)
		return
	}
	// Check TOTP if enabled
	if s.totp != nil && s.totp.IsEnabled(user.ID) {
		if req.Code == "" {
			http.Error(w, `{"error":"TOTP code required"}`, http.StatusUnauthorized)
			return
		}
		valid, err := s.totp.Verify(user.ID, req.Code)
		if err != nil || !valid {
			http.Error(w, `{"error":"invalid TOTP code"}`, http.StatusUnauthorized)
			return
		}
	}
	token, err := s.jwt.Generate(user)
	if err != nil {
		http.Error(w, `{"error":"token generation failed"}`, http.StatusInternalServerError)
		return
	}
	// Set cookie + return token
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		HttpOnly: true,
		Path:     "/",
		MaxAge:   86400,
		SameSite: http.SameSiteLaxMode,
	})
	writeJSON(w, map[string]any{"token": token, "user": map[string]any{
		"id": user.ID, "username": user.Username, "role": user.Role,
	}})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name: "token", Value: "", MaxAge: -1, Path: "/",
	})
	writeJSON(w, map[string]any{"ok": true})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	writeJSON(w, map[string]any{
		"id": claims.UserID, "username": claims.Username, "role": claims.Role,
	})
}

func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.users.List()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, users)
}

func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string    `json:"username"`
		Password string    `json:"password"`
		Role     auth.Role `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, 400)
		return
	}
	if req.Role == "" {
		req.Role = auth.RoleViewer
	}
	user, err := s.users.Create(req.Username, req.Password, req.Role)
	if err != nil {
		http.Error(w, `{"error":"failed to create user"}`, 500)
		return
	}
	writeJSON(w, user)
}

func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	_, err := s.db.Exec(`DELETE FROM users WHERE id = ?`, id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

// Data handlers
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{"status": "ok", "time": time.Now().Format(time.RFC3339)})
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	since := time.Now().Add(-24 * time.Hour)
	stats, err := s.db.GetStats(since)
	if err != nil { http.Error(w, err.Error(), 500); return }
	writeJSON(w, stats)
}

func (s *Server) handleTopIPs(w http.ResponseWriter, r *http.Request) {
	since := time.Now().Add(-24 * time.Hour)
	data, err := s.db.GetTopIPs(since, 10)
	if err != nil { http.Error(w, err.Error(), 500); return }
	writeJSON(w, data)
}

func (s *Server) handleTopPaths(w http.ResponseWriter, r *http.Request) {
	since := time.Now().Add(-24 * time.Hour)
	data, err := s.db.GetTopPaths(since, 10)
	if err != nil { http.Error(w, err.Error(), 500); return }
	writeJSON(w, data)
}

func (s *Server) handleStatusCodes(w http.ResponseWriter, r *http.Request) {
	since := time.Now().Add(-24 * time.Hour)
	data, err := s.db.GetStatusCodes(since)
	if err != nil { http.Error(w, err.Error(), 500); return }
	writeJSON(w, data)
}

func (s *Server) handleSecurityStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.db.GetSecurityStatsFixed()
	if err != nil { http.Error(w, err.Error(), 500); return }
	writeJSON(w, stats)
}

func (s *Server) handleRecentThreats(w http.ResponseWriter, r *http.Request) {
	events, err := s.db.GetRecentThreats(50)
	if err != nil { http.Error(w, err.Error(), 500); return }
	writeJSON(w, events)
}

func (s *Server) handleTopAttackers(w http.ResponseWriter, r *http.Request) {
	data, err := s.db.GetTopAttackersFixed(10)
	if err != nil { http.Error(w, err.Error(), 500); return }
	writeJSON(w, data)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func (s *Server) SetTOTP(t *auth.TOTPManager) {
	s.totp = t
}
