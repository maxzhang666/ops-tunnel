//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func autostartPath() string {
	appData := os.Getenv("APPDATA")
	return filepath.Join(appData, "Microsoft", "Windows", "Start Menu", "Programs", "Startup", "OpsTunnel.lnk")
}

func autostartEnable() error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable: %w", err)
	}

	lnkPath := autostartPath()
	script := fmt.Sprintf(
		`$ws = New-Object -ComObject WScript.Shell; $s = $ws.CreateShortcut('%s'); $s.TargetPath = '%s'; $s.Save()`,
		lnkPath, exe,
	)
	return exec.Command("powershell", "-NoProfile", "-Command", script).Run()
}

func autostartDisable() error {
	err := os.Remove(autostartPath())
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
