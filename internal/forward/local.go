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
	mapping    config.Mapping
	listenAddr string // actual bound address after Start
	logFn      LogFunc

	mu       sync.RWMutex
	listener net.Listener
	state    string // "stopped" | "listening" | "error"
	lastErr  string
	done     chan struct{}

	active    sync.WaitGroup
	activeCnt atomic.Int32
	totalCnt  atomic.Int64
	bytesIn   atomic.Int64
	bytesOut  atomic.Int64
}

func (f *LocalForwarder) SetLogger(fn LogFunc) { f.logFn = fn }

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
		BytesIn:     f.bytesIn.Load(),
		BytesOut:    f.bytesOut.Load(),
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
	f.listenAddr = ln.Addr().String()
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

	biCopyCount(local, remote, &f.bytesIn, &f.bytesOut)
}

func biCopy(local, remote net.Conn) {
	biCopyCount(local, remote, nil, nil)
}

func biCopyCount(local, remote net.Conn, bytesIn, bytesOut *atomic.Int64) {
	ch := make(chan struct{}, 1)
	go func() {
		n, _ := io.Copy(remote, local)
		if bytesOut != nil {
			bytesOut.Add(n)
		}
		remote.Close()
		ch <- struct{}{}
	}()
	n, _ := io.Copy(local, remote)
	if bytesIn != nil {
		bytesIn.Add(n)
	}
	local.Close()
	<-ch
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
