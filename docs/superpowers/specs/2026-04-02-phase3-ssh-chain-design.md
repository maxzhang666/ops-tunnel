# Phase 3: SSH Multi-Hop Chain + Test Connection

## Goal

Implement real SSH connections: single-hop and multi-hop chains (hop1 → hop2 → ... → last), each hop with independent authentication. Include a "Test Connection" API endpoint. Upgrade the supervisor from stub to real SSH.

## Scope

**In scope:**
- `internal/ssh/auth.go`: config.Auth → []ssh.AuthMethod
- `internal/ssh/hostkey.go`: HostKeyCallback factory + JSON-based known hosts store
- `internal/ssh/keepalive.go`: KeepAlive goroutine
- `internal/ssh/chain.go`: BuildChain — sequential multi-hop connection builder
- `internal/ssh/test.go`: TestConnection — single SSH connection test
- `internal/api/handler_ssh.go`: Add `POST /ssh-connections/{id}/test` endpoint
- `internal/engine/supervisor.go`: Upgrade Start/Stop to use real SSH chain

**Out of scope:**
- Port forwarding (Phase 4-6)
- Auto-reconnect/backoff (Phase 7)

## SSH Auth (`internal/ssh/auth.go`)

```go
func AuthMethods(a config.Auth) ([]ssh.AuthMethod, error)
```

- `password`: returns `ssh.Password(a.Password)`
- `privateKey` + inline: parse PEM with `ssh.ParsePrivateKey` (or `ssh.ParsePrivateKeyWithPassphrase`)
- `privateKey` + file: read file, then parse
- `none`: returns empty slice

## Host Key (`internal/ssh/hostkey.go`)

```go
type HostKeyStore interface {
    Lookup(hostport string) ([]byte, bool)
    Add(hostport string, key []byte) error
}

func NewJSONHostKeyStore(path string) HostKeyStore

func HostKeyCallback(mode config.HostKeyVerifyMode, store HostKeyStore, hostport string) ssh.HostKeyCallback
```

- `insecure`: accept any key (`ssh.InsecureIgnoreHostKey()`)
- `acceptNew`: if no stored key → store and accept; if stored → compare
- `strict`: must match stored key, reject if missing

Store implementation: JSON file `data/known_hosts.json` as `map[string]string` (hostport → base64 key fingerprint). Atomic write like config store.

## KeepAlive (`internal/ssh/keepalive.go`)

```go
func StartKeepAlive(ctx context.Context, client *ssh.Client, interval time.Duration, maxMissed int) <-chan error
```

- Goroutine sends `keepalive@openssh.com` global request at `interval`
- Counts consecutive failures
- If failures >= maxMissed, sends error on returned channel and stops
- Cancelled by ctx

## Chain Builder (`internal/ssh/chain.go`)

```go
type ChainResult struct {
    Clients []*ssh.Client  // all clients, ordered hop1..hopN
}

func (r *ChainResult) Last() *ssh.Client  // returns the last client (target)
func (r *ChainResult) Close() error       // closes all clients in reverse order

func BuildChain(ctx context.Context, conns []config.SSHConnection, hostKeyStore HostKeyStore, bus engine.EventBus) (*ChainResult, error)
```

### Algorithm:
1. First hop: `net.DialTimeout("tcp", hop1.endpoint, timeout)` → `ssh.NewClientConn` → `ssh.NewClient`
2. Each subsequent hop: `prevClient.Dial("tcp", nextHop.endpoint)` → `ssh.NewClientConn` → `ssh.NewClient`
3. Each hop: publish `EventChainConnected` on success or `EventChainError` on failure
4. On failure at any hop: close all previously established clients, return error
5. After each successful connection: start KeepAlive goroutine

### Context cancellation:
- `BuildChain` respects ctx — if cancelled mid-chain, cleanup and return

## Test Connection (`internal/ssh/test.go`)

```go
func TestConnection(ctx context.Context, conn config.SSHConnection, hostKeyStore HostKeyStore) error
```

- Dial → authenticate → close
- Returns nil on success, error with details on failure
- Timeout from `conn.DialTimeoutMs`

## API: Test Connection Endpoint

`POST /api/v1/ssh-connections/{id}/test`

Response:
```json
// Success
{"status": "ok", "message": "connected successfully", "latencyMs": 120}

// Failure  
{"status": "error", "message": "auth failed: password rejected"}
```

## Supervisor Upgrade

Replace the Phase 2 stub with real SSH chain management:

```go
type tunnelSupervisor struct {
    tunnel    config.Tunnel
    conns     []config.SSHConnection  // resolved from chain IDs
    bus       engine.EventBus
    hostKeys  ssh.HostKeyStore
    
    mu        sync.RWMutex
    state     TunnelState
    since     time.Time
    lastErr   string
    chain     *ssh.ChainResult        // nil when stopped
    cancelKA  context.CancelFunc      // cancel keepalive goroutines
}
```

- `Start()`: resolve chain IDs → BuildChain → store result → set state running
- `Stop()`: cancel keepalives → chain.Close() → set state stopped
- `Status()`: populate HopStatus from chain state (connected/disconnected)

### Engine changes:
- `NewEngine` takes additional `hostKeyStorePath string` parameter
- Creates `ssh.NewJSONHostKeyStore(path)`
- Resolves SSHConnection IDs to actual objects when creating supervisors
- Passes resolved connections to supervisor

## Acceptance Criteria

1. Single SSH connection (password auth) → start tunnel → status shows chain hop as "connected"
2. Single SSH connection (privateKey file) → start tunnel → chain connected
3. Multi-hop: 2+ SSH connections in chain → all hops connected sequentially
4. Wrong password → start fails, status shows error with detail
5. Unreachable host → start fails with timeout error
6. `POST /ssh-connections/{id}/test` → returns latency on success, error message on failure
7. KeepAlive: connection stays alive > 1 minute under idle
8. Stop tunnel → all SSH clients closed
9. Host key `acceptNew`: first connect stores key, second connect with different key rejects (strict mode)
