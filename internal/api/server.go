package api

import (
	"context"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/maxzhang666/ops-tunnel/internal/config"
	"github.com/maxzhang666/ops-tunnel/internal/engine"
	tunnelssh "github.com/maxzhang666/ops-tunnel/internal/ssh"
)

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	ListenAddr string
	UIDir      string
	UIEmbed    fs.FS
	Token      string
}

// Server is the HTTP API server.
type Server struct {
	cfg      ServerConfig
	store    config.Store
	eng      engine.Engine
	hostKeys tunnelssh.HostKeyStore
	mu       sync.RWMutex
	data     *config.Config
	router   chi.Router
	http     *http.Server
}

// NewServer creates an API server with the given config store.
func NewServer(cfg ServerConfig, store config.Store, data *config.Config, eng engine.Engine, hostKeys tunnelssh.HostKeyStore) *Server {
	r := chi.NewRouter()
	r.Use(SecurityHeaders)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(CORS)

	s := &Server{
		cfg:      cfg,
		store:    store,
		eng:      eng,
		hostKeys: hostKeys,
		data:     data,
		router:   r,
	}
	s.registerRoutes()
	return s
}

// saveConfig validates and persists the current in-memory config.
// Caller must hold s.mu write lock.
func (s *Server) saveConfig(ctx context.Context) (*config.ValidationResult, error) {
	vr := config.ValidateConfig(s.data)
	if vr.HasErrors() {
		return vr, nil
	}
	if err := s.store.Save(ctx, s.data); err != nil {
		return nil, err
	}
	return vr, nil
}

// Handler returns the HTTP handler for use by external servers (e.g., Wails AssetServer).
func (s *Server) Handler() http.Handler {
	return s.router
}

func (s *Server) Run(ctx context.Context) error {
	s.http = &http.Server{
		Addr:    s.cfg.ListenAddr,
		Handler: s.router,
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
	}

	slog.Info("server starting", "addr", s.cfg.ListenAddr)
	if err := s.http.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.http != nil {
		return s.http.Shutdown(ctx)
	}
	return nil
}
