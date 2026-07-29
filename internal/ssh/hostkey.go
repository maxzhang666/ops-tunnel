package ssh

import (
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/maxzhang666/ops-tunnel/internal/config"
	gossh "golang.org/x/crypto/ssh"
)

// KnownHost describes one trusted entry, derived from its stored wire bytes.
type KnownHost struct {
	HostPort    string `json:"hostPort"`
	Fingerprint string `json:"fingerprint"`
	KeyType     string `json:"keyType"`
}

// HostKeyStore persists trusted host keys and tracks keys a server offered
// that have not been confirmed by the user.
type HostKeyStore interface {
	Lookup(hostport string) ([]byte, bool)
	Add(hostport string, key []byte) error
	Delete(hostport string) error
	List() []KnownHost

	// SetPending records a key that failed verification. In-memory only —
	// unconfirmed key material is never persisted.
	SetPending(hostport string, key []byte)
	Pending(hostport string) ([]byte, bool)
	ClearPending(hostport string)
}

// HostPort formats an endpoint as the host:port key used by the host key store.
func HostPort(e config.Endpoint) string {
	return fmt.Sprintf("%s:%d", e.Host, e.Port)
}

// Fingerprint returns the SHA256 fingerprint and algorithm name of a
// wire-format host key.
func Fingerprint(marshaled []byte) (string, string, error) {
	pk, err := gossh.ParsePublicKey(marshaled)
	if err != nil {
		return "", "", err
	}
	return gossh.FingerprintSHA256(pk), pk.Type(), nil
}

// JSONHostKeyStore stores trusted host keys in a JSON file.
type JSONHostKeyStore struct {
	path string
	mu   sync.RWMutex
	keys map[string]string

	pendingMu sync.RWMutex
	pending   map[string][]byte
}

// NewJSONHostKeyStore loads a host key store, or creates an empty one when the
// file does not exist. A corrupt file is an error rather than a silent reset:
// starting with an empty store would re-trust every host on first connect.
func NewJSONHostKeyStore(path string) (*JSONHostKeyStore, error) {
	s := &JSONHostKeyStore{
		path:    path,
		keys:    make(map[string]string),
		pending: make(map[string][]byte),
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *JSONHostKeyStore) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read known hosts %s: %w", s.path, err)
	}
	if len(data) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, &s.keys); err != nil {
		return fmt.Errorf("parse known hosts %s: %w", s.path, err)
	}
	// Tighten permissions on stores written by earlier versions (0o644).
	os.Chmod(s.path, 0o600)
	return nil
}

func (s *JSONHostKeyStore) save() error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s.keys, "", "  ")
	if err != nil {
		return err
	}
	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmpPath, s.path)
}

func (s *JSONHostKeyStore) Lookup(hostport string) ([]byte, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	encoded, ok := s.keys[hostport]
	if !ok {
		return nil, false
	}
	key, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, false
	}
	return key, true
}

func (s *JSONHostKeyStore) Add(hostport string, key []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.keys[hostport] = base64.StdEncoding.EncodeToString(key)
	return s.save()
}

// Delete removes a trusted entry. Deleting a missing entry is not an error.
func (s *JSONHostKeyStore) Delete(hostport string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.keys[hostport]; !ok {
		return nil
	}
	delete(s.keys, hostport)
	return s.save()
}

// List returns all trusted entries sorted by host:port. Entries whose stored
// bytes cannot be parsed are reported with keyType "unparseable" rather than
// omitted, so they stay visible and revocable.
func (s *JSONHostKeyStore) List() []KnownHost {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]KnownHost, 0, len(s.keys))
	for hostport, encoded := range s.keys {
		kh := KnownHost{HostPort: hostport, KeyType: "unparseable"}
		if raw, err := base64.StdEncoding.DecodeString(encoded); err == nil {
			if fp, keyType, err := Fingerprint(raw); err == nil {
				kh.Fingerprint = fp
				kh.KeyType = keyType
			}
		}
		out = append(out, kh)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].HostPort < out[j].HostPort })
	return out
}

func (s *JSONHostKeyStore) SetPending(hostport string, key []byte) {
	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()
	s.pending[hostport] = key
}

func (s *JSONHostKeyStore) Pending(hostport string) ([]byte, bool) {
	s.pendingMu.RLock()
	defer s.pendingMu.RUnlock()
	key, ok := s.pending[hostport]
	return key, ok
}

func (s *JSONHostKeyStore) ClearPending(hostport string) {
	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()
	delete(s.pending, hostport)
}

// Reasons a host key verification can fail.
const (
	HostKeyReasonChanged = "changed" // a key was trusted and the server offered a different one
	HostKeyReasonUnknown = "unknown" // strict mode with no trusted key on file
)

// HostKeyMismatch is the structured detail of a host key verification failure.
type HostKeyMismatch struct {
	SSHConnID     string `json:"sshConnId,omitempty"`
	SSHConnName   string `json:"sshConnName,omitempty"`
	HostPort      string `json:"hostPort"`
	Reason        string `json:"reason"`
	StoredFP      string `json:"storedFingerprint,omitempty"`
	StoredKeyType string `json:"storedKeyType,omitempty"`
	OfferedFP     string `json:"offeredFingerprint"`
	OfferedType   string `json:"offeredKeyType"`
}

