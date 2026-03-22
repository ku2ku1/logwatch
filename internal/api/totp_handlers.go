package api

import (
	"encoding/json"
	"net/http"

	"github.com/yourusername/logwatch/internal/auth"
)

func (s *Server) handleTOTPSetup(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	setup, err := s.totp.Generate(claims.UserID, claims.Username)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, setup)
}

func (s *Server) handleTOTPEnable(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var req struct{ Code string `json:"code"` }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if err := s.totp.Enable(claims.UserID, req.Code); err != nil {
		http.Error(w, "invalid code", http.StatusBadRequest)
		return
	}
	writeJSON(w, map[string]string{"status": "2FA enabled"})
}

func (s *Server) handleTOTPDisable(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var req struct{ Code string `json:"code"` }
	json.NewDecoder(r.Body).Decode(&req)
	valid, _ := s.totp.Verify(claims.UserID, req.Code)
	if !valid {
		http.Error(w, "invalid code", http.StatusBadRequest)
		return
	}
	s.totp.Disable(claims.UserID)
	writeJSON(w, map[string]string{"status": "2FA disabled"})
}

func (s *Server) handleTOTPStatus(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	enabled := s.totp.IsEnabled(claims.UserID)
	writeJSON(w, map[string]bool{"enabled": enabled})
}

func (s *Server) handleRateLimitStats(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]interface{}{
		"status": "active",
		"limits": map[string]string{
			"api":   "100 req/min",
			"login": "5 req/min → 5min block",
		},
	})
}
