package ssh

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maxzhang666/ops-tunnel/internal/config"
	gossh "golang.org/x/crypto/ssh"
)

// testHostKey returns a real wire-format host key and its SHA256 fingerprint.
func testHostKey(t *testing.T) ([]byte, string) {
	t.Helper()
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	pk, err := gossh.NewPublicKey(pub)
	if err != nil {
		t.Fatalf("wrap key: %v", err)
	}
	return pk.Marshal(), gossh.FingerprintSHA256(pk)
}

func newTestStore(t *testing.T) (*JSONHostKeyStore, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "known_hosts.json")
	store, err := NewJSONHostKeyStore(path)
	if err != nil {
		t.Fatalf("NewJSONHostKeyStore error: %v", err)
	}
	return store, path
}

func TestJSONHostKeyStore_AddLookup(t *testing.T) {
	store, _ := newTestStore(t)
	key, _ := testHostKey(t)

	if err := store.Add("example.com:22", key); err != nil {
		t.Fatalf("Add error: %v", err)
	}

	got, ok := store.Lookup("example.com:22")
	if !ok {
		t.Fatal("expected to find key")
	}
	if !bytes.Equal(got, key) {
		t.Error("key mismatch")
	}
}

func TestJSONHostKeyStore_LookupMissing(t *testing.T) {
	store, _ := newTestStore(t)
	if _, ok := store.Lookup("nonexistent:22"); ok {
		t.Error("expected not found")
	}
}

func TestJSONHostKeyStore_Persistence(t *testing.T) {
	store1, path := newTestStore(t)
	key, _ := testHostKey(t)
	if err := store1.Add("host:22", key); err != nil {
		t.Fatalf("Add error: %v", err)
	}

	store2, err := NewJSONHostKeyStore(path)
	if err != nil {
		t.Fatalf("reload error: %v", err)
	}
	got, ok := store2.Lookup("host:22")
	if !ok {
		t.Fatal("key not persisted")
	}
	if !bytes.Equal(got, key) {
		t.Error("persisted key mismatch")
	}
}

func TestJSONHostKeyStore_FileMode(t *testing.T) {
	store, path := newTestStore(t)
	key, _ := testHostKey(t)
	if err := store.Add("host:22", key); err != nil {
		t.Fatalf("Add error: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat error: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Errorf("mode = %o, want 600", mode)
	}
}

func TestJSONHostKeyStore_Delete(t *testing.T) {
	store, _ := newTestStore(t)
	key, _ := testHostKey(t)
	store.Add("host:22", key)

	if err := store.Delete("host:22"); err != nil {
		t.Fatalf("Delete error: %v", err)
	}
	if _, ok := store.Lookup("host:22"); ok {
		t.Error("expected key to be gone")
	}
	if err := store.Delete("host:22"); err != nil {
		t.Errorf("Delete of missing key should be a no-op, got %v", err)
	}
}

func TestJSONHostKeyStore_List(t *testing.T) {
	store, _ := newTestStore(t)
	keyB, fpB := testHostKey(t)
	keyA, fpA := testHostKey(t)
	store.Add("b-host:22", keyB)
	store.Add("a-host:22", keyA)

	hosts := store.List()
	if len(hosts) != 2 {
		t.Fatalf("len = %d, want 2", len(hosts))
	}
	if hosts[0].HostPort != "a-host:22" || hosts[1].HostPort != "b-host:22" {
		t.Errorf("not sorted by hostPort: %+v", hosts)
	}
	if hosts[0].Fingerprint != fpA || hosts[1].Fingerprint != fpB {
		t.Error("fingerprint mismatch")
	}
	if hosts[0].KeyType != "ssh-ed25519" {
		t.Errorf("keyType = %q, want ssh-ed25519", hosts[0].KeyType)
	}
}

func TestJSONHostKeyStore_ListUnparseable(t *testing.T) {
	store, _ := newTestStore(t)
	store.Add("bad:22", []byte("not-a-wire-format-key"))

	hosts := store.List()
	if len(hosts) != 1 {
		t.Fatalf("len = %d, want 1", len(hosts))
	}
	if hosts[0].KeyType != "unparseable" {
		t.Errorf("keyType = %q, want unparseable", hosts[0].KeyType)
	}
	if hosts[0].Fingerprint != "" {
		t.Errorf("fingerprint = %q, want empty", hosts[0].Fingerprint)
	}
}

func TestJSONHostKeyStore_Pending(t *testing.T) {
	store, path := newTestStore(t)
	key, _ := testHostKey(t)

	if _, ok := store.Pending("host:22"); ok {
		t.Error("expected no pending key")
	}

	store.SetPending("host:22", key)
	got, ok := store.Pending("host:22")
	if !ok || !bytes.Equal(got, key) {
		t.Error("pending key not returned")
	}

	// Pending keys must never be persisted.
	if data, err := os.ReadFile(path); err == nil && bytes.Contains(data, []byte("host:22")) {
		t.Error("pending key leaked to disk")
	}

	store.ClearPending("host:22")
	if _, ok := store.Pending("host:22"); ok {
		t.Error("expected pending key to be cleared")
	}
}

func TestJSONHostKeyStore_CorruptFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "known_hosts.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatalf("write error: %v", err)
	}
	if _, err := NewJSONHostKeyStore(path); err == nil {
		t.Error("expected error for corrupt store, got nil")
	}
}

