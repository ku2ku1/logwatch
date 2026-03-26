package api

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/yourusername/logvance/internal/reports"
)

func (s *Server) handleExportJSON(w http.ResponseWriter, r *http.Request) {
	since := time.Now().Add(-24 * time.Hour)
	rangeParam := r.URL.Query().Get("range")
	switch rangeParam {
	case "7d":
		since = time.Now().Add(-7 * 24 * time.Hour)
	case "30d":
		since = time.Now().Add(-30 * 24 * time.Hour)
	case "all":
		since = time.Time{}
	}

	stats, _ := s.db.GetStats(since)
	paths, _ := s.db.GetTopPaths(since, 100)
	ips, _ := s.db.GetTopIPs(since, 100)
	codes, _ := s.db.GetStatusCodes(since)
	secStats, _ := s.db.GetSecurityStatsFixed()
	threats, _ := s.db.GetRecentThreats(1000)

	export := map[string]any{
		"generated_at":    time.Now().Format(time.RFC3339),
		"range":           rangeParam,
		"stats":           stats,
		"top_paths":       paths,
		"top_ips":         ips,
		"status_codes":    codes,
		"security_stats":  secStats,
		"recent_threats":  threats,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="logvance-export-%s.json"`, time.Now().Format("2006-01-02")))
	json.NewEncoder(w).Encode(export)
}

func (s *Server) handleExportCSV(w http.ResponseWriter, r *http.Request) {
	since := time.Now().Add(-24 * time.Hour)
	rangeParam := r.URL.Query().Get("range")
	dataType := r.URL.Query().Get("type") // paths, ips, threats

	switch rangeParam {
	case "7d":
		since = time.Now().Add(-7 * 24 * time.Hour)
	case "30d":
		since = time.Now().Add(-30 * 24 * time.Hour)
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="logvance-%s-%s.csv"`, dataType, time.Now().Format("2006-01-02")))

	cw := csv.NewWriter(w)
	defer cw.Flush()

	switch dataType {
	case "ips":
		cw.Write([]string{"rank", "ip", "requests"})
		data, _ := s.db.GetTopIPs(since, 1000)
		for i, d := range data {
			cw.Write([]string{fmt.Sprintf("%d", i+1), d.Key, fmt.Sprintf("%d", d.Count)})
		}
	case "threats":
		cw.Write([]string{"timestamp", "ip", "path", "threat_type", "severity", "score", "description"})
		threats, _ := s.db.GetRecentThreats(1000)
		for _, t := range threats {
			cw.Write([]string{
				fmt.Sprintf("%v", t.Timestamp),
				t.IP, t.Path, t.ThreatType,
				t.Severity, fmt.Sprintf("%d", t.Score),
				t.Description,
			})
		}
	default: // paths
		cw.Write([]string{"rank", "path", "requests"})
		data, _ := s.db.GetTopPaths(since, 1000)
		for i, d := range data {
			cw.Write([]string{fmt.Sprintf("%d", i+1), d.Key, fmt.Sprintf("%d", d.Count)})
		}
	}
}

func (s *Server) handleExportPDF(w http.ResponseWriter, r *http.Request) {
	since := time.Now().Add(-24 * time.Hour)
	rangeParam := r.URL.Query().Get("range")
	switch rangeParam {
	case "7d":
		since = time.Now().Add(-7 * 24 * time.Hour)
	case "30d":
		since = time.Now().Add(-30 * 24 * time.Hour)
	case "all":
		since = time.Time{}
	}

	stats, _ := s.db.GetStats(since)
	paths, _ := s.db.GetTopPaths(since, 10)
	ips, _ := s.db.GetTopIPs(since, 10)
	threats, _ := s.db.GetRecentThreats(20)

	filename := fmt.Sprintf("/tmp/logvance-report-%s.pdf", time.Now().Format("20060102-150405"))
	if err := reports.GeneratePDF(stats, paths, ips, threats, filename); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="logvance-report-%s.pdf"`, time.Now().Format("2006-01-02")))
	http.ServeFile(w, r, filename)
}
