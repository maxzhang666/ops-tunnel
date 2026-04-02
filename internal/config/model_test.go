package config

import (
	"testing"
)

func TestApplySSHConnectionDefaults(t *testing.T) {
	c := &SSHConnection{}
	ApplySSHConnectionDefaults(c)
	if c.DialTimeoutMs != 10000 {
		t.Errorf("DialTimeoutMs = %d, want 10000", c.DialTimeoutMs)
	}
	if c.KeepAlive.IntervalMs != 15000 {
		t.Errorf("KeepAlive.IntervalMs = %d, want 15000", c.KeepAlive.IntervalMs)
	}
	if c.KeepAlive.MaxMissed != 3 {
		t.Errorf("KeepAlive.MaxMissed = %d, want 3", c.KeepAlive.MaxMissed)
	}
	if c.HostKeyVerification.Mode != HostKeyAcceptNew {
		t.Errorf("HostKeyVerification.Mode = %s, want acceptNew", c.HostKeyVerification.Mode)
	}
}

func TestApplySSHConnectionDefaults_NoOverwrite(t *testing.T) {
	c := &SSHConnection{DialTimeoutMs: 5000}
	ApplySSHConnectionDefaults(c)
	if c.DialTimeoutMs != 5000 {
		t.Errorf("DialTimeoutMs = %d, want 5000 (should not overwrite)", c.DialTimeoutMs)
	}
}

func TestApplyTunnelDefaults(t *testing.T) {
	tun := &Tunnel{
		Mappings: []Mapping{{Listen: Endpoint{Port: 8080}}},
	}
	ApplyTunnelDefaults(tun)
	if tun.Mappings[0].Listen.Host != "127.0.0.1" {
		t.Errorf("Listen.Host = %s, want 127.0.0.1", tun.Mappings[0].Listen.Host)
	}
	if tun.Policy.RestartBackoff.MinMs != 500 {
		t.Errorf("RestartBackoff.MinMs = %d, want 500", tun.Policy.RestartBackoff.MinMs)
	}
}

func TestValidateSSHConnection_Valid(t *testing.T) {
	c := &SSHConnection{
		ID:       "test",
		Name:     "test-conn",
		Endpoint: Endpoint{Host: "1.2.3.4", Port: 22},
		Auth:     Auth{Type: AuthPassword, Username: "user", Password: "pass"},
	}
	r := ValidateSSHConnection(c, "")
	if r.HasErrors() {
		t.Errorf("expected no errors, got: %v", r.Errors)
	}
}

func TestValidateSSHConnection_InvalidPort(t *testing.T) {
	c := &SSHConnection{
		ID:       "test",
		Name:     "test-conn",
		Endpoint: Endpoint{Host: "1.2.3.4", Port: 0},
		Auth:     Auth{Type: AuthNone},
	}
	r := ValidateSSHConnection(c, "")
	if !r.HasErrors() {
		t.Error("expected validation errors for port 0")
	}
}

func TestValidateSSHConnection_PrivateKeyInline(t *testing.T) {
	c := &SSHConnection{
		ID:       "test",
		Name:     "test-conn",
		Endpoint: Endpoint{Host: "1.2.3.4", Port: 22},
		Auth: Auth{
			Type:     AuthPrivateKey,
			Username: "user",
			PrivateKey: &PrivateKey{
				Source: KeySourceInline,
				KeyPEM: "-----BEGIN OPENSSH PRIVATE KEY-----\ntest\n-----END OPENSSH PRIVATE KEY-----",
			},
		},
	}
	r := ValidateSSHConnection(c, "")
	if r.HasErrors() {
		t.Errorf("expected no errors, got: %v", r.Errors)
	}
}

func TestValidateSSHConnection_PrivateKeyFileMissing(t *testing.T) {
	c := &SSHConnection{
		ID:       "test",
		Name:     "test-conn",
		Endpoint: Endpoint{Host: "1.2.3.4", Port: 22},
		Auth: Auth{
			Type:       AuthPrivateKey,
			Username:   "user",
			PrivateKey: &PrivateKey{Source: KeySourceFile, FilePath: ""},
		},
	}
	r := ValidateSSHConnection(c, "")
	if !r.HasErrors() {
		t.Error("expected validation error for empty filePath")
	}
}

func TestValidateTunnel_Valid(t *testing.T) {
	sshIDs := map[string]bool{"ssh-1": true}
	tun := &Tunnel{
		ID:   "tun-1",
		Name: "test-tunnel",
		Mode: ModeLocal,
		Chain: []string{"ssh-1"},
		Mappings: []Mapping{
			{ID: "m1", Listen: Endpoint{Host: "127.0.0.1", Port: 15432}, Connect: Endpoint{Host: "127.0.0.1", Port: 5432}},
		},
		Policy: Policy{
			RestartBackoff:     RestartBackoff{MinMs: 500, MaxMs: 15000, Factor: 1.7},
			MaxRestartsPerHour: 60,
		},
	}
	r := ValidateTunnel(tun, sshIDs, "")
	if r.HasErrors() {
		t.Errorf("expected no errors, got: %v", r.Errors)
	}
}

