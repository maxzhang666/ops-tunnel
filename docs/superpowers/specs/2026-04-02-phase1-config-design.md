# Phase 1: Config Model + Storage + Validation + CRUD API

## Goal

Define the complete data model (SSHConnection + Tunnel), implement JSON file storage with atomic write, validation, sensitive field redaction, and REST CRUD endpoints for both entities.

## Scope

**In scope:**
- `internal/config/` package: model, defaults, validate, store, redact
- `internal/api/handler_ssh.go`: SSH Connection CRUD + reference check on delete
- `internal/api/handler_tunnel.go`: Tunnel CRUD
- Integration: `api.Server` receives `config.Store`, `main.go` initializes it

**Out of scope:**
- Engine, supervisor, SSH connections, forwarding (Phase 2+)
- `POST /ssh-connections/{id}/test` (Phase 3, needs SSH package)
- WebSocket, events (Phase 2)

## Data Model

### Primitives

```go
type Endpoint struct {
    Host string `json:"host"`
    Port int    `json:"port"`
}

type AuthType string

const (
    AuthPassword   AuthType = "password"
    AuthPrivateKey AuthType = "privateKey"
    AuthNone       AuthType = "none"
)

type PrivateKeySource string

const (
    KeySourceInline PrivateKeySource = "inline"
    KeySourceFile   PrivateKeySource = "file"
)

type PrivateKey struct {
    Source     PrivateKeySource `json:"source"`
    KeyPEM     string          `json:"keyPem,omitempty"`
    FilePath   string          `json:"filePath,omitempty"`
    Passphrase string          `json:"passphrase,omitempty"`
}

type Auth struct {
    Type       AuthType    `json:"type"`
    Username   string      `json:"username"`
    Password   string      `json:"password,omitempty"`
    PrivateKey *PrivateKey `json:"privateKey,omitempty"`
}

type HostKeyVerifyMode string

const (
    HostKeyInsecure  HostKeyVerifyMode = "insecure"
    HostKeyAcceptNew HostKeyVerifyMode = "acceptNew"
    HostKeyStrict    HostKeyVerifyMode = "strict"
)

type HostKeyVerification struct {
    Mode HostKeyVerifyMode `json:"mode"`
}

type KeepAlive struct {
    IntervalMs int `json:"intervalMs"`
    MaxMissed  int `json:"maxMissed"`
}
```

### SSHConnection

```go
type SSHConnection struct {
    ID                  string              `json:"id"`
    Name                string              `json:"name"`
    Endpoint            Endpoint            `json:"endpoint"`
    Auth                Auth                `json:"auth"`
    HostKeyVerification HostKeyVerification `json:"hostKeyVerification"`
    DialTimeoutMs       int                 `json:"dialTimeoutMs"`
    KeepAlive           KeepAlive           `json:"keepAlive"`
}
```

### Tunnel

```go
type TunnelMode string

const (
    ModeLocal   TunnelMode = "local"
    ModeRemote  TunnelMode = "remote"
    ModeDynamic TunnelMode = "dynamic"
)

type Socks5Auth string

const (
    Socks5None     Socks5Auth = "none"
    Socks5UserPass Socks5Auth = "userpass"
)

type Socks5Cfg struct {
    Auth       Socks5Auth `json:"auth"`
    Username   string     `json:"username,omitempty"`
    Password   string     `json:"password,omitempty"`
    AllowCIDRs []string   `json:"allowCIDRs,omitempty"`
    DenyCIDRs  []string   `json:"denyCIDRs,omitempty"`
}

type Mapping struct {
    ID      string     `json:"id"`
    Listen  Endpoint   `json:"listen"`
    Connect Endpoint   `json:"connect,omitempty"`
    Socks5  *Socks5Cfg `json:"socks5,omitempty"`
    Notes   string     `json:"notes,omitempty"`
}

type RestartBackoff struct {
    MinMs  int     `json:"minMs"`
    MaxMs  int     `json:"maxMs"`
    Factor float64 `json:"factor"`
}

type Policy struct {
    AutoStart             bool           `json:"autoStart"`
    AutoRestart           bool           `json:"autoRestart"`
    RestartBackoff        RestartBackoff `json:"restartBackoff"`
    MaxRestartsPerHour    int            `json:"maxRestartsPerHour"`
    GracefulStopTimeoutMs int            `json:"gracefulStopTimeoutMs"`
}

type Tunnel struct {
    ID       string     `json:"id"`
    Name     string     `json:"name"`
    Mode     TunnelMode `json:"mode"`
    Chain    []string   `json:"chain"`
    Mappings []Mapping  `json:"mappings"`
    Policy   Policy     `json:"policy"`
}
```

