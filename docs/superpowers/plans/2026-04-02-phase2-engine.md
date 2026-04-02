# Phase 2: EventBus + Engine + WebSocket — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement tunnel lifecycle engine with state machine, event bus, WebSocket endpoint. Start/stop are stubs (no real SSH).

**Architecture:** `internal/engine/` provides EventBus (pub/sub), Engine (orchestrates supervisors), and per-tunnel supervisors (state machines). `internal/api/` adds control endpoints (start/stop/restart/status) and a WebSocket endpoint that streams events.

**Tech Stack:** Go 1.26, chi v5, `github.com/coder/websocket`, `log/slog`

---

## File Map

| File | Purpose |
|------|---------|
| `internal/engine/events.go` | EventBus interface + implementation |
| `internal/engine/state.go` | TunnelState, TunnelStatus, HopStatus, MappingStatus |
| `internal/engine/supervisor.go` | Per-tunnel supervisor (stub: state transitions only) |
| `internal/engine/engine.go` | Engine interface + implementation |
| `internal/engine/events_test.go` | EventBus tests |
| `internal/engine/engine_test.go` | Engine + supervisor tests |
| `internal/api/handler_control.go` | start/stop/restart/status handlers |
| `internal/api/ws.go` | WebSocket endpoint |
| `internal/api/server.go` | **Modify:** add engine field |
| `internal/api/routes.go` | **Modify:** register control + ws routes |
| `cmd/tunnel-server/main.go` | **Modify:** create engine, pass to server |

---

## Task 1: Add websocket dependency + commit

- [ ] **Step 1: Add dependency (already done via `go get`)**

```bash
cd /Users/maxzhang/Tools/ops-tunnel
git add go.mod go.sum
git commit -m "chore: add coder/websocket dependency"
```

---

## Task 2: EventBus

**Files:**
- Create: `internal/engine/events.go`

- [ ] **Step 1: Create `internal/engine/events.go`**

```go
package engine

import (
	"sync"
	"time"
)

// EventType identifies the kind of event.
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

// Event is a single notification emitted by the engine.
type Event struct {
	Type     EventType      `json:"type"`
	TunnelID string         `json:"tunnelId,omitempty"`
	Level    string         `json:"level,omitempty"`
	TS       time.Time      `json:"ts"`
	Message  string         `json:"message"`
	Fields   map[string]any `json:"fields,omitempty"`
}

// EventBus supports publishing events and subscribing to them.
type EventBus interface {
	Publish(e Event)
	Subscribe(bufSize int) (ch <-chan Event, cancel func())
}

type subscriber struct {
	ch chan Event
}

type eventBus struct {
	mu   sync.RWMutex
	subs map[*subscriber]struct{}
}

// NewEventBus creates a new event bus.
func NewEventBus() EventBus {
	return &eventBus{
		subs: make(map[*subscriber]struct{}),
	}
}

func (b *eventBus) Publish(e Event) {
	if e.TS.IsZero() {
		e.TS = time.Now().UTC()
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	for sub := range b.subs {
		// Non-blocking send: drop if subscriber is slow
		select {
		case sub.ch <- e:
		default:
		}
	}
}

func (b *eventBus) Subscribe(bufSize int) (<-chan Event, func()) {
	if bufSize <= 0 {
		bufSize = 64
	}
	sub := &subscriber{ch: make(chan Event, bufSize)}

	b.mu.Lock()
	b.subs[sub] = struct{}{}
	b.mu.Unlock()

	cancel := func() {
		b.mu.Lock()
		delete(b.subs, sub)
		b.mu.Unlock()
		// Drain remaining events
		for range sub.ch {
		}
	}

	return sub.ch, cancel
}
```

- [ ] **Step 2: Verify**

```bash
go build ./internal/engine/
```

- [ ] **Step 3: Commit**

```bash
git add internal/engine/events.go
git commit -m "feat(engine): add EventBus with fan-out pub/sub"
```

---

## Task 3: State Types

