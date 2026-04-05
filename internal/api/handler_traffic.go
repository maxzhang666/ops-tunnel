package api

import (
	"net/http"
	"time"

	"github.com/maxzhang666/ops-tunnel/internal/traffic"
)

func (s *Server) getRealtimeTraffic(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Sampler == nil {
		writeJSON(w, http.StatusOK, map[string]any{"samples": []any{}, "interval": 1})
		return
	}
	samples := s.cfg.Sampler.GetRealtime()
	writeJSON(w, http.StatusOK, map[string]any{
		"samples":  samples,
		"interval": 1,
	})
}

func (s *Server) getTrafficHistory(w http.ResponseWriter, r *http.Request) {
	if s.cfg.TrafficDB == nil {
		writeJSON(w, http.StatusOK, map[string]any{"series": []any{}})
		return
	}

	rangeStr := r.URL.Query().Get("range")
	stepStr := r.URL.Query().Get("step")

	rangeDur := parseDuration(rangeStr, 24*time.Hour)
	stepDur := parseDuration(stepStr, 5*time.Minute)

	now := time.Now().UTC()
	from := now.Add(-rangeDur)

	series, err := s.cfg.TrafficDB.Query(from, now, stepDur)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	if series == nil {
		series = []traffic.Sample{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"series": series})
}

func parseDuration(s string, fallback time.Duration) time.Duration {
	switch s {
	case "1h":
		return time.Hour
	case "6h":
		return 6 * time.Hour
	case "24h":
		return 24 * time.Hour
	case "7d":
		return 7 * 24 * time.Hour
	case "1m":
		return time.Minute
	case "5m":
		return 5 * time.Minute
	case "1hr":
		return time.Hour
	default:
		return fallback
	}
}
