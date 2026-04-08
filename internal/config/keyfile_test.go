package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadOrCreateKeyfile_CreatesNew(t *testing.T) {
	dir := t.TempDir()
	key, err := LoadOrCreateKeyfile(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(key) != keyfileSize {
		t.Fatalf("key length = %d, want %d", len(key), keyfileSize)
	}

	// Verify file exists with correct permissions
	info, err := os.Stat(filepath.Join(dir, keyfileName))
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("keyfile permissions = %o, want 0600", perm)
	}
}

func TestLoadOrCreateKeyfile_ReadsExisting(t *testing.T) {
	dir := t.TempDir()
	key1, _ := LoadOrCreateKeyfile(dir)
	key2, _ := LoadOrCreateKeyfile(dir)

	if string(key1) != string(key2) {
		t.Error("second call should return same key")
	}
}

func TestLoadOrCreateKeyfile_BadSize(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, keyfileName), make([]byte, 16), 0o600)

	_, err := LoadOrCreateKeyfile(dir)
	if err == nil {
		t.Error("should reject keyfile with wrong size")
	}
}