**Files:**
- Create: `internal/engine/state.go`

- [ ] **Step 1: Create `internal/engine/state.go`**

```go
package engine

import "time"

// TunnelState represents the runtime state of a tunnel.
type TunnelState string

const (
	StateStopped  TunnelState = "stopped"
	StateStarting TunnelState = "starting"
	StateRunning  TunnelState = "running"
	StateDegraded TunnelState = "degraded"
	StateError    TunnelState = "error"
	StateStopping TunnelState = "stopping"
)

// HopStatus represents the runtime state of a single hop in the chain.
type HopStatus struct {
	SSHConnID string `json:"sshConnId"`
	State     string `json:"state"`
	LatencyMs int    `json:"latencyMs,omitempty"`
	Detail    string `json:"detail,omitempty"`
}

// MappingStatus represents the runtime state of a single port mapping.
type MappingStatus struct {
	MappingID string `json:"mappingId"`
	State     string `json:"state"`
	Listen    string `json:"listen"`
	Detail    string `json:"detail,omitempty"`
}

// TunnelStatus is the full runtime status of a tunnel.
type TunnelStatus struct {
	ID        string          `json:"id"`
	State     TunnelState     `json:"state"`
	Since     time.Time       `json:"since"`
	Chain     []HopStatus     `json:"chain"`
	Mappings  []MappingStatus `json:"mappings"`
	LastError string          `json:"lastError,omitempty"`
}
```

- [ ] **Step 2: Verify and commit**

```bash
go build ./internal/engine/
git add internal/engine/state.go
git commit -m "feat(engine): add tunnel runtime state types"
```

---

## Task 4: Supervisor Stub

**Files:**
- Create: `internal/engine/supervisor.go`

- [ ] **Step 1: Create `internal/engine/supervisor.go`**

```go
package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/maxzhang666/ops-tunnel/internal/config"
)

// tunnelSupervisor manages the lifecycle of a single tunnel.
// Phase 2: stub implementation — state transitions only, no real SSH.
type tunnelSupervisor struct {
	tunnel config.Tunnel
	bus    EventBus

	mu    sync.RWMutex
	state TunnelState
	since time.Time
	lastErr string
}

func newSupervisor(t config.Tunnel, bus EventBus) *tunnelSupervisor {
	return &tunnelSupervisor{
		tunnel: t,
		bus:    bus,
		state:  StateStopped,
		since:  time.Now().UTC(),
	}
}

func (s *tunnelSupervisor) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state == StateRunning || s.state == StateStarting {
		return fmt.Errorf("tunnel %s is already %s", s.tunnel.ID, s.state)
	}

	s.setState(StateStarting)
	// Phase 2 stub: immediately transition to running
	s.setState(StateRunning)
	s.lastErr = ""
	return nil
}

func (s *tunnelSupervisor) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state == StateStopped || s.state == StateStopping {
		return nil
	}

	s.setState(StateStopping)
	// Phase 2 stub: immediately transition to stopped
	s.setState(StateStopped)
	return nil
}

func (s *tunnelSupervisor) Status() TunnelStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	chain := make([]HopStatus, len(s.tunnel.Chain))
	for i, connID := range s.tunnel.Chain {
		st := "disconnected"
		if s.state == StateRunning {
			st = "connected"
		}
		chain[i] = HopStatus{SSHConnID: connID, State: st}
	}

	mappings := make([]MappingStatus, len(s.tunnel.Mappings))
	for i, m := range s.tunnel.Mappings {
		st := "stopped"
		listen := fmt.Sprintf("%s:%d", m.Listen.Host, m.Listen.Port)
		if s.state == StateRunning {
			st = "listening"
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

// setState updates state, timestamp, and publishes an event.
// Caller must hold s.mu write lock.
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

- [ ] **Step 2: Verify and commit**

```bash
go build ./internal/engine/
git add internal/engine/supervisor.go
git commit -m "feat(engine): add tunnel supervisor stub with state machine"
```

---

## Task 5: Engine

**Files:**
- Create: `internal/engine/engine.go`

- [ ] **Step 1: Create `internal/engine/engine.go`**

```go
package engine

