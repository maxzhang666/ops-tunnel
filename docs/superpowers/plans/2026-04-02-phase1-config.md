# Phase 1: Config Model + Storage + Validation + CRUD API — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the complete config layer (SSHConnection + Tunnel data model) with JSON file storage, validation, redaction, and REST CRUD endpoints.

**Architecture:** `internal/config/` package defines all types and handles persistence. `internal/api/` handlers operate on an in-memory `*config.Config` (loaded at startup), validate mutations, save to disk, and return redacted responses. The `api.Server` holds both the store and the live config.

**Tech Stack:** Go 1.26, chi v5, `github.com/rs/xid` (ID generation), `log/slog`, stdlib `encoding/json`

---

## File Map

| File | Purpose |
|------|---------|
| `internal/config/model.go` | All type definitions (SSHConnection, Tunnel, Mapping, etc.) |
| `internal/config/defaults.go` | `ApplySSHConnectionDefaults`, `ApplyTunnelDefaults` |
| `internal/config/validate.go` | `ValidateSSHConnection`, `ValidateTunnel`, `ValidateConfig` |
| `internal/config/store.go` | `Store` interface, `FileStore` (atomic write) |
| `internal/config/redact.go` | `RedactSSHConnection`, `RedactTunnel`, `RedactConfig` |
| `internal/config/model_test.go` | Tests for defaults, validation, redaction |
| `internal/config/store_test.go` | Tests for FileStore load/save |
| `internal/api/response.go` | Shared JSON response/error helpers |
| `internal/api/handler_ssh.go` | SSH Connection CRUD handlers |
| `internal/api/handler_tunnel.go` | Tunnel CRUD handlers |
| `internal/api/server.go` | **Modify:** add store + config fields, update NewServer |
| `internal/api/routes.go` | **Modify:** register new routes |
| `cmd/tunnel-server/main.go` | **Modify:** init FileStore, load config, pass to Server |

---

## Task 1: Add xid Dependency

**Files:**
- Modify: `go.mod`

- [ ] **Step 1: Add xid**

```bash
cd /Users/maxzhang/Tools/ops-tunnel
go get github.com/rs/xid
```

- [ ] **Step 2: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add xid dependency for ID generation"
```

---

## Task 2: Config Model

**Files:**
- Create: `internal/config/model.go`

- [ ] **Step 1: Create `internal/config/model.go`**

```go
package config

// Endpoint represents a host:port pair.
type Endpoint struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

// AuthType defines SSH authentication method.
type AuthType string

const (
	AuthPassword   AuthType = "password"
	AuthPrivateKey AuthType = "privateKey"
	AuthNone       AuthType = "none"
)

// PrivateKeySource defines where the key content comes from.
type PrivateKeySource string

const (
	KeySourceInline PrivateKeySource = "inline"
	KeySourceFile   PrivateKeySource = "file"
)

// PrivateKey holds SSH private key configuration.
type PrivateKey struct {
	Source     PrivateKeySource `json:"source"`
	KeyPEM     string          `json:"keyPem,omitempty"`
	FilePath   string          `json:"filePath,omitempty"`
	Passphrase string          `json:"passphrase,omitempty"`
}

// Auth holds SSH authentication configuration.
type Auth struct {
	Type       AuthType    `json:"type"`
	Username   string      `json:"username"`
	Password   string      `json:"password,omitempty"`
	PrivateKey *PrivateKey `json:"privateKey,omitempty"`
}

// HostKeyVerifyMode defines host key verification strategy.
type HostKeyVerifyMode string

const (
	HostKeyInsecure  HostKeyVerifyMode = "insecure"
	HostKeyAcceptNew HostKeyVerifyMode = "acceptNew"
	HostKeyStrict    HostKeyVerifyMode = "strict"
)

// HostKeyVerification holds host key verification settings.
type HostKeyVerification struct {
	Mode HostKeyVerifyMode `json:"mode"`
}

// KeepAlive holds SSH keep-alive settings.
type KeepAlive struct {
	IntervalMs int `json:"intervalMs"`
	MaxMissed  int `json:"maxMissed"`
}

// SSHConnection is an independently managed SSH connection configuration.
type SSHConnection struct {
	ID                  string              `json:"id"`
	Name                string              `json:"name"`
	Endpoint            Endpoint            `json:"endpoint"`
	Auth                Auth                `json:"auth"`
	HostKeyVerification HostKeyVerification `json:"hostKeyVerification"`
	DialTimeoutMs       int                 `json:"dialTimeoutMs"`
	KeepAlive           KeepAlive           `json:"keepAlive"`
}

// TunnelMode defines the forwarding type.
type TunnelMode string

const (
	ModeLocal   TunnelMode = "local"
	ModeRemote  TunnelMode = "remote"
	ModeDynamic TunnelMode = "dynamic"
)

// Socks5Auth defines SOCKS5 authentication method.
type Socks5Auth string

const (
	Socks5None     Socks5Auth = "none"
	Socks5UserPass Socks5Auth = "userpass"
)

// Socks5Cfg holds SOCKS5 proxy configuration for dynamic tunnels.
type Socks5Cfg struct {
	Auth       Socks5Auth `json:"auth"`
	Username   string     `json:"username,omitempty"`
	Password   string     `json:"password,omitempty"`
	AllowCIDRs []string   `json:"allowCIDRs,omitempty"`
	DenyCIDRs  []string   `json:"denyCIDRs,omitempty"`
}

// Mapping defines a single port forwarding rule within a tunnel.
type Mapping struct {
	ID      string     `json:"id"`
	Listen  Endpoint   `json:"listen"`
	Connect Endpoint   `json:"connect,omitempty"`
	Socks5  *Socks5Cfg `json:"socks5,omitempty"`
	Notes   string     `json:"notes,omitempty"`
}

// RestartBackoff holds exponential backoff parameters.
type RestartBackoff struct {
	MinMs  int     `json:"minMs"`
	MaxMs  int     `json:"maxMs"`
	Factor float64 `json:"factor"`
}

// Policy holds tunnel runtime behavior settings.
type Policy struct {
	AutoStart             bool           `json:"autoStart"`
	AutoRestart           bool           `json:"autoRestart"`
	RestartBackoff        RestartBackoff `json:"restartBackoff"`
	MaxRestartsPerHour    int            `json:"maxRestartsPerHour"`
	GracefulStopTimeoutMs int            `json:"gracefulStopTimeoutMs"`
}

// Tunnel references SSH connections and defines forwarding rules.
type Tunnel struct {
	ID       string     `json:"id"`
	Name     string     `json:"name"`
	Mode     TunnelMode `json:"mode"`
	Chain    []string   `json:"chain"`
	Mappings []Mapping  `json:"mappings"`
	Policy   Policy     `json:"policy"`
}