### Config

```go
type Config struct {
    Version        int              `json:"version"`
    SSHConnections []SSHConnection  `json:"sshConnections"`
    Tunnels        []Tunnel         `json:"tunnels"`
}
```

## Defaults (`internal/config/defaults.go`)

When creating/loading, fill missing values:

| Field | Default |
|-------|---------|
| SSHConnection.DialTimeoutMs | 10000 |
| SSHConnection.KeepAlive.IntervalMs | 15000 |
| SSHConnection.KeepAlive.MaxMissed | 3 |
| SSHConnection.HostKeyVerification.Mode | "acceptNew" |
| Mapping.Listen.Host | "127.0.0.1" |
| Policy.AutoRestart | true |
| Policy.RestartBackoff.MinMs | 500 |
| Policy.RestartBackoff.MaxMs | 15000 |
| Policy.RestartBackoff.Factor | 1.7 |
| Policy.MaxRestartsPerHour | 60 |
| Policy.GracefulStopTimeoutMs | 5000 |

## Validation (`internal/config/validate.go`)

Returns `[]ValidationError` where each has `Field` and `Message`.

Separate from errors, validation can return `[]ValidationWarning` for non-blocking issues.

### SSHConnection rules:
- `id`: non-empty, unique across all SSH connections
- `name`: non-empty
- `endpoint.host`: non-empty
- `endpoint.port`: 1-65535
- `auth.type`: must be "password", "privateKey", or "none"
- `auth.username`: non-empty (except for type "none")
- If type "password": `auth.password` non-empty
- If type "privateKey": `auth.privateKey` must exist
  - If source "inline": `keyPem` non-empty
  - If source "file": `filePath` non-empty

### Tunnel rules:
- `id`: non-empty, unique across all tunnels
- `name`: non-empty
- `mode`: must be "local", "remote", or "dynamic"
- `chain`: non-empty array; every ID must reference an existing SSHConnection
- `mappings`: at least one mapping
- Each mapping `id`: non-empty, unique within tunnel
- Each mapping validated per mode:
  - **local**: `listen.port` required (1-65535), `connect.host` non-empty, `connect.port` required (1-65535)
  - **remote**: same as local
  - **dynamic**: `listen.port` required (1-65535), `socks5` must exist, `socks5.auth` must be "none" or "userpass"
    - If "userpass": username and password non-empty
- Policy: `restartBackoff.minMs` > 0, `maxMs` >= `minMs`, `factor` >= 1.0, `maxRestartsPerHour` > 0

### Warnings (non-blocking):
- Dynamic mapping: listen.host is "0.0.0.0" AND socks5.auth is "none" AND (allowCIDRs is empty OR contains "0.0.0.0/0") → "SOCKS5 proxy is publicly accessible without authentication"

## Storage (`internal/config/store.go`)

```go
type Store interface {
    Load(ctx context.Context) (*Config, error)
    Save(ctx context.Context, cfg *Config) error
}

type FileStore struct {
    path string
    mu   sync.RWMutex
}

func NewFileStore(path string) *FileStore
```