import (
	"context"
	"fmt"
	"sync"

	"github.com/maxzhang666/ops-tunnel/internal/config"
)

// Engine manages all tunnel lifecycles.
type Engine interface {
	StartTunnel(ctx context.Context, id string) error
	StopTunnel(ctx context.Context, id string) error
	RestartTunnel(ctx context.Context, id string) error
	GetStatus(id string) (TunnelStatus, bool)
	ListStatus() []TunnelStatus
	Events() EventBus
	Shutdown(ctx context.Context) error
}

type engine struct {
	cfg  *config.Config
	bus  EventBus

	mu   sync.RWMutex
	sups map[string]*tunnelSupervisor
}

// NewEngine creates a new engine.
func NewEngine(cfg *config.Config, bus EventBus) Engine {
	return &engine{
		cfg:  cfg,
		bus:  bus,
		sups: make(map[string]*tunnelSupervisor),
	}
}

func (e *engine) findTunnel(id string) (*config.Tunnel, error) {
	for i := range e.cfg.Tunnels {
		if e.cfg.Tunnels[i].ID == id {
			return &e.cfg.Tunnels[i], nil
		}
	}
	return nil, fmt.Errorf("tunnel '%s' not found", id)
}

func (e *engine) getOrCreateSupervisor(t *config.Tunnel) *tunnelSupervisor {
	if sup, ok := e.sups[t.ID]; ok {
		return sup
	}
	sup := newSupervisor(*t, e.bus)
	e.sups[t.ID] = sup
	return sup
}

func (e *engine) StartTunnel(ctx context.Context, id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	t, err := e.findTunnel(id)
	if err != nil {
		return err
	}

	sup := e.getOrCreateSupervisor(t)
	return sup.Start(ctx)
}

func (e *engine) StopTunnel(ctx context.Context, id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	t, err := e.findTunnel(id)
	if err != nil {
		return err
	}

	sup := e.getOrCreateSupervisor(t)
	return sup.Stop(ctx)
}

func (e *engine) RestartTunnel(ctx context.Context, id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	t, err := e.findTunnel(id)
	if err != nil {
		return err
	}

	sup := e.getOrCreateSupervisor(t)
	if err := sup.Stop(ctx); err != nil {
		return err
	}
	return sup.Start(ctx)
}

func (e *engine) GetStatus(id string) (TunnelStatus, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if sup, ok := e.sups[id]; ok {
		return sup.Status(), true
	}

	// Return stopped status for tunnels that exist but have no supervisor
	for _, t := range e.cfg.Tunnels {
		if t.ID == id {
			return TunnelStatus{
				ID:    id,
				State: StateStopped,
				Chain: []HopStatus{},
				Mappings: []MappingStatus{},
			}, true
		}
	}

	return TunnelStatus{}, false
}

func (e *engine) ListStatus() []TunnelStatus {
	e.mu.RLock()
	defer e.mu.RUnlock()

	statuses := make([]TunnelStatus, 0, len(e.cfg.Tunnels))
	for _, t := range e.cfg.Tunnels {
		if sup, ok := e.sups[t.ID]; ok {
			statuses = append(statuses, sup.Status())
		} else {
			statuses = append(statuses, TunnelStatus{
				ID:       t.ID,
				State:    StateStopped,
				Chain:    []HopStatus{},
				Mappings: []MappingStatus{},
			})
		}
	}
	return statuses
}

func (e *engine) Events() EventBus {
	return e.bus
}

func (e *engine) Shutdown(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	for _, sup := range e.sups {
		sup.Stop(ctx)
	}
	e.sups = make(map[string]*tunnelSupervisor)
	return nil
}
```

- [ ] **Step 2: Verify and commit**

```bash
go build ./internal/engine/
git add internal/engine/engine.go
git commit -m "feat(engine): add Engine interface and implementation"
```

---

## Task 6: Engine Tests

**Files:**
- Create: `internal/engine/events_test.go`
- Create: `internal/engine/engine_test.go`

- [ ] **Step 1: Create `internal/engine/events_test.go`**

```go
package engine