// Config is the top-level configuration persisted to disk.
type Config struct {
	Version        int              `json:"version"`
	SSHConnections []SSHConnection  `json:"sshConnections"`
	Tunnels        []Tunnel         `json:"tunnels"`
}

// NewConfig returns an empty config with version set.
func NewConfig() *Config {
	return &Config{
		Version:        1,
		SSHConnections: []SSHConnection{},
		Tunnels:        []Tunnel{},
	}
}
```

- [ ] **Step 2: Verify compilation**

```bash
cd /Users/maxzhang/Tools/ops-tunnel
go build ./internal/config/
```

- [ ] **Step 3: Commit**

```bash
git add internal/config/model.go
git commit -m "feat(config): add data model types"
```

---

## Task 3: Defaults

**Files:**
- Create: `internal/config/defaults.go`

- [ ] **Step 1: Create `internal/config/defaults.go`**

```go
package config

// ApplySSHConnectionDefaults fills zero-value fields with sensible defaults.
func ApplySSHConnectionDefaults(c *SSHConnection) {
	if c.DialTimeoutMs == 0 {
		c.DialTimeoutMs = 10000
	}
	if c.KeepAlive.IntervalMs == 0 {
		c.KeepAlive.IntervalMs = 15000
	}
	if c.KeepAlive.MaxMissed == 0 {
		c.KeepAlive.MaxMissed = 3
	}
	if c.HostKeyVerification.Mode == "" {
		c.HostKeyVerification.Mode = HostKeyAcceptNew
	}
}

// ApplyMappingDefaults fills zero-value fields on a mapping.
func ApplyMappingDefaults(m *Mapping) {
	if m.Listen.Host == "" {
		m.Listen.Host = "127.0.0.1"
	}
}

// ApplyTunnelDefaults fills zero-value fields with sensible defaults.
func ApplyTunnelDefaults(t *Tunnel) {
	for i := range t.Mappings {
		ApplyMappingDefaults(&t.Mappings[i])
	}
	p := &t.Policy
	if !p.AutoRestart {
		// AutoRestart defaults to true only when Policy is zero-value
		// We check if the entire backoff is zero to detect "not explicitly set"
		if p.RestartBackoff.MinMs == 0 && p.RestartBackoff.MaxMs == 0 {
			p.AutoRestart = true
		}
	}
	if p.RestartBackoff.MinMs == 0 {
		p.RestartBackoff.MinMs = 500
	}
	if p.RestartBackoff.MaxMs == 0 {
		p.RestartBackoff.MaxMs = 15000
	}
	if p.RestartBackoff.Factor == 0 {
		p.RestartBackoff.Factor = 1.7
	}
	if p.MaxRestartsPerHour == 0 {
		p.MaxRestartsPerHour = 60
	}
	if p.GracefulStopTimeoutMs == 0 {
		p.GracefulStopTimeoutMs = 5000
	}
}

// ApplyConfigDefaults applies defaults to all entities in a config.
func ApplyConfigDefaults(cfg *Config) {
	for i := range cfg.SSHConnections {
		ApplySSHConnectionDefaults(&cfg.SSHConnections[i])
	}
	for i := range cfg.Tunnels {
		ApplyTunnelDefaults(&cfg.Tunnels[i])
	}
}
```

- [ ] **Step 2: Verify compilation**

```bash
go build ./internal/config/
```

- [ ] **Step 3: Commit**

```bash
git add internal/config/defaults.go
git commit -m "feat(config): add default value logic"
```

---

## Task 4: Validation

**Files:**
- Create: `internal/config/validate.go`

- [ ] **Step 1: Create `internal/config/validate.go`**

```go
package config

import (
	"fmt"
	"strings"
)

// ValidationError represents a single field validation failure.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationResult holds errors (blocking) and warnings (non-blocking).
type ValidationResult struct {
	Errors   []ValidationError `json:"errors,omitempty"`
	Warnings []string          `json:"warnings,omitempty"`
}

// HasErrors returns true if there are blocking validation errors.
func (r *ValidationResult) HasErrors() bool {
	return len(r.Errors) > 0
}

func (r *ValidationResult) addError(field, msg string) {
	r.Errors = append(r.Errors, ValidationError{Field: field, Message: msg})
}

func (r *ValidationResult) addWarning(msg string) {
	r.Warnings = append(r.Warnings, msg)
}

func validatePort(port int) bool {
	return port >= 1 && port <= 65535
}

// ValidateSSHConnection validates a single SSH connection (without uniqueness check).
func ValidateSSHConnection(c *SSHConnection, prefix string) *ValidationResult {
	r := &ValidationResult{}
	p := prefix

	if c.Name == "" {
		r.addError(p+"name", "must not be empty")
	}
	if c.Endpoint.Host == "" {
		r.addError(p+"endpoint.host", "must not be empty")
	}
	if !validatePort(c.Endpoint.Port) {
		r.addError(p+"endpoint.port", "must be between 1 and 65535")
	}

	switch c.Auth.Type {
	case AuthPassword:
		if c.Auth.Username == "" {
			r.addError(p+"auth.username", "must not be empty for password auth")
		}
		if c.Auth.Password == "" {
			r.addError(p+"auth.password", "must not be empty for password auth")
		}
	case AuthPrivateKey:
		if c.Auth.Username == "" {
			r.addError(p+"auth.username", "must not be empty for privateKey auth")
		}
		if c.Auth.PrivateKey == nil {
			r.addError(p+"auth.privateKey", "must be provided for privateKey auth")
		} else {
			switch c.Auth.PrivateKey.Source {
			case KeySourceInline:
				if c.Auth.PrivateKey.KeyPEM == "" {
					r.addError(p+"auth.privateKey.keyPem", "must not be empty for inline source")
				}
			case KeySourceFile:
				if c.Auth.PrivateKey.FilePath == "" {
					r.addError(p+"auth.privateKey.filePath", "must not be empty for file source")
				}
			default:
				r.addError(p+"auth.privateKey.source", "must be 'inline' or 'file'")
			}
		}
	case AuthNone:
		// no additional fields required
	default:
		r.addError(p+"auth.type", "must be 'password', 'privateKey', or 'none'")
	}

	return r
}

