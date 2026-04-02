# Phase 3: SSH Multi-Hop Chain — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement real SSH connections with multi-hop chain building, KeepAlive, host key verification, and test connection. Upgrade supervisor from stub to real SSH.

**Architecture:** `internal/ssh/` package handles all SSH-specific logic (auth, host keys, keepalive, chain building). The engine supervisor calls `ssh.BuildChain()` on Start and `chain.Close()` on Stop. API adds a test-connection endpoint.

**Tech Stack:** Go 1.26, `golang.org/x/crypto/ssh`, `log/slog`

---

## File Map

| File | Purpose |
|------|---------|
| `internal/ssh/auth.go` | config.Auth → []ssh.AuthMethod conversion |
| `internal/ssh/hostkey.go` | HostKeyStore (JSON) + HostKeyCallback factory |
| `internal/ssh/keepalive.go` | KeepAlive goroutine |
| `internal/ssh/chain.go` | BuildChain: multi-hop SSH connection builder |
| `internal/ssh/test.go` | TestConnection: single SSH test |
| `internal/ssh/auth_test.go` | Auth conversion tests |
| `internal/ssh/hostkey_test.go` | HostKeyStore tests |
| `internal/engine/supervisor.go` | **Rewrite:** real SSH chain management |
| `internal/engine/engine.go` | **Modify:** add hostKeyStore, resolve SSH connections |
| `internal/api/handler_ssh.go` | **Modify:** add testSSHConnection handler |
| `internal/api/routes.go` | **Modify:** add test-connection route |
| `internal/api/server.go` | **Modify:** add hostKeyStorePath to Engine init |
| `cmd/tunnel-server/main.go` | **Modify:** pass data-dir to engine for hostkey store |

---

## Task 1: SSH Auth

**Files:**
- Create: `internal/ssh/auth.go`

- [ ] **Step 1: Create `internal/ssh/auth.go`**

```go
package ssh

import (
	"fmt"
	"os"

	"github.com/maxzhang666/ops-tunnel/internal/config"
	gossh "golang.org/x/crypto/ssh"
)

// AuthMethods converts config.Auth to SSH auth methods.
func AuthMethods(a config.Auth) ([]gossh.AuthMethod, error) {
	switch a.Type {
	case config.AuthPassword:
		return []gossh.AuthMethod{gossh.Password(a.Password)}, nil

	case config.AuthPrivateKey:
		if a.PrivateKey == nil {
			return nil, fmt.Errorf("privateKey config is nil")
		}
		var pemBytes []byte
		switch a.PrivateKey.Source {
		case config.KeySourceInline:
			pemBytes = []byte(a.PrivateKey.KeyPEM)
		case config.KeySourceFile:
			var err error
			pemBytes, err = os.ReadFile(a.PrivateKey.FilePath)
			if err != nil {
				return nil, fmt.Errorf("read key file: %w", err)
			}
		default:
			return nil, fmt.Errorf("unknown key source: %s", a.PrivateKey.Source)
		}

		var signer gossh.Signer
		var err error
		if a.PrivateKey.Passphrase != "" {
			signer, err = gossh.ParsePrivateKeyWithPassphrase(pemBytes, []byte(a.PrivateKey.Passphrase))
		} else {
			signer, err = gossh.ParsePrivateKey(pemBytes)
		}
		if err != nil {
			return nil, fmt.Errorf("parse private key: %w", err)
		}
		return []gossh.AuthMethod{gossh.PublicKeys(signer)}, nil

	case config.AuthNone:
		return nil, nil

	default:
		return nil, fmt.Errorf("unknown auth type: %s", a.Type)
	}
}
```

- [ ] **Step 2: Verify and commit**

```bash
go build ./internal/ssh/
git add internal/ssh/auth.go
git commit -m "feat(ssh): add auth method conversion"
```

---

## Task 2: Host Key Store

**Files:**
- Create: `internal/ssh/hostkey.go`

- [ ] **Step 1: Create `internal/ssh/hostkey.go`**

