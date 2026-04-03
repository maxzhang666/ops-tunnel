package api

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

func (s *Server) registerRoutes() {
	s.router.Get("/healthz", s.handleHealthz)
	s.router.Get("/ws", s.handleWebSocket)

	s.router.Route("/api/v1", func(r chi.Router) {
		if s.cfg.Token != "" {
			r.Use(TokenAuth(s.cfg.Token))
		}
		r.Use(MaxBodySize(1 << 20))

		r.Route("/ssh-connections", func(r chi.Router) {
			r.Get("/", s.listSSHConnections)
			r.Post("/", s.createSSHConnection)
			r.Get("/{id}", s.getSSHConnection)
			r.Put("/{id}", s.updateSSHConnection)
			r.Patch("/{id}", s.patchSSHConnection)
			r.Delete("/{id}", s.deleteSSHConnection)
			r.Post("/{id}/test", s.testSSHConnection)
		})

		r.Route("/tunnels", func(r chi.Router) {
			r.Get("/", s.listTunnels)
			r.Post("/", s.createTunnel)
			r.Get("/{id}", s.getTunnel)
			r.Put("/{id}", s.updateTunnel)
			r.Patch("/{id}", s.patchTunnel)
			r.Delete("/{id}", s.deleteTunnel)
		})

		r.Post("/tunnels/{id}/start", s.startTunnel)
		r.Post("/tunnels/{id}/stop", s.stopTunnel)
		r.Post("/tunnels/{id}/restart", s.restartTunnel)
		r.Get("/tunnels/{id}/status", s.getTunnelStatus)

		r.Get("/settings", s.getSettings)
		r.Patch("/settings", s.patchSettings)
		r.Get("/version", s.getVersion)
	})

	if s.cfg.UIEmbed != nil {
		s.serveEmbedSPA(s.cfg.UIEmbed)
	} else if s.cfg.UIDir != "" {
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

func (s *Server) serveEmbedSPA(fsys fs.FS) {
	fileServer := http.FileServer(http.FS(fsys))

	s.router.NotFound(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path != "" {
			if f, err := fsys.Open(path); err == nil {
				f.Close()
				fileServer.ServeHTTP(w, r)
				return
			}
		}
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
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