// validateMapping validates a single mapping for the given tunnel mode.
func validateMapping(m *Mapping, mode TunnelMode, prefix string) *ValidationResult {
	r := &ValidationResult{}
	p := prefix

	if !validatePort(m.Listen.Port) {
		r.addError(p+"listen.port", "must be between 1 and 65535")
	}

	switch mode {
	case ModeLocal, ModeRemote:
		if m.Connect.Host == "" {
			r.addError(p+"connect.host", "must not be empty for "+string(mode)+" mode")
		}
		if !validatePort(m.Connect.Port) {
			r.addError(p+"connect.port", "must be between 1 and 65535")
		}
	case ModeDynamic:
		if m.Socks5 == nil {
			r.addError(p+"socks5", "must be provided for dynamic mode")
		} else {
			switch m.Socks5.Auth {
			case Socks5UserPass:
				if m.Socks5.Username == "" {
					r.addError(p+"socks5.username", "must not be empty for userpass auth")
				}
				if m.Socks5.Password == "" {
					r.addError(p+"socks5.password", "must not be empty for userpass auth")
				}
			case Socks5None:
				// check for security warning
				if m.Listen.Host == "0.0.0.0" {
					wideOpen := len(m.Socks5.AllowCIDRs) == 0
					if !wideOpen {
						for _, cidr := range m.Socks5.AllowCIDRs {
							if cidr == "0.0.0.0/0" {
								wideOpen = true
								break
							}
						}
					}
					if wideOpen {
						r.addWarning("SOCKS5 proxy is publicly accessible without authentication")
					}
				}
			default:
				r.addError(p+"socks5.auth", "must be 'none' or 'userpass'")
			}
		}
	}

	return r
}

// ValidateTunnel validates a single tunnel. sshIDs is the set of known SSH connection IDs.
func ValidateTunnel(t *Tunnel, sshIDs map[string]bool, prefix string) *ValidationResult {
	r := &ValidationResult{}
	p := prefix

	if t.Name == "" {
		r.addError(p+"name", "must not be empty")
	}

	switch t.Mode {
	case ModeLocal, ModeRemote, ModeDynamic:
	default:
		r.addError(p+"mode", "must be 'local', 'remote', or 'dynamic'")
	}

	if len(t.Chain) == 0 {
		r.addError(p+"chain", "must contain at least one SSH connection")
	} else {
		for i, id := range t.Chain {
			if !sshIDs[id] {
				r.addError(fmt.Sprintf("%schain[%d]", p, i), fmt.Sprintf("SSH connection '%s' not found", id))
			}
		}
	}

	if len(t.Mappings) == 0 {
		r.addError(p+"mappings", "must contain at least one mapping")
	} else {
		seenMappingIDs := make(map[string]bool)
		for i, m := range t.Mappings {
			mp := fmt.Sprintf("%smappings[%d].", p, i)
			if m.ID == "" {
				r.addError(mp+"id", "must not be empty")
			} else if seenMappingIDs[m.ID] {
				r.addError(mp+"id", "must be unique within tunnel")
			}
			seenMappingIDs[m.ID] = true

			mr := validateMapping(&m, t.Mode, mp)
			r.Errors = append(r.Errors, mr.Errors...)
			r.Warnings = append(r.Warnings, mr.Warnings...)
		}
	}

	// Policy validation
	pp := p + "policy."
	if t.Policy.RestartBackoff.MinMs <= 0 {
		r.addError(pp+"restartBackoff.minMs", "must be greater than 0")
	}
	if t.Policy.RestartBackoff.MaxMs < t.Policy.RestartBackoff.MinMs {
		r.addError(pp+"restartBackoff.maxMs", "must be greater than or equal to minMs")
	}
	if t.Policy.RestartBackoff.Factor < 1.0 {
		r.addError(pp+"restartBackoff.factor", "must be greater than or equal to 1.0")
	}
	if t.Policy.MaxRestartsPerHour <= 0 {
		r.addError(pp+"maxRestartsPerHour", "must be greater than 0")
	}

	return r
}

// ValidateConfig validates the entire config for internal consistency.
func ValidateConfig(cfg *Config) *ValidationResult {
	r := &ValidationResult{}

	// Check SSH connection ID uniqueness
	sshIDs := make(map[string]bool, len(cfg.SSHConnections))
	for i, c := range cfg.SSHConnections {
		prefix := fmt.Sprintf("sshConnections[%d].", i)
		if c.ID == "" {
			r.addError(prefix+"id", "must not be empty")
		} else if sshIDs[c.ID] {
			r.addError(prefix+"id", "must be unique")
		}
		sshIDs[c.ID] = true

		cr := ValidateSSHConnection(&c, prefix)
		r.Errors = append(r.Errors, cr.Errors...)
		r.Warnings = append(r.Warnings, cr.Warnings...)
	}

	// Check tunnel ID uniqueness and validate each
	tunnelIDs := make(map[string]bool, len(cfg.Tunnels))
	for i, t := range cfg.Tunnels {
		prefix := fmt.Sprintf("tunnels[%d].", i)
		if t.ID == "" {
			r.addError(prefix+"id", "must not be empty")
		} else if tunnelIDs[t.ID] {
			r.addError(prefix+"id", "must be unique")
		}
		tunnelIDs[t.ID] = true

		tr := ValidateTunnel(&t, sshIDs, prefix)
		r.Errors = append(r.Errors, tr.Errors...)
		r.Warnings = append(r.Warnings, tr.Warnings...)
	}

	return r
}

// FindSSHConnectionReferences returns tunnel names that reference the given SSH connection ID.
func FindSSHConnectionReferences(cfg *Config, sshID string) []string {
	var names []string
	for _, t := range cfg.Tunnels {
		for _, chainID := range t.Chain {
			if chainID == sshID {
				names = append(names, t.Name)
				break
			}
		}
	}
	return names
}

// FormatErrors returns a human-readable summary of validation errors.
func FormatErrors(errs []ValidationError) string {
	parts := make([]string, len(errs))
	for i, e := range errs {
		parts[i] = e.Field + ": " + e.Message
	}
	return strings.Join(parts, "; ")
}
```

- [ ] **Step 2: Verify compilation**

```bash
go build ./internal/config/
```

- [ ] **Step 3: Commit**

```bash
git add internal/config/validate.go
git commit -m "feat(config): add validation with errors and warnings"
```

---

## Task 5: Redaction

**Files:**
- Create: `internal/config/redact.go`

- [ ] **Step 1: Create `internal/config/redact.go`**

```go
package config

const redacted = "***"

// RedactSSHConnection returns a copy with sensitive fields masked.
func RedactSSHConnection(c SSHConnection) SSHConnection {
	if c.Auth.Password != "" {
		c.Auth.Password = redacted
	}
	if c.Auth.PrivateKey != nil {
		pk := *c.Auth.PrivateKey
		if pk.KeyPEM != "" {
			pk.KeyPEM = redacted
		}
		if pk.Passphrase != "" {
			pk.Passphrase = redacted
		}
		c.Auth.PrivateKey = &pk
	}
	return c
}

