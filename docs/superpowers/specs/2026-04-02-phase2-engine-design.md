# Phase 2: EventBus + Engine Skeleton + WebSocket

## Goal

Implement the tunnel lifecycle engine with state machine, event bus for real-time notifications, and WebSocket endpoint for frontend consumption. Start/stop are stubs (state changes only, no SSH connections).

## Scope

**In scope:**
- `internal/engine/events.go`: EventBus (fan-out pub/sub)
- `internal/engine/state.go`: Runtime status types
- `internal/engine/engine.go`: Engine interface + implementation
- `internal/engine/supervisor.go`: Per-tunnel supervisor stub
- `internal/api/handler_control.go`: start/stop/restart/status endpoints
- `internal/api/ws.go`: WebSocket endpoint
- Integration: Server receives Engine, main.go creates it

**Out of scope:**
- Real SSH connections (Phase 3)
- Real forwarding (Phase 4-6)
- Auto-reconnect/backoff (Phase 7)

## EventBus (`internal/engine/events.go`)

```go
type EventType string

const (
    EventTunnelStateChanged EventType = "tunnel.stateChanged"
    EventTunnelLog          EventType = "tunnel.log"
    EventForwardListening   EventType = "tunnel.forwardListening"
    EventForwardError       EventType = "tunnel.forwardError"
    EventChainConnected     EventType = "tunnel.chainConnected"
    EventChainError         EventType = "tunnel.chainError"
    EventCoreHealth         EventType = "core.health"
)

type Event struct {
    Type     EventType      `json:"type"`
    TunnelID string         `json:"tunnelId,omitempty"`
    Level    string         `json:"level,omitempty"`
    TS       time.Time      `json:"ts"`
    Message  string         `json:"message"`
    Fields   map[string]any `json:"fields,omitempty"`
}

type EventBus interface {
    Publish(e Event)
    Subscribe(bufSize int) (ch <-chan Event, cancel func())
}
```

### Implementation:
- Internal: `sync.RWMutex` + map of subscriber channels
- `Publish`: fan-out to all subscribers; if a subscriber's channel is full, drop the event (non-blocking send)
- `Subscribe`: creates a buffered channel, returns it + a cancel function that removes the subscription
- Thread-safe

## State Types (`internal/engine/state.go`)

```go
type TunnelState string

const (
    StateStopped  TunnelState = "stopped"
    StateStarting TunnelState = "starting"
    StateRunning  TunnelState = "running"
    StateDegraded TunnelState = "degraded"
    StateError    TunnelState = "error"
    StateStopping TunnelState = "stopping"
)

type HopStatus struct {
    SSHConnID string `json:"sshConnId"`
    State     string `json:"state"`
    LatencyMs int    `json:"latencyMs,omitempty"`
    Detail    string `json:"detail,omitempty"`
}

type MappingStatus struct {
    MappingID string `json:"mappingId"`
    State     string `json:"state"`
    Listen    string `json:"listen"`
    Detail    string `json:"detail,omitempty"`
}

type TunnelStatus struct {
    ID         string          `json:"id"`
    State      TunnelState     `json:"state"`
    Since      time.Time       `json:"since"`
    Chain      []HopStatus     `json:"chain"`
    Mappings   []MappingStatus `json:"mappings"`
    LastError  string          `json:"lastError,omitempty"`
}
```

## Engine (`internal/engine/engine.go`)

```go
type Engine interface {
    StartTunnel(ctx context.Context, id string) error
    StopTunnel(ctx context.Context, id string) error
    RestartTunnel(ctx context.Context, id string) error
    GetStatus(id string) (TunnelStatus, bool)
    ListStatus() []TunnelStatus
    Events() EventBus
    Shutdown(ctx context.Context) error
}
```

### Implementation:
- Holds: `config *config.Config`, `bus EventBus`, `supervisors map[string]*tunnelSupervisor`, `mu sync.RWMutex`
- `StartTunnel`: lookup tunnel config by ID → create supervisor if not exists → call supervisor.Start()
- `StopTunnel`: lookup supervisor → call supervisor.Stop()
- `RestartTunnel`: Stop then Start
- `GetStatus`: delegate to supervisor.Status()
- `ListStatus`: return status for all tunnels (stopped for those without supervisors)
- `Shutdown`: stop all supervisors

## Supervisor Stub (`internal/engine/supervisor.go`)

Phase 2 stub — no real SSH, just state transitions:

```go
type tunnelSupervisor struct {
    tunnel config.Tunnel
    bus    EventBus
    state  TunnelState
    since  time.Time
    mu     sync.RWMutex
}
```

- `Start()`: set state to `starting` → publish event → set state to `running` → publish event
- `Stop()`: set state to `stopping` → publish event → set state to `stopped` → publish event
- `Status()`: return TunnelStatus with current state (chain and mappings are empty stubs)

## API Endpoints

| Method | Path | Handler | Response |
|--------|------|---------|----------|
| POST | `/api/v1/tunnels/{id}/start` | startTunnel | 200 `{"status":"ok"}` or 404 |
| POST | `/api/v1/tunnels/{id}/stop` | stopTunnel | 200 `{"status":"ok"}` or 404 |
| POST | `/api/v1/tunnels/{id}/restart` | restartTunnel | 200 `{"status":"ok"}` or 404 |
| GET | `/api/v1/tunnels/{id}/status` | getTunnelStatus | 200 + TunnelStatus or 404 |
| GET | `/ws` | websocketHandler | Upgrade to WebSocket |

### WebSocket behavior:
- On connect: subscribe to EventBus
- Stream events as JSON lines
- On disconnect: cancel subscription
- Use `github.com/coder/websocket` (already planned in tech stack)

## Integration Changes

### `api.Server`
- Add `engine engine.Engine` field
- `NewServer` takes additional `engine.Engine` parameter
- Register control routes and WS route

### `main.go`
- Create `engine.NewEventBus(256)`
- Create `engine.NewEngine(cfg, bus)`
- Pass engine to `api.NewServer`

## Acceptance Criteria

1. `POST /tunnels/{id}/start` → `GET /tunnels/{id}/status` returns `"state":"running"`
2. `POST /tunnels/{id}/stop` → status returns `"state":"stopped"`
3. `POST /tunnels/{id}/restart` → status returns `"state":"running"`
4. Start non-existent tunnel ID → 404
5. WebSocket at `/ws` receives `tunnel.stateChanged` events in real-time when tunnels are started/stopped
6. Multiple WS clients receive the same events simultaneously
7. Slow WS client doesn't block event delivery to other clients (non-blocking publish)
