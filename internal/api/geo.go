package api

import (
	"net/http"
	"time"

	"github.com/yourusername/logvance/internal/geoip"
	"github.com/yourusername/logvance/internal/storage"
)

func (s *Server) handleGeoMap(w http.ResponseWriter, r *http.Request) {
	if s.geo == nil {
		writeJSON(w, []storage.GeoEntry{})
		return
	}

	since := time.Now().Add(-24 * time.Hour)
	ips, err := s.db.GetTopIPs(since, 500)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	var result []storage.GeoEntry
	seen := map[string]bool{}

	for _, ip := range ips {
		loc := s.geo.Lookup(ip.Key)
		if loc == nil {
			continue
		}
		key := loc.CountryCode
		if seen[key] {
			// Aggregate by country
			for i, e := range result {
				if e.CountryCode == key {
					result[i].Count += ip.Count
					break
				}
			}
			continue
		}
		seen[key] = true
		result = append(result, storage.GeoEntry{
			IP:          ip.Key,
			Count:       ip.Count,
			Country:     loc.Country,
			CountryCode: loc.CountryCode,
			City:        loc.City,
			Lat:         loc.Latitude,
			Lon:         loc.Longitude,
		})
	}

	writeJSON(w, result)
}

// SetGeo sets the GeoIP resolver
func (s *Server) SetGeo(g *geoip.Resolver) {
	s.geo = g
}

// GetHub returns the WebSocket hub
func (s *Server) GetHub() *Hub {
	return s.hub
}
