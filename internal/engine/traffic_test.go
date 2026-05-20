package engine

import (
	"context"
	"testing"
)

// mockEngine satisfies Engine just enough for TrafficSampler tests.
type mockEngine struct {
	statuses []TunnelStatus
}

func (m *mockEngine) StartTunnel(context.Context, string) error   { return nil }
func (m *mockEngine) StopTunnel(context.Context, string) error    { return nil }
func (m *mockEngine) RestartTunnel(context.Context, string) error { return nil }
func (m *mockEngine) GetStatus(string) (TunnelStatus, bool)       { return TunnelStatus{}, false }
func (m *mockEngine) ListStatus() []TunnelStatus                  { return m.statuses }
func (m *mockEngine) Events() EventBus                            { return nil }
func (m *mockEngine) Shutdown(context.Context) error              { return nil }

func TestTrafficSampler_PerTunnelDelta(t *testing.T) {
	eng := &mockEngine{}
	s := NewTrafficSampler(eng, nil)

	// First sample: establishes baseline. With no prev, deltas equal current counters.
	eng.statuses = []TunnelStatus{
		{ID: "a", BytesIn: 100, BytesOut: 50},
		{ID: "b", BytesIn: 0, BytesOut: 0},
	}
	s.sample()

	// Second sample: only "a" moved.
	eng.statuses = []TunnelStatus{
		{ID: "a", BytesIn: 300, BytesOut: 80},
		{ID: "b", BytesIn: 0, BytesOut: 0},
	}
	s.sample()

	samples := s.GetRealtime()
	if len(samples) != 2 {
		t.Fatalf("samples = %d, want 2", len(samples))
	}

	last := samples[1]
	if last.BytesIn != 200 || last.BytesOut != 30 {
		t.Errorf("totals = (%d,%d), want (200,30)", last.BytesIn, last.BytesOut)
	}
	if len(last.PerTunnel) != 1 {
		t.Fatalf("perTunnel len = %d, want 1 (b should be filtered as zero-delta)", len(last.PerTunnel))
	}
	if last.PerTunnel[0].TunnelID != "a" || last.PerTunnel[0].BytesIn != 200 || last.PerTunnel[0].BytesOut != 30 {
		t.Errorf("perTunnel[0] = %+v, want {a 200 30}", last.PerTunnel[0])
	}
}

func TestTrafficSampler_PerTunnelSumEqualsTotal(t *testing.T) {
	eng := &mockEngine{}
	s := NewTrafficSampler(eng, nil)

	eng.statuses = []TunnelStatus{
		{ID: "a", BytesIn: 0, BytesOut: 0},
		{ID: "b", BytesIn: 0, BytesOut: 0},
		{ID: "c", BytesIn: 0, BytesOut: 0},
	}
	s.sample()

	eng.statuses = []TunnelStatus{
		{ID: "a", BytesIn: 1000, BytesOut: 200},
		{ID: "b", BytesIn: 500, BytesOut: 100},
		{ID: "c", BytesIn: 0, BytesOut: 0},
	}
	s.sample()

	last := s.GetRealtime()[1]
	var sumIn, sumOut int64
	for _, d := range last.PerTunnel {
		sumIn += d.BytesIn
		sumOut += d.BytesOut
	}
	if sumIn != last.BytesIn || sumOut != last.BytesOut {
		t.Errorf("perTunnel sum (%d,%d) != totals (%d,%d)", sumIn, sumOut, last.BytesIn, last.BytesOut)
	}
}

func TestTrafficSampler_CounterResetTreatedAsCurrent(t *testing.T) {
	// Simulates a tunnel restart: BytesIn drops from 1000 -> 200.
	// The sample should treat 200 as the delta, not a negative number.
	eng := &mockEngine{statuses: []TunnelStatus{{ID: "a", BytesIn: 1000, BytesOut: 1000}}}
	s := NewTrafficSampler(eng, nil)
	s.sample()

	eng.statuses = []TunnelStatus{{ID: "a", BytesIn: 200, BytesOut: 200}}
	s.sample()

	last := s.GetRealtime()[1]
	if last.BytesIn != 200 || last.BytesOut != 200 {
		t.Errorf("post-reset totals = (%d,%d), want (200,200)", last.BytesIn, last.BytesOut)
	}
	if len(last.PerTunnel) != 1 || last.PerTunnel[0].BytesIn != 200 {
		t.Errorf("perTunnel after reset = %+v, want single entry with 200", last.PerTunnel)
	}
}

func TestTrafficSampler_NewTunnelStartsFromZero(t *testing.T) {
	eng := &mockEngine{statuses: []TunnelStatus{{ID: "a", BytesIn: 100, BytesOut: 100}}}
	s := NewTrafficSampler(eng, nil)
	s.sample()

	// New tunnel "b" appears with non-zero counters: full counters count as delta.
	eng.statuses = []TunnelStatus{
		{ID: "a", BytesIn: 150, BytesOut: 130},
		{ID: "b", BytesIn: 400, BytesOut: 250},
	}
	s.sample()

	last := s.GetRealtime()[1]
	if len(last.PerTunnel) != 2 {
		t.Fatalf("perTunnel len = %d, want 2", len(last.PerTunnel))
	}
	got := map[string][2]int64{}
	for _, d := range last.PerTunnel {
		got[d.TunnelID] = [2]int64{d.BytesIn, d.BytesOut}
	}
	if got["a"] != [2]int64{50, 30} {
		t.Errorf("a delta = %v, want [50 30]", got["a"])
	}
	if got["b"] != [2]int64{400, 250} {
		t.Errorf("b delta = %v, want [400 250]", got["b"])
	}
}

func TestTrafficSampler_DeletedTunnelDropped(t *testing.T) {
	eng := &mockEngine{statuses: []TunnelStatus{
		{ID: "a", BytesIn: 100, BytesOut: 100},
		{ID: "b", BytesIn: 100, BytesOut: 100},
	}}
	s := NewTrafficSampler(eng, nil)
	s.sample()

	// "b" disappears (removed); "a" continues.
	eng.statuses = []TunnelStatus{{ID: "a", BytesIn: 150, BytesOut: 150}}
	s.sample()

	last := s.GetRealtime()[1]
	if len(last.PerTunnel) != 1 || last.PerTunnel[0].TunnelID != "a" {
		t.Errorf("perTunnel = %+v, want only [a]", last.PerTunnel)
	}
}