func TestValidateTunnel_MissingSSHRef(t *testing.T) {
	sshIDs := map[string]bool{}
	tun := &Tunnel{
		ID:    "tun-1",
		Name:  "test",
		Mode:  ModeLocal,
		Chain: []string{"nonexistent"},
		Mappings: []Mapping{
			{ID: "m1", Listen: Endpoint{Host: "127.0.0.1", Port: 15432}, Connect: Endpoint{Host: "127.0.0.1", Port: 5432}},
		},
		Policy: Policy{
			RestartBackoff:     RestartBackoff{MinMs: 500, MaxMs: 15000, Factor: 1.7},
			MaxRestartsPerHour: 60,
		},
	}
	r := ValidateTunnel(tun, sshIDs, "")
	if !r.HasErrors() {
		t.Error("expected validation error for missing SSH ref")
	}
}

func TestValidateTunnel_DynamicWarning(t *testing.T) {
	sshIDs := map[string]bool{"ssh-1": true}
	tun := &Tunnel{
		ID:    "tun-1",
		Name:  "test",
		Mode:  ModeDynamic,
		Chain: []string{"ssh-1"},
		Mappings: []Mapping{
			{
				ID:     "m1",
				Listen: Endpoint{Host: "0.0.0.0", Port: 1080},
				Socks5: &Socks5Cfg{Auth: Socks5None},
			},
		},
		Policy: Policy{
			RestartBackoff:     RestartBackoff{MinMs: 500, MaxMs: 15000, Factor: 1.7},
			MaxRestartsPerHour: 60,
		},
	}
	r := ValidateTunnel(tun, sshIDs, "")
	if r.HasErrors() {
		t.Errorf("expected no errors, got: %v", r.Errors)
	}
	if len(r.Warnings) == 0 {
		t.Error("expected SOCKS5 security warning")
	}
}

func TestRedactSSHConnection(t *testing.T) {
	c := SSHConnection{
		ID:   "test",
		Auth: Auth{Type: AuthPassword, Username: "user", Password: "secret123"},
	}
	redacted := RedactSSHConnection(c)
	if redacted.Auth.Password != "***" {
		t.Errorf("Password = %s, want ***", redacted.Auth.Password)
	}
	if c.Auth.Password != "secret123" {
		t.Error("original password was mutated")
	}
}

func TestRedactSSHConnection_PrivateKey(t *testing.T) {
	c := SSHConnection{
		ID: "test",
		Auth: Auth{
			Type: AuthPrivateKey,
			PrivateKey: &PrivateKey{
				Source:     KeySourceInline,
				KeyPEM:     "-----BEGIN KEY-----",
				Passphrase: "mypass",
			},
		},
	}
	redacted := RedactSSHConnection(c)
	if redacted.Auth.PrivateKey.KeyPEM != "***" {
		t.Errorf("KeyPEM = %s, want ***", redacted.Auth.PrivateKey.KeyPEM)
	}
	if redacted.Auth.PrivateKey.Passphrase != "***" {
		t.Errorf("Passphrase = %s, want ***", redacted.Auth.PrivateKey.Passphrase)
	}
}

func TestRedactTunnel_Socks5(t *testing.T) {
	tun := Tunnel{
		ID: "test",
		Mappings: []Mapping{
			{
				ID:     "m1",
				Socks5: &Socks5Cfg{Auth: Socks5UserPass, Username: "user", Password: "secret"},
			},
		},
	}
	redacted := RedactTunnel(tun)
	if redacted.Mappings[0].Socks5.Password != "***" {
		t.Errorf("Socks5.Password = %s, want ***", redacted.Mappings[0].Socks5.Password)
	}
	if tun.Mappings[0].Socks5.Password != "secret" {
		t.Error("original socks5 password was mutated")
	}
}

func TestFindSSHConnectionReferences(t *testing.T) {
	cfg := &Config{
		Tunnels: []Tunnel{
			{Name: "tun-a", Chain: []string{"ssh-1", "ssh-2"}},
			{Name: "tun-b", Chain: []string{"ssh-2"}},
			{Name: "tun-c", Chain: []string{"ssh-3"}},
		},
	}
	refs := FindSSHConnectionReferences(cfg, "ssh-2")
	if len(refs) != 2 {
		t.Errorf("expected 2 references, got %d", len(refs))
	}
}
