package ssh

import (
	"path/filepath"
	"testing"
)

func TestJSONHostKeyStore_AddLookup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "known_hosts.json")
	store := NewJSONHostKeyStore(path)

	key := []byte("test-key-data")
	if err := store.Add("example.com:22", key); err != nil {
		t.Fatalf("Add error: %v", err)
	}

	got, ok := store.Lookup("example.com:22")
	if !ok {
		t.Fatal("expected to find key")
	}
	if !bytesEqual(got, key) {
		t.Error("key mismatch")
	}
}

func TestJSONHostKeyStore_LookupMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "known_hosts.json")
	store := NewJSONHostKeyStore(path)

	_, ok := store.Lookup("nonexistent:22")
	if ok {
		t.Error("expected not found")
	}
}

func TestJSONHostKeyStore_Persistence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "known_hosts.json")

	store1 := NewJSONHostKeyStore(path)
	store1.Add("host:22", []byte("key1"))

	store2 := NewJSONHostKeyStore(path)
	got, ok := store2.Lookup("host:22")
	if !ok {
		t.Fatal("key not persisted")
	}
	if !bytesEqual(got, []byte("key1")) {
		t.Error("persisted key mismatch")
	}
}

func TestNoopHostKeyStore(t *testing.T) {
	store := NewNoopHostKeyStore()
	_, ok := store.Lookup("any:22")
	if ok {
		t.Error("noop store should never find keys")
	}
	if err := store.Add("any:22", []byte("key")); err != nil {
		t.Errorf("noop store Add should not error: %v", err)
	}
}
