package ssh

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"

	"github.com/maxzhang666/ops-tunnel/internal/config"
	gossh "golang.org/x/crypto/ssh"
)

// HostKeyStore persists known host key fingerprints.
type HostKeyStore interface {
	Lookup(hostport string) ([]byte, bool)
	Add(hostport string, key []byte) error
}

// JSONHostKeyStore stores host keys in a JSON file.
type JSONHostKeyStore struct {
	path string
	mu   sync.RWMutex
	keys map[string]string
}

// NewJSONHostKeyStore creates or loads a host key store.
func NewJSONHostKeyStore(path string) *JSONHostKeyStore {
	s := &JSONHostKeyStore{
		path: path,
		keys: make(map[string]string),
	}
	s.load()
	return s
}

func (s *JSONHostKeyStore) load() {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	json.Unmarshal(data, &s.keys)
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
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
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

// HostKeyCallback creates an ssh.HostKeyCallback based on the verification mode.
func HostKeyCallback(mode config.HostKeyVerifyMode, store HostKeyStore, hostport string) gossh.HostKeyCallback {
	switch mode {
	case config.HostKeyInsecure:
		return gossh.InsecureIgnoreHostKey()

	case config.HostKeyAcceptNew:
		return func(hostname string, remote net.Addr, key gossh.PublicKey) error {
			marshaledKey := key.Marshal()
			stored, found := store.Lookup(hostport)
			if !found {
				return store.Add(hostport, marshaledKey)
			}
			if !bytesEqual(stored, marshaledKey) {
				return fmt.Errorf("host key mismatch for %s (key changed since first connection)", hostport)
			}
			return nil
		}

	case config.HostKeyStrict:
		return func(hostname string, remote net.Addr, key gossh.PublicKey) error {
			marshaledKey := key.Marshal()
			stored, found := store.Lookup(hostport)
			if !found {
				return fmt.Errorf("no known host key for %s (strict mode requires pre-registered key)", hostport)
			}
			if !bytesEqual(stored, marshaledKey) {
				return fmt.Errorf("host key mismatch for %s", hostport)
			}
			return nil
		}

	default:
		return gossh.InsecureIgnoreHostKey()
	}
}

type noopHostKeyStore struct{}

func (noopHostKeyStore) Lookup(string) ([]byte, bool) { return nil, false }
func (noopHostKeyStore) Add(string, []byte) error      { return nil }

// NewNoopHostKeyStore returns a no-op store (for testing).
func NewNoopHostKeyStore() HostKeyStore { return noopHostKeyStore{} }

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
