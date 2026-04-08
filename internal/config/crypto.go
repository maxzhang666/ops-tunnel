package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
)

const encPrefix = "ENC:"

// Encryptor provides symmetric encryption for credential fields.
type Encryptor interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(ciphertext string) (string, error)
}

// NopEncryptor is a passthrough encryptor for graceful degradation.
type NopEncryptor struct{}

func (NopEncryptor) Encrypt(s string) (string, error) { return s, nil }
func (NopEncryptor) Decrypt(s string) (string, error) { return s, nil }

type aesEncryptor struct {
	gcm cipher.AEAD
}

// NewAESEncryptor creates an AES-256-GCM encryptor from a 32-byte key.
func NewAESEncryptor(key []byte) (Encryptor, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("key must be 32 bytes, got %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &aesEncryptor{gcm: gcm}, nil
}

func (e *aesEncryptor) Encrypt(plaintext string) (string, error) {
	nonce := make([]byte, e.gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	sealed := e.gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return encPrefix + base64.StdEncoding.EncodeToString(sealed), nil
}

func (e *aesEncryptor) Decrypt(ciphertext string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(ciphertext, encPrefix))
	if err != nil {
		return "", fmt.Errorf("base64 decode: %w", err)
	}
	ns := e.gcm.NonceSize()
	if len(raw) < ns {
		return "", fmt.Errorf("ciphertext too short")
	}
	plaintext, err := e.gcm.Open(nil, raw[:ns], raw[ns:], nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// IsEncrypted reports whether s carries the encryption prefix.
func IsEncrypted(s string) bool {
	return strings.HasPrefix(s, encPrefix)
}

// encryptConfig returns a deep copy of cfg with all sensitive fields encrypted.
// The original cfg is not modified.
func encryptConfig(cfg *Config, enc Encryptor) (*Config, error) {
	out := *cfg

	conns := make([]SSHConnection, len(cfg.SSHConnections))
	copy(conns, cfg.SSHConnections)
	for i := range conns {
		if err := encryptAuth(&conns[i].Auth, enc); err != nil {
			return nil, fmt.Errorf("ssh connection %s: %w", conns[i].ID, err)
		}
	}
	out.SSHConnections = conns

	tunnels := make([]Tunnel, len(cfg.Tunnels))
	copy(tunnels, cfg.Tunnels)
	for i := range tunnels {
		mappings := make([]Mapping, len(tunnels[i].Mappings))
		copy(mappings, tunnels[i].Mappings)
		for j := range mappings {
			if mappings[j].Socks5 != nil {
				s5 := *mappings[j].Socks5
				if err := encryptField(&s5.Password, enc); err != nil {
					return nil, fmt.Errorf("tunnel %s mapping %s socks5: %w", tunnels[i].ID, mappings[j].ID, err)
				}
				mappings[j].Socks5 = &s5
			}
		}
		tunnels[i].Mappings = mappings
	}
	out.Tunnels = tunnels

	return &out, nil
}

// decryptConfig decrypts all sensitive fields in cfg in-place.
// Non-encrypted (plaintext) fields pass through unchanged for migration.
func decryptConfig(cfg *Config, enc Encryptor) error {
	for i := range cfg.SSHConnections {
		if err := decryptAuth(&cfg.SSHConnections[i].Auth, enc); err != nil {
			return fmt.Errorf("ssh connection %s: %w", cfg.SSHConnections[i].ID, err)
		}
	}
	for i := range cfg.Tunnels {
		for j := range cfg.Tunnels[i].Mappings {
			m := &cfg.Tunnels[i].Mappings[j]
			if m.Socks5 != nil && IsEncrypted(m.Socks5.Password) {
				p, err := enc.Decrypt(m.Socks5.Password)
				if err != nil {
					return fmt.Errorf("tunnel %s mapping %s socks5: %w", cfg.Tunnels[i].ID, m.ID, err)
				}
				m.Socks5.Password = p
			}
		}
	}
	return nil
}

func encryptAuth(a *Auth, enc Encryptor) error {
	if err := encryptField(&a.Password, enc); err != nil {
		return fmt.Errorf("password: %w", err)
	}
	if a.PrivateKey != nil {
		pk := *a.PrivateKey
		if err := encryptField(&pk.KeyPEM, enc); err != nil {
			return fmt.Errorf("keyPem: %w", err)
		}
		if err := encryptField(&pk.Passphrase, enc); err != nil {
			return fmt.Errorf("passphrase: %w", err)
		}
		a.PrivateKey = &pk
	}
	return nil
}

func decryptAuth(a *Auth, enc Encryptor) error {
	if err := decryptField(&a.Password, enc); err != nil {
		return fmt.Errorf("password: %w", err)
	}
	if a.PrivateKey != nil {
		if err := decryptField(&a.PrivateKey.KeyPEM, enc); err != nil {
			return fmt.Errorf("keyPem: %w", err)
		}
		if err := decryptField(&a.PrivateKey.Passphrase, enc); err != nil {
			return fmt.Errorf("passphrase: %w", err)
		}
	}
	return nil
}

func encryptField(field *string, enc Encryptor) error {
	if *field == "" || IsEncrypted(*field) {
		return nil
	}
	v, err := enc.Encrypt(*field)
	if err != nil {
		return err
	}
	*field = v
	return nil
}

func decryptField(field *string, enc Encryptor) error {
	if !IsEncrypted(*field) {
		return nil
	}
	v, err := enc.Decrypt(*field)
	if err != nil {
		return err
	}
	*field = v
	return nil
}
