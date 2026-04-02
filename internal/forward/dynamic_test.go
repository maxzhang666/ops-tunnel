package forward

import (
	"context"
	"io"
	"net"
	"testing"
	"time"

	"github.com/maxzhang666/ops-tunnel/internal/config"
)

func testDynamicMapping() config.Mapping {
	return config.Mapping{
		ID:     "dm1",
		Listen: config.Endpoint{Host: "127.0.0.1", Port: 0},
		Socks5: &config.Socks5Cfg{
			Auth: config.Socks5None,
		},
	}
}

func TestDynamicForwarder_InitialStatus(t *testing.T) {
	fwd := NewDynamicForwarder(testDynamicMapping())
	st := fwd.Status()
	if st.MappingID != "dm1" {
		t.Errorf("MappingID = %q, want %q", st.MappingID, "dm1")
	}
	if st.State != "stopped" {
		t.Errorf("State = %q, want %q", st.State, "stopped")
	}
}

func TestDynamicForwarder_StartStop(t *testing.T) {
	fwd := NewDynamicForwarder(testDynamicMapping())
	err := fwd.Start(context.Background(), nil)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	st := fwd.Status()
	if st.State != "listening" {
		t.Errorf("State = %q, want %q", st.State, "listening")
	}
	err = fwd.Stop(context.Background())
	if err != nil {
		t.Fatalf("Stop: %v", err)
	}
	st = fwd.Status()
	if st.State != "stopped" {
		t.Errorf("State = %q, want %q", st.State, "stopped")
	}
}

func TestDynamicForwarder_StopWhenStopped(t *testing.T) {
	fwd := NewDynamicForwarder(testDynamicMapping())
	if err := fwd.Stop(context.Background()); err != nil {
		t.Fatalf("Stop on stopped: %v", err)
	}
}

func TestDynamicForwarder_NilSocks5Config(t *testing.T) {
	fwd := NewDynamicForwarder(config.Mapping{
		ID:     "dm-nil",
		Listen: config.Endpoint{Host: "127.0.0.1", Port: 0},
	})
	err := fwd.Start(context.Background(), nil)
	if err != nil {
		t.Fatalf("Start with nil Socks5: %v", err)
	}
	fwd.Stop(context.Background())
}

func TestDynamicForwarder_InvalidACL(t *testing.T) {
	fwd := NewDynamicForwarder(config.Mapping{
		ID:     "dm-bad-acl",
		Listen: config.Endpoint{Host: "127.0.0.1", Port: 0},
		Socks5: &config.Socks5Cfg{
			AllowCIDRs: []string{"not-a-cidr"},
		},
	})
	err := fwd.Start(context.Background(), nil)
	if err == nil {
		fwd.Stop(context.Background())
		t.Fatal("expected error for invalid ACL CIDR")
	}
}

// --- SOCKS5 Protocol Integration Tests ---

