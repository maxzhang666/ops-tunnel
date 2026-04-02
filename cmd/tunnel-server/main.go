package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/maxzhang666/ops-tunnel/internal/api"
	"github.com/maxzhang666/ops-tunnel/internal/config"
)

func main() {
	listen := flag.String("listen", "127.0.0.1:8080", "HTTP listen address")
	dataDir := flag.String("data-dir", "./data", "data directory")
	uiDir := flag.String("ui-dir", "", "static UI files directory")
	token := flag.String("token", "", "bearer token for API auth")
	flag.Parse()

	if err := os.MkdirAll(*dataDir, 0o755); err != nil {
		slog.Error("failed to create data dir", "path", *dataDir, "err", err)
		os.Exit(1)
	}

	store := config.NewFileStore(filepath.Join(*dataDir, "config.json"))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := store.Load(ctx)
	if err != nil {
		slog.Error("failed to load config", "err", err)
		os.Exit(1)
	}

	srv := api.NewServer(api.ServerConfig{
		ListenAddr: *listen,
		UIDir:      *uiDir,
		Token:      *token,
	}, store, cfg)

	go func() {
		if err := srv.Run(ctx); err != nil {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down...")

	shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutCtx); err != nil {
		slog.Error("shutdown error", "err", err)
	}
	slog.Info("server stopped")
}
