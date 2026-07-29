package engine

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/maxzhang666/ops-tunnel/internal/config"
	tunnelssh "github.com/maxzhang666/ops-tunnel/internal/ssh"
	gossh "golang.org/x/crypto/ssh"
)

// startSSHServer runs an in-process SSH server on a random loopback port and
// returns its address plus the fingerprint and type of its host key.
//
// Host key verification happens during key exchange, before authentication, so
// a server that only completes KEX is enough to drive the verifier: the client
// aborts the handshake from inside the host key callback.
func startSSHServer(t *testing.T) (addr, fingerprint, keyType string) {
	t.Helper()

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate host key: %v", err)
	}
	signer, err := gossh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatalf("build signer: %v", err)
	}

	srvCfg := &gossh.ServerConfig{NoClientAuth: true}
	srvCfg.AddHostKey(signer)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { ln.Close() })

	go func() {
		for {
			netConn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				// The client rejects the host key mid-KEX, so this always
				// fails. Nothing to do but drop the connection.
				sshConn, chans, reqs, err := gossh.NewServerConn(c, srvCfg)
				if err != nil {
					return
				}
				go gossh.DiscardRequests(reqs)
				for nc := range chans {
					nc.Reject(gossh.UnknownChannelType, "unsupported")
				}
				sshConn.Close()
			}(netConn)
		}
	}()

	return ln.Addr().String(), gossh.FingerprintSHA256(signer.PublicKey()), signer.PublicKey().Type()
}

// hostKeyTestConfig builds a single-hop tunnel pointing at addr in strict mode,
// with auto-restart enabled so the suppression behaviour is observable.
func hostKeyTestConfig(t *testing.T, addr string) *config.Config {
	t.Helper()

	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("split host port: %v", err)
	}
	var port int
	if _, err := fmt.Sscanf(portStr, "%d", &port); err != nil {
		t.Fatalf("parse port: %v", err)
	}

	return &config.Config{
		Version: 1,
		SSHConnections: []config.SSHConnection{
			{
				ID:                  "ssh-1",
				Name:                "bastion",
				Endpoint:            config.Endpoint{Host: host, Port: port},
				Auth:                config.Auth{Type: config.AuthNone},
				HostKeyVerification: config.HostKeyVerification{Mode: config.HostKeyStrict},
				DialTimeoutMs:       5000,
			},
		},
		Tunnels: []config.Tunnel{
			{
				ID:    "tun-1",
				Name:  "test-tunnel",
				Mode:  config.ModeLocal,
				Chain: []string{"ssh-1"},
				Policy: config.Policy{
					AutoRestart:           true,
					RestartBackoff:        config.RestartBackoff{MinMs: 20, MaxMs: 50, Factor: 1.5},
					MaxRestartsPerHour:    60,
					GracefulStopTimeoutMs: 500,
				},
			},
		},
	}
}

func newHostKeyStore(t *testing.T) tunnelssh.HostKeyStore {
	t.Helper()
	store, err := tunnelssh.NewJSONHostKeyStore(filepath.Join(t.TempDir(), "known_hosts.json"))
	if err != nil {
		t.Fatalf("host key store: %v", err)
	}
	return store
}

func TestSupervisor_StatusHostKeyNilWithoutMismatch(t *testing.T) {
	cfg := testConfig()
	sup := newSupervisor(cfg.Tunnels[0], cfg.SSHConnections, NewEventBus(), tunnelssh.NewNoopHostKeyStore())

	if hk := sup.Status().HostKey; hk != nil {
		t.Errorf("HostKey = %+v, want nil for a fresh supervisor", hk)
	}
}

func TestSupervisor_StatusSurfacesHostKeyMismatch(t *testing.T) {
	cfg := testConfig()
	sup := newSupervisor(cfg.Tunnels[0], cfg.SSHConnections, NewEventBus(), tunnelssh.NewNoopHostKeyStore())

	want := &tunnelssh.HostKeyMismatch{
		SSHConnID:   "ssh-1",
		SSHConnName: "bastion",
		HostPort:    "10.0.0.1:22",
		Reason:      tunnelssh.HostKeyReasonChanged,
		StoredFP:    "SHA256:stored",
		OfferedFP:   "SHA256:offered",
		OfferedType: "ssh-ed25519",
	}

	sup.mu.Lock()
	sup.hostKeyMismatch = want
	sup.mu.Unlock()

	got := sup.Status().HostKey
	if got == nil {
		t.Fatal("HostKey = nil, want the recorded mismatch")
	}
	if *got != *want {
		t.Errorf("HostKey = %+v, want %+v", got, want)
	}
}

