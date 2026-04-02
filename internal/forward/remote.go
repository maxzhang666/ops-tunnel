package forward

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/maxzhang666/ops-tunnel/internal/config"
	gossh "golang.org/x/crypto/ssh"
)

// RemoteForwarder implements remote (-R) port forwarding.
type RemoteForwarder struct {
	mapping    config.Mapping
	listenAddr string // actual remote address returned by sshClient.Listen

	mu       sync.RWMutex
	listener net.Listener
	state    string // "stopped" | "listening" | "error"
	lastErr  string
	done     chan struct{}

	active    sync.WaitGroup
	activeCnt atomic.Int32
	totalCnt  atomic.Int64
}

// NewRemoteForwarder creates a new remote forwarder for the given mapping.
func NewRemoteForwarder(m config.Mapping) *RemoteForwarder {
	return &RemoteForwarder{
		mapping: m,
		state:   "stopped",
	}
}

// Status returns the current state of this forwarder.
func (f *RemoteForwarder) Status() Status {
	f.mu.RLock()
	defer f.mu.RUnlock()
	listen := f.listenAddr
	if listen == "" {
		listen = fmt.Sprintf("%s:%d", f.mapping.Listen.Host, f.mapping.Listen.Port)
	}
	return Status{
		MappingID:   f.mapping.ID,
		State:       f.state,
		Listen:      listen,
		ActiveConns: int(f.activeCnt.Load()),
		TotalConns:  f.totalCnt.Load(),
		LastError:   f.lastErr,
	}
}

// Start requests the remote SSH server to listen and begins forwarding.
func (f *RemoteForwarder) Start(ctx context.Context, sshClient *gossh.Client) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.state == "listening" {
		return fmt.Errorf("already listening")
	}

	if sshClient == nil {
		return fmt.Errorf("no SSH client")
	}

	requestAddr := fmt.Sprintf("%s:%d", f.mapping.Listen.Host, f.mapping.Listen.Port)
	ln, err := sshClient.Listen("tcp", requestAddr)
	if err != nil {
		return fmt.Errorf("remote listen %s: %w", requestAddr, err)
	}

	f.listener = ln
	f.listenAddr = ln.Addr().String()
	f.state = "listening"
	f.lastErr = ""
	f.done = make(chan struct{})

	if warn := detectGatewayPortsRestriction(requestAddr, f.listenAddr); warn != "" {
		f.lastErr = warn
	}

	go f.acceptLoop()
	return nil
}

func (f *RemoteForwarder) acceptLoop() {
	defer close(f.done)
	for {
		conn, err := f.listener.Accept()
		if err != nil {
			return // listener closed
		}
		f.active.Add(1)
		f.activeCnt.Add(1)
		go f.handleConn(conn)
	}
}

func (f *RemoteForwarder) handleConn(remote net.Conn) {
	defer func() {
		f.activeCnt.Add(-1)
		f.totalCnt.Add(1)
		f.active.Done()
	}()

	connectAddr := fmt.Sprintf("%s:%d", f.mapping.Connect.Host, f.mapping.Connect.Port)
	local, err := net.DialTimeout("tcp", connectAddr, 10*time.Second)
	if err != nil {
		remote.Close()
		f.mu.Lock()
		f.lastErr = fmt.Sprintf("dial local %s: %s", connectAddr, err)
		f.mu.Unlock()
		return
	}

	biCopy(remote, local)
}

// Stop closes the remote listener and waits for active connections to drain.
func (f *RemoteForwarder) Stop(ctx context.Context) error {
	f.mu.Lock()

	if f.state != "listening" {
		f.mu.Unlock()
		return nil
	}

	f.listener.Close()
	f.mu.Unlock()

	select {
	case <-f.done:
	case <-ctx.Done():
	}

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

// detectGatewayPortsRestriction checks if the SSH server restricted the listen address.
func detectGatewayPortsRestriction(requested, actual string) string {
	if !strings.HasPrefix(requested, "0.0.0.0:") {
		return ""
	}
	if strings.HasPrefix(actual, "127.0.0.1:") || strings.HasPrefix(actual, "[::1]:") {
		return fmt.Sprintf("GatewayPorts may be disabled: requested %s but got %s", requested, actual)
	}
	return ""
}

var _ Forwarder = (*RemoteForwarder)(nil)
