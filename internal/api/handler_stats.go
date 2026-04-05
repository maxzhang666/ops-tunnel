package api

import (
	"net/http"
	"runtime"
)

func (s *Server) getStats(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	writeJSON(w, http.StatusOK, map[string]any{
		"memAlloc":   m.Alloc,
		"memSys":     m.Sys,
		"goroutines": runtime.NumGoroutine(),
	})
}