```go
package ssh

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
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
	keys map[string]string // hostport → base64-encoded marshal of public key
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

// NoopHostKeyStore is a store that never finds or stores keys.
// Used when no persistence path is available.
var _ HostKeyStore = (*noopHostKeyStore)(nil)

type noopHostKeyStore struct{}

func (noopHostKeyStore) Lookup(string) ([]byte, bool) { return nil, false }
func (noopHostKeyStore) Add(string, []byte) error      { return nil }

// NewNoopHostKeyStore returns a no-op store (for testing).
func NewNoopHostKeyStore() HostKeyStore { return noopHostKeyStore{} }

// fileExists checks if a file exists and is not a directory.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return false
	}
	return err == nil && !info.IsDir()
}

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
```

- [ ] **Step 2: Verify and commit**

```bash
go build ./internal/ssh/
git add internal/ssh/hostkey.go
git commit -m "feat(ssh): add host key store and verification callbacks"
```

---

## Task 3: KeepAlive

**Files:**
- Create: `internal/ssh/keepalive.go`

- [ ] **Step 1: Create `internal/ssh/keepalive.go`**

```go
package ssh

import (
	"context"
	"log/slog"
	"time"

	gossh "golang.org/x/crypto/ssh"
)

// StartKeepAlive sends periodic keepalive requests to the SSH server.
// Returns a channel that receives an error if maxMissed consecutive keepalives fail.
// The goroutine stops when ctx is cancelled.
func StartKeepAlive(ctx context.Context, client *gossh.Client, interval time.Duration, maxMissed int) <-chan error {
	errCh := make(chan error, 1)

	go func() {
		defer close(errCh)
		missed := 0
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_, _, err := client.SendRequest("keepalive@openssh.com", true, nil)
				if err != nil {
					missed++
					slog.Debug("keepalive failed", "missed", missed, "err", err)
					if missed >= maxMissed {
						errCh <- err
						return
					}
				} else {
					missed = 0
				}
			}
		}
	}()

	return errCh
}
```

- [ ] **Step 2: Verify and commit**

```bash
go build ./internal/ssh/
git add internal/ssh/keepalive.go
git commit -m "feat(ssh): add keepalive goroutine"
```

---

## Task 4: Chain Builder

**Files:**
- Create: `internal/ssh/chain.go`

- [ ] **Step 1: Create `internal/ssh/chain.go`**

```go
package ssh

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/maxzhang666/ops-tunnel/internal/config"
	gossh "golang.org/x/crypto/ssh"
)

// ChainResult holds all SSH clients created for a multi-hop chain.
type ChainResult struct {
	Clients    []*gossh.Client
	KACancel   context.CancelFunc // cancels all keepalive goroutines
	KAErrors   []<-chan error     // keepalive error channels per hop
}

// Last returns the final client in the chain (the target).
func (r *ChainResult) Last() *gossh.Client {
	if len(r.Clients) == 0 {
		return nil
	}
	return r.Clients[len(r.Clients)-1]
}

// Close shuts down all clients in reverse order.
func (r *ChainResult) Close() error {
	if r.KACancel != nil {
		r.KACancel()
	}
	var lastErr error
	for i := len(r.Clients) - 1; i >= 0; i-- {
		if err := r.Clients[i].Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// BuildChain establishes a chain of SSH connections through the given hops.
// Each hop uses the previous client's Dial to reach the next.
func BuildChain(ctx context.Context, conns []config.SSHConnection, hostKeyStore HostKeyStore) (*ChainResult, error) {
	if len(conns) == 0 {
		return nil, fmt.Errorf("chain must have at least one connection")
	}

	kaCtx, kaCancel := context.WithCancel(ctx)
	result := &ChainResult{
		KACancel: kaCancel,
	}

	var prevClient *gossh.Client

	for i, conn := range conns {
		addr := fmt.Sprintf("%s:%d", conn.Endpoint.Host, conn.Endpoint.Port)
		timeout := time.Duration(conn.DialTimeoutMs) * time.Millisecond
		if timeout == 0 {
			timeout = 10 * time.Second
		}

		slog.Info("connecting hop", "hop", i+1, "name", conn.Name, "addr", addr)

		authMethods, err := AuthMethods(conn.Auth)
		if err != nil {
			result.Close()
			return nil, fmt.Errorf("hop %d (%s): auth config error: %w", i+1, conn.Name, err)
		}

		hostport := addr
		hkCallback := HostKeyCallback(conn.HostKeyVerification.Mode, hostKeyStore, hostport)

		sshConfig := &gossh.ClientConfig{
			User:            conn.Auth.Username,
			Auth:            authMethods,
			HostKeyCallback: hkCallback,
			Timeout:         timeout,
		}

		var netConn net.Conn
		if prevClient == nil {
			// First hop: direct TCP dial
			dialer := net.Dialer{Timeout: timeout}
			dialCtx, dialCancel := context.WithTimeout(ctx, timeout)
			netConn, err = dialer.DialContext(dialCtx, "tcp", addr)
			dialCancel()
		} else {
			// Subsequent hops: dial through previous client
			netConn, err = prevClient.DialContext(ctx, "tcp", addr)
		}
		if err != nil {
			result.Close()
			return nil, fmt.Errorf("hop %d (%s): dial failed: %w", i+1, conn.Name, err)
		}

		sshConn, chans, reqs, err := gossh.NewClientConn(netConn, addr, sshConfig)
		if err != nil {
			netConn.Close()
			result.Close()
			return nil, fmt.Errorf("hop %d (%s): SSH handshake failed: %w", i+1, conn.Name, err)
		}

		client := gossh.NewClient(sshConn, chans, reqs)
		result.Clients = append(result.Clients, client)

		// Start keepalive if configured
		if conn.KeepAlive.IntervalMs > 0 && conn.KeepAlive.MaxMissed > 0 {
			interval := time.Duration(conn.KeepAlive.IntervalMs) * time.Millisecond
			kaErrCh := StartKeepAlive(kaCtx, client, interval, conn.KeepAlive.MaxMissed)
			result.KAErrors = append(result.KAErrors, kaErrCh)
		}

		slog.Info("hop connected", "hop", i+1, "name", conn.Name)
		prevClient = client
	}

	return result, nil
}
```

