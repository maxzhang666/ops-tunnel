package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/maxzhang666/ops-tunnel/internal/api"
	"github.com/maxzhang666/ops-tunnel/internal/config"
	"github.com/maxzhang666/ops-tunnel/internal/engine"
	tunnelssh "github.com/maxzhang666/ops-tunnel/internal/ssh"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

var version = "dev"

func main() {
	dataDir := flag.String("data-dir", "./data", "data directory")
	flag.Parse()

	logLevel := new(slog.LevelVar)
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel})))

	if err := os.MkdirAll(*dataDir, 0o755); err != nil {
		slog.Error("failed to create data dir", "err", err)
		os.Exit(1)
	}

	store := config.NewFileStore(filepath.Join(*dataDir, "config.json"))

	ctx := context.Background()
	cfg, err := store.Load(ctx)
	if err != nil {
		slog.Error("failed to load config", "err", err)
		os.Exit(1)
	}
	if err := store.Save(ctx, cfg); err != nil {
		slog.Error("failed to save initial config", "err", err)
		os.Exit(1)
	}

	bus := engine.NewEventBus()
	hostKeys := tunnelssh.NewJSONHostKeyStore(filepath.Join(*dataDir, "known_hosts.json"))
	eng := engine.NewEngine(cfg, bus, hostKeys)

	// Find random available port for the API server
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		slog.Error("failed to find available port", "err", err)
		os.Exit(1)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	uiFS, err := frontendFS()
	if err != nil {
		slog.Error("failed to load frontend assets", "err", err)
		os.Exit(1)
	}

	srv := api.NewServer(api.ServerConfig{
		ListenAddr:  fmt.Sprintf("127.0.0.1:%d", port),
		UIEmbed:     uiFS,
		Version:     version,
		Mode:        "desktop",
		LogLevelVar: logLevel,
	}, store, cfg, eng, bus, hostKeys)

	go func() {
		if err := srv.Run(ctx); err != nil {
			slog.Error("server error", "err", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	// AutoStart tunnels
	for _, t := range cfg.Tunnels {
		if t.Policy.AutoStart {
			go eng.StartTunnel(context.Background(), t.ID)
		}
	}

	app := NewApp(cfg, store, eng, bus)

	slog.Info("desktop starting", "api", fmt.Sprintf("http://localhost:%d", port))

	if err := wails.Run(&options.App{
		Title:            "OpsTunnel",
		Width:            1060,
		Height:           832,
		DisableResize:    true,
		StartHidden:      false,
		AssetServer: &assetserver.Options{
			Assets:  frontendAssets,
			Handler: srv.Handler(),
		},
		OnStartup:     app.Startup,
		OnBeforeClose: app.BeforeClose,
		OnShutdown: func(ctx context.Context) {
			if app.trayEnd != nil {
				app.trayEnd()
			}
			shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			eng.Shutdown(shutCtx)
			srv.Shutdown(shutCtx)
		},
	}); err != nil {
		slog.Error("wails error", "err", err)
		os.Exit(1)
	}
}
