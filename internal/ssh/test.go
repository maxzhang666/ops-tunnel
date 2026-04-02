package ssh

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/maxzhang666/ops-tunnel/internal/config"
	gossh "golang.org/x/crypto/ssh"
)

// TestResult holds the result of a connection test.
type TestResult struct {
	OK        bool   `json:"ok"`
	LatencyMs int64  `json:"latencyMs,omitempty"`
	Error     string `json:"error,omitempty"`
}

// TestConnection attempts to connect and authenticate to a single SSH server.
func TestConnection(ctx context.Context, conn config.SSHConnection, hostKeyStore HostKeyStore) TestResult {
	addr := fmt.Sprintf("%s:%d", conn.Endpoint.Host, conn.Endpoint.Port)
	timeout := time.Duration(conn.DialTimeoutMs) * time.Millisecond
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	authMethods, err := AuthMethods(conn.Auth)
	if err != nil {
		return TestResult{Error: fmt.Sprintf("auth config: %s", err)}
	}

	hkCallback := HostKeyCallback(conn.HostKeyVerification.Mode, hostKeyStore, addr)

	sshConfig := &gossh.ClientConfig{
		User:            conn.Auth.Username,
		Auth:            authMethods,
		HostKeyCallback: hkCallback,
		Timeout:         timeout,
	}

	start := time.Now()

	dialCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	dialer := net.Dialer{Timeout: timeout}
	netConn, err := dialer.DialContext(dialCtx, "tcp", addr)
	if err != nil {
		return TestResult{Error: fmt.Sprintf("dial: %s", err)}
	}

	sshConn, chans, reqs, err := gossh.NewClientConn(netConn, addr, sshConfig)
	if err != nil {
		netConn.Close()
		return TestResult{Error: fmt.Sprintf("SSH handshake: %s", err)}
	}

	client := gossh.NewClient(sshConn, chans, reqs)
	latency := time.Since(start).Milliseconds()
	client.Close()

	return TestResult{OK: true, LatencyMs: latency}
}
