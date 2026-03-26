package api

import (
	"net/http"
	"time"

	"github.com/yourusername/logvance/internal/collector"
)

func (s *Server) handleServices(w http.ResponseWriter, r *http.Request) {
	services := collector.AutoDetect()
	writeJSON(w, services)
}

func (s *Server) handleFail2banStats(w http.ResponseWriter, r *http.Request) {
	path := "/var/log/fail2ban.log"
	since := time.Now().Add(-24 * time.Hour)
	stats, err := collector.GetFail2banStats(path, since)
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error(), "available": false})
		return
	}
	writeJSON(w, map[string]any{"available": true, "data": stats})
}

func (s *Server) handleUFWStats(w http.ResponseWriter, r *http.Request) {
	path := "/var/log/ufw.log"
	since := time.Now().Add(-24 * time.Hour)
	stats, err := collector.GetUFWStats(path, since)
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error(), "available": false})
		return
	}
	writeJSON(w, map[string]any{"available": true, "data": stats})
}
