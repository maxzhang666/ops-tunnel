package config

import (
	"testing"
)

func TestValidateSSHConnection_HostKeyMode(t *testing.T) {
	for _, mode := range []HostKeyVerifyMode{"", HostKeyInsecure, HostKeyAcceptNew, HostKeyStrict} {
		c := &SSHConnection{
			ID:                  "test",
			Name:                "test-conn",
			Endpoint:            Endpoint{Host: "10.0.0.1", Port: 22},
			Auth:                Auth{Type: AuthNone},
			HostKeyVerification: HostKeyVerification{Mode: mode},
		}
		if r := ValidateSSHConnection(c, ""); r.HasErrors() {
			t.Errorf("mode %q should be valid, got: %v", mode, r.Errors)
		}
	}

	c := &SSHConnection{
		ID:                  "test",
		Name:                "test-conn",
		Endpoint:            Endpoint{Host: "10.0.0.1", Port: 22},
		Auth:                Auth{Type: AuthNone},
		HostKeyVerification: HostKeyVerification{Mode: HostKeyVerifyMode("acceptnew")},
	}
	r := ValidateSSHConnection(c, "")
	if !r.HasErrors() {
		t.Fatal("expected validation error for invalid hostKeyVerification.mode")
	}
	if r.Errors[0].Field != "hostKeyVerification.mode" {
		t.Errorf("Field = %s, want hostKeyVerification.mode", r.Errors[0].Field)
	}
}