import (
	"testing"
	"time"
)

func TestEventBus_PublishSubscribe(t *testing.T) {
	bus := NewEventBus()
	ch, cancel := bus.Subscribe(16)
	defer cancel()

	bus.Publish(Event{Type: EventTunnelStateChanged, TunnelID: "t1", Message: "started"})

	select {
	case e := <-ch:
		if e.TunnelID != "t1" {
			t.Errorf("TunnelID = %s, want t1", e.TunnelID)
		}
		if e.TS.IsZero() {
			t.Error("TS should be auto-set")
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}
}

func TestEventBus_MultipleSubscribers(t *testing.T) {
	bus := NewEventBus()
	ch1, cancel1 := bus.Subscribe(16)
	defer cancel1()
	ch2, cancel2 := bus.Subscribe(16)
	defer cancel2()

	bus.Publish(Event{Type: EventTunnelLog, Message: "test"})

	for _, ch := range []<-chan Event{ch1, ch2} {
		select {
		case e := <-ch:
			if e.Message != "test" {
				t.Errorf("Message = %s, want test", e.Message)
			}
		case <-time.After(time.Second):
			t.Fatal("timeout")
		}
	}
}

func TestEventBus_NonBlockingPublish(t *testing.T) {
	bus := NewEventBus()
	// Subscribe with buffer of 1
	ch, cancel := bus.Subscribe(1)
	defer cancel()

	// Fill the buffer
	bus.Publish(Event{Message: "first"})
	// This should not block even though buffer is full
	bus.Publish(Event{Message: "second"})

	// Should get first event
	select {
	case e := <-ch:
		if e.Message != "first" {
			t.Errorf("Message = %s, want first", e.Message)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

func TestEventBus_CancelSubscription(t *testing.T) {
	bus := NewEventBus()
	ch, cancel := bus.Subscribe(16)
	cancel()

	bus.Publish(Event{Message: "after cancel"})

	// Channel should be closed/drained, no new events
	select {
	case _, ok := <-ch:
		if ok {
			t.Error("should not receive events after cancel")
		}
	case <-time.After(100 * time.Millisecond):
		// Expected: no event
	}
}
```

- [ ] **Step 2: Create `internal/engine/engine_test.go`**

```go
package engine

import (
	"context"
	"testing"
	"time"

	"github.com/maxzhang666/ops-tunnel/internal/config"
)

func testConfig() *config.Config {
	return &config.Config{
		Version: 1,
		SSHConnections: []config.SSHConnection{
			{ID: "ssh-1", Name: "test-ssh", Endpoint: config.Endpoint{Host: "1.2.3.4", Port: 22}},
		},
		Tunnels: []config.Tunnel{
			{
				ID:    "tun-1",
				Name:  "test-tunnel",
				Mode:  config.ModeLocal,
				Chain: []string{"ssh-1"},
				Mappings: []config.Mapping{
					{ID: "m1", Listen: config.Endpoint{Host: "127.0.0.1", Port: 15432}, Connect: config.Endpoint{Host: "127.0.0.1", Port: 5432}},
				},
				Policy: config.Policy{
					RestartBackoff:     config.RestartBackoff{MinMs: 500, MaxMs: 15000, Factor: 1.7},
					MaxRestartsPerHour: 60,
				},
			},
		},
	}
}

func TestEngine_StartStop(t *testing.T) {
	bus := NewEventBus()
	eng := NewEngine(testConfig(), bus)
	ctx := context.Background()

	// Start
	if err := eng.StartTunnel(ctx, "tun-1"); err != nil {
		t.Fatalf("StartTunnel error: %v", err)
	}

	st, ok := eng.GetStatus("tun-1")
	if !ok {
		t.Fatal("tunnel status not found")
	}
	if st.State != StateRunning {
		t.Errorf("State = %s, want running", st.State)
	}

	// Stop
	if err := eng.StopTunnel(ctx, "tun-1"); err != nil {
		t.Fatalf("StopTunnel error: %v", err)
	}

	st, _ = eng.GetStatus("tun-1")
	if st.State != StateStopped {
		t.Errorf("State = %s, want stopped", st.State)
	}
}

func TestEngine_Restart(t *testing.T) {
	bus := NewEventBus()
	eng := NewEngine(testConfig(), bus)
	ctx := context.Background()

	if err := eng.StartTunnel(ctx, "tun-1"); err != nil {
		t.Fatalf("StartTunnel error: %v", err)
	}
	if err := eng.RestartTunnel(ctx, "tun-1"); err != nil {
		t.Fatalf("RestartTunnel error: %v", err)
	}

	st, _ := eng.GetStatus("tun-1")
	if st.State != StateRunning {
		t.Errorf("State = %s, want running after restart", st.State)
	}
}

func TestEngine_StartNonExistent(t *testing.T) {
	bus := NewEventBus()
	eng := NewEngine(testConfig(), bus)

	err := eng.StartTunnel(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for non-existent tunnel")
	}
}

func TestEngine_EventsReceived(t *testing.T) {
	bus := NewEventBus()
	eng := NewEngine(testConfig(), bus)
	ch, cancel := bus.Subscribe(64)
	defer cancel()

	eng.StartTunnel(context.Background(), "tun-1")

	// Should receive at least 2 events: starting + running
	received := 0
	timeout := time.After(time.Second)
	for received < 2 {
		select {
		case <-ch:
			received++
		case <-timeout:
			t.Fatalf("expected at least 2 events, got %d", received)
		}
	}
}

func TestEngine_ListStatus(t *testing.T) {
	bus := NewEventBus()
	eng := NewEngine(testConfig(), bus)

	statuses := eng.ListStatus()
	if len(statuses) != 1 {
		t.Fatalf("expected 1 status, got %d", len(statuses))
	}
	if statuses[0].State != StateStopped {
		t.Errorf("State = %s, want stopped (not started yet)", statuses[0].State)
	}
}

func TestEngine_Shutdown(t *testing.T) {
	bus := NewEventBus()
	eng := NewEngine(testConfig(), bus)
	ctx := context.Background()

	eng.StartTunnel(ctx, "tun-1")
	eng.Shutdown(ctx)

	st, _ := eng.GetStatus("tun-1")
	if st.State != StateStopped {
		t.Errorf("State = %s, want stopped after shutdown", st.State)
	}
}
```

- [ ] **Step 3: Run tests**

```bash
go test ./internal/engine/ -v
```

Expected: all pass.

- [ ] **Step 4: Commit**

```bash
git add internal/engine/events_test.go internal/engine/engine_test.go
git commit -m "test(engine): add EventBus and Engine tests"
```

---

## Task 7: Control Handlers + WebSocket

**Files:**
- Create: `internal/api/handler_control.go`
- Create: `internal/api/ws.go`
- Modify: `internal/api/server.go`
- Modify: `internal/api/routes.go`

- [ ] **Step 1: Create `internal/api/handler_control.go`**

```go
package api

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

func (s *Server) startTunnel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.eng.StartTunnel(r.Context(), id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeNotFound(w, "tunnel", id)
			return
		}
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) stopTunnel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.eng.StopTunnel(r.Context(), id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeNotFound(w, "tunnel", id)
			return
		}
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) restartTunnel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.eng.RestartTunnel(r.Context(), id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeNotFound(w, "tunnel", id)
			return
		}
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) getTunnelStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	st, ok := s.eng.GetStatus(id)
	if !ok {
		writeNotFound(w, "tunnel", id)
		return
	}
	writeJSON(w, http.StatusOK, st)
}
```

- [ ] **Step 2: Create `internal/api/ws.go`**

```go
package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/coder/websocket"
)

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true, // Allow connections from any origin (dev mode)
	})
	if err != nil {
		slog.Error("websocket accept failed", "err", err)
		return
	}
	defer conn.CloseNow()

	ch, cancel := s.eng.Events().Subscribe(256)
	defer cancel()

	ctx := conn.CloseRead(r.Context())

	for {
		select {
		case <-ctx.Done():
			conn.Close(websocket.StatusNormalClosure, "")
			return
		case evt, ok := <-ch:
			if !ok {
				conn.Close(websocket.StatusNormalClosure, "")
				return
			}
			data, err := json.Marshal(evt)
			if err != nil {
				slog.Error("marshal event failed", "err", err)
				continue
			}
			if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
				slog.Debug("websocket write failed", "err", err)
				return
			}
		}
	}
}
```

- [ ] **Step 3: Update `internal/api/server.go`**

Add the engine field. Read the file first, then add `eng engine.Engine` field and update `NewServer`:

In `server.go`, add import `"github.com/maxzhang666/ops-tunnel/internal/engine"` and update the struct/constructor:

The Server struct needs an `eng` field:
```go
type Server struct {
	cfg    ServerConfig
	store  config.Store
	eng    engine.Engine
	mu     sync.RWMutex
	data   *config.Config
	router chi.Router
	http   *http.Server
}
```

And `NewServer` needs the engine parameter:
```go
func NewServer(cfg ServerConfig, store config.Store, data *config.Config, eng engine.Engine) *Server {
```

Assign `eng: eng` in the struct literal.

- [ ] **Step 4: Update `internal/api/routes.go`**

Add these routes inside `registerRoutes()`, after the tunnels route group:

```go
	// Tunnel control
	s.router.Post("/api/v1/tunnels/{id}/start", s.startTunnel)
	s.router.Post("/api/v1/tunnels/{id}/stop", s.stopTunnel)
	s.router.Post("/api/v1/tunnels/{id}/restart", s.restartTunnel)
	s.router.Get("/api/v1/tunnels/{id}/status", s.getTunnelStatus)

	// WebSocket
	s.router.Get("/ws", s.handleWebSocket)
```

- [ ] **Step 5: Verify compilation**

```bash
go build ./internal/api/
```

This will fail until main.go is updated (NewServer signature changed). That's expected.

- [ ] **Step 6: Commit**

```bash
git add internal/api/handler_control.go internal/api/ws.go internal/api/server.go internal/api/routes.go
git commit -m "feat(api): add control handlers (start/stop/restart/status) and WebSocket"
```

---

## Task 8: Update main.go

**Files:**
- Modify: `cmd/tunnel-server/main.go`

- [ ] **Step 1: Update `cmd/tunnel-server/main.go`**

Add engine import and creation. Change `NewServer` call to include engine.

Add import: `"github.com/maxzhang666/ops-tunnel/internal/engine"`

After config load/save, before creating server:
```go
	bus := engine.NewEventBus()
	eng := engine.NewEngine(cfg, bus)
```

Update NewServer call:
```go
	srv := api.NewServer(api.ServerConfig{
		ListenAddr: *listen,
		UIDir:      *uiDir,
		Token:      *token,
	}, store, cfg, eng)
```

Before shutdown, add engine shutdown:
```go
	if err := eng.Shutdown(shutCtx); err != nil {
		slog.Error("engine shutdown error", "err", err)
	}
```

- [ ] **Step 2: Build and verify**

```bash
go build -o bin/tunnel-server ./cmd/tunnel-server
go test ./... -v
```

Expected: compiles, all tests pass.

- [ ] **Step 3: Commit**

```bash
git add cmd/tunnel-server/main.go
git commit -m "feat: integrate engine into server startup"
```

---

## Task 9: Commit docs

- [ ] **Step 1: Commit spec and plan**

```bash
git add docs/
git commit -m "docs: add Phase 2 spec and plan"
```
