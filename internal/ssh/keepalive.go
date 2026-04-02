package ssh

import (
	"context"
	"log/slog"
	"time"

	gossh "golang.org/x/crypto/ssh"
)

// StartKeepAlive sends periodic keepalive requests to the SSH server.
func StartKeepAlive(ctx context.Context, client *gossh.Client, interval time.Duration, maxMissed int) <-chan error {
	errCh := make(chan error, 1)

	go func() {
		defer close(errCh)
		missed := 0
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_, _, err := client.SendRequest("keepalive@openssh.com", true, nil)
				if err != nil {
					missed++
					slog.Debug("keepalive failed", "missed", missed, "err", err)
					if missed >= maxMissed {
						errCh <- err
						return
					}
				} else {
					missed = 0
				}
			}
		}
	}()

	return errCh
}
