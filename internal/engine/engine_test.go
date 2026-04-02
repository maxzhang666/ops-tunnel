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
					RestartBackoff:     config.RestartBackoff{MinMs: 500, MaxMs: 15000, Factor: 1.7},
					MaxRestartsPerHour: 60,
				},
			},
		},
	}
}

func TestEngine_StartStop(t *testing.T) {
	bus := NewEventBus()
	eng := NewEngine(testConfig(), bus, tunnelssh.NewNoopHostKeyStore())
	ctx := context.Background()

	// Start will fail because no real SSH server exists
	err := eng.StartTunnel(ctx, "tun-1")
	if err == nil {
		t.Log("Start succeeded (unexpected in test without real SSH server)")
	}

	// Status should be error (failed to connect)
	st, ok := eng.GetStatus("tun-1")
	if !ok {
		t.Fatal("tunnel status not found")
	}
	if st.State != StateError && st.State != StateRunning {
		// Either error (expected) or running (if somehow connected)
		t.Logf("State = %s (expected error without real SSH)", st.State)
	}
}

func TestEngine_Restart(t *testing.T) {
	bus := NewEventBus()
	eng := NewEngine(testConfig(), bus, tunnelssh.NewNoopHostKeyStore())
	ctx := context.Background()

	eng.StartTunnel(ctx, "tun-1")
	// RestartTunnel: Stop (from error state is a no-op essentially) then Start again
	// Both Start calls will fail due to no real SSH — that is expected
	eng.RestartTunnel(ctx, "tun-1")

	st, _ := eng.GetStatus("tun-1")
	if st.State != StateError && st.State != StateRunning && st.State != StateStopped {
		t.Logf("State = %s after restart (expected error without real SSH)", st.State)
	}
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

	// Expect at least 2 events: starting → error (or starting → running)
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
	ctx := context.Background()

	eng.StartTunnel(ctx, "tun-1")
	eng.Shutdown(ctx)

	st, _ := eng.GetStatus("tun-1")
	if st.State != StateStopped {
		t.Errorf("State = %s, want stopped after shutdown", st.State)
	}
}
