package config

// ApplySSHConnectionDefaults fills zero-value fields with sensible defaults.
func ApplySSHConnectionDefaults(c *SSHConnection) {
	if c.DialTimeoutMs == 0 {
		c.DialTimeoutMs = 10000
	}
	if c.KeepAlive.IntervalMs == 0 {
		c.KeepAlive.IntervalMs = 15000
	}
	if c.KeepAlive.MaxMissed == 0 {
		c.KeepAlive.MaxMissed = 3
	}
	if c.HostKeyVerification.Mode == "" {
		c.HostKeyVerification.Mode = HostKeyAcceptNew
	}
}

// ApplyMappingDefaults fills zero-value fields on a mapping.
func ApplyMappingDefaults(m *Mapping) {
	if m.Listen.Host == "" {
		m.Listen.Host = "127.0.0.1"
	}
}

// ApplyTunnelDefaults fills zero-value fields with sensible defaults.
func ApplyTunnelDefaults(t *Tunnel) {
	for i := range t.Mappings {
		ApplyMappingDefaults(&t.Mappings[i])
	}
	p := &t.Policy
	if !p.AutoRestart {
		if p.RestartBackoff.MinMs == 0 && p.RestartBackoff.MaxMs == 0 {
			p.AutoRestart = true
		}
	}
	if p.RestartBackoff.MinMs == 0 {
		p.RestartBackoff.MinMs = 500
	}
	if p.RestartBackoff.MaxMs == 0 {
		p.RestartBackoff.MaxMs = 15000
	}
	if p.RestartBackoff.Factor == 0 {
		p.RestartBackoff.Factor = 1.7
	}
	if p.MaxRestartsPerHour == 0 {
		p.MaxRestartsPerHour = 60
	}
	if p.GracefulStopTimeoutMs == 0 {
		p.GracefulStopTimeoutMs = 5000
	}
}

// ApplyConfigDefaults applies defaults to all entities in a config.
func ApplyConfigDefaults(cfg *Config) {
	if cfg.Desktop.CloseAction == "" {
		cfg.Desktop.CloseAction = "ask"
	}
	for i := range cfg.SSHConnections {
		ApplySSHConnectionDefaults(&cfg.SSHConnections[i])
	}
	for i := range cfg.Tunnels {
		ApplyTunnelDefaults(&cfg.Tunnels[i])
	}
}