// RedactTunnel returns a copy with sensitive fields masked.
func RedactTunnel(t Tunnel) Tunnel {
	mappings := make([]Mapping, len(t.Mappings))
	for i, m := range t.Mappings {
		if m.Socks5 != nil {
			s := *m.Socks5
			if s.Password != "" {
				s.Password = redacted
			}
			m.Socks5 = &s
		}
		mappings[i] = m
	}
	t.Mappings = mappings
	return t
}

// RedactConfig returns a full copy with all sensitive fields masked.
func RedactConfig(cfg Config) Config {
	out := Config{
		Version:        cfg.Version,
		SSHConnections: make([]SSHConnection, len(cfg.SSHConnections)),
		Tunnels:        make([]Tunnel, len(cfg.Tunnels)),
	}
	for i, c := range cfg.SSHConnections {
		out.SSHConnections[i] = RedactSSHConnection(c)
	}
	for i, t := range cfg.Tunnels {
		out.Tunnels[i] = RedactTunnel(t)
	}
	return out
}
```

- [ ] **Step 2: Verify compilation**

```bash
go build ./internal/config/
```

- [ ] **Step 3: Commit**

```bash
git add internal/config/redact.go
git commit -m "feat(config): add sensitive field redaction"
```

---

## Task 6: File Store

**Files:**
- Create: `internal/config/store.go`

- [ ] **Step 1: Create `internal/config/store.go`**

```go
package config

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

// Store defines config persistence operations.
type Store interface {
	Load(ctx context.Context) (*Config, error)
	Save(ctx context.Context, cfg *Config) error
}

// FileStore persists config as a JSON file with atomic writes.
type FileStore struct {
	path string
	mu   sync.RWMutex
}

// NewFileStore creates a store backed by the given file path.
func NewFileStore(path string) *FileStore {
	return &FileStore{path: path}
}

// Load reads config from disk. Returns empty config if file doesn't exist.
func (s *FileStore) Load(_ context.Context) (*Config, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return NewConfig(), nil
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Ensure slices are never nil after load
	if cfg.SSHConnections == nil {
		cfg.SSHConnections = []SSHConnection{}
	}
	if cfg.Tunnels == nil {
		cfg.Tunnels = []Tunnel{}
	}

	ApplyConfigDefaults(&cfg)
	return &cfg, nil
}

// Save writes config to disk atomically: write temp → fsync → rename.
func (s *FileStore) Save(_ context.Context, cfg *Config) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	tmpPath := s.path + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return err
	}

	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return err
	}

	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return err
	}

	if err := f.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}

	return os.Rename(tmpPath, s.path)
}
```

- [ ] **Step 2: Verify compilation**

```bash
go build ./internal/config/
```

- [ ] **Step 3: Commit**

```bash
git add internal/config/store.go
git commit -m "feat(config): add FileStore with atomic write"
```

---

## Task 7: Config Tests

**Files:**
- Create: `internal/config/model_test.go`
- Create: `internal/config/store_test.go`

- [ ] **Step 1: Create `internal/config/model_test.go`**

Tests for defaults, validation, and redaction:

```go
package config

import (
	"testing"
)

func TestApplySSHConnectionDefaults(t *testing.T) {
	c := &SSHConnection{}
	ApplySSHConnectionDefaults(c)

	if c.DialTimeoutMs != 10000 {
		t.Errorf("DialTimeoutMs = %d, want 10000", c.DialTimeoutMs)
	}
	if c.KeepAlive.IntervalMs != 15000 {
		t.Errorf("KeepAlive.IntervalMs = %d, want 15000", c.KeepAlive.IntervalMs)
	}
	if c.KeepAlive.MaxMissed != 3 {
		t.Errorf("KeepAlive.MaxMissed = %d, want 3", c.KeepAlive.MaxMissed)
	}
	if c.HostKeyVerification.Mode != HostKeyAcceptNew {
		t.Errorf("HostKeyVerification.Mode = %s, want acceptNew", c.HostKeyVerification.Mode)
	}
}

func TestApplySSHConnectionDefaults_NoOverwrite(t *testing.T) {
	c := &SSHConnection{DialTimeoutMs: 5000}
	ApplySSHConnectionDefaults(c)
	if c.DialTimeoutMs != 5000 {
		t.Errorf("DialTimeoutMs = %d, want 5000 (should not overwrite)", c.DialTimeoutMs)
	}
}

func TestApplyTunnelDefaults(t *testing.T) {
	tun := &Tunnel{
		Mappings: []Mapping{{Listen: Endpoint{Port: 8080}}},
	}
	ApplyTunnelDefaults(tun)

	if tun.Mappings[0].Listen.Host != "127.0.0.1" {
		t.Errorf("Listen.Host = %s, want 127.0.0.1", tun.Mappings[0].Listen.Host)
	}
	if tun.Policy.RestartBackoff.MinMs != 500 {
		t.Errorf("RestartBackoff.MinMs = %d, want 500", tun.Policy.RestartBackoff.MinMs)
	}
}

func TestValidateSSHConnection_Valid(t *testing.T) {
	c := &SSHConnection{
		ID:       "test",
		Name:     "test-conn",
		Endpoint: Endpoint{Host: "1.2.3.4", Port: 22},
		Auth:     Auth{Type: AuthPassword, Username: "user", Password: "pass"},
	}
	r := ValidateSSHConnection(c, "")
	if r.HasErrors() {
		t.Errorf("expected no errors, got: %v", r.Errors)
	}
}

func TestValidateSSHConnection_InvalidPort(t *testing.T) {
	c := &SSHConnection{
		ID:       "test",
		Name:     "test-conn",
		Endpoint: Endpoint{Host: "1.2.3.4", Port: 0},
		Auth:     Auth{Type: AuthNone},
	}
	r := ValidateSSHConnection(c, "")
	if !r.HasErrors() {
		t.Error("expected validation errors for port 0")
	}
}

func TestValidateSSHConnection_PrivateKeyInline(t *testing.T) {
	c := &SSHConnection{
		ID:       "test",
		Name:     "test-conn",
		Endpoint: Endpoint{Host: "1.2.3.4", Port: 22},
		Auth: Auth{
			Type:     AuthPrivateKey,
			Username: "user",
			PrivateKey: &PrivateKey{
				Source: KeySourceInline,
				KeyPEM: "-----BEGIN OPENSSH PRIVATE KEY-----\ntest\n-----END OPENSSH PRIVATE KEY-----",
			},
		},
	}
	r := ValidateSSHConnection(c, "")
	if r.HasErrors() {
		t.Errorf("expected no errors, got: %v", r.Errors)
	}
}

