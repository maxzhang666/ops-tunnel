package config

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

// WebAuth holds admin credentials for Web UI authentication.
type WebAuth struct {
	Username     string `json:"username"`
	PasswordHash string `json:"passwordHash"`
}

// AuthStore persists WebAuth as a JSON file.
type AuthStore struct {
	path string
	mu   sync.RWMutex
}

// NewAuthStore creates an auth store backed by the given file path.
func NewAuthStore(path string) *AuthStore {
	return &AuthStore{path: path}
}

// Load reads auth from disk. Returns nil if file doesn't exist.
func (s *AuthStore) Load() (*WebAuth, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var auth WebAuth
	if err := json.Unmarshal(data, &auth); err != nil {
		return nil, err
	}
	return &auth, nil
}

// Save writes auth to disk atomically.
func (s *AuthStore) Save(auth *WebAuth) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(auth, "", "  ")
	if err != nil {
		return err
	}

	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmpPath, s.path)
}
