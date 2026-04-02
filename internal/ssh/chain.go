package ssh

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/maxzhang666/ops-tunnel/internal/config"
	gossh "golang.org/x/crypto/ssh"
)

// ChainResult holds all SSH clients created for a multi-hop chain.
type ChainResult struct {
	Clients  []*gossh.Client
	KACancel context.CancelFunc
	KAErrors []<-chan error
}

// Last returns the final client in the chain.
func (r *ChainResult) Last() *gossh.Client {
	if len(r.Clients) == 0 {
		return nil
	}
	return r.Clients[len(r.Clients)-1]
}

// Close shuts down all clients in reverse order.
func (r *ChainResult) Close() error {
	if r.KACancel != nil {
		r.KACancel()
	}
	var lastErr error
	for i := len(r.Clients) - 1; i >= 0; i-- {
		if err := r.Clients[i].Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// BuildChain establishes a chain of SSH connections through the given hops.
func BuildChain(ctx context.Context, conns []config.SSHConnection, hostKeyStore HostKeyStore) (*ChainResult, error) {
	if len(conns) == 0 {
		return nil, fmt.Errorf("chain must have at least one connection")
	}

	kaCtx, kaCancel := context.WithCancel(context.Background())
	result := &ChainResult{
		KACancel: kaCancel,
	}

	var prevClient *gossh.Client

	for i, conn := range conns {
		addr := fmt.Sprintf("%s:%d", conn.Endpoint.Host, conn.Endpoint.Port)
		timeout := time.Duration(conn.DialTimeoutMs) * time.Millisecond
		if timeout == 0 {
			timeout = 10 * time.Second
		}

		slog.Info("connecting hop", "hop", i+1, "name", conn.Name, "addr", addr)

		authMethods, err := AuthMethods(conn.Auth)
		if err != nil {
			result.Close()
			return nil, fmt.Errorf("hop %d (%s): auth config error: %w", i+1, conn.Name, err)
		}

		hkCallback := HostKeyCallback(conn.HostKeyVerification.Mode, hostKeyStore, addr)

		sshConfig := &gossh.ClientConfig{
			User:            conn.Auth.Username,
			Auth:            authMethods,
			HostKeyCallback: hkCallback,
			Timeout:         timeout,
		}

		var netConn net.Conn
		if prevClient == nil {
			dialer := net.Dialer{Timeout: timeout}
			dialCtx, dialCancel := context.WithTimeout(ctx, timeout)
			netConn, err = dialer.DialContext(dialCtx, "tcp", addr)
			dialCancel()
		} else {
			netConn, err = prevClient.Dial("tcp", addr)
		}
		if err != nil {
			result.Close()
			return nil, fmt.Errorf("hop %d (%s): dial failed: %w", i+1, conn.Name, err)
		}

		sshConn, chans, reqs, err := gossh.NewClientConn(netConn, addr, sshConfig)
		if err != nil {
			netConn.Close()
			result.Close()
			return nil, fmt.Errorf("hop %d (%s): SSH handshake failed: %w", i+1, conn.Name, err)
		}

		client := gossh.NewClient(sshConn, chans, reqs)
		result.Clients = append(result.Clients, client)

		if conn.KeepAlive.IntervalMs > 0 && conn.KeepAlive.MaxMissed > 0 {
			interval := time.Duration(conn.KeepAlive.IntervalMs) * time.Millisecond
			kaErrCh := StartKeepAlive(kaCtx, client, interval, conn.KeepAlive.MaxMissed)
			result.KAErrors = append(result.KAErrors, kaErrCh)
		}

		slog.Info("hop connected", "hop", i+1, "name", conn.Name)
		prevClient = client
	}

	return result, nil
}
