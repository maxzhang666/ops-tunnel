package config

import (
	"strings"
	"testing"
)

func testKey() []byte {
	k := make([]byte, 32)
	for i := range k {
		k[i] = byte(i)
	}
	return k
}

func TestAESEncryptor_RoundTrip(t *testing.T) {
	enc, err := NewAESEncryptor(testKey())
	if err != nil {
		t.Fatal(err)
	}

	for _, plain := range []string{"hello", "", "a longer test string with unicode: 你好"} {
		ct, err := enc.Encrypt(plain)
		if err != nil {
			t.Fatalf("Encrypt(%q): %v", plain, err)
		}
		got, err := enc.Decrypt(ct)
		if err != nil {
			t.Fatalf("Decrypt: %v", err)
		}
		if got != plain {
			t.Errorf("round-trip failed: got %q, want %q", got, plain)
		}
	}
}

func TestAESEncryptor_HasPrefix(t *testing.T) {
	enc, _ := NewAESEncryptor(testKey())
	ct, _ := enc.Encrypt("secret")
	if !strings.HasPrefix(ct, encPrefix) {
		t.Errorf("ciphertext %q missing prefix %q", ct, encPrefix)
	}
}

func TestAESEncryptor_UniqueNonces(t *testing.T) {
	enc, _ := NewAESEncryptor(testKey())
	a, _ := enc.Encrypt("same")
	b, _ := enc.Encrypt("same")
	if a == b {
		t.Error("encrypting same plaintext twice produced identical ciphertext")
	}
}

func TestAESEncryptor_BadKeyLength(t *testing.T) {
	for _, n := range []int{0, 16, 31, 33, 64} {
		_, err := NewAESEncryptor(make([]byte, n))
		if err == nil {
			t.Errorf("NewAESEncryptor(len=%d) should fail", n)
		}
	}
}

func TestAESEncryptor_TamperedCiphertext(t *testing.T) {
	enc, _ := NewAESEncryptor(testKey())
	ct, _ := enc.Encrypt("secret")

	// Flip a character in the base64 body
	tampered := ct[:len(ct)-2] + "XX"
	_, err := enc.Decrypt(tampered)
	if err == nil {
		t.Error("Decrypt of tampered ciphertext should fail")
	}
}

func TestIsEncrypted(t *testing.T) {
	if !IsEncrypted("ENC:abc") {
		t.Error("should detect ENC: prefix")
	}
	if IsEncrypted("plaintext") {
		t.Error("should not detect plaintext as encrypted")
	}
	if IsEncrypted("") {
		t.Error("should not detect empty as encrypted")
	}
}

func TestNopEncryptor(t *testing.T) {
	var nop NopEncryptor
	ct, _ := nop.Encrypt("hello")
	if ct != "hello" {
		t.Errorf("NopEncryptor.Encrypt changed value: %q", ct)
	}
	pt, _ := nop.Decrypt("hello")
	if pt != "hello" {
		t.Errorf("NopEncryptor.Decrypt changed value: %q", pt)
	}
}

func TestEncryptDecryptConfig(t *testing.T) {
	enc, _ := NewAESEncryptor(testKey())

	cfg := &Config{
		SSHConnections: []SSHConnection{
			{
				ID:   "ssh-1",
				Auth: Auth{Type: AuthPassword, Password: "mypass"},
			},
			{
				ID: "ssh-2",
				Auth: Auth{
					Type: AuthPrivateKey,
					PrivateKey: &PrivateKey{
						KeyPEM:     "PEM-DATA",
						Passphrase: "keypass",
					},
				},
			},
		},
		Tunnels: []Tunnel{
			{
				ID: "tun-1",
				Mappings: []Mapping{
					{
						ID:    "m-1",
						Socks5: &Socks5Cfg{Password: "s5pass"},
					},
				},
			},
		},
	}

	encrypted, err := encryptConfig(cfg, enc)
	if err != nil {
		t.Fatal(err)
	}

	// Verify all sensitive fields are encrypted
	if !IsEncrypted(encrypted.SSHConnections[0].Auth.Password) {
		t.Error("password should be encrypted")
	}
	if !IsEncrypted(encrypted.SSHConnections[1].Auth.PrivateKey.KeyPEM) {
		t.Error("keyPem should be encrypted")
	}
	if !IsEncrypted(encrypted.SSHConnections[1].Auth.PrivateKey.Passphrase) {
		t.Error("passphrase should be encrypted")
	}
	if !IsEncrypted(encrypted.Tunnels[0].Mappings[0].Socks5.Password) {
		t.Error("socks5 password should be encrypted")
	}

	// Verify original is NOT mutated
	if IsEncrypted(cfg.SSHConnections[0].Auth.Password) {
		t.Error("original config should not be mutated")
	}

	// Decrypt and verify round-trip
	if err := decryptConfig(encrypted, enc); err != nil {
		t.Fatal(err)
	}
	if encrypted.SSHConnections[0].Auth.Password != "mypass" {
		t.Errorf("password = %q, want mypass", encrypted.SSHConnections[0].Auth.Password)
	}
	if encrypted.SSHConnections[1].Auth.PrivateKey.KeyPEM != "PEM-DATA" {
		t.Error("keyPem round-trip failed")
	}
	if encrypted.SSHConnections[1].Auth.PrivateKey.Passphrase != "keypass" {
		t.Error("passphrase round-trip failed")
	}
	if encrypted.Tunnels[0].Mappings[0].Socks5.Password != "s5pass" {
		t.Error("socks5 password round-trip failed")
	}
}

func TestEncryptConfig_SkipsEmpty(t *testing.T) {
	enc, _ := NewAESEncryptor(testKey())

	cfg := &Config{
		SSHConnections: []SSHConnection{
			{ID: "ssh-1", Auth: Auth{Type: AuthPassword, Password: ""}},
		},
	}

	encrypted, err := encryptConfig(cfg, enc)
	if err != nil {
		t.Fatal(err)
	}
	if encrypted.SSHConnections[0].Auth.Password != "" {
		t.Error("empty password should remain empty after encryption")
	}
}
