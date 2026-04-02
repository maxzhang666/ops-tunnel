package forward

import (
	"context"
	"io"
	"net"
	"testing"
	"time"

	"github.com/maxzhang666/ops-tunnel/internal/config"
)

func testMapping() config.Mapping {
	return config.Mapping{
		ID:      "m1",
		Listen:  config.Endpoint{Host: "127.0.0.1", Port: 0},
		Connect: config.Endpoint{Host: "127.0.0.1", Port: 5432},
	}
}

func TestLocalForwarder_InitialStatus(t *testing.T) {
	fwd := NewLocalForwarder(testMapping())
	st := fwd.Status()

	if st.MappingID != "m1" {
		t.Errorf("MappingID = %q, want %q", st.MappingID, "m1")
	}
	if st.State != "stopped" {
		t.Errorf("State = %q, want %q", st.State, "stopped")
	}
	if st.ActiveConns != 0 {
		t.Errorf("ActiveConns = %d, want 0", st.ActiveConns)
	}
	if st.TotalConns != 0 {
		t.Errorf("TotalConns = %d, want 0", st.TotalConns)
	}
}

func TestLocalForwarder_StartStop(t *testing.T) {
	fwd := NewLocalForwarder(testMapping())

	// Start with nil SSH client — only tests listener lifecycle.
	// handleConn calls sshClient.Dial only when a connection arrives.
	err := fwd.Start(context.Background(), nil)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	st := fwd.Status()
	if st.State != "listening" {
		t.Errorf("State = %q, want %q", st.State, "listening")
	}

	err = fwd.Stop(context.Background())
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	st = fwd.Status()
	if st.State != "stopped" {
		t.Errorf("State = %q after stop, want %q", st.State, "stopped")
	}
}

func TestLocalForwarder_PortInUse(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	addr := ln.Addr().(*net.TCPAddr)

	fwd := NewLocalForwarder(config.Mapping{
		ID:      "m-conflict",
		Listen:  config.Endpoint{Host: "127.0.0.1", Port: addr.Port},
		Connect: config.Endpoint{Host: "127.0.0.1", Port: 5432},
	})

	err = fwd.Start(context.Background(), nil)
	if err == nil {
		fwd.Stop(context.Background())
		t.Fatal("expected error for port in use")
	}
}

func TestBiCopy(t *testing.T) {
	localClient, localServer := net.Pipe()
	remoteClient, remoteServer := net.Pipe()

	done := make(chan struct{})
	go func() {
		biCopy(localServer, remoteClient)
		close(done)
	}()

	msg := []byte("hello forward")
	go func() {
		localClient.Write(msg)
		localClient.Close()
	}()

	buf := make([]byte, 64)
	n, err := remoteServer.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatalf("remote read error: %v", err)
	}
	if string(buf[:n]) != "hello forward" {
		t.Errorf("got %q, want %q", string(buf[:n]), "hello forward")
	}
	remoteServer.Close()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("biCopy did not exit in time")
	}
}
