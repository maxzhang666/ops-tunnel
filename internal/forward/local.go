package forward

import (
	"context"
	"fmt"
	"io"
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
func (f *LocalForwarder) Start(ctx context.Context, sshClient *gossh.Client) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.state == "listening" {
		return fmt.Errorf("already listening")
	}

	addr := fmt.Sprintf("%s:%d", f.mapping.Listen.Host, f.mapping.Listen.Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", addr, err)
	}

	f.listener = ln
	f.state = "listening"
	f.lastErr = ""
	f.done = make(chan struct{})

	go f.acceptLoop(sshClient)
	return nil
}

func (f *LocalForwarder) acceptLoop(sshClient *gossh.Client) {
	defer close(f.done)
	for {
		conn, err := f.listener.Accept()
		if err != nil {
			return // listener closed
		}
		f.active.Add(1)
		f.activeCnt.Add(1)
		go f.handleConn(conn, sshClient)
	}
}

func (f *LocalForwarder) handleConn(local net.Conn, sshClient *gossh.Client) {
	defer func() {
		f.activeCnt.Add(-1)
		f.totalCnt.Add(1)
		f.active.Done()
	}()

	if sshClient == nil {
		local.Close()
		f.mu.Lock()
		f.lastErr = "no SSH client"
		f.mu.Unlock()
		return
	}

	connectAddr := fmt.Sprintf("%s:%d", f.mapping.Connect.Host, f.mapping.Connect.Port)
	remote, err := sshClient.Dial("tcp", connectAddr)
	if err != nil {
		local.Close()
		f.mu.Lock()
		f.lastErr = fmt.Sprintf("dial %s: %s", connectAddr, err)
		f.mu.Unlock()
		return
	}

	biCopy(local, remote)
}

func biCopy(local, remote net.Conn) {
	ch := make(chan struct{}, 1)
	go func() {
		io.Copy(remote, local)
		ch <- struct{}{}
	}()
	io.Copy(local, remote)
	<-ch
	local.Close()
	remote.Close()
}

// Stop closes the listener and waits for active connections to drain.
func (f *LocalForwarder) Stop(ctx context.Context) error {
	f.mu.Lock()

	if f.state != "listening" {
		f.mu.Unlock()
		return nil
	}

	f.listener.Close()
	f.mu.Unlock()

	// Wait for accept loop to exit.
	select {
	case <-f.done:
	case <-ctx.Done():
	}

	// Wait for active connections to drain.
	waitDone := make(chan struct{})
	go func() {
		f.active.Wait()
		close(waitDone)
	}()

	select {
	case <-waitDone:
	case <-ctx.Done():
	}

	f.mu.Lock()
	f.state = "stopped"
	f.mu.Unlock()
	return nil
}

// Ensure interface compliance.
var _ Forwarder = (*LocalForwarder)(nil)