func TestValidateSSHConnection_PrivateKeyFileMissing(t *testing.T) {
	c := &SSHConnection{
		ID:       "test",
		Name:     "test-conn",
		Endpoint: Endpoint{Host: "1.2.3.4", Port: 22},
		Auth: Auth{
			Type:       AuthPrivateKey,
			Username:   "user",
			PrivateKey: &PrivateKey{Source: KeySourceFile, FilePath: ""},
		},
	}
	r := ValidateSSHConnection(c, "")
	if !r.HasErrors() {
		t.Error("expected validation error for empty filePath")
	}
}

func TestValidateTunnel_Valid(t *testing.T) {
	sshIDs := map[string]bool{"ssh-1": true}
	tun := &Tunnel{
		ID:   "tun-1",
		Name: "test-tunnel",
		Mode: ModeLocal,
		Chain: []string{"ssh-1"},
		Mappings: []Mapping{
			{ID: "m1", Listen: Endpoint{Host: "127.0.0.1", Port: 15432}, Connect: Endpoint{Host: "127.0.0.1", Port: 5432}},
		},
		Policy: Policy{
			RestartBackoff:     RestartBackoff{MinMs: 500, MaxMs: 15000, Factor: 1.7},
			MaxRestartsPerHour: 60,
		},
	}
	r := ValidateTunnel(tun, sshIDs, "")
	if r.HasErrors() {
		t.Errorf("expected no errors, got: %v", r.Errors)
	}
}

func TestValidateTunnel_MissingSSHRef(t *testing.T) {
	sshIDs := map[string]bool{}
	tun := &Tunnel{
		ID:    "tun-1",
		Name:  "test",
		Mode:  ModeLocal,
		Chain: []string{"nonexistent"},
		Mappings: []Mapping{
			{ID: "m1", Listen: Endpoint{Host: "127.0.0.1", Port: 15432}, Connect: Endpoint{Host: "127.0.0.1", Port: 5432}},
		},
		Policy: Policy{
			RestartBackoff:     RestartBackoff{MinMs: 500, MaxMs: 15000, Factor: 1.7},
			MaxRestartsPerHour: 60,
		},
	}
	r := ValidateTunnel(tun, sshIDs, "")
	if !r.HasErrors() {
		t.Error("expected validation error for missing SSH ref")
	}
}

func TestValidateTunnel_DynamicWarning(t *testing.T) {
	sshIDs := map[string]bool{"ssh-1": true}
	tun := &Tunnel{
		ID:    "tun-1",
		Name:  "test",
		Mode:  ModeDynamic,
		Chain: []string{"ssh-1"},
		Mappings: []Mapping{
			{
				ID:     "m1",
				Listen: Endpoint{Host: "0.0.0.0", Port: 1080},
				Socks5: &Socks5Cfg{Auth: Socks5None},
			},
		},
		Policy: Policy{
			RestartBackoff:     RestartBackoff{MinMs: 500, MaxMs: 15000, Factor: 1.7},
			MaxRestartsPerHour: 60,
		},
	}
	r := ValidateTunnel(tun, sshIDs, "")
	if r.HasErrors() {
		t.Errorf("expected no errors, got: %v", r.Errors)
	}
	if len(r.Warnings) == 0 {
		t.Error("expected SOCKS5 security warning")
	}
}

func TestRedactSSHConnection(t *testing.T) {
	c := SSHConnection{
		ID:   "test",
		Auth: Auth{Type: AuthPassword, Username: "user", Password: "secret123"},
	}
	redacted := RedactSSHConnection(c)
	if redacted.Auth.Password != "***" {
		t.Errorf("Password = %s, want ***", redacted.Auth.Password)
	}
	// Original unchanged
	if c.Auth.Password != "secret123" {
		t.Error("original password was mutated")
	}
}

func TestRedactSSHConnection_PrivateKey(t *testing.T) {
	c := SSHConnection{
		ID: "test",
		Auth: Auth{
			Type: AuthPrivateKey,
			PrivateKey: &PrivateKey{
				Source:     KeySourceInline,
				KeyPEM:     "-----BEGIN KEY-----",
				Passphrase: "mypass",
			},
		},
	}
	redacted := RedactSSHConnection(c)
	if redacted.Auth.PrivateKey.KeyPEM != "***" {
		t.Errorf("KeyPEM = %s, want ***", redacted.Auth.PrivateKey.KeyPEM)
	}
	if redacted.Auth.PrivateKey.Passphrase != "***" {
		t.Errorf("Passphrase = %s, want ***", redacted.Auth.PrivateKey.Passphrase)
	}
}

func TestRedactTunnel_Socks5(t *testing.T) {
	tun := Tunnel{
		ID: "test",
		Mappings: []Mapping{
			{
				ID:     "m1",
				Socks5: &Socks5Cfg{Auth: Socks5UserPass, Username: "user", Password: "secret"},
			},
		},
	}
	redacted := RedactTunnel(tun)
	if redacted.Mappings[0].Socks5.Password != "***" {
		t.Errorf("Socks5.Password = %s, want ***", redacted.Mappings[0].Socks5.Password)
	}
	// Original unchanged
	if tun.Mappings[0].Socks5.Password != "secret" {
		t.Error("original socks5 password was mutated")
	}
}

func TestFindSSHConnectionReferences(t *testing.T) {
	cfg := &Config{
		Tunnels: []Tunnel{
			{Name: "tun-a", Chain: []string{"ssh-1", "ssh-2"}},
			{Name: "tun-b", Chain: []string{"ssh-2"}},
			{Name: "tun-c", Chain: []string{"ssh-3"}},
		},
	}
	refs := FindSSHConnectionReferences(cfg, "ssh-2")
	if len(refs) != 2 {
		t.Errorf("expected 2 references, got %d", len(refs))
	}
}
```

- [ ] **Step 2: Create `internal/config/store_test.go`**

```go
package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFileStore_LoadEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	store := NewFileStore(path)

	cfg, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if cfg.Version != 1 {
		t.Errorf("Version = %d, want 1", cfg.Version)
	}
	if len(cfg.SSHConnections) != 0 {
		t.Errorf("SSHConnections should be empty, got %d", len(cfg.SSHConnections))
	}
	if len(cfg.Tunnels) != 0 {
		t.Errorf("Tunnels should be empty, got %d", len(cfg.Tunnels))
	}
}

