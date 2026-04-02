package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFileStore_LoadEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	store := NewFileStore(path)

	cfg, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if cfg.Version != 1 {
		t.Errorf("Version = %d, want 1", cfg.Version)
	}
	if len(cfg.SSHConnections) != 0 {
		t.Errorf("SSHConnections should be empty, got %d", len(cfg.SSHConnections))
	}
	if len(cfg.Tunnels) != 0 {
		t.Errorf("Tunnels should be empty, got %d", len(cfg.Tunnels))
	}
}

func TestFileStore_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	store := NewFileStore(path)
	ctx := context.Background()

	cfg := NewConfig()
	cfg.SSHConnections = append(cfg.SSHConnections, SSHConnection{
		ID:       "ssh-1",
		Name:     "test-conn",
		Endpoint: Endpoint{Host: "1.2.3.4", Port: 22},
		Auth:     Auth{Type: AuthPassword, Username: "user", Password: "secret"},
	})

	if err := store.Save(ctx, cfg); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("config file not created: %v", err)
	}

	loaded, err := store.Load(ctx)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if len(loaded.SSHConnections) != 1 {
		t.Fatalf("expected 1 SSH connection, got %d", len(loaded.SSHConnections))
	}
	if loaded.SSHConnections[0].Auth.Password != "secret" {
		t.Error("password not persisted correctly")
	}
	if loaded.SSHConnections[0].DialTimeoutMs != 10000 {
		t.Errorf("defaults not applied: DialTimeoutMs = %d", loaded.SSHConnections[0].DialTimeoutMs)
	}
}

func TestFileStore_AtomicWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	store := NewFileStore(path)
	ctx := context.Background()

	cfg := NewConfig()
	if err := store.Save(ctx, cfg); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	tmpPath := path + ".tmp"
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Error("temp file should not exist after successful save")
	}
}
