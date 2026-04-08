package config

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/crypto/hkdf"
)

const (
	keyfileSize = 32
	keyfileName = ".keyfile"
	hkdfSalt    = "ops-tunnel-credential-encryption-v1"
	hkdfInfo    = "aes-256-gcm"
)

// InitEncryptor creates the appropriate Encryptor for the given runtime mode.
// Desktop mode derives the key from the machine ID; server/Docker mode uses a keyfile.
func InitEncryptor(mode, dataDir string) (Encryptor, error) {
	var key []byte
	var err error
	if mode == "desktop" {
		key, err = DeriveDesktopKey(dataDir)
	} else {
		key, err = LoadOrCreateKeyfile(dataDir)
	}
	if err != nil {
		return nil, err
	}
	return NewAESEncryptor(key)
}

// LoadOrCreateKeyfile reads or generates a random 32-byte keyfile in dataDir.
func LoadOrCreateKeyfile(dataDir string) ([]byte, error) {
	p := filepath.Join(dataDir, keyfileName)

	data, err := os.ReadFile(p)
	if err == nil {
		if len(data) != keyfileSize {
			return nil, fmt.Errorf("keyfile %s has invalid size %d (expected %d)", p, len(data), keyfileSize)
		}
		return data, nil
	}
	if !os.IsNotExist(err) {
		return nil, err
	}

	key := make([]byte, keyfileSize)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("generate key: %w", err)
	}
	if err := os.WriteFile(p, key, 0o600); err != nil {
		return nil, fmt.Errorf("write keyfile: %w", err)
	}
	slog.Info("encryption keyfile created", "path", p)
	return key, nil
}

// DeriveDesktopKey derives an encryption key from the machine ID via HKDF-SHA256.
// Falls back to a keyfile if machine ID is unavailable.
func DeriveDesktopKey(dataDir string) ([]byte, error) {
	mid, err := readMachineID()
	if err != nil {
		slog.Warn("machine-id unavailable, falling back to keyfile", "err", err)
		return LoadOrCreateKeyfile(dataDir)
	}

	r := hkdf.New(sha256.New, []byte(mid), []byte(hkdfSalt), []byte(hkdfInfo))
	key := make([]byte, keyfileSize)
	if _, err := io.ReadFull(r, key); err != nil {
		return nil, fmt.Errorf("hkdf derive: %w", err)
	}
	return key, nil
}

// readMachineID returns a stable machine identifier across platforms.
func readMachineID() (string, error) {
	// Linux
	for _, p := range []string{"/etc/machine-id", "/var/lib/dbus/machine-id"} {
		if data, err := os.ReadFile(p); err == nil {
			if id := strings.TrimSpace(string(data)); id != "" {
				return id, nil
			}
		}
	}

	// macOS: IOPlatformUUID
	if out, err := exec.Command("ioreg", "-rd1", "-c", "IOPlatformExpertDevice").Output(); err == nil {
		re := regexp.MustCompile(`"IOPlatformUUID"\s*=\s*"([^"]+)"`)
		if m := re.FindSubmatch(out); len(m) == 2 {
			return string(m[1]), nil
		}
	}

	// Windows: MachineGuid
	if out, err := exec.Command("reg", "query",
		`HKLM\SOFTWARE\Microsoft\Cryptography`, "/v", "MachineGuid").Output(); err == nil {
		re := regexp.MustCompile(`MachineGuid\s+REG_SZ\s+(\S+)`)
		if m := re.FindSubmatch(out); len(m) == 2 {
			return string(m[1]), nil
		}
	}

	return "", fmt.Errorf("no machine-id source found")
}