- [ ] **Step 2: Verify and commit**

```bash
go build ./internal/ssh/
git add internal/ssh/chain.go
git commit -m "feat(ssh): add multi-hop chain builder"
```

---

## Task 5: Test Connection

**Files:**
- Create: `internal/ssh/test.go`

- [ ] **Step 1: Create `internal/ssh/test.go`**

```go
package ssh

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/maxzhang666/ops-tunnel/internal/config"
	gossh "golang.org/x/crypto/ssh"
)

// TestResult holds the result of a connection test.
type TestResult struct {
	OK        bool   `json:"ok"`
	LatencyMs int64  `json:"latencyMs,omitempty"`
	Error     string `json:"error,omitempty"`
}

// TestConnection attempts to connect and authenticate to a single SSH server.
func TestConnection(ctx context.Context, conn config.SSHConnection, hostKeyStore HostKeyStore) TestResult {
	addr := fmt.Sprintf("%s:%d", conn.Endpoint.Host, conn.Endpoint.Port)
	timeout := time.Duration(conn.DialTimeoutMs) * time.Millisecond
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	authMethods, err := AuthMethods(conn.Auth)
	if err != nil {
		return TestResult{Error: fmt.Sprintf("auth config: %s", err)}
	}

	hkCallback := HostKeyCallback(conn.HostKeyVerification.Mode, hostKeyStore, addr)

	sshConfig := &gossh.ClientConfig{
		User:            conn.Auth.Username,
		Auth:            authMethods,
		HostKeyCallback: hkCallback,
		Timeout:         timeout,
	}

	start := time.Now()

	dialCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	dialer := net.Dialer{Timeout: timeout}
	netConn, err := dialer.DialContext(dialCtx, "tcp", addr)
	if err != nil {
		return TestResult{Error: fmt.Sprintf("dial: %s", err)}
	}

	sshConn, chans, reqs, err := gossh.NewClientConn(netConn, addr, sshConfig)
	if err != nil {
		netConn.Close()
		return TestResult{Error: fmt.Sprintf("SSH handshake: %s", err)}
	}

	client := gossh.NewClient(sshConn, chans, reqs)
	latency := time.Since(start).Milliseconds()
	client.Close()

	return TestResult{OK: true, LatencyMs: latency}
}
```