### Behaviors:
- **Load**: If file doesn't exist, return empty config `{version:1, sshConnections:[], tunnels:[]}`. If file exists, read and unmarshal. Apply defaults after load.
- **Save**: Marshal with indent → write to `<path>.tmp` → fsync → rename to `<path>`. Hold write lock during save.
- **Concurrency**: `sync.RWMutex` — read lock on Load, write lock on Save.

## Redaction (`internal/config/redact.go`)

```go
func RedactSSHConnection(conn SSHConnection) SSHConnection
func RedactTunnel(t Tunnel) Tunnel
func RedactConfig(cfg Config) Config
```

Fields replaced with `"***"`:
- `Auth.Password`
- `PrivateKey.KeyPEM`
- `PrivateKey.Passphrase`
- `Socks5Cfg.Password`

Returns a copy — does not mutate the original.

## API Endpoints

### SSH Connections

| Method | Path | Handler | Response |
|--------|------|---------|----------|
| GET | `/api/v1/ssh-connections` | listSSHConnections | 200 + `[]SSHConnection` (redacted) |
| POST | `/api/v1/ssh-connections` | createSSHConnection | 201 + created (redacted), auto-generate ID |
| GET | `/api/v1/ssh-connections/{id}` | getSSHConnection | 200 + SSHConnection (redacted) |
| PUT | `/api/v1/ssh-connections/{id}` | updateSSHConnection | 200 + updated (redacted) |
| DELETE | `/api/v1/ssh-connections/{id}` | deleteSSHConnection | 204, or 409 if referenced |

### Tunnels

| Method | Path | Handler | Response |
|--------|------|---------|----------|
| GET | `/api/v1/tunnels` | listTunnels | 200 + `[]Tunnel` (redacted) |
| POST | `/api/v1/tunnels` | createTunnel | 201 + created (redacted), auto-generate ID + mapping IDs |
| GET | `/api/v1/tunnels/{id}` | getTunnel | 200 + Tunnel (redacted) |
| PUT | `/api/v1/tunnels/{id}` | updateTunnel | 200 + updated (redacted) |
| DELETE | `/api/v1/tunnels/{id}` | deleteTunnel | 204 |

### Error Response Format

```json
{
  "error": "validation_failed",
  "details": [
    {"field": "endpoint.port", "message": "must be between 1 and 65535"}
  ]
}
```

Error codes:
- `validation_failed` (400): request body fails validation
- `not_found` (404): ID doesn't exist
- `conflict` (409): SSH connection referenced by tunnel(s); include tunnel names in details
- `internal_error` (500): unexpected server error

### Warnings in Response

Create/Update responses may include a `warnings` field:
```json
{
  "data": { ... },
  "warnings": ["SOCKS5 proxy is publicly accessible without authentication"]
}
```

## Integration Changes

### `api.Server`

`NewServer` signature changes to accept a `config.Store`:

```go
func NewServer(cfg Config, store config.Store) *Server
```

The server loads config on startup and holds it in memory. All CRUD operations:
1. Modify the in-memory config
2. Validate the full config
3. If valid, save to disk via store
4. Return redacted response

### `main.go`

```go
store := config.NewFileStore(filepath.Join(*dataDir, "config.json"))
cfg, err := store.Load(ctx)
// handle err
srv := api.NewServer(api.Config{...}, store)
```

## Acceptance Criteria

1. On startup with no config file → auto-creates `data/config.json` with `{"version":1,"sshConnections":[],"tunnels":[]}`
2. `POST /api/v1/ssh-connections` with valid body → 201, ID auto-generated
3. `GET /api/v1/ssh-connections` → list with password/keyPem redacted as `"***"`
4. `POST /api/v1/tunnels` referencing existing SSH connections → 201
5. `POST /api/v1/tunnels` referencing non-existent SSH connection ID → 400 validation error
6. `DELETE /api/v1/ssh-connections/{id}` when referenced by tunnel → 409 with tunnel names
7. Restart server → data persists (read from disk)
8. Invalid port (0 or 99999) → 400 with structured error
9. Dynamic mapping with wide-open SOCKS5 → response includes warning
