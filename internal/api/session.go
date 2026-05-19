package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Session represents an authenticated browser session.
type Session struct {
	Token     string    `json:"token"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// SessionStore manages sessions. When path is non-empty, sessions are
// persisted to a JSON file so they survive process/container restarts.
type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	path     string // "" = in-memory only (used by tests and desktop mode)
}

// NewSessionStore returns an in-memory session store (no persistence).
func NewSessionStore() *SessionStore {
	return &SessionStore{sessions: make(map[string]*Session)}
}

// NewFileSessionStore loads existing sessions from path and persists future
// changes back to it. Missing file is not an error. Expired sessions are
// dropped on load.
func NewFileSessionStore(path string) (*SessionStore, error) {
	s := &SessionStore{sessions: make(map[string]*Session), path: path}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

type sessionFile struct {
	Sessions []*Session `json:"sessions"`
}

func (s *SessionStore) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	var f sessionFile
	if err := json.Unmarshal(data, &f); err != nil {
		return err
	}
	now := time.Now()
	for _, sess := range f.Sessions {
		if sess == nil || sess.Token == "" {
			continue
		}
		if now.After(sess.ExpiresAt) {
			continue
		}
		s.sessions[sess.Token] = sess
	}
	return nil
}

// persistLocked writes the in-memory sessions to disk atomically.
// Caller must hold s.mu. No-op when path is empty. Errors are logged but
// never returned so callers keep working if the disk is temporarily broken.
func (s *SessionStore) persistLocked() {
	if s.path == "" {
		return
	}
	list := make([]*Session, 0, len(s.sessions))
	for _, sess := range s.sessions {
		list = append(list, sess)
	}
	data, err := json.MarshalIndent(sessionFile{Sessions: list}, "", "  ")
	if err != nil {
		slog.Error("session persist marshal failed", "err", err)
		return
	}
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		slog.Error("session persist mkdir failed", "err", err)
		return
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		slog.Error("session persist write failed", "err", err)
		return
	}
	if err := os.Rename(tmp, s.path); err != nil {
		slog.Error("session persist rename failed", "err", err)
	}
}

// Create generates and stores a new session. Existing sessions for the same
// user are NOT invalidated, so multi-device login coexists.
func (s *SessionStore) Create(ttl time.Duration) *Session {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	token := hex.EncodeToString(b)
	now := time.Now()
	sess := &Session{Token: token, CreatedAt: now, ExpiresAt: now.Add(ttl)}
	s.mu.Lock()
	s.sessions[token] = sess
	s.persistLocked()
	s.mu.Unlock()
	return sess
}

func (s *SessionStore) Valid(token string) bool {
	s.mu.RLock()
	sess, ok := s.sessions[token]
	s.mu.RUnlock()
	if !ok {
		return false
	}
	if time.Now().After(sess.ExpiresAt) {
		s.Delete(token)
		return false
	}
	return true
}

func (s *SessionStore) Delete(token string) {
	s.mu.Lock()
	if _, ok := s.sessions[token]; ok {
		delete(s.sessions, token)
		s.persistLocked()
	}
	s.mu.Unlock()
}

func (s *SessionStore) Cleanup() {
	now := time.Now()
	s.mu.Lock()
	changed := false
	for token, sess := range s.sessions {
		if now.After(sess.ExpiresAt) {
			delete(s.sessions, token)
			changed = true
		}
	}
	if changed {
		s.persistLocked()
	}
	s.mu.Unlock()
}

func (s *SessionStore) StartCleanup(interval time.Duration, done <-chan struct{}) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.Cleanup()
			case <-done:
				return
			}
		}
	}()
}
