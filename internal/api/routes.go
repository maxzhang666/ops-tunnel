package api

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/go-chi/chi/v5"
)

func (s *Server) registerRoutes() {
	s.router.Get("/healthz", s.handleHealthz)

	s.router.Route("/api/v1/ssh-connections", func(r chi.Router) {
		r.Get("/", s.listSSHConnections)
		r.Post("/", s.createSSHConnection)
		r.Get("/{id}", s.getSSHConnection)
		r.Put("/{id}", s.updateSSHConnection)
		r.Delete("/{id}", s.deleteSSHConnection)
		r.Post("/{id}/test", s.testSSHConnection)
	})

	s.router.Route("/api/v1/tunnels", func(r chi.Router) {
		r.Get("/", s.listTunnels)
		r.Post("/", s.createTunnel)
		r.Get("/{id}", s.getTunnel)
		r.Put("/{id}", s.updateTunnel)
		r.Delete("/{id}", s.deleteTunnel)
	})

	// Tunnel control
	s.router.Post("/api/v1/tunnels/{id}/start", s.startTunnel)
	s.router.Post("/api/v1/tunnels/{id}/stop", s.stopTunnel)
	s.router.Post("/api/v1/tunnels/{id}/restart", s.restartTunnel)
	s.router.Get("/api/v1/tunnels/{id}/status", s.getTunnelStatus)

	// WebSocket
	s.router.Get("/ws", s.handleWebSocket)

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

		indexPath := filepath.Join(absDir, "index.html")
		if _, err := fs.Stat(os.DirFS(absDir), "index.html"); err == nil {
			http.ServeFile(w, r, indexPath)
			return
		}

		http.NotFound(w, r)
	})
}
