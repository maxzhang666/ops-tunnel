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
	"github.com/maxzhang666/ops-tunnel/internal/traffic"
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

	trafficStore, err := traffic.NewStore(filepath.Join(*dataDir, "traffic.db"))
	if err != nil {
		slog.Error("failed to open traffic db", "err", err)
		os.Exit(1)
	}
	defer trafficStore.Close()
	trafficStore.Prune(30 * 24 * time.Hour)

	sampler := engine.NewTrafficSampler(eng, trafficStore)
	samplerCtx, samplerCancel := context.WithCancel(context.Background())
	defer samplerCancel()
	go sampler.Run(samplerCtx)

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
		WsPort:      port,
		LogLevelVar: logLevel,
		Sampler:     sampler,
		TrafficDB:   trafficStore,
	}, store, cfg, eng, bus, hostKeys)

	go func() {
		if err := srv.Run(ctx); err != nil {
			slog.Error("server error", "err", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	// Auto-start tunnels with policy.autoStart enabled
	for _, t := range cfg.Tunnels {
		if t.Policy.AutoStart {
			go eng.StartTunnel(context.Background(), t.ID)
		}
	}

	InitI18n(cfg)
	app := NewApp(cfg, store, eng, bus)

	slog.Info("desktop starting", "api", fmt.Sprintf("http://localhost:%d", port))

	if err := wails.Run(&options.App{
		Title:            "OpsTunnel",
		Width:            1060,
		Height:           832,
		DisableResize:    true,
		StartHidden:      false,
		Bind: []interface{}{app},
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
			// Cleanup runs in background so the window closes instantly
			go func() {
				shutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				done := make(chan struct{})
				go func() {
					eng.Shutdown(shutCtx)
					close(done)
				}()
				srv.Shutdown(shutCtx)
				<-done
			}()
		},
	}); err != nil {
		slog.Error("wails error", "err", err)
		os.Exit(1)
	}
}
