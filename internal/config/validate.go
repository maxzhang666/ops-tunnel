package config

import (
	"fmt"
	"strings"
)

// ValidationError represents a single field validation failure.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationResult holds errors (blocking) and warnings (non-blocking).
type ValidationResult struct {
	Errors   []ValidationError `json:"errors,omitempty"`
	Warnings []string          `json:"warnings,omitempty"`
}

// HasErrors returns true if there are blocking validation errors.
func (r *ValidationResult) HasErrors() bool {
	return len(r.Errors) > 0
}

func (r *ValidationResult) addError(field, msg string) {
	r.Errors = append(r.Errors, ValidationError{Field: field, Message: msg})
}

func (r *ValidationResult) addWarning(msg string) {
	r.Warnings = append(r.Warnings, msg)
}

func validatePort(port int) bool {
	return port >= 1 && port <= 65535
}

// ValidateSSHConnection validates a single SSH connection (without uniqueness check).
func ValidateSSHConnection(c *SSHConnection, prefix string) *ValidationResult {
	r := &ValidationResult{}
	p := prefix

	if c.Name == "" {
		r.addError(p+"name", "must not be empty")
	}
	if c.Endpoint.Host == "" {
		r.addError(p+"endpoint.host", "must not be empty")
	}
	if !validatePort(c.Endpoint.Port) {
		r.addError(p+"endpoint.port", "must be between 1 and 65535")
	}

	switch c.Auth.Type {
	case AuthPassword:
		if c.Auth.Username == "" {
			r.addError(p+"auth.username", "must not be empty for password auth")
		}
		if c.Auth.Password == "" {
			r.addError(p+"auth.password", "must not be empty for password auth")
		}
	case AuthPrivateKey:
		if c.Auth.Username == "" {
			r.addError(p+"auth.username", "must not be empty for privateKey auth")
		}
		if c.Auth.PrivateKey == nil {
			r.addError(p+"auth.privateKey", "must be provided for privateKey auth")
		} else {
			switch c.Auth.PrivateKey.Source {
			case KeySourceInline:
				if c.Auth.PrivateKey.KeyPEM == "" {
					r.addError(p+"auth.privateKey.keyPem", "must not be empty for inline source")
				}
			case KeySourceFile:
				if c.Auth.PrivateKey.FilePath == "" {
					r.addError(p+"auth.privateKey.filePath", "must not be empty for file source")
				}
			default:
				r.addError(p+"auth.privateKey.source", "must be 'inline' or 'file'")
			}
		}
	case AuthNone:
		// no additional fields required
	default:
		r.addError(p+"auth.type", "must be 'password', 'privateKey', or 'none'")
	}

	return r
}

// validateMapping validates a single mapping for the given tunnel mode.
func validateMapping(m *Mapping, mode TunnelMode, prefix string) *ValidationResult {
	r := &ValidationResult{}
	p := prefix

	if !validatePort(m.Listen.Port) {
		r.addError(p+"listen.port", "must be between 1 and 65535")
	}

	switch mode {
	case ModeLocal, ModeRemote:
		if m.Connect.Host == "" {
			r.addError(p+"connect.host", "must not be empty for "+string(mode)+" mode")
		}
		if !validatePort(m.Connect.Port) {
			r.addError(p+"connect.port", "must be between 1 and 65535")
		}
	case ModeDynamic:
		if m.Socks5 == nil {
			r.addError(p+"socks5", "must be provided for dynamic mode")
		} else {
			switch m.Socks5.Auth {
			case Socks5UserPass:
				if m.Socks5.Username == "" {
					r.addError(p+"socks5.username", "must not be empty for userpass auth")
				}
				if m.Socks5.Password == "" {
					r.addError(p+"socks5.password", "must not be empty for userpass auth")
				}
			case Socks5None:
				if m.Listen.Host == "0.0.0.0" {
					wideOpen := len(m.Socks5.AllowCIDRs) == 0
					if !wideOpen {
						for _, cidr := range m.Socks5.AllowCIDRs {
							if cidr == "0.0.0.0/0" {
								wideOpen = true
								break
							}
						}
					}
					if wideOpen {
						r.addWarning("SOCKS5 proxy is publicly accessible without authentication")
					}
				}
			default:
				r.addError(p+"socks5.auth", "must be 'none' or 'userpass'")
			}
		}
	}

	return r
}

