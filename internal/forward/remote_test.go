package forward

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/maxzhang666/ops-tunnel/internal/config"
)

func testRemoteMapping() config.Mapping {
	return config.Mapping{
		ID:      "rm1",
		Listen:  config.Endpoint{Host: "0.0.0.0", Port: 18080},
		Connect: config.Endpoint{Host: "127.0.0.1", Port: 8080},
	}
}

func TestRemoteForwarder_InitialStatus(t *testing.T) {
	fwd := NewRemoteForwarder(testRemoteMapping())
	st := fwd.Status()

	if st.MappingID != "rm1" {
		t.Errorf("MappingID = %q, want %q", st.MappingID, "rm1")
	}
	if st.State != "stopped" {
		t.Errorf("State = %q, want %q", st.State, "stopped")
	}
	if st.Listen != "0.0.0.0:18080" {
		t.Errorf("Listen = %q, want %q", st.Listen, "0.0.0.0:18080")
	}
	if st.ActiveConns != 0 {
		t.Errorf("ActiveConns = %d, want 0", st.ActiveConns)
	}
	if st.TotalConns != 0 {
		t.Errorf("TotalConns = %d, want 0", st.TotalConns)
	}
}

func TestRemoteForwarder_StopWhenStopped(t *testing.T) {
	fwd := NewRemoteForwarder(testRemoteMapping())
	err := fwd.Stop(context.Background())
	if err != nil {
		t.Fatalf("Stop on stopped forwarder should be no-op, got: %v", err)
	}
}

func TestRemoteForwarder_HandleConn_LocalDial(t *testing.T) {
	// Start a local server that reads a message and writes a response.
	svcLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer svcLn.Close()

	svcDone := make(chan struct{})
	go func() {
		defer close(svcDone)
		c, err := svcLn.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		buf := make([]byte, 64)
		n, _ := c.Read(buf)
		c.Write(buf[:n])
	}()

	svcAddr := svcLn.Addr().(*net.TCPAddr)

	fwd := NewRemoteForwarder(config.Mapping{
		ID:      "rm-echo",
		Listen:  config.Endpoint{Host: "127.0.0.1", Port: 0},
		Connect: config.Endpoint{Host: "127.0.0.1", Port: svcAddr.Port},
	})

	// Manually set up the forwarder with a local listener (simulating sshClient.Listen)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	fwd.mu.Lock()
	fwd.listener = ln
	fwd.listenAddr = ln.Addr().String()
	fwd.state = "listening"
	fwd.done = make(chan struct{})
	fwd.mu.Unlock()

	go fwd.acceptLoop()

	conn, err := net.DialTimeout("tcp", ln.Addr().String(), time.Second)
	if err != nil {
		t.Fatalf("dial forwarder: %v", err)
	}

	msg := []byte("hello remote")
	conn.Write(msg)

	buf := make([]byte, 64)
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if string(buf[:n]) != "hello remote" {
		t.Errorf("got %q, want %q", string(buf[:n]), "hello remote")
	}

	// Wait for the service goroutine to close its side, then close ours.
	<-svcDone
	conn.Close()
	time.Sleep(100 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	fwd.Stop(ctx)

	st := fwd.Status()
	if st.TotalConns != 1 {
		t.Errorf("TotalConns = %d, want 1", st.TotalConns)
	}
}

func TestRemoteForwarder_HandleConn_LocalUnreachable(t *testing.T) {
	fwd := NewRemoteForwarder(config.Mapping{
		ID:      "rm-unreachable",
		Listen:  config.Endpoint{Host: "127.0.0.1", Port: 0},
		Connect: config.Endpoint{Host: "127.0.0.1", Port: 1},
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	fwd.mu.Lock()
	fwd.listener = ln
	fwd.listenAddr = ln.Addr().String()
	fwd.state = "listening"
	fwd.done = make(chan struct{})
	fwd.mu.Unlock()

	go fwd.acceptLoop()

	conn, err := net.DialTimeout("tcp", ln.Addr().String(), time.Second)
	if err != nil {
		t.Fatalf("dial forwarder: %v", err)
	}

	buf := make([]byte, 64)
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, err = conn.Read(buf)
	if err == nil {
		t.Fatal("expected read error after local dial failure")
	}
	conn.Close()

	time.Sleep(100 * time.Millisecond)
	fwd.Stop(context.Background())

	st := fwd.Status()
	if st.LastError == "" {
		t.Error("expected lastErr to be set after local dial failure")
	}
	if st.TotalConns != 1 {
		t.Errorf("TotalConns = %d, want 1", st.TotalConns)
	}
}

func TestRemoteForwarder_GatewayPortsWarning(t *testing.T) {
	warn := detectGatewayPortsRestriction("0.0.0.0:18080", "127.0.0.1:18080")
	if warn == "" {
		t.Error("expected warning when 0.0.0.0 was requested but 127.0.0.1 was returned")
	}
	if !strings.Contains(warn, "GatewayPorts") {
		t.Errorf("warning should mention GatewayPorts, got: %q", warn)
	}

	warn = detectGatewayPortsRestriction("127.0.0.1:18080", "127.0.0.1:18080")
	if warn != "" {
		t.Errorf("expected no warning for matching addresses, got: %q", warn)
	}

	warn = detectGatewayPortsRestriction("10.0.0.1:18080", "127.0.0.1:18080")
	if warn != "" {
		t.Errorf("expected no warning for non-0.0.0.0 request, got: %q", warn)
	}
}

func TestRemoteForwarder_DoubleStop(t *testing.T) {
	fwd := NewRemoteForwarder(testRemoteMapping())

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	fwd.mu.Lock()
	fwd.listener = ln
	fwd.listenAddr = ln.Addr().String()
	fwd.state = "listening"
	fwd.done = make(chan struct{})
	fwd.mu.Unlock()
	go fwd.acceptLoop()

	if err := fwd.Stop(context.Background()); err != nil {
		t.Fatalf("first stop: %v", err)
	}
	if err := fwd.Stop(context.Background()); err != nil {
		t.Fatalf("second stop should be no-op: %v", err)
	}
}
