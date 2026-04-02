package ssh

import (
	"fmt"
	"os"

	"github.com/maxzhang666/ops-tunnel/internal/config"
	gossh "golang.org/x/crypto/ssh"
)

// AuthMethods converts config.Auth to SSH auth methods.
func AuthMethods(a config.Auth) ([]gossh.AuthMethod, error) {
	switch a.Type {
	case config.AuthPassword:
		return []gossh.AuthMethod{gossh.Password(a.Password)}, nil

	case config.AuthPrivateKey:
		if a.PrivateKey == nil {
			return nil, fmt.Errorf("privateKey config is nil")
		}
		var pemBytes []byte
		switch a.PrivateKey.Source {
		case config.KeySourceInline:
			pemBytes = []byte(a.PrivateKey.KeyPEM)
		case config.KeySourceFile:
			var err error
			pemBytes, err = os.ReadFile(a.PrivateKey.FilePath)
			if err != nil {
				return nil, fmt.Errorf("read key file: %w", err)
			}
		default:
			return nil, fmt.Errorf("unknown key source: %s", a.PrivateKey.Source)
		}

		var signer gossh.Signer
		var err error
		if a.PrivateKey.Passphrase != "" {
			signer, err = gossh.ParsePrivateKeyWithPassphrase(pemBytes, []byte(a.PrivateKey.Passphrase))
		} else {
			signer, err = gossh.ParsePrivateKey(pemBytes)
		}
		if err != nil {
			return nil, fmt.Errorf("parse private key: %w", err)
		}
		return []gossh.AuthMethod{gossh.PublicKeys(signer)}, nil

	case config.AuthNone:
		return nil, nil

	default:
		return nil, fmt.Errorf("unknown auth type: %s", a.Type)
	}
}
