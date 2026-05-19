package api

import (
	"os"
	"path/filepath"
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

func TestFileSessionStore_PersistAcrossRestart(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sessions.json")

	store1, err := NewFileSessionStore(path)
	if err != nil {
		t.Fatalf("NewFileSessionStore: %v", err)
	}
	sess := store1.Create(time.Hour)

	// Simulate process restart by constructing a fresh store from the same file.
	store2, err := NewFileSessionStore(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if !store2.Valid(sess.Token) {
		t.Error("session should survive restart")
	}
}

func TestFileSessionStore_FilePermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sessions.json")

	store, err := NewFileSessionStore(path)
	if err != nil {
		t.Fatalf("NewFileSessionStore: %v", err)
	}
	store.Create(time.Hour)

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("file perm = %o, want 0600", perm)
	}
}

func TestFileSessionStore_ExpiredDroppedOnLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sessions.json")

	store1, _ := NewFileSessionStore(path)
	alive := store1.Create(time.Hour)
	dead := store1.Create(time.Hour)

	// Force one session to be expired on disk by mutating ExpiresAt and re-persisting.
	store1.mu.Lock()
	store1.sessions[dead.Token].ExpiresAt = time.Now().Add(-time.Minute)
	store1.persistLocked()
	store1.mu.Unlock()

	store2, err := NewFileSessionStore(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if !store2.Valid(alive.Token) {
		t.Error("alive session should be loaded")
	}
	if store2.Valid(dead.Token) {
		t.Error("expired session should be dropped on load")
	}
}

func TestFileSessionStore_MissingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "does-not-exist.json")

	store, err := NewFileSessionStore(path)
	if err != nil {
		t.Fatalf("missing file should not error: %v", err)
	}
	// Store should be usable and start empty.
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("load should not create the file until first write")
	}
	sess := store.Create(time.Hour)
	if !store.Valid(sess.Token) {
		t.Error("Create should work after missing-file load")
	}
}

func TestFileSessionStore_CorruptFileReturnsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sessions.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatalf("seed corrupt file: %v", err)
	}
	if _, err := NewFileSessionStore(path); err == nil {
		t.Error("corrupt JSON should surface an error to main so it can fall back to in-memory")
	}
}

func TestFileSessionStore_DeletePersists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sessions.json")

	store1, _ := NewFileSessionStore(path)
	sess := store1.Create(time.Hour)
	store1.Delete(sess.Token)

	store2, err := NewFileSessionStore(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if store2.Valid(sess.Token) {
		t.Error("deleted session should not reappear after restart")
	}
}

func TestFileSessionStore_MultipleSessionsCoexist(t *testing.T) {
	// Multi-device login: several sessions for the same user coexist.
	dir := t.TempDir()
	path := filepath.Join(dir, "sessions.json")

	store, _ := NewFileSessionStore(path)
	a := store.Create(time.Hour)
	b := store.Create(time.Hour)
	c := store.Create(time.Hour)

	reloaded, err := NewFileSessionStore(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	for _, tok := range []string{a.Token, b.Token, c.Token} {
		if !reloaded.Valid(tok) {
			t.Errorf("session %s should survive restart (multi-device)", tok[:8])
		}
	}
}
