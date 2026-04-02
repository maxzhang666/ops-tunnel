package forward

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"

	"github.com/maxzhang666/ops-tunnel/internal/config"
	gossh "golang.org/x/crypto/ssh"
)

// LocalForwarder implements local (-L) port forwarding.
type LocalForwarder struct {
	mapping config.Mapping

	mu       sync.RWMutex
	listener net.Listener
	state    string // "stopped" | "listening" | "error"
	lastErr  string
	done     chan struct{}

	active    sync.WaitGroup
	activeCnt atomic.Int32
	totalCnt  atomic.Int64
}

// NewLocalForwarder creates a new local forwarder for the given mapping.
func NewLocalForwarder(m config.Mapping) *LocalForwarder {
	return &LocalForwarder{
		mapping: m,
		state:   "stopped",
	}
}

// Status returns the current state of this forwarder.
func (f *LocalForwarder) Status() Status {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return Status{
		MappingID:   f.mapping.ID,
		State:       f.state,
		Listen:      fmt.Sprintf("%s:%d", f.mapping.Listen.Host, f.mapping.Listen.Port),
		ActiveConns: int(f.activeCnt.Load()),
		TotalConns:  f.totalCnt.Load(),
		LastError:   f.lastErr,
	}
}

// Start begins listening and forwarding connections through the SSH client.
func (f *LocalForwarder) Start(_ context.Context, _ *gossh.Client) error {
	return nil // stub - implemented in Task 3
}

// Stop closes the listener and waits for active connections to drain.
func (f *LocalForwarder) Stop(_ context.Context) error {
	return nil // stub - implemented in Task 3
}

func (f *LocalForwarder) handleConn(_ net.Conn, _ *gossh.Client) {
	// stub - implemented in Task 3
}

func biCopy(_, _ net.Conn) {
	// stub - implemented in Task 3
}

// Ensure interface compliance.
var _ Forwarder = (*LocalForwarder)(nil)
