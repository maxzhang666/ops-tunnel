package engine

import (
	"context"
	"sync"
	"time"
)

const (
	realtimeSamplesCap = 600 // 10 minutes at 1s interval
	flushInterval      = 60 * time.Second
)

// TrafficSample represents a single bandwidth sample.
//
// BytesIn / BytesOut are the totals across all tunnels for this 1-second window.
// PerTunnel breaks the same window down by tunnel; tunnels with zero delta
// are omitted to keep the payload small.
type TrafficSample struct {
	TS        time.Time     `json:"ts"`
	BytesIn   int64         `json:"bytesIn"`
	BytesOut  int64         `json:"bytesOut"`
	PerTunnel []TunnelDelta `json:"perTunnel,omitempty"`
}

// TunnelDelta is the per-tunnel contribution to a TrafficSample window.
type TunnelDelta struct {
	TunnelID string `json:"tunnelId"`
	BytesIn  int64  `json:"bytesIn"`
	BytesOut int64  `json:"bytesOut"`
}

// TrafficRecorder persists aggregated traffic data.
type TrafficRecorder interface {
	Record(tunnelID string, bytesIn, bytesOut, conns int64) error
}

// TrafficSampler periodically snapshots tunnel traffic and stores deltas.
type TrafficSampler struct {
	eng      Engine
	recorder TrafficRecorder // optional, may be nil

	mu      sync.RWMutex
	samples []TrafficSample
	prev    map[string][2]int64 // previous 1s snapshot for realtime deltas

	flushMu     sync.Mutex
	lastFlushed map[string][2]int64 // snapshot at last flush for persistence deltas
}

// NewTrafficSampler creates a sampler that reads from the given engine.
func NewTrafficSampler(eng Engine, recorder TrafficRecorder) *TrafficSampler {
	return &TrafficSampler{
		eng:         eng,
		recorder:    recorder,
		prev:        make(map[string][2]int64),
		lastFlushed: make(map[string][2]int64),
	}
}

// Run starts the sampling loop. Blocks until ctx is cancelled.
func (s *TrafficSampler) Run(ctx context.Context) {
	sampleTicker := time.NewTicker(time.Second)
	defer sampleTicker.Stop()

	flushTicker := time.NewTicker(flushInterval)
	defer flushTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.flush()
			return
		case <-sampleTicker.C:
			s.sample()
		case <-flushTicker.C:
			s.flush()
		}
	}
}

func (s *TrafficSampler) sample() {
	statuses := s.eng.ListStatus()

	s.mu.Lock()
	defer s.mu.Unlock()

	curr := make(map[string][2]int64, len(statuses))
	perTunnel := make([]TunnelDelta, 0, len(statuses))
	var deltaIn, deltaOut int64

	for _, st := range statuses {
		bi, bo := st.BytesIn, st.BytesOut
		curr[st.ID] = [2]int64{bi, bo}
		p := s.prev[st.ID]
		dIn := bi - p[0]
		dOut := bo - p[1]
		// Counter reset (process/tunnel restart): treat current value as the delta.
		if dIn < 0 {
			dIn = bi
		}
		if dOut < 0 {
			dOut = bo
		}
		deltaIn += dIn
		deltaOut += dOut
		if dIn > 0 || dOut > 0 {
			perTunnel = append(perTunnel, TunnelDelta{TunnelID: st.ID, BytesIn: dIn, BytesOut: dOut})
		}
	}

	s.prev = curr
	s.samples = append(s.samples, TrafficSample{
		TS:        time.Now().UTC(),
		BytesIn:   deltaIn,
		BytesOut:  deltaOut,
		PerTunnel: perTunnel,
	})
	if len(s.samples) > realtimeSamplesCap {
		s.samples = s.samples[len(s.samples)-realtimeSamplesCap:]
	}
}

func (s *TrafficSampler) flush() {
	if s.recorder == nil {
		return
	}

	statuses := s.eng.ListStatus()

	s.flushMu.Lock()
	defer s.flushMu.Unlock()

	for _, st := range statuses {
		prev := s.lastFlushed[st.ID]
		dIn := st.BytesIn - prev[0]
		dOut := st.BytesOut - prev[1]
		if dIn < 0 {
			dIn = st.BytesIn
		}
		if dOut < 0 {
			dOut = st.BytesOut
		}
		if dIn > 0 || dOut > 0 {
			s.recorder.Record(st.ID, dIn, dOut, int64(len(st.Mappings)))
		}
		s.lastFlushed[st.ID] = [2]int64{st.BytesIn, st.BytesOut}
	}
}

// GetRealtime returns the in-memory sample buffer.
func (s *TrafficSampler) GetRealtime() []TrafficSample {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]TrafficSample, len(s.samples))
	copy(out, s.samples)
	return out
}
