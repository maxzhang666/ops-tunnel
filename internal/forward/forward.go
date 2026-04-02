package forward

import (
	"context"

	gossh "golang.org/x/crypto/ssh"
)

// Forwarder manages a single port forwarding for one mapping.
type Forwarder interface {
	Start(ctx context.Context, sshClient *gossh.Client) error
	Stop(ctx context.Context) error
	Status() Status
}

// Status reports the current state of a forwarder.
type Status struct {
	MappingID   string `json:"mappingId"`
	State       string `json:"state"`       // "listening" | "stopped" | "error"
	Listen      string `json:"listen"`
	ActiveConns int    `json:"activeConns"`
	TotalConns  int64  `json:"totalConns"`
	LastError   string `json:"lastError,omitempty"`
}