func TestFileStore_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	store := NewFileStore(path)
	ctx := context.Background()

	cfg := NewConfig()
	cfg.SSHConnections = append(cfg.SSHConnections, SSHConnection{
		ID:       "ssh-1",
		Name:     "test-conn",
		Endpoint: Endpoint{Host: "1.2.3.4", Port: 22},
		Auth:     Auth{Type: AuthPassword, Username: "user", Password: "secret"},
	})

	if err := store.Save(ctx, cfg); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("config file not created: %v", err)
	}

	// Load it back
	loaded, err := store.Load(ctx)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if len(loaded.SSHConnections) != 1 {
		t.Fatalf("expected 1 SSH connection, got %d", len(loaded.SSHConnections))
	}
	if loaded.SSHConnections[0].Auth.Password != "secret" {
		t.Error("password not persisted correctly")
	}
	// Defaults should be applied
	if loaded.SSHConnections[0].DialTimeoutMs != 10000 {
		t.Errorf("defaults not applied: DialTimeoutMs = %d", loaded.SSHConnections[0].DialTimeoutMs)
	}
}

func TestFileStore_AtomicWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	store := NewFileStore(path)
	ctx := context.Background()

	// Save initial
	cfg := NewConfig()
	if err := store.Save(ctx, cfg); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	// Verify no .tmp file left behind
	tmpPath := path + ".tmp"
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Error("temp file should not exist after successful save")
	}
}
```

- [ ] **Step 3: Run tests**

```bash
cd /Users/maxzhang/Tools/ops-tunnel
go test ./internal/config/ -v
```

Expected: all tests pass.

- [ ] **Step 4: Commit**

```bash
git add internal/config/model_test.go internal/config/store_test.go
git commit -m "test(config): add tests for model, validation, redaction, and store"
```

---

## Task 8: API Response Helpers

**Files:**
- Create: `internal/api/response.go`

- [ ] **Step 1: Create `internal/api/response.go`**

```go
package api

import (
	"encoding/json"
	"net/http"

	"github.com/maxzhang666/ops-tunnel/internal/config"
)

// ErrorResponse is the standard error envelope.
type ErrorResponse struct {
	Error   string                   `json:"error"`
	Details []config.ValidationError `json:"details,omitempty"`
}

