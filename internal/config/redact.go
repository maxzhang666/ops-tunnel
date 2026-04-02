package config

const redacted = "***"

// RedactSSHConnection returns a copy with sensitive fields masked.
func RedactSSHConnection(c SSHConnection) SSHConnection {
	if c.Auth.Password != "" {
		c.Auth.Password = redacted
	}
	if c.Auth.PrivateKey != nil {
		pk := *c.Auth.PrivateKey
		if pk.KeyPEM != "" {
			pk.KeyPEM = redacted
		}
		if pk.Passphrase != "" {
			pk.Passphrase = redacted
		}
		c.Auth.PrivateKey = &pk
	}
	return c
}

// RedactTunnel returns a copy with sensitive fields masked.
func RedactTunnel(t Tunnel) Tunnel {
	mappings := make([]Mapping, len(t.Mappings))
	for i, m := range t.Mappings {
		if m.Socks5 != nil {
			s := *m.Socks5
			if s.Password != "" {
				s.Password = redacted
			}
			m.Socks5 = &s
		}
		mappings[i] = m
	}
	t.Mappings = mappings
	return t
}

// RedactConfig returns a full copy with all sensitive fields masked.
func RedactConfig(cfg Config) Config {
	out := Config{
		Version:        cfg.Version,
		SSHConnections: make([]SSHConnection, len(cfg.SSHConnections)),
		Tunnels:        make([]Tunnel, len(cfg.Tunnels)),
	}
	for i, c := range cfg.SSHConnections {
		out.SSHConnections[i] = RedactSSHConnection(c)
	}
	for i, t := range cfg.Tunnels {
		out.Tunnels[i] = RedactTunnel(t)
	}
	return out
}