- [ ] **Step 2: Verify and commit**

```bash
go build ./internal/ssh/
git add internal/ssh/test.go
git commit -m "feat(ssh): add test connection function"
```

---

## Task 6: SSH Unit Tests

**Files:**
- Create: `internal/ssh/auth_test.go`
- Create: `internal/ssh/hostkey_test.go`

- [ ] **Step 1: Create `internal/ssh/auth_test.go`**

```go
package ssh

import (
	"testing"

	"github.com/maxzhang666/ops-tunnel/internal/config"
)

func TestAuthMethods_Password(t *testing.T) {
	methods, err := AuthMethods(config.Auth{
		Type:     config.AuthPassword,
		Username: "user",
		Password: "pass",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(methods) != 1 {
		t.Errorf("expected 1 method, got %d", len(methods))
	}
}

func TestAuthMethods_None(t *testing.T) {
	methods, err := AuthMethods(config.Auth{Type: config.AuthNone})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if methods != nil {
		t.Errorf("expected nil methods for none auth, got %v", methods)
	}
}

func TestAuthMethods_PrivateKeyInline(t *testing.T) {
	// Use a minimal test key (this won't connect to anything, just tests parsing)
	_, err := AuthMethods(config.Auth{
		Type:     config.AuthPrivateKey,
		Username: "user",
		PrivateKey: &config.PrivateKey{
			Source: config.KeySourceInline,
			KeyPEM: "not-a-valid-key",
		},
	})
	// Should fail to parse, but not panic
	if err == nil {
		t.Error("expected error for invalid key PEM")
	}
}

func TestAuthMethods_PrivateKeyNilConfig(t *testing.T) {
	_, err := AuthMethods(config.Auth{
		Type:     config.AuthPrivateKey,
		Username: "user",
	})
	if err == nil {
		t.Error("expected error for nil privateKey")
	}
}

func TestAuthMethods_UnknownType(t *testing.T) {
	_, err := AuthMethods(config.Auth{Type: "unknown"})
	if err == nil {
		t.Error("expected error for unknown auth type")
	}
}
```

- [ ] **Step 2: Create `internal/ssh/hostkey_test.go`**

```go
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

	// Reload from disk
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
```

- [ ] **Step 3: Run tests**

```bash
go test ./internal/ssh/ -v
```

- [ ] **Step 4: Commit**

```bash
git add internal/ssh/auth_test.go internal/ssh/hostkey_test.go
git commit -m "test(ssh): add auth and host key store tests"
```

---

## Task 7: Upgrade Supervisor + Engine

**Files:**
- Rewrite: `internal/engine/supervisor.go`
- Modify: `internal/engine/engine.go`

- [ ] **Step 1: Rewrite `internal/engine/supervisor.go`**

Replace the entire file:

```go
package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/maxzhang666/ops-tunnel/internal/config"
	tunnelssh "github.com/maxzhang666/ops-tunnel/internal/ssh"
)

type tunnelSupervisor struct {
	tunnel   config.Tunnel
	conns    []config.SSHConnection
	bus      EventBus
	hostKeys tunnelssh.HostKeyStore

	mu       sync.RWMutex
	state    TunnelState
	since    time.Time
	lastErr  string
	chain    *tunnelssh.ChainResult
	kaCtx    context.Context
	kaCancel context.CancelFunc
}

func newSupervisor(t config.Tunnel, conns []config.SSHConnection, bus EventBus, hostKeys tunnelssh.HostKeyStore) *tunnelSupervisor {
	return &tunnelSupervisor{
		tunnel:   t,
		conns:    conns,
		bus:      bus,
		hostKeys: hostKeys,
		state:    StateStopped,
		since:    time.Now().UTC(),
	}
}

func (s *tunnelSupervisor) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state == StateRunning || s.state == StateStarting {
		return fmt.Errorf("tunnel %s is already %s", s.tunnel.ID, s.state)
	}

	s.setState(StateStarting)

	chainCtx, chainCancel := context.WithCancel(context.Background())
	s.kaCtx = chainCtx
	s.kaCancel = chainCancel

	chain, err := tunnelssh.BuildChain(ctx, s.conns, s.hostKeys)
	if err != nil {
		chainCancel()
		s.lastErr = err.Error()
		s.setState(StateError)
		return fmt.Errorf("build chain: %w", err)
	}

	s.chain = chain
	s.lastErr = ""
	s.setState(StateRunning)

	// Monitor keepalive errors
	for i, kaErr := range chain.KAErrors {
		go func(hopIdx int, errCh <-chan error) {
			select {
			case <-chainCtx.Done():
				return
			case err, ok := <-errCh:
				if !ok || err == nil {
					return
				}
				s.mu.Lock()
				if s.state == StateRunning {
					s.lastErr = fmt.Sprintf("hop %d keepalive failed: %s", hopIdx+1, err)
					s.setState(StateDegraded)
				}
				s.mu.Unlock()
			}
		}(i, kaErr)
	}

	return nil
}

func (s *tunnelSupervisor) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state == StateStopped || s.state == StateStopping {
		return nil
	}

	s.setState(StateStopping)

	if s.kaCancel != nil {
		s.kaCancel()
	}
	if s.chain != nil {
		s.chain.Close()
		s.chain = nil
	}

	s.setState(StateStopped)
	return nil
}

func (s *tunnelSupervisor) Status() TunnelStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	chain := make([]HopStatus, len(s.conns))
	for i, conn := range s.conns {
		st := "disconnected"
		if s.state == StateRunning && s.chain != nil && i < len(s.chain.Clients) {
			st = "connected"
		}
		chain[i] = HopStatus{SSHConnID: conn.ID, State: st}
	}

	mappings := make([]MappingStatus, len(s.tunnel.Mappings))
	for i, m := range s.tunnel.Mappings {
		st := "stopped"
		listen := fmt.Sprintf("%s:%d", m.Listen.Host, m.Listen.Port)
		if s.state == StateRunning {
			st = "listening" // stub until forwarding is implemented
		}
		mappings[i] = MappingStatus{MappingID: m.ID, State: st, Listen: listen}
	}

	return TunnelStatus{
		ID:        s.tunnel.ID,
		State:     s.state,
		Since:     s.since,
		Chain:     chain,
		Mappings:  mappings,
		LastError: s.lastErr,
	}
}

func (s *tunnelSupervisor) setState(state TunnelState) {
	s.state = state
	s.since = time.Now().UTC()
	s.bus.Publish(Event{
		Type:     EventTunnelStateChanged,
		TunnelID: s.tunnel.ID,
		Level:    "info",
		Message:  fmt.Sprintf("tunnel %s: %s", s.tunnel.Name, state),
		Fields:   map[string]any{"state": string(state)},
	})
}
```

- [ ] **Step 2: Update `internal/engine/engine.go`**

Read the existing file. Make these changes:

1. Add import: `tunnelssh "github.com/maxzhang666/ops-tunnel/internal/ssh"`
2. Add `hostKeys tunnelssh.HostKeyStore` field to `eng` struct
3. Update `NewEngine` signature: `func NewEngine(cfg *config.Config, bus EventBus, hostKeys tunnelssh.HostKeyStore) Engine`
4. Set `hostKeys: hostKeys` in the struct literal
5. Add a helper to resolve SSH connections from chain IDs:

```go
func (e *eng) resolveChain(t *config.Tunnel) ([]config.SSHConnection, error) {
	conns := make([]config.SSHConnection, 0, len(t.Chain))
	for _, id := range t.Chain {
		found := false
		for _, c := range e.cfg.SSHConnections {
			if c.ID == id {
				conns = append(conns, c)
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("SSH connection '%s' not found", id)
		}
	}
	return conns, nil
}
```

6. Update `getOrCreateSupervisor` to resolve connections and pass hostKeys:

```go
func (e *eng) getOrCreateSupervisor(t *config.Tunnel) (*tunnelSupervisor, error) {
	if sup, ok := e.sups[t.ID]; ok {
		return sup, nil
	}
	conns, err := e.resolveChain(t)
	if err != nil {
		return nil, err
	}
	sup := newSupervisor(*t, conns, e.bus, e.hostKeys)
	e.sups[t.ID] = sup
	return sup, nil
}
```

7. Update all callers of `getOrCreateSupervisor` to handle the error return (StartTunnel, StopTunnel, RestartTunnel).