// DataResponse wraps a successful response, optionally with warnings.
type DataResponse struct {
	Data     any      `json:"data"`
	Warnings []string `json:"warnings,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeData(w http.ResponseWriter, status int, data any, warnings []string) {
	if len(warnings) > 0 {
		writeJSON(w, status, DataResponse{Data: data, Warnings: warnings})
	} else {
		writeJSON(w, status, data)
	}
}

func writeValidationError(w http.ResponseWriter, errs []config.ValidationError) {
	writeJSON(w, http.StatusBadRequest, ErrorResponse{
		Error:   "validation_failed",
		Details: errs,
	})
}

func writeNotFound(w http.ResponseWriter, resource, id string) {
	writeJSON(w, http.StatusNotFound, ErrorResponse{
		Error:   "not_found",
		Details: []config.ValidationError{{Field: resource, Message: "'" + id + "' not found"}},
	})
}

func writeConflict(w http.ResponseWriter, details []config.ValidationError) {
	writeJSON(w, http.StatusConflict, ErrorResponse{
		Error:   "conflict",
		Details: details,
	})
}

func writeInternalError(w http.ResponseWriter) {
	writeJSON(w, http.StatusInternalServerError, ErrorResponse{
		Error: "internal_error",
	})
}

func decodeBody(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}
```

- [ ] **Step 2: Verify compilation**

```bash
go build ./internal/api/
```

- [ ] **Step 3: Commit**

```bash
git add internal/api/response.go
git commit -m "feat(api): add shared JSON response helpers"
```

---

## Task 9: SSH Connection Handlers

**Files:**
- Create: `internal/api/handler_ssh.go`
- Modify: `internal/api/server.go` — add store and config fields
- Modify: `internal/api/routes.go` — register SSH routes

- [ ] **Step 1: Update `internal/api/server.go`**

Replace the full content:

```go
package api

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/maxzhang666/ops-tunnel/internal/config"
)

// ServerConfig holds HTTP server settings (renamed to avoid clash with config.Config).
type ServerConfig struct {
	ListenAddr string
	UIDir      string
	Token      string
}

// Server is the HTTP API server.
type Server struct {
	cfg    ServerConfig
	store  config.Store
	mu     sync.RWMutex
	data   *config.Config
	router chi.Router
	http   *http.Server
}

// NewServer creates an API server with the given config store.
// The caller must load the config and pass it in.
func NewServer(cfg ServerConfig, store config.Store, data *config.Config) *Server {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	s := &Server{
		cfg:    cfg,
		store:  store,
		data:   data,
		router: r,
	}
	s.registerRoutes()
	return s
}

// saveConfig validates and persists the current in-memory config.
// Caller must hold s.mu write lock.
func (s *Server) saveConfig(ctx context.Context) (*config.ValidationResult, error) {
	vr := config.ValidateConfig(s.data)
	if vr.HasErrors() {
		return vr, nil
	}
	if err := s.store.Save(ctx, s.data); err != nil {
		return nil, err
	}
	return vr, nil
}

func (s *Server) Run(ctx context.Context) error {
	s.http = &http.Server{
		Addr:    s.cfg.ListenAddr,
		Handler: s.router,
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
	}

	slog.Info("server starting", "addr", s.cfg.ListenAddr)
	if err := s.http.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.http != nil {
		return s.http.Shutdown(ctx)
	}
	return nil
}
```

- [ ] **Step 2: Create `internal/api/handler_ssh.go`**

```go
package api

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/xid"
	"github.com/maxzhang666/ops-tunnel/internal/config"
)

func (s *Server) listSSHConnections(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	conns := make([]config.SSHConnection, len(s.data.SSHConnections))
	for i, c := range s.data.SSHConnections {
		conns[i] = config.RedactSSHConnection(c)
	}
	s.mu.RUnlock()

	writeJSON(w, http.StatusOK, conns)
}

func (s *Server) createSSHConnection(w http.ResponseWriter, r *http.Request) {
	var conn config.SSHConnection
	if err := decodeBody(r, &conn); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid_json", Details: []config.ValidationError{{Field: "body", Message: err.Error()}}})
		return
	}

	conn.ID = xid.New().String()
	config.ApplySSHConnectionDefaults(&conn)

	s.mu.Lock()
	defer s.mu.Unlock()

	s.data.SSHConnections = append(s.data.SSHConnections, conn)

	vr, err := s.saveConfig(r.Context())
	if err != nil {
		// Rollback
		s.data.SSHConnections = s.data.SSHConnections[:len(s.data.SSHConnections)-1]
		slog.Error("failed to save config", "err", err)
		writeInternalError(w)
		return
	}
	if vr.HasErrors() {
		s.data.SSHConnections = s.data.SSHConnections[:len(s.data.SSHConnections)-1]
		writeValidationError(w, vr.Errors)
		return
	}

	writeData(w, http.StatusCreated, config.RedactSSHConnection(conn), vr.Warnings)
}

func (s *Server) getSSHConnection(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, c := range s.data.SSHConnections {
		if c.ID == id {
			writeJSON(w, http.StatusOK, config.RedactSSHConnection(c))
			return
		}
	}
	writeNotFound(w, "ssh-connection", id)
}

func (s *Server) updateSSHConnection(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var conn config.SSHConnection
	if err := decodeBody(r, &conn); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid_json", Details: []config.ValidationError{{Field: "body", Message: err.Error()}}})
		return
	}
	conn.ID = id
	config.ApplySSHConnectionDefaults(&conn)

	s.mu.Lock()
	defer s.mu.Unlock()

	idx := -1
	for i, c := range s.data.SSHConnections {
		if c.ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		writeNotFound(w, "ssh-connection", id)
		return
	}

	old := s.data.SSHConnections[idx]
	s.data.SSHConnections[idx] = conn

	vr, err := s.saveConfig(r.Context())
	if err != nil {
		s.data.SSHConnections[idx] = old
		slog.Error("failed to save config", "err", err)
		writeInternalError(w)
		return
	}
	if vr.HasErrors() {
		s.data.SSHConnections[idx] = old
		writeValidationError(w, vr.Errors)
		return
	}

	writeData(w, http.StatusOK, config.RedactSSHConnection(conn), vr.Warnings)
}

func (s *Server) deleteSSHConnection(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	s.mu.Lock()
	defer s.mu.Unlock()

	idx := -1
	for i, c := range s.data.SSHConnections {
		if c.ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		writeNotFound(w, "ssh-connection", id)
		return
	}

	// Check references
	refs := config.FindSSHConnectionReferences(s.data, id)
	if len(refs) > 0 {
		details := make([]config.ValidationError, len(refs))
		for i, name := range refs {
			details[i] = config.ValidationError{Field: "tunnel", Message: "referenced by tunnel '" + name + "'"}
		}
		writeConflict(w, details)
		return
	}

	s.data.SSHConnections = append(s.data.SSHConnections[:idx], s.data.SSHConnections[idx+1:]...)

	if _, err := s.saveConfig(r.Context()); err != nil {
		slog.Error("failed to save config", "err", err)
		writeInternalError(w)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
```

- [ ] **Step 3: Update `internal/api/routes.go`**

Replace the full content:

```go
package api

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func (s *Server) registerRoutes() {
	s.router.Get("/healthz", s.handleHealthz)

	// SSH Connection CRUD
	s.router.Route("/api/v1/ssh-connections", func(r chi.Router) {
		r.Get("/", s.listSSHConnections)
		r.Post("/", s.createSSHConnection)
		r.Get("/{id}", s.getSSHConnection)
		r.Put("/{id}", s.updateSSHConnection)
		r.Delete("/{id}", s.deleteSSHConnection)
	})

	// Tunnel CRUD
	s.router.Route("/api/v1/tunnels", func(r chi.Router) {
		r.Get("/", s.listTunnels)
		r.Post("/", s.createTunnel)
		r.Get("/{id}", s.getTunnel)
		r.Put("/{id}", s.updateTunnel)
		r.Delete("/{id}", s.deleteTunnel)
	})

	if s.cfg.UIDir != "" {
		s.serveSPA(s.cfg.UIDir)
	}
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status": "ok",
		"ts":     time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) serveSPA(dir string) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return
	}

	fileServer := http.FileServer(http.Dir(absDir))

	s.router.NotFound(func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join(absDir, r.URL.Path)
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			fileServer.ServeHTTP(w, r)
			return
		}

		indexPath := filepath.Join(absDir, "index.html")
		if _, err := fs.Stat(os.DirFS(absDir), "index.html"); err == nil {
			http.ServeFile(w, r, indexPath)
			return
		}

		http.NotFound(w, r)
	})
}
```

Note: `routes.go` now needs access to chi.Router type directly for `Route()`. Add this import at the top:

```go
import (
	"encoding/json"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/go-chi/chi/v5"
)
```

- [ ] **Step 4: Verify compilation**

```bash
go build ./internal/api/
```

This will fail because `listTunnels`, `createTunnel`, etc. don't exist yet. That's expected — they're in the next task.

- [ ] **Step 5: Commit (partial — SSH handlers only)**

```bash
git add internal/api/server.go internal/api/handler_ssh.go internal/api/response.go internal/api/routes.go
git commit -m "feat(api): add SSH connection CRUD handlers and response helpers"
```

---

## Task 10: Tunnel Handlers

**Files:**
- Create: `internal/api/handler_tunnel.go`

- [ ] **Step 1: Create `internal/api/handler_tunnel.go`**

```go
package api

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/xid"
	"github.com/maxzhang666/ops-tunnel/internal/config"
)

func (s *Server) listTunnels(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	tunnels := make([]config.Tunnel, len(s.data.Tunnels))
	for i, t := range s.data.Tunnels {
		tunnels[i] = config.RedactTunnel(t)
	}
	s.mu.RUnlock()

	writeJSON(w, http.StatusOK, tunnels)
}

func (s *Server) createTunnel(w http.ResponseWriter, r *http.Request) {
	var tun config.Tunnel
	if err := decodeBody(r, &tun); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid_json", Details: []config.ValidationError{{Field: "body", Message: err.Error()}}})
		return
	}

	tun.ID = xid.New().String()
	// Auto-generate mapping IDs if empty
	for i := range tun.Mappings {
		if tun.Mappings[i].ID == "" {
			tun.Mappings[i].ID = xid.New().String()
		}
	}
	config.ApplyTunnelDefaults(&tun)

	s.mu.Lock()
	defer s.mu.Unlock()

	s.data.Tunnels = append(s.data.Tunnels, tun)

	vr, err := s.saveConfig(r.Context())
	if err != nil {
		s.data.Tunnels = s.data.Tunnels[:len(s.data.Tunnels)-1]
		slog.Error("failed to save config", "err", err)
		writeInternalError(w)
		return
	}
	if vr.HasErrors() {
		s.data.Tunnels = s.data.Tunnels[:len(s.data.Tunnels)-1]
		writeValidationError(w, vr.Errors)
		return
	}

	writeData(w, http.StatusCreated, config.RedactTunnel(tun), vr.Warnings)
}

func (s *Server) getTunnel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, t := range s.data.Tunnels {
		if t.ID == id {
			writeJSON(w, http.StatusOK, config.RedactTunnel(t))
			return
		}
	}
	writeNotFound(w, "tunnel", id)
}

func (s *Server) updateTunnel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var tun config.Tunnel
	if err := decodeBody(r, &tun); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid_json", Details: []config.ValidationError{{Field: "body", Message: err.Error()}}})
		return
	}
	tun.ID = id
	// Auto-generate mapping IDs if empty
	for i := range tun.Mappings {
		if tun.Mappings[i].ID == "" {
			tun.Mappings[i].ID = xid.New().String()
		}
	}
	config.ApplyTunnelDefaults(&tun)

	s.mu.Lock()
	defer s.mu.Unlock()

	idx := -1
	for i, t := range s.data.Tunnels {
		if t.ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		writeNotFound(w, "tunnel", id)
		return
	}

	old := s.data.Tunnels[idx]
	s.data.Tunnels[idx] = tun

	vr, err := s.saveConfig(r.Context())
	if err != nil {
		s.data.Tunnels[idx] = old
		slog.Error("failed to save config", "err", err)
		writeInternalError(w)
		return
	}
	if vr.HasErrors() {
		s.data.Tunnels[idx] = old
		writeValidationError(w, vr.Errors)
		return
	}

	writeData(w, http.StatusOK, config.RedactTunnel(tun), vr.Warnings)
}

func (s *Server) deleteTunnel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	s.mu.Lock()
	defer s.mu.Unlock()

	idx := -1
	for i, t := range s.data.Tunnels {
		if t.ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		writeNotFound(w, "tunnel", id)
		return
	}

	s.data.Tunnels = append(s.data.Tunnels[:idx], s.data.Tunnels[idx+1:]...)

	if _, err := s.saveConfig(r.Context()); err != nil {
		slog.Error("failed to save config", "err", err)
		writeInternalError(w)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
```

- [ ] **Step 2: Verify compilation**

```bash
go build ./internal/api/
```

Expected: success (all handler functions now defined).

- [ ] **Step 3: Commit**

```bash
git add internal/api/handler_tunnel.go
git commit -m "feat(api): add Tunnel CRUD handlers"
```

---

## Task 11: Update main.go Integration

**Files:**
- Modify: `cmd/tunnel-server/main.go`

- [ ] **Step 1: Update `cmd/tunnel-server/main.go`**

Replace the full content:

```go
package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/maxzhang666/ops-tunnel/internal/api"
	"github.com/maxzhang666/ops-tunnel/internal/config"
)

func main() {
	listen := flag.String("listen", "127.0.0.1:8080", "HTTP listen address")
	dataDir := flag.String("data-dir", "./data", "data directory")
	uiDir := flag.String("ui-dir", "", "static UI files directory")
	token := flag.String("token", "", "bearer token for API auth")
	flag.Parse()

	if err := os.MkdirAll(*dataDir, 0o755); err != nil {
		slog.Error("failed to create data dir", "path", *dataDir, "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	store := config.NewFileStore(filepath.Join(*dataDir, "config.json"))
	data, err := store.Load(ctx)
	if err != nil {
		slog.Error("failed to load config", "err", err)
		os.Exit(1)
	}

	// Save on first run to create the config file
	if err := store.Save(ctx, data); err != nil {
		slog.Error("failed to save initial config", "err", err)
		os.Exit(1)
	}

	slog.Info("config loaded",
		"sshConnections", len(data.SSHConnections),
		"tunnels", len(data.Tunnels),
	)

	srv := api.NewServer(api.ServerConfig{
		ListenAddr: *listen,
		UIDir:      *uiDir,
		Token:      *token,
	}, store, data)

	go func() {
		if err := srv.Run(ctx); err != nil {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down...")

	shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutCtx); err != nil {
		slog.Error("shutdown error", "err", err)
	}
	slog.Info("server stopped")
}
```

- [ ] **Step 2: Build and verify**

```bash
cd /Users/maxzhang/Tools/ops-tunnel
go build -o bin/tunnel-server ./cmd/tunnel-server
```

- [ ] **Step 3: Commit**

```bash
git add cmd/tunnel-server/main.go
git commit -m "feat: integrate config store into server startup"
```

---

## Task 12: End-to-End Verification

- [ ] **Step 1: Run all tests**

```bash
cd /Users/maxzhang/Tools/ops-tunnel
go test ./... -v
```

Expected: all tests pass.

- [ ] **Step 2: Start server and test CRUD**

```bash
rm -rf data/
bin/tunnel-server --listen 127.0.0.1:8080 &
SERVER_PID=$!
sleep 1
```

Verify config file auto-created:
```bash
cat data/config.json
```
Expected: `{"version":1,"sshConnections":[],"tunnels":[]}`

Create SSH connection:
```bash
curl -s -X POST http://127.0.0.1:8080/api/v1/ssh-connections \
  -H "Content-Type: application/json" \
  -d '{"name":"test-server","endpoint":{"host":"1.2.3.4","port":22},"auth":{"type":"password","username":"root","password":"secret123"}}'
```
Expected: 201, ID auto-generated, password redacted as `"***"`.

List SSH connections:
```bash
curl -s http://127.0.0.1:8080/api/v1/ssh-connections
```
Expected: array with 1 item, password is `"***"`.

Create tunnel referencing it (use the ID from above, e.g., replace `SSH_ID`):
```bash
curl -s -X POST http://127.0.0.1:8080/api/v1/tunnels \
  -H "Content-Type: application/json" \
  -d '{"name":"test-tunnel","mode":"local","chain":["SSH_ID"],"mappings":[{"listen":{"port":15432},"connect":{"host":"127.0.0.1","port":5432}}]}'
```
Expected: 201, tunnel created.

Test referential integrity — delete SSH connection:
```bash
curl -s -X DELETE http://127.0.0.1:8080/api/v1/ssh-connections/SSH_ID
```
Expected: 409 conflict (referenced by tunnel).

Test validation — invalid port:
```bash
curl -s -X POST http://127.0.0.1:8080/api/v1/ssh-connections \
  -H "Content-Type: application/json" \
  -d '{"name":"bad","endpoint":{"host":"1.2.3.4","port":0},"auth":{"type":"password","username":"root","password":"x"}}'
```
Expected: 400 with `{"error":"validation_failed","details":[...]}`.

```bash
kill $SERVER_PID
```

- [ ] **Step 3: Verify persistence**

```bash
bin/tunnel-server --listen 127.0.0.1:8080 &
SERVER_PID=$!
sleep 1
curl -s http://127.0.0.1:8080/api/v1/ssh-connections
kill $SERVER_PID
```
Expected: the SSH connection created earlier still exists after restart.

- [ ] **Step 4: Commit docs update**

```bash
git add docs/
git commit -m "docs: add Phase 1 spec and plan"
```
