package api

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// Session represents an authenticated browser session.
type Session struct {
	Token     string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// SessionStore manages in-memory sessions.
type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

func NewSessionStore() *SessionStore {
	return &SessionStore{sessions: make(map[string]*Session)}
}

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
	delete(s.sessions, token)
	s.mu.Unlock()
}

func (s *SessionStore) Cleanup() {
	now := time.Now()
	s.mu.Lock()
	for token, sess := range s.sessions {
		if now.After(sess.ExpiresAt) {
			delete(s.sessions, token)
		}
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
