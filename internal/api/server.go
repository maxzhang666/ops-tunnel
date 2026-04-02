package api

import (
	"context"
	"log/slog"
	"net"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Config struct {
	ListenAddr string
	UIDir      string
	Token      string
}

type Server struct {
	cfg    Config
	router chi.Router
	http   *http.Server
}

func NewServer(cfg Config) *Server {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	s := &Server{
		cfg:    cfg,
		router: r,
	}
	s.registerRoutes()
	return s
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