func TestDynamicForwarder_SOCKS5_WrongAuth(t *testing.T) {
	fwd := NewDynamicForwarder(config.Mapping{
		ID:     "dm-auth",
		Listen: config.Endpoint{Host: "127.0.0.1", Port: 0},
		Socks5: &config.Socks5Cfg{
			Auth:     config.Socks5UserPass,
			Username: "admin",
			Password: "secret",
		},
	})
	err := fwd.Start(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer fwd.Stop(context.Background())

	addr := fwd.Status().Listen

	// Offer only no-auth when userpass is required
	conn, err := net.DialTimeout("tcp", addr, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	conn.Write([]byte{0x05, 0x01, 0x00})
	resp := make([]byte, 2)
	io.ReadFull(conn, resp)
	if resp[1] != 0xFF {
		t.Errorf("expected 0xFF (no acceptable methods), got %d", resp[1])
	}
	conn.Close()
	time.Sleep(50 * time.Millisecond)
}

func TestDynamicForwarder_SOCKS5_UserPassAuth(t *testing.T) {
	fwd := NewDynamicForwarder(config.Mapping{
		ID:     "dm-userpass",
		Listen: config.Endpoint{Host: "127.0.0.1", Port: 0},
		Socks5: &config.Socks5Cfg{
			Auth:     config.Socks5UserPass,
			Username: "admin",
			Password: "secret",
		},
	})
	err := fwd.Start(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer fwd.Stop(context.Background())

	addr := fwd.Status().Listen

	// Wrong credentials
	conn, err := net.DialTimeout("tcp", addr, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	conn.Write([]byte{0x05, 0x01, 0x02})
	resp := make([]byte, 2)
	io.ReadFull(conn, resp)
	if resp[1] != 0x02 {
		t.Fatalf("expected method 0x02, got %d", resp[1])
	}
	conn.Write([]byte{0x01, 0x05, 'w', 'r', 'o', 'n', 'g', 0x03, 'b', 'a', 'd'})
	authResp := make([]byte, 2)
	io.ReadFull(conn, authResp)
	if authResp[1] == 0x00 {
		t.Error("expected auth failure for wrong credentials")
	}
	conn.Close()

	// Correct credentials
	conn2, err := net.DialTimeout("tcp", addr, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	conn2.Write([]byte{0x05, 0x01, 0x02})
	resp2 := make([]byte, 2)
	io.ReadFull(conn2, resp2)
	conn2.Write([]byte{0x01, 0x05, 'a', 'd', 'm', 'i', 'n', 0x06, 's', 'e', 'c', 'r', 'e', 't'})
	authResp2 := make([]byte, 2)
	io.ReadFull(conn2, authResp2)
	if authResp2[1] != 0x00 {
		t.Errorf("expected auth success, got status %d", authResp2[1])
	}
	conn2.Close()
	time.Sleep(50 * time.Millisecond)
}

func TestDynamicForwarder_SOCKS5_ACLReject(t *testing.T) {
	fwd := NewDynamicForwarder(config.Mapping{
		ID:     "dm-acl",
		Listen: config.Endpoint{Host: "127.0.0.1", Port: 0},
		Socks5: &config.Socks5Cfg{
			Auth:       config.Socks5None,
			AllowCIDRs: []string{"10.0.0.0/8"},
		},
	})
	err := fwd.Start(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer fwd.Stop(context.Background())

	addr := fwd.Status().Listen
	conn, err := net.DialTimeout("tcp", addr, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	// Handshake
	conn.Write([]byte{0x05, 0x01, 0x00})
	resp := make([]byte, 2)
	io.ReadFull(conn, resp)

	// CONNECT to 8.8.8.8:80 (not in allow list)
	conn.Write([]byte{0x05, 0x01, 0x00, 0x01, 8, 8, 8, 8, 0x00, 0x50})
	reply := make([]byte, 10)
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	io.ReadFull(conn, reply)
	if reply[1] != RepNotAllowed {
		t.Errorf("expected RepNotAllowed (%d), got %d", RepNotAllowed, reply[1])
	}
	conn.Close()
	time.Sleep(50 * time.Millisecond)
}

func TestDynamicForwarder_SOCKS5_UDPAssociateRejected(t *testing.T) {
	fwd := NewDynamicForwarder(testDynamicMapping())
	err := fwd.Start(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer fwd.Stop(context.Background())

	addr := fwd.Status().Listen
	conn, err := net.DialTimeout("tcp", addr, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	conn.Write([]byte{0x05, 0x01, 0x00})
	resp := make([]byte, 2)
	io.ReadFull(conn, resp)

	// UDP ASSOCIATE
	conn.Write([]byte{0x05, 0x03, 0x00, 0x01, 0, 0, 0, 0, 0x00, 0x00})
	reply := make([]byte, 10)
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	io.ReadFull(conn, reply)
	if reply[1] != RepCmdNotSupported {
		t.Errorf("expected RepCmdNotSupported (%d), got %d", RepCmdNotSupported, reply[1])
	}
	conn.Close()
	time.Sleep(50 * time.Millisecond)
}
