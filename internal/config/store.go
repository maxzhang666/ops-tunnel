package config

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

// Store defines config persistence operations.
type Store interface {
	Load(ctx context.Context) (*Config, error)
	Save(ctx context.Context, cfg *Config) error
}

// FileStore persists config as a JSON file with atomic writes.
type FileStore struct {
	path string
	mu   sync.RWMutex
}

// NewFileStore creates a store backed by the given file path.
func NewFileStore(path string) *FileStore {
	return &FileStore{path: path}
}

// Load reads config from disk. Returns empty config if file doesn't exist.
func (s *FileStore) Load(_ context.Context) (*Config, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return NewConfig(), nil
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	if cfg.SSHConnections == nil {
		cfg.SSHConnections = []SSHConnection{}
	}
	if cfg.Tunnels == nil {
		cfg.Tunnels = []Tunnel{}
	}

	migrateEnabledToAutoStart(data, &cfg)
	ApplyConfigDefaults(&cfg)
	return &cfg, nil
}

// migrateEnabledToAutoStart reads the legacy "enabled" field from raw JSON
// and sets policy.autoStart=true for tunnels that had enabled=true.
func migrateEnabledToAutoStart(data []byte, cfg *Config) {
	var raw struct {
		Tunnels []struct {
			ID      string `json:"id"`
			Enabled bool   `json:"enabled"`
		} `json:"tunnels"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return
	}
	for _, rt := range raw.Tunnels {
		if !rt.Enabled {
			continue
		}
		for i := range cfg.Tunnels {
			if cfg.Tunnels[i].ID == rt.ID && !cfg.Tunnels[i].Policy.AutoStart {
				cfg.Tunnels[i].Policy.AutoStart = true
			}
		}
	}
}

// Save writes config to disk atomically: write temp -> fsync -> rename.
func (s *FileStore) Save(_ context.Context, cfg *Config) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	tmpPath := s.path + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return err
	}

	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return err
	}

	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return err
	}

	if err := f.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}

	return os.Rename(tmpPath, s.path)
}