// ValidateTunnel validates a single tunnel. sshIDs is the set of known SSH connection IDs.
func ValidateTunnel(t *Tunnel, sshIDs map[string]bool, prefix string) *ValidationResult {
	r := &ValidationResult{}
	p := prefix

	if t.Name == "" {
		r.addError(p+"name", "must not be empty")
	}

	switch t.Mode {
	case ModeLocal, ModeRemote, ModeDynamic:
	default:
		r.addError(p+"mode", "must be 'local', 'remote', or 'dynamic'")
	}

	if len(t.Chain) == 0 {
		r.addError(p+"chain", "must contain at least one SSH connection")
	} else {
		for i, id := range t.Chain {
			if !sshIDs[id] {
				r.addError(fmt.Sprintf("%schain[%d]", p, i), fmt.Sprintf("SSH connection '%s' not found", id))
			}
		}
	}

	if len(t.Mappings) == 0 {
		r.addError(p+"mappings", "must contain at least one mapping")
	} else {
		seenMappingIDs := make(map[string]bool)
		for i, m := range t.Mappings {
			mp := fmt.Sprintf("%smappings[%d].", p, i)
			if m.ID == "" {
				r.addError(mp+"id", "must not be empty")
			} else if seenMappingIDs[m.ID] {
				r.addError(mp+"id", "must be unique within tunnel")
			}
			seenMappingIDs[m.ID] = true

			mr := validateMapping(&m, t.Mode, mp)
			r.Errors = append(r.Errors, mr.Errors...)
			r.Warnings = append(r.Warnings, mr.Warnings...)
		}
	}

	pp := p + "policy."
	if t.Policy.RestartBackoff.MinMs <= 0 {
		r.addError(pp+"restartBackoff.minMs", "must be greater than 0")
	}
	if t.Policy.RestartBackoff.MaxMs < t.Policy.RestartBackoff.MinMs {
		r.addError(pp+"restartBackoff.maxMs", "must be greater than or equal to minMs")
	}
	if t.Policy.RestartBackoff.Factor < 1.0 {
		r.addError(pp+"restartBackoff.factor", "must be greater than or equal to 1.0")
	}
	if t.Policy.MaxRestartsPerHour <= 0 {
		r.addError(pp+"maxRestartsPerHour", "must be greater than 0")
	}

	return r
}

// ValidateConfig validates the entire config for internal consistency.
func ValidateConfig(cfg *Config) *ValidationResult {
	r := &ValidationResult{}

	sshIDs := make(map[string]bool, len(cfg.SSHConnections))
	for i, c := range cfg.SSHConnections {
		prefix := fmt.Sprintf("sshConnections[%d].", i)
		if c.ID == "" {
			r.addError(prefix+"id", "must not be empty")
		} else if sshIDs[c.ID] {
			r.addError(prefix+"id", "must be unique")
		}
		sshIDs[c.ID] = true

		cr := ValidateSSHConnection(&c, prefix)
		r.Errors = append(r.Errors, cr.Errors...)
		r.Warnings = append(r.Warnings, cr.Warnings...)
	}

	tunnelIDs := make(map[string]bool, len(cfg.Tunnels))
	for i, t := range cfg.Tunnels {
		prefix := fmt.Sprintf("tunnels[%d].", i)
		if t.ID == "" {
			r.addError(prefix+"id", "must not be empty")
		} else if tunnelIDs[t.ID] {
			r.addError(prefix+"id", "must be unique")
		}
		tunnelIDs[t.ID] = true

		tr := ValidateTunnel(&t, sshIDs, prefix)
		r.Errors = append(r.Errors, tr.Errors...)
		r.Warnings = append(r.Warnings, tr.Warnings...)
	}

	return r
}

// FindSSHConnectionReferences returns tunnel names that reference the given SSH connection ID.
func FindSSHConnectionReferences(cfg *Config, sshID string) []string {
	var names []string
	for _, t := range cfg.Tunnels {
		for _, chainID := range t.Chain {
			if chainID == sshID {
				names = append(names, t.Name)
				break
			}
		}
	}
	return names
}

// FormatErrors returns a human-readable summary of validation errors.
func FormatErrors(errs []ValidationError) string {
	parts := make([]string, len(errs))
	for i, e := range errs {
		parts[i] = e.Field + ": " + e.Message
	}
	return strings.Join(parts, "; ")
}