func (m *HostKeyMismatch) String() string {
	if m.Reason == HostKeyReasonUnknown {
		return fmt.Sprintf("no trusted host key for %s; server offered %s (%s)",
			m.HostPort, m.OfferedFP, m.OfferedType)
	}
	return fmt.Sprintf("host key changed for %s: trusted %s (%s), server offered %s (%s)",
		m.HostPort, m.StoredFP, m.StoredKeyType, m.OfferedFP, m.OfferedType)
}

// HostKeyError is returned when a chain hop fails host key verification.
// Callers extract it with errors.As to reach the structured detail.
type HostKeyError struct {
	Hop      int
	Mismatch *HostKeyMismatch
}

func (e *HostKeyError) Error() string {
	name := e.Mismatch.SSHConnName
	if name == "" {
		name = e.Mismatch.HostPort
	}
	return fmt.Sprintf("hop %d (%s): %s", e.Hop, name, e.Mismatch.String())
}

// HostKeyVerifier builds an ssh.HostKeyCallback and records the outcome of the
// handshake it is used for. One verifier per handshake attempt.
//
// The recorded mismatch — not the returned error — is the source of structured
// detail: x/crypto returns the callback's error through its own wrapping, so
// relying on errors.As across that boundary would be fragile.
type HostKeyVerifier struct {
	mode     config.HostKeyVerifyMode
	store    HostKeyStore
	hostport string

	mu       sync.Mutex
	mismatch *HostKeyMismatch
}

func NewHostKeyVerifier(mode config.HostKeyVerifyMode, store HostKeyStore, hostport string) *HostKeyVerifier {
	return &HostKeyVerifier{mode: mode, store: store, hostport: hostport}
}

// Mismatch returns the recorded verification failure, or nil when verification
// passed or the handshake failed for an unrelated reason.
func (v *HostKeyVerifier) Mismatch() *HostKeyMismatch {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.mismatch
}

// Callback returns the ssh.HostKeyCallback for this verifier.
func (v *HostKeyVerifier) Callback() gossh.HostKeyCallback {
	switch v.mode {
	case config.HostKeyInsecure:
		return gossh.InsecureIgnoreHostKey()

	case config.HostKeyAcceptNew:
		return func(_ string, _ net.Addr, key gossh.PublicKey) error {
			stored, found := v.store.Lookup(v.hostport)
			if !found {
				return v.store.Add(v.hostport, key.Marshal())
			}
			if subtle.ConstantTimeCompare(stored, key.Marshal()) != 1 {
				return v.record(HostKeyReasonChanged, key, stored)
			}
			return nil
		}

	case config.HostKeyStrict:
		return func(_ string, _ net.Addr, key gossh.PublicKey) error {
			stored, found := v.store.Lookup(v.hostport)
			if !found {
				return v.record(HostKeyReasonUnknown, key, nil)
			}
			if subtle.ConstantTimeCompare(stored, key.Marshal()) != 1 {
				return v.record(HostKeyReasonChanged, key, stored)
			}
			return nil
		}

	default:
		// Fail closed: an unrecognised mode must never mean "skip verification".
		return func(string, net.Addr, gossh.PublicKey) error {
			return fmt.Errorf("invalid host key verification mode %q for %s", v.mode, v.hostport)
		}
	}
}

// record stores the mismatch detail, parks the offered key as pending so the
// user can confirm it, and returns the error to reject the handshake with.
func (v *HostKeyVerifier) record(reason string, offered gossh.PublicKey, stored []byte) error {
	mm := &HostKeyMismatch{
		HostPort:    v.hostport,
		Reason:      reason,
		OfferedFP:   gossh.FingerprintSHA256(offered),
		OfferedType: offered.Type(),
	}
	if len(stored) > 0 {
		if fp, keyType, err := Fingerprint(stored); err == nil {
			mm.StoredFP = fp
			mm.StoredKeyType = keyType
		}
	}

	v.store.SetPending(v.hostport, offered.Marshal())

	v.mu.Lock()
	v.mismatch = mm
	v.mu.Unlock()

	return errors.New(mm.String())
}

type noopHostKeyStore struct{}

func (noopHostKeyStore) Lookup(string) ([]byte, bool)  { return nil, false }
func (noopHostKeyStore) Add(string, []byte) error      { return nil }
func (noopHostKeyStore) Delete(string) error           { return nil }
func (noopHostKeyStore) List() []KnownHost             { return nil }
func (noopHostKeyStore) SetPending(string, []byte)     {}
func (noopHostKeyStore) Pending(string) ([]byte, bool) { return nil, false }
func (noopHostKeyStore) ClearPending(string)           {}

// NewNoopHostKeyStore returns a no-op store (for testing).
func NewNoopHostKeyStore() HostKeyStore { return noopHostKeyStore{} }
