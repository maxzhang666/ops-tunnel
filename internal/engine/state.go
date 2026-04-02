package engine

import "time"

type TunnelState string

const (
	StateStopped  TunnelState = "stopped"
	StateStarting TunnelState = "starting"
	StateRunning  TunnelState = "running"
	StateDegraded TunnelState = "degraded"
	StateError    TunnelState = "error"
	StateStopping TunnelState = "stopping"
)

type HopStatus struct {
	SSHConnID string `json:"sshConnId"`
	State     string `json:"state"`
	LatencyMs int    `json:"latencyMs,omitempty"`
	Detail    string `json:"detail,omitempty"`
}

type MappingStatus struct {
	MappingID string `json:"mappingId"`
	State     string `json:"state"`
	Listen    string `json:"listen"`
	Detail    string `json:"detail,omitempty"`
}

type TunnelStatus struct {
	ID        string          `json:"id"`
	State     TunnelState     `json:"state"`
	Since     time.Time       `json:"since"`
	Chain     []HopStatus     `json:"chain"`
	Mappings  []MappingStatus `json:"mappings"`
	LastError string          `json:"lastError,omitempty"`
}
