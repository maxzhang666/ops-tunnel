package engine

import (
	"context"
	"fmt"
	"log/slog"
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
	fwds     []forward.Forwarder

	loopCtx    context.Context
	loopCancel context.CancelFunc
	loopDone   chan struct{}

	restartCount int
	restartTimes []time.Time
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

	s.loopCtx, s.loopCancel = context.WithCancel(context.Background())
	s.loopDone = make(chan struct{})
	s.restartCount = 0
	s.restartTimes = nil
	s.setState(StateStarting)

	go s.runLoop()
	return nil
}

func (s *tunnelSupervisor) Stop(ctx context.Context) error {
	s.mu.Lock()

	if s.state == StateStopped || s.state == StateStopping {
		s.mu.Unlock()
		return nil
	}

	s.setState(StateStopping)
	s.loopCancel()
	s.mu.Unlock()

	select {
	case <-s.loopDone:
	case <-ctx.Done():
		return ctx.Err()
	}

	s.mu.Lock()
	if s.state != StateStopped {
		s.setState(StateStopped)
	}
	s.mu.Unlock()

	return nil
}

func (s *tunnelSupervisor) runLoop() {
	defer close(s.loopDone)

	backoff := BackoffCalc{
		MinMs:  s.tunnel.Policy.RestartBackoff.MinMs,
		MaxMs:  s.tunnel.Policy.RestartBackoff.MaxMs,
		Factor: s.tunnel.Policy.RestartBackoff.Factor,
	}

	for {
		// 1. Build chain
		s.publishLog("info", "connecting SSH chain...")
		chain, err := tunnelssh.BuildChain(s.loopCtx, s.conns, s.hostKeys)
		if err != nil {
			if s.loopCtx.Err() != nil {
				s.mu.Lock()
				s.setState(StateStopped)
				s.mu.Unlock()
				return
			}
			s.mu.Lock()
			s.lastErr = err.Error()
			s.setState(StateError)
			s.mu.Unlock()

			s.publishLog("error", fmt.Sprintf("chain failed: %s", err))

			if !s.tunnel.Policy.AutoRestart {
				return
			}

			if s.rateLimitExceeded() {
				s.mu.Lock()
				s.lastErr = fmt.Sprintf("restart rate limit exceeded (%d/hour)", s.tunnel.Policy.MaxRestartsPerHour)
				s.mu.Unlock()
				s.publishLog("error", s.lastErr)
				return
			}

			if !s.backoffSleep(backoff) {
				return
			}

			s.publishLog("info", "retrying...")
			s.mu.Lock()
			s.setState(StateStarting)
			s.mu.Unlock()
			continue
		}

		s.publishLog("info", fmt.Sprintf("SSH chain established (%d hops)", len(s.conns)))

		// 2. Start forwards
		s.mu.Lock()
		s.chain = chain
		s.mu.Unlock()

		fwds, err := s.startForwards(chain)
		if err != nil {
			s.mu.Lock()
			s.lastErr = err.Error()
			s.chain = nil
			s.setState(StateError)
			s.mu.Unlock()
			chain.Close()

			s.publishLog("error", fmt.Sprintf("forward failed: %s", err))

			if !s.tunnel.Policy.AutoRestart {
				return
			}

			if s.rateLimitExceeded() {
				s.mu.Lock()
				s.lastErr = fmt.Sprintf("restart rate limit exceeded (%d/hour)", s.tunnel.Policy.MaxRestartsPerHour)
				s.mu.Unlock()
				s.publishLog("error", s.lastErr)
				return
			}

			if !s.backoffSleep(backoff) {
				return
			}

			s.publishLog("info", "retrying...")
			s.mu.Lock()
			s.setState(StateStarting)
			s.mu.Unlock()
			continue
		}

		// 3. Running
		s.mu.Lock()
		s.fwds = fwds
		s.lastErr = ""
		s.restartCount = 0
		s.setState(StateRunning)
		s.mu.Unlock()

		s.publishLog("info", "tunnel running")

		// 4. Wait for error or stop
		errMsg := s.waitForError(chain)

		// 5. Cleanup
		s.mu.Lock()
		if errMsg != "" {
			s.lastErr = errMsg
			s.setState(StateDegraded)
		}
		s.mu.Unlock()

		if errMsg != "" {
			s.publishLog("warn", errMsg)
		}

		s.cleanup()

		s.mu.Lock()
		s.chain = nil
		s.fwds = nil
		s.mu.Unlock()

		if s.loopCtx.Err() != nil {
			s.mu.Lock()
			s.setState(StateStopped)
			s.mu.Unlock()
			return
		}

		// 6. Rate limit + backoff
		if !s.tunnel.Policy.AutoRestart {
			s.mu.Lock()
			s.setState(StateError)
			s.mu.Unlock()
			return
		}

		if s.rateLimitExceeded() {
			s.mu.Lock()
			s.lastErr = fmt.Sprintf("restart rate limit exceeded (%d/hour)", s.tunnel.Policy.MaxRestartsPerHour)
			s.setState(StateError)
			s.mu.Unlock()
			return
		}

		if !s.backoffSleep(backoff) {
			return
		}

		s.mu.Lock()
		s.setState(StateStarting)
		s.mu.Unlock()
	}
}

