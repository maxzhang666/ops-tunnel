package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAuthStore_LoadNonExistent(t *testing.T) {
	s := NewAuthStore(filepath.Join(t.TempDir(), "auth.json"))
	auth, err := s.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if auth != nil {
		t.Fatal("expected nil for non-existent file")
	}
}

func TestAuthStore_SaveAndLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth.json")
	s := NewAuthStore(path)

	want := &WebAuth{Username: "admin", PasswordHash: "$2a$10$fake"}
	if err := s.Save(want); err != nil {
		t.Fatalf("save error: %v", err)
	}

	got, err := s.Load()
	if err != nil {
		t.Fatalf("load error: %v", err)
	}
	if got.Username != want.Username || got.PasswordHash != want.PasswordHash {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestAuthStore_SaveCreatesDir(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "auth.json")
	s := NewAuthStore(path)
	if err := s.Save(&WebAuth{Username: "admin", PasswordHash: "hash"}); err != nil {
		t.Fatalf("save should create parent dir: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file should exist: %v", err)
	}
}