func TestJSONHostKeyStore_MissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist.json")
	store, err := NewJSONHostKeyStore(path)
	if err != nil {
		t.Fatalf("missing file should not error, got %v", err)
	}
	if len(store.List()) != 0 {
		t.Error("expected empty store")
	}
}

func TestFingerprint(t *testing.T) {
	key, want := testHostKey(t)
	fp, keyType, err := Fingerprint(key)
	if err != nil {
		t.Fatalf("Fingerprint error: %v", err)
	}
	if fp != want {
		t.Errorf("fingerprint = %q, want %q", fp, want)
	}
	if keyType != "ssh-ed25519" {
		t.Errorf("keyType = %q, want ssh-ed25519", keyType)
	}
	if _, _, err := Fingerprint([]byte("garbage")); err == nil {
		t.Error("expected error for garbage input")
	}
}

func TestHostPort(t *testing.T) {
	got := HostPort(config.Endpoint{Host: "10.0.0.1", Port: 2222})
	if got != "10.0.0.1:2222" {
		t.Errorf("HostPort = %q, want 10.0.0.1:2222", got)
	}
}

func TestNoopHostKeyStore(t *testing.T) {
	store := NewNoopHostKeyStore()
	if _, ok := store.Lookup("any:22"); ok {
		t.Error("noop store should never find keys")
	}
	if err := store.Add("any:22", []byte("key")); err != nil {
		t.Errorf("noop Add should not error: %v", err)
	}
	if err := store.Delete("any:22"); err != nil {
		t.Errorf("noop Delete should not error: %v", err)
	}
	if store.List() != nil {
		t.Error("noop List should return nil")
	}
	store.SetPending("any:22", []byte("key"))
	if _, ok := store.Pending("any:22"); ok {
		t.Error("noop Pending should never find keys")
	}
}

// callbackFor runs a verifier's callback against an offered key and returns the
// verifier so the caller can inspect the recorded mismatch.
func callbackFor(t *testing.T, mode config.HostKeyVerifyMode, store HostKeyStore, offered []byte) (*HostKeyVerifier, error) {
	t.Helper()
	pk, err := gossh.ParsePublicKey(offered)
	if err != nil {
		t.Fatalf("parse offered key: %v", err)
	}
	v := NewHostKeyVerifier(mode, store, "host:22")
	return v, v.Callback()("host:22", nil, pk)
}

func TestVerifier_AcceptNew_TOFU(t *testing.T) {
	store, _ := newTestStore(t)
	key, _ := testHostKey(t)

	v, err := callbackFor(t, config.HostKeyAcceptNew, store, key)
	if err != nil {
		t.Fatalf("first connect should be accepted, got %v", err)
	}
	if v.Mismatch() != nil {
		t.Error("expected no mismatch")
	}
	if _, ok := store.Lookup("host:22"); !ok {
		t.Error("expected key to be trusted on first connect")
	}
}

