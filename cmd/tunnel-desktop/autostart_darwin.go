//go:build darwin

package main

import (
	"fmt"
	"os"
	"path/filepath"
)

const launchAgentLabel = "com.opstunnel.app"

func autostartPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents", launchAgentLabel+".plist")
}

func autostartEnable() error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable: %w", err)
	}

	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>%s</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
</dict>
</plist>
`, launchAgentLabel, exe)

	dir := filepath.Dir(autostartPath())
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(autostartPath(), []byte(plist), 0o644)
}

func autostartDisable() error {
	err := os.Remove(autostartPath())
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
