package ssh

import (
	"testing"

	"github.com/maxzhang666/ops-tunnel/internal/config"
)

func TestAuthMethods_Password(t *testing.T) {
	methods, err := AuthMethods(config.Auth{
		Type:     config.AuthPassword,
		Username: "user",
		Password: "pass",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(methods) != 1 {
		t.Errorf("expected 1 method, got %d", len(methods))
	}
}

func TestAuthMethods_None(t *testing.T) {
	methods, err := AuthMethods(config.Auth{Type: config.AuthNone})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if methods != nil {
		t.Errorf("expected nil methods for none auth, got %v", methods)
	}
}

func TestAuthMethods_PrivateKeyInline_Invalid(t *testing.T) {
	_, err := AuthMethods(config.Auth{
		Type:     config.AuthPrivateKey,
		Username: "user",
		PrivateKey: &config.PrivateKey{
			Source: config.KeySourceInline,
			KeyPEM: "not-a-valid-key",
		},
	})
	if err == nil {
		t.Error("expected error for invalid key PEM")
	}
}

func TestAuthMethods_PrivateKeyNilConfig(t *testing.T) {
	_, err := AuthMethods(config.Auth{
		Type:     config.AuthPrivateKey,
		Username: "user",
	})
	if err == nil {
		t.Error("expected error for nil privateKey")
	}
}

func TestAuthMethods_UnknownType(t *testing.T) {
	_, err := AuthMethods(config.Auth{Type: "unknown"})
	if err == nil {
		t.Error("expected error for unknown auth type")
	}
}