func TestVerifier_AcceptNew_SameKey(t *testing.T) {
	store, _ := newTestStore(t)
	key, _ := testHostKey(t)
	store.Add("host:22", key)

	v, err := callbackFor(t, config.HostKeyAcceptNew, store, key)
	if err != nil {
		t.Fatalf("matching key should be accepted, got %v", err)
	}
	if v.Mismatch() != nil {
		t.Error("expected no mismatch")
	}
}

func TestVerifier_AcceptNew_Changed(t *testing.T) {
	store, _ := newTestStore(t)
	oldKey, oldFP := testHostKey(t)
	newKey, newFP := testHostKey(t)
	store.Add("host:22", oldKey)

	v, err := callbackFor(t, config.HostKeyAcceptNew, store, newKey)
	if err == nil {
		t.Fatal("changed key must be rejected")
	}

	mm := v.Mismatch()
	if mm == nil {
		t.Fatal("expected recorded mismatch")
	}
	if mm.Reason != HostKeyReasonChanged {
		t.Errorf("reason = %q, want changed", mm.Reason)
	}
	if mm.StoredFP != oldFP {
		t.Errorf("storedFP = %q, want %q", mm.StoredFP, oldFP)
	}
	if mm.OfferedFP != newFP {
		t.Errorf("offeredFP = %q, want %q", mm.OfferedFP, newFP)
	}
	if mm.HostPort != "host:22" {
		t.Errorf("hostPort = %q", mm.HostPort)
	}

	// The trusted key must be untouched, and the offered key parked as pending.
	if got, _ := store.Lookup("host:22"); !bytes.Equal(got, oldKey) {
		t.Error("trusted key must not be overwritten on mismatch")
	}
	pending, ok := store.Pending("host:22")
	if !ok || !bytes.Equal(pending, newKey) {
		t.Error("offered key should be parked as pending")
	}
}

func TestVerifier_Strict_Unknown(t *testing.T) {
	store, _ := newTestStore(t)
	key, fp := testHostKey(t)

	v, err := callbackFor(t, config.HostKeyStrict, store, key)
	if err == nil {
		t.Fatal("strict mode with no trusted key must be rejected")
	}
	mm := v.Mismatch()
	if mm == nil {
		t.Fatal("expected recorded mismatch")
	}
	if mm.Reason != HostKeyReasonUnknown {
		t.Errorf("reason = %q, want unknown", mm.Reason)
	}
	if mm.StoredFP != "" {
		t.Errorf("storedFP = %q, want empty", mm.StoredFP)
	}
	if mm.OfferedFP != fp {
		t.Errorf("offeredFP = %q, want %q", mm.OfferedFP, fp)
	}
	if _, ok := store.Lookup("host:22"); ok {
		t.Error("strict mode must not auto-trust")
	}
	if _, ok := store.Pending("host:22"); !ok {
		t.Error("offered key should be parked as pending")
	}
}

func TestVerifier_Insecure(t *testing.T) {
	store, _ := newTestStore(t)
	oldKey, _ := testHostKey(t)
	newKey, _ := testHostKey(t)
	store.Add("host:22", oldKey)

	v, err := callbackFor(t, config.HostKeyInsecure, store, newKey)
	if err != nil {
		t.Fatalf("insecure mode should accept anything, got %v", err)
	}
	if v.Mismatch() != nil {
		t.Error("insecure mode should record nothing")
	}
}

func TestVerifier_UnknownModeFailsClosed(t *testing.T) {
	store, _ := newTestStore(t)
	key, _ := testHostKey(t)

	if _, err := callbackFor(t, config.HostKeyVerifyMode("typo"), store, key); err == nil {
		t.Fatal("unknown mode must reject, not skip verification")
	}
	if _, ok := store.Lookup("host:22"); ok {
		t.Error("unknown mode must not trust anything")
	}
}

func TestHostKeyError_Message(t *testing.T) {
	e := &HostKeyError{
		Hop: 2,
		Mismatch: &HostKeyMismatch{
			SSHConnName: "bastion",
			HostPort:    "10.0.0.1:22",
			Reason:      HostKeyReasonChanged,
			StoredFP:    "SHA256:old",
			OfferedFP:   "SHA256:new",
		},
	}
	msg := e.Error()
	for _, want := range []string{"hop 2", "bastion", "SHA256:old", "SHA256:new"} {
		if !strings.Contains(msg, want) {
			t.Errorf("message %q missing %q", msg, want)
		}
	}
}
