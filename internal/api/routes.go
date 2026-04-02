package api

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func (s *Server) registerRoutes() {
	s.router.Get("/healthz", s.handleHealthz)

	if s.cfg.UIDir != "" {
		s.serveSPA(s.cfg.UIDir)
	}
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status": "ok",
		"ts":     time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) serveSPA(dir string) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return
	}

	fileServer := http.FileServer(http.Dir(absDir))

	s.router.NotFound(func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join(absDir, r.URL.Path)
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			fileServer.ServeHTTP(w, r)
			return
		}

		// SPA fallback: serve index.html for non-file routes
		indexPath := filepath.Join(absDir, "index.html")
		if _, err := fs.Stat(os.DirFS(absDir), "index.html"); err == nil {
			http.ServeFile(w, r, indexPath)
			return
		}

		http.NotFound(w, r)
	})
}
