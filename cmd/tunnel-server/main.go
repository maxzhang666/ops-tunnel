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

	"golang.org/x/crypto/bcrypt"

	"github.com/maxzhang666/ops-tunnel/internal/api"
	"github.com/maxzhang666/ops-tunnel/internal/config"
	"github.com/maxzhang666/ops-tunnel/internal/engine"
	tunnelssh "github.com/maxzhang666/ops-tunnel/internal/ssh"
	"github.com/maxzhang666/ops-tunnel/internal/traffic"
)

var version = "dev"

func main() {
	listenFlag := flag.String("listen", "127.0.0.1:9876", "HTTP listen address")
	dataDirFlag := flag.String("data-dir", "./data", "data directory")
	uiDirFlag := flag.String("ui-dir", "", "static UI files directory")
	tokenFlag := flag.String("token", "", "bearer token for API auth")
	flag.Parse()

	explicit := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) { explicit[f.Name] = true })

	resolve := func(flagName, flagVal, envKey string) string {
		if explicit[flagName] {
			return flagVal
		}
		if v := os.Getenv(envKey); v != "" {
			return v
		}
		return flagVal
	}

	listen := resolve("listen", *listenFlag, "TUNNEL_LISTEN")
	dataDir := resolve("data-dir", *dataDirFlag, "TUNNEL_DATA_DIR")
	uiDir := resolve("ui-dir", *uiDirFlag, "TUNNEL_UI_DIR")
	token := resolve("token", *tokenFlag, "TUNNEL_TOKEN")

	logLevel := new(slog.LevelVar)
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel})))

	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		slog.Error("failed to create data dir", "path", dataDir, "err", err)
		os.Exit(1)
	}

	store := config.NewFileStore(filepath.Join(dataDir, "config.json"))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := store.Load(ctx)
	if err != nil {
		slog.Error("failed to load config", "err", err)
		os.Exit(1)
	}

	// Save on first run to create the config file
	if err := store.Save(ctx, cfg); err != nil {
		slog.Error("failed to save initial config", "err", err)
		os.Exit(1)
	}

	slog.Info("config loaded",
		"sshConnections", len(cfg.SSHConnections),
		"tunnels", len(cfg.Tunnels),
	)

	authStore := config.NewAuthStore(filepath.Join(dataDir, "auth.json"))

	if adminPass := os.Getenv("TUNNEL_ADMIN_PASSWORD"); adminPass != "" {
		hash, hashErr := bcrypt.GenerateFromPassword([]byte(adminPass), bcrypt.DefaultCost)
		if hashErr != nil {
			slog.Error("failed to hash admin password", "err", hashErr)
			os.Exit(1)
		}
		username := os.Getenv("TUNNEL_ADMIN_USERNAME")
		if username == "" {
			username = "admin"
		}
		if saveErr := authStore.Save(&config.WebAuth{
			Username:     username,
			PasswordHash: string(hash),
		}); saveErr != nil {
			slog.Error("failed to save auth config", "err", saveErr)
			os.Exit(1)
		}
		slog.Info("admin credentials updated from environment")
	}

	webAuth, authLoadErr := authStore.Load()
	if authLoadErr != nil {
		slog.Error("failed to load auth config", "err", authLoadErr)
		os.Exit(1)
	}
	if webAuth != nil {
		slog.Info("web authentication enabled", "username", webAuth.Username)
	}

	bus := engine.NewEventBus()
	hostKeys := tunnelssh.NewJSONHostKeyStore(filepath.Join(dataDir, "known_hosts.json"))
	eng := engine.NewEngine(cfg, bus, hostKeys)

	trafficStore, err := traffic.NewStore(filepath.Join(dataDir, "traffic.db"))
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

	serverCfg := api.ServerConfig{
		ListenAddr:  listen,
		UIDir:       uiDir,
		Token:       token,
		Version:     version,
		Mode:        "server",
		LogLevelVar: logLevel,
		Sampler:     sampler,
		TrafficDB:   trafficStore,
		WebAuth:     webAuth,
	}
	if uiDir == "" {
		if uiFS, err := frontendFS(); err == nil {
			serverCfg.UIEmbed = uiFS
		}
	}

	srv := api.NewServer(serverCfg, store, cfg, eng, bus, hostKeys)

	cleanupDone := make(chan struct{})
	defer close(cleanupDone)
	srv.StartSessionCleanup(10*time.Minute, cleanupDone)

	// Auto-start tunnels with policy.autoStart enabled
	for _, t := range cfg.Tunnels {
		if t.Policy.AutoStart {
			go eng.StartTunnel(context.Background(), t.ID)
		}
	}

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

	if err := eng.Shutdown(shutCtx); err != nil {
		slog.Error("engine shutdown error", "err", err)
	}
	if err := srv.Shutdown(shutCtx); err != nil {
		slog.Error("shutdown error", "err", err)
	}
	slog.Info("server stopped")
}
