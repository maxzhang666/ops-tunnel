package api

import (
	"testing"
	"time"
)

func TestSessionStore_CreateAndValid(t *testing.T) {
	ss := NewSessionStore()
	s := ss.Create(time.Hour)
	if len(s.Token) != 64 {
		t.Errorf("token length = %d, want 64 hex chars", len(s.Token))
	}
	if !ss.Valid(s.Token) {
		t.Error("session should be valid")
	}
}

func TestSessionStore_ExpiredSession(t *testing.T) {
	ss := NewSessionStore()
	s := ss.Create(-time.Second)
	if ss.Valid(s.Token) {
		t.Error("expired session should be invalid")
	}
}

func TestSessionStore_Delete(t *testing.T) {
	ss := NewSessionStore()
	s := ss.Create(time.Hour)
	ss.Delete(s.Token)
	if ss.Valid(s.Token) {
		t.Error("deleted session should be invalid")
	}
}

func TestSessionStore_InvalidToken(t *testing.T) {
	ss := NewSessionStore()
	if ss.Valid("nonexistent") {
		t.Error("unknown token should be invalid")
	}
}

func TestSessionStore_Cleanup(t *testing.T) {
	ss := NewSessionStore()
	ss.Create(-time.Second)
	ss.Create(-time.Second)
	alive := ss.Create(time.Hour)
	ss.Cleanup()
	if !ss.Valid(alive.Token) {
		t.Error("alive session should survive cleanup")
	}
	ss.mu.RLock()
	count := len(ss.sessions)
	ss.mu.RUnlock()
	if count != 1 {
		t.Errorf("sessions count = %d, want 1 after cleanup", count)
	}
}
