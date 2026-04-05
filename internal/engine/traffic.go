package engine

import (
	"context"
	"sync"
	"time"
)

const (
	realtimeSamplesCap = 300 // 5 minutes at 1s interval
	flushInterval      = 60 * time.Second
)

// TrafficSample represents a single bandwidth sample.
type TrafficSample struct {
	TS       time.Time `json:"ts"`
	BytesIn  int64     `json:"bytesIn"`
	BytesOut int64     `json:"bytesOut"`
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

	var totalIn, totalOut int64
	curr := make(map[string][2]int64, len(statuses))
	for _, st := range statuses {
		curr[st.ID] = [2]int64{st.BytesIn, st.BytesOut}
		totalIn += st.BytesIn
		totalOut += st.BytesOut
	}

	var prevTotalIn, prevTotalOut int64
	for _, v := range s.prev {
		prevTotalIn += v[0]
		prevTotalOut += v[1]
	}
	deltaIn := totalIn - prevTotalIn
	deltaOut := totalOut - prevTotalOut
	if deltaIn < 0 {
		deltaIn = totalIn
	}
	if deltaOut < 0 {
		deltaOut = totalOut
	}

	s.mu.Lock()
	s.prev = curr
	s.samples = append(s.samples, TrafficSample{
		TS:       time.Now().UTC(),
		BytesIn:  deltaIn,
		BytesOut: deltaOut,
	})
	if len(s.samples) > realtimeSamplesCap {
		s.samples = s.samples[len(s.samples)-realtimeSamplesCap:]
	}
	s.mu.Unlock()
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
