package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/maxzhang666/ops-tunnel/internal/config"
)

type tunnelSupervisor struct {
	tunnel  config.Tunnel
	bus     EventBus
	mu      sync.RWMutex
	state   TunnelState
	since   time.Time
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