// The supervisor pulls the mismatch out of the chain error with errors.As, so
// the error type must stay reachable both bare and wrapped.
func TestHostKeyErrorExtractableWithErrorsAs(t *testing.T) {
	mm := &tunnelssh.HostKeyMismatch{HostPort: "10.0.0.1:22", Reason: tunnelssh.HostKeyReasonUnknown}
	hkErr := &tunnelssh.HostKeyError{Hop: 1, Mismatch: mm}

	for name, err := range map[string]error{
		"bare":    hkErr,
		"wrapped": fmt.Errorf("build chain: %w", hkErr),
	} {
		var target *tunnelssh.HostKeyError
		if !errors.As(err, &target) {
			t.Errorf("%s: errors.As failed to extract *HostKeyError", name)
			continue
		}
		if target.Mismatch != mm {
			t.Errorf("%s: extracted mismatch = %+v, want %+v", name, target.Mismatch, mm)
		}
	}

	var target *tunnelssh.HostKeyError
	if errors.As(errors.New("unrelated failure"), &target) {
		t.Error("errors.As matched an unrelated error")
	}
}

// End-to-end over a real handshake: an untrusted host key must land in
// TunnelStatus.HostKey, publish EventChainError, and stop the supervisor from
// retrying even though AutoRestart is on.
func TestSupervisor_HostKeyMismatchSuppressesAutoRestart(t *testing.T) {
	addr, wantFP, wantType := startSSHServer(t)
	cfg := hostKeyTestConfig(t, addr)
	store := newHostKeyStore(t)

	bus := NewEventBus()
	events, cancel := bus.Subscribe(128)
	defer cancel()

	eng := NewEngine(cfg, bus, store)
	if err := eng.StartTunnel(context.Background(), "tun-1"); err != nil {
		t.Fatalf("StartTunnel: %v", err)
	}
	t.Cleanup(func() { eng.Shutdown(context.Background()) })

	waitForState(t, eng, "tun-1", StateError, 10*time.Second)

	st, _ := eng.GetStatus("tun-1")
	mm := st.HostKey
	if mm == nil {
		t.Fatalf("HostKey = nil, want a mismatch; lastError = %q", st.LastError)
	}
	if mm.Reason != tunnelssh.HostKeyReasonUnknown {
		t.Errorf("reason = %q, want %q", mm.Reason, tunnelssh.HostKeyReasonUnknown)
	}
	if mm.OfferedFP != wantFP {
		t.Errorf("offeredFingerprint = %q, want %q", mm.OfferedFP, wantFP)
	}
	if mm.OfferedType != wantType {
		t.Errorf("offeredKeyType = %q, want %q", mm.OfferedType, wantType)
	}
	if mm.HostPort != addr {
		t.Errorf("hostPort = %q, want %q", mm.HostPort, addr)
	}
	if mm.SSHConnID != "ssh-1" || mm.SSHConnName != "bastion" {
		t.Errorf("connection identity = %q/%q, want ssh-1/bastion", mm.SSHConnID, mm.SSHConnName)
	}
	if mm.StoredFP != "" {
		t.Errorf("storedFingerprint = %q, want empty for an unknown host", mm.StoredFP)
	}

	// The offered key is parked for the user to confirm, never auto-trusted.
	if _, ok := store.Lookup(addr); ok {
		t.Error("strict mode must not trust the offered key")
	}
	if _, ok := store.Pending(addr); !ok {
		t.Error("offered key should be parked as pending")
	}

	// Retrying a changed host key is pointless, so exactly one chain error must
	// be published and the state must not churn back to starting.
	deadline := time.After(700 * time.Millisecond)
	chainErrors := 0
	for done := false; !done; {
		select {
		case ev := <-events:
			if ev.Type == EventChainError {
				chainErrors++
				if ev.Fields["offeredFingerprint"] != wantFP {
					t.Errorf("event offeredFingerprint = %v, want %q", ev.Fields["offeredFingerprint"], wantFP)
				}
				if ev.Fields["hostPort"] != addr {
					t.Errorf("event hostPort = %v, want %q", ev.Fields["hostPort"], addr)
				}
				if ev.Fields["reason"] != tunnelssh.HostKeyReasonUnknown {
					t.Errorf("event reason = %v, want %q", ev.Fields["reason"], tunnelssh.HostKeyReasonUnknown)
				}
			}
		case <-deadline:
			done = true
		}
	}
	if chainErrors != 1 {
		t.Errorf("EventChainError count = %d, want exactly 1 (no retry)", chainErrors)
	}

	st, _ = eng.GetStatus("tun-1")
	if st.State != StateError {
		t.Errorf("State = %s, want error to persist without retrying", st.State)
	}
	if st.HostKey == nil {
		t.Error("HostKey must remain set while the tunnel is wedged")
	}
}