func (s *tunnelSupervisor) startForwards(chain *tunnelssh.ChainResult) ([]forward.Forwarder, error) {
	if s.tunnel.Mode != config.ModeLocal && s.tunnel.Mode != config.ModeRemote && s.tunnel.Mode != config.ModeDynamic {
		return nil, nil
	}

	fwds := make([]forward.Forwarder, 0, len(s.tunnel.Mappings))
	for _, m := range s.tunnel.Mappings {
		fwd := createForwarder(s.tunnel.Mode, m)
		fwd.SetLogger(func(level, message string) {
			s.bus.Publish(Event{
				Type:     EventTunnelLog,
				TunnelID: s.tunnel.ID,
				Level:    level,
				Message:  message,
			})
		})
		if err := fwd.Start(s.loopCtx, chain.Last()); err != nil {
			s.bus.Publish(Event{
				Type:     EventForwardError,
				TunnelID: s.tunnel.ID,
				Level:    "error",
				Message:  fmt.Sprintf("forward %s failed: %s", m.ID, err),
				Fields:   map[string]any{"mappingId": m.ID, "error": err.Error()},
			})
			for j := len(fwds) - 1; j >= 0; j-- {
				fwds[j].Stop(s.loopCtx)
			}
			return nil, fmt.Errorf("forward %s: %w", m.ID, err)
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
	return fwds, nil
}

func (s *tunnelSupervisor) waitForError(chain *tunnelssh.ChainResult) string {
	if len(chain.KAErrors) == 0 {
		<-s.loopCtx.Done()
		return ""
	}

	merged := make(chan string, 1)
	for i, kaErr := range chain.KAErrors {
		go func(hopIdx int, errCh <-chan error) {
			select {
			case <-s.loopCtx.Done():
			case err, ok := <-errCh:
				if ok && err != nil {
					select {
					case merged <- fmt.Sprintf("hop %d keepalive failed: %s", hopIdx+1, err):
					default:
					}
				}
			}
		}(i, kaErr)
	}

	select {
	case <-s.loopCtx.Done():
		return ""
	case msg := <-merged:
		return msg
	}
}

func (s *tunnelSupervisor) cleanup() {
	graceful := time.Duration(s.tunnel.Policy.GracefulStopTimeoutMs) * time.Millisecond
	stopCtx, cancel := context.WithTimeout(context.Background(), graceful)
	defer cancel()

	s.mu.RLock()
	fwds := s.fwds
	chain := s.chain
	s.mu.RUnlock()

	for i := len(fwds) - 1; i >= 0; i-- {
		fwds[i].Stop(stopCtx)
	}
	if chain != nil {
		chain.Close()
	}
}

func (s *tunnelSupervisor) rateLimitExceeded() bool {
	now := time.Now()
	cutoff := now.Add(-time.Hour)

	valid := s.restartTimes[:0]
	for _, t := range s.restartTimes {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	valid = append(valid, now)
	s.restartTimes = valid

	return len(s.restartTimes) >= s.tunnel.Policy.MaxRestartsPerHour
}

func (s *tunnelSupervisor) backoffSleep(b BackoffCalc) bool {
	delay := b.Delay(s.restartCount)
	s.restartCount++

	slog.Info("supervisor backoff", "tunnel", s.tunnel.Name, "delay", delay, "attempt", s.restartCount)

	select {
	case <-time.After(delay):
		return true
	case <-s.loopCtx.Done():
		s.mu.Lock()
		s.setState(StateStopped)
		s.mu.Unlock()
		return false
	}
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

	var totalIn, totalOut int64
	mappings := make([]MappingStatus, len(s.tunnel.Mappings))
	if len(s.fwds) > 0 {
		for i, fwd := range s.fwds {
			st := fwd.Status()
			totalIn += st.BytesIn
			totalOut += st.BytesOut
			mappings[i] = MappingStatus{
				MappingID:   st.MappingID,
				State:       st.State,
				Listen:      st.Listen,
				BytesIn:     st.BytesIn,
				BytesOut:    st.BytesOut,
				ActiveConns: st.ActiveConns,
				Detail:      st.LastError,
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
		BytesIn:   totalIn,
		BytesOut:  totalOut,
		Since:     s.since,
		Chain:     chain,
		Mappings:  mappings,
		LastError: s.lastErr,
	}
}

func createForwarder(mode config.TunnelMode, m config.Mapping) forward.Forwarder {
	switch mode {
	case config.ModeLocal:
		return forward.NewLocalForwarder(m)
	case config.ModeRemote:
		return forward.NewRemoteForwarder(m)
	case config.ModeDynamic:
		return forward.NewDynamicForwarder(m)
	default:
		return nil
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

func (s *tunnelSupervisor) publishLog(level, message string) {
	s.bus.Publish(Event{
		Type:     EventTunnelLog,
		TunnelID: s.tunnel.ID,
		Level:    level,
		Message:  message,
	})
}
