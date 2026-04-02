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

	eng.StartTunnel(ctx, "tun-1")
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
		t.Errorf("State = %s, want stopped", statuses[0].State)
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
