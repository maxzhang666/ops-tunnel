package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/maxzhang666/ops-tunnel/internal/config"
	"github.com/maxzhang666/ops-tunnel/internal/forward"
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
	fwds     []forward.Forwarder
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

	// Start forwards for local mode
	if s.tunnel.Mode == config.ModeLocal {
		fwds := make([]forward.Forwarder, 0, len(s.tunnel.Mappings))
		for _, m := range s.tunnel.Mappings {
			fwd := forward.NewLocalForwarder(m)
			if err := fwd.Start(ctx, chain.Last()); err != nil {
				s.bus.Publish(Event{
					Type:     EventForwardError,
					TunnelID: s.tunnel.ID,
					Level:    "error",
					Message:  fmt.Sprintf("forward %s failed: %s", m.ID, err),
					Fields:   map[string]any{"mappingId": m.ID, "error": err.Error()},
				})
				for j := len(fwds) - 1; j >= 0; j-- {
					fwds[j].Stop(ctx)
				}
				chain.Close()
				chainCancel()
				s.chain = nil
				s.lastErr = fmt.Sprintf("forward %s: %s", m.ID, err)
				s.setState(StateError)
				return fmt.Errorf("start forward %s: %w", m.ID, err)
			}
			fwds = append(fwds, fwd)
			st := fwd.Status()
			s.bus.Publish(Event{
				Type:     EventForwardListening,
				TunnelID: s.tunnel.ID,
				Level:    "info",
				Message:  fmt.Sprintf("forward %s listening on %s", m.ID, st.Listen),
				Fields:   map[string]any{"mappingId": m.ID, "listen": st.Listen},
			})
		}
		s.fwds = fwds
	}

	s.lastErr = ""
	s.setState(StateRunning)

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

	// Stop forwards in reverse order before closing chain
	for i := len(s.fwds) - 1; i >= 0; i-- {
		s.fwds[i].Stop(ctx)
	}
	s.fwds = nil

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
	if len(s.fwds) > 0 {
		for i, fwd := range s.fwds {
			st := fwd.Status()
			mappings[i] = MappingStatus{
				MappingID: st.MappingID,
				State:     st.State,
				Listen:    st.Listen,
				Detail:    st.LastError,
			}
		}
	} else {
		for i, m := range s.tunnel.Mappings {
			mappings[i] = MappingStatus{
				MappingID: m.ID,
				State:     "stopped",
				Listen:    fmt.Sprintf("%s:%d", m.Listen.Host, m.Listen.Port),
			}
		}
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
