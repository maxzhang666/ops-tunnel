//go:build linux

package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func autostartPath() string {
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		home, _ := os.UserHomeDir()
		configDir = filepath.Join(home, ".config")
	}
	return filepath.Join(configDir, "autostart", "opstunnel.desktop")
}

func autostartEnable() error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable: %w", err)
	}

	entry := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=OpsTunnel
Exec=%s
X-GNOME-Autostart-enabled=true
`, exe)

	dir := filepath.Dir(autostartPath())
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(autostartPath(), []byte(entry), 0o644)
}

func autostartDisable() error {
	err := os.Remove(autostartPath())
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
