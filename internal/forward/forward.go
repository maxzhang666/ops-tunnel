package forward

import (
	"context"

	gossh "golang.org/x/crypto/ssh"
)

// LogFunc is a callback for forwarder log messages.
type LogFunc func(level, message string)

// Forwarder manages a single port forwarding for one mapping.
type Forwarder interface {
	Start(ctx context.Context, sshClient *gossh.Client) error
	Stop(ctx context.Context) error
	Status() Status
	SetLogger(LogFunc)
}

// Status reports the current state of a forwarder.
type Status struct {
	MappingID   string `json:"mappingId"`
	State       string `json:"state"`       // "listening" | "stopped" | "error"
	Listen      string `json:"listen"`
	ActiveConns int    `json:"activeConns"`
	TotalConns  int64  `json:"totalConns"`
	BytesIn     int64  `json:"bytesIn"`
	BytesOut    int64  `json:"bytesOut"`
	LastError   string `json:"lastError,omitempty"`
}
