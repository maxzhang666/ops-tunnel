package engine

import (
	"context"
	"testing"
	"time"

	"github.com/maxzhang666/ops-tunnel/internal/config"
	tunnelssh "github.com/maxzhang666/ops-tunnel/internal/ssh"
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
					AutoRestart:           false,
					RestartBackoff:        config.RestartBackoff{MinMs: 100, MaxMs: 1000, Factor: 1.5},
					MaxRestartsPerHour:    60,
					GracefulStopTimeoutMs: 1000,
				},
			},
		},
	}
}

func waitForState(t *testing.T, eng Engine, id string, want TunnelState, timeout time.Duration) {
	t.Helper()
	deadline := time.After(timeout)
	for {
		st, ok := eng.GetStatus(id)
		if !ok {
			t.Fatalf("tunnel %s not found", id)
		}
		if st.State == want {
			return
		}
		select {
		case <-deadline:
			t.Fatalf("timeout waiting for state %s, got %s", want, st.State)
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func TestEngine_StartAsync(t *testing.T) {
	bus := NewEventBus()
	eng := NewEngine(testConfig(), bus, tunnelssh.NewNoopHostKeyStore())

	err := eng.StartTunnel(context.Background(), "tun-1")
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	waitForState(t, eng, "tun-1", StateError, 5*time.Second)
}

func TestEngine_StopFromError(t *testing.T) {
	bus := NewEventBus()
	eng := NewEngine(testConfig(), bus, tunnelssh.NewNoopHostKeyStore())

	eng.StartTunnel(context.Background(), "tun-1")
	waitForState(t, eng, "tun-1", StateError, 5*time.Second)

	err := eng.StopTunnel(context.Background(), "tun-1")
	if err != nil {
		t.Fatalf("Stop: %v", err)
	}

	st, _ := eng.GetStatus("tun-1")
	if st.State != StateStopped {
		t.Errorf("State = %s, want stopped", st.State)
	}
}

func TestEngine_Restart(t *testing.T) {
	bus := NewEventBus()
	eng := NewEngine(testConfig(), bus, tunnelssh.NewNoopHostKeyStore())

	eng.StartTunnel(context.Background(), "tun-1")
	waitForState(t, eng, "tun-1", StateError, 5*time.Second)

	eng.RestartTunnel(context.Background(), "tun-1")
	waitForState(t, eng, "tun-1", StateError, 5*time.Second)
}

func TestEngine_StartNonExistent(t *testing.T) {
	bus := NewEventBus()
	eng := NewEngine(testConfig(), bus, tunnelssh.NewNoopHostKeyStore())
	err := eng.StartTunnel(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for non-existent tunnel")
	}
}

func TestEngine_EventsReceived(t *testing.T) {
	bus := NewEventBus()
	eng := NewEngine(testConfig(), bus, tunnelssh.NewNoopHostKeyStore())
	ch, cancel := bus.Subscribe(64)
	defer cancel()

	eng.StartTunnel(context.Background(), "tun-1")

	received := 0
	timeout := time.After(5 * time.Second)
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
	eng := NewEngine(testConfig(), bus, tunnelssh.NewNoopHostKeyStore())
	statuses := eng.ListStatus()
	if len(statuses) != 1 {
		t.Fatalf("expected 1 status, got %d", len(statuses))
	}
	if statuses[0].State != StateStopped {
		t.Errorf("State = %s, want stopped", statuses[0].State)
	}
}

func TestEngine_Shutdown(t *testing.T) {
	bus := NewEventBus()
	eng := NewEngine(testConfig(), bus, tunnelssh.NewNoopHostKeyStore())

	eng.StartTunnel(context.Background(), "tun-1")
	time.Sleep(100 * time.Millisecond)

	eng.Shutdown(context.Background())

	st, _ := eng.GetStatus("tun-1")
	if st.State != StateStopped {
		t.Errorf("State = %s, want stopped after shutdown", st.State)
	}
}

func TestEngine_AutoRestartBackoff(t *testing.T) {
	cfg := testConfig()
	cfg.Tunnels[0].Policy.AutoRestart = true
	cfg.Tunnels[0].Policy.MaxRestartsPerHour = 3
	cfg.Tunnels[0].Policy.RestartBackoff.MinMs = 50
	cfg.Tunnels[0].Policy.RestartBackoff.MaxMs = 200

	bus := NewEventBus()
	eng := NewEngine(cfg, bus, tunnelssh.NewNoopHostKeyStore())

	eng.StartTunnel(context.Background(), "tun-1")
	waitForState(t, eng, "tun-1", StateError, 10*time.Second)

	st, _ := eng.GetStatus("tun-1")
	if st.LastError == "" {
		t.Error("expected lastErr to mention rate limit")
	}
}

func TestEngine_StopDuringBackoff(t *testing.T) {
	cfg := testConfig()
	cfg.Tunnels[0].Policy.AutoRestart = true
	cfg.Tunnels[0].Policy.RestartBackoff.MinMs = 5000
	cfg.Tunnels[0].Policy.RestartBackoff.MaxMs = 10000

	bus := NewEventBus()
	eng := NewEngine(cfg, bus, tunnelssh.NewNoopHostKeyStore())

	eng.StartTunnel(context.Background(), "tun-1")
	time.Sleep(500 * time.Millisecond)

	start := time.Now()
	eng.StopTunnel(context.Background(), "tun-1")
	elapsed := time.Since(start)

	if elapsed > 2*time.Second {
		t.Errorf("Stop took %v, expected near-instant", elapsed)
	}

	st, _ := eng.GetStatus("tun-1")
	if st.State != StateStopped {
		t.Errorf("State = %s, want stopped", st.State)
	}
}