- [ ] **Step 3: Verify compilation**

```bash
go build ./internal/engine/
```

- [ ] **Step 4: Update engine tests**

In `internal/engine/engine_test.go`, update `NewEngine` calls to pass a noop host key store:

```go
import tunnelssh "github.com/maxzhang666/ops-tunnel/internal/ssh"

// In each test:
eng := NewEngine(testConfig(), bus, tunnelssh.NewNoopHostKeyStore())
```

Run tests:
```bash
go test ./internal/engine/ -v
```

Note: Engine tests will still pass because `Start` will fail (no real SSH server) but the state machine and event flow still work for tests that don't call Start. For `TestEngine_StartStop`, the test should be updated to expect an error from Start since there's no SSH server. Update the test to verify that start with unreachable host returns error and sets state to error.

- [ ] **Step 5: Commit**

```bash
git add internal/engine/supervisor.go internal/engine/engine.go internal/engine/engine_test.go
git commit -m "feat(engine): upgrade supervisor to real SSH chain, resolve connections"
```

---

## Task 8: Test Connection API + Integration

**Files:**
- Modify: `internal/api/handler_ssh.go` — add testSSHConnection handler
- Modify: `internal/api/routes.go` — add route
- Modify: `internal/api/server.go` — add hostKeyStore field
- Modify: `cmd/tunnel-server/main.go` — create hostKeyStore, pass to engine

- [ ] **Step 1: Add test handler to `internal/api/handler_ssh.go`**

Add this function at the end of the file:

```go
func (s *Server) testSSHConnection(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	s.mu.RLock()
	var conn *config.SSHConnection
	for i, c := range s.data.SSHConnections {
		if c.ID == id {
			conn = &s.data.SSHConnections[i]
			break
		}
	}
	s.mu.RUnlock()

	if conn == nil {
		writeNotFound(w, "ssh-connection", id)
		return
	}

	result := tunnelssh.TestConnection(r.Context(), *conn, s.hostKeys)
	if result.OK {
		writeJSON(w, http.StatusOK, map[string]any{
			"status":    "ok",
			"message":   "connected successfully",
			"latencyMs": result.LatencyMs,
		})
	} else {
		writeJSON(w, http.StatusOK, map[string]any{
			"status":  "error",
			"message": result.Error,
		})
	}
}
```

Add import to handler_ssh.go: `tunnelssh "github.com/maxzhang666/ops-tunnel/internal/ssh"`

- [ ] **Step 2: Add route in `internal/api/routes.go`**

Inside the ssh-connections route group, add:
```go
r.Post("/{id}/test", s.testSSHConnection)
```

- [ ] **Step 3: Update `internal/api/server.go`**

Add `hostKeys tunnelssh.HostKeyStore` field to Server.
Add import: `tunnelssh "github.com/maxzhang666/ops-tunnel/internal/ssh"`
Update NewServer signature: add `hostKeys tunnelssh.HostKeyStore` parameter.
Set `hostKeys: hostKeys` in struct literal.

- [ ] **Step 4: Update `cmd/tunnel-server/main.go`**

Add import: `tunnelssh "github.com/maxzhang666/ops-tunnel/internal/ssh"`

After engine creation, create host key store:
```go
hostKeys := tunnelssh.NewJSONHostKeyStore(filepath.Join(*dataDir, "known_hosts.json"))
```

Update `NewEngine` call: pass `hostKeys`
```go
eng := engine.NewEngine(cfg, bus, hostKeys)
```

Update `NewServer` call: pass `hostKeys`
```go
srv := api.NewServer(api.ServerConfig{...}, store, cfg, eng, hostKeys)
```

- [ ] **Step 5: Build and test**

```bash
go build -o bin/tunnel-server ./cmd/tunnel-server
go test ./... -v
```

- [ ] **Step 6: Commit**

```bash
git add internal/api/ internal/engine/ cmd/tunnel-server/main.go
git commit -m "feat: add test-connection endpoint, wire hostkey store through stack"
```

---

## Task 9: Commit docs

- [ ] **Step 1: Commit**

```bash
git add -f docs/
git commit -m "docs: add Phase 3 spec and plan"
```
