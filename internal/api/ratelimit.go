package api

import (
	"net/http"
	"fmt"
	"sync"
	"time"
)

type visitor struct {
	count    int
	firstSeen time.Time
	lastSeen  time.Time
	blocked   bool
	blockUntil time.Time
}

type RateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	limit    int
	window   time.Duration
	blockFor time.Duration
}

func NewRateLimiter(limit int, window, blockFor time.Duration) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		limit:    limit,
		window:   window,
		blockFor: blockFor,
	}
	go rl.cleanup()
	return rl
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, v := range rl.visitors {
			if now.Sub(v.lastSeen) > rl.window*2 {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *RateLimiter) Allow(ip string) (bool, int) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	v, exists := rl.visitors[ip]
	if !exists {
		rl.visitors[ip] = &visitor{count: 1, firstSeen: now, lastSeen: now}
		return true, rl.limit - 1
	}

	// Still blocked?
	if v.blocked {
		if now.Before(v.blockUntil) {
			return false, 0
		}
		v.blocked = false
		v.count = 0
		v.firstSeen = now
	}

	// Reset window
	if now.Sub(v.firstSeen) > rl.window {
		v.count = 0
		v.firstSeen = now
	}

	v.count++
	v.lastSeen = now

	if v.count > rl.limit {
		v.blocked = true
		v.blockUntil = now.Add(rl.blockFor)
		return false, 0
	}

	return true, rl.limit - v.count
}

func (rl *RateLimiter) IsBlocked(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	v, ok := rl.visitors[ip]
	if !ok {
		return false
	}
	return v.blocked && time.Now().Before(v.blockUntil)
}

// Middleware — different limits for different routes
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := realIP(r)
		allowed, remaining := rl.Allow(ip)
		w.Header().Set("X-RateLimit-Limit", "100")
		w.Header().Set("X-RateLimit-Remaining", itoa(remaining))
		if !allowed {
			w.Header().Set("Retry-After", "60")
			http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Strict limiter for auth endpoints
func (rl *RateLimiter) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := realIP(r)
		allowed, _ := rl.Allow(ip)
		if !allowed {
			w.Header().Set("Retry-After", "300")
			http.Error(w, `{"error":"too many login attempts, try after 5 minutes"}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func realIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	return r.RemoteAddr
}

func itoa(n int) string {
	if n < 0 {
		return "0"
	}
	return fmt.Sprintf("%d", n)
}
