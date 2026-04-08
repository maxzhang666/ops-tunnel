package main

import (
	"log/slog"

	"github.com/maxzhang666/ops-tunnel/internal/config"
	"github.com/maxzhang666/ops-tunnel/internal/engine"
)

// syncAutoStart syncs OS auto-start state with config on boot,
// then listens for settings changes to keep it in sync at runtime.
func syncAutoStart(cfg *config.Config, bus engine.EventBus) {
	applyAutoStart(cfg.General.AutoStart)

	ch, cancel := bus.Subscribe(16)
	go func() {
		defer cancel()
		for e := range ch {
			if e.Type != engine.EventSettingsChanged {
				continue
			}
			applyAutoStart(cfg.General.AutoStart)
		}
	}()
}

func applyAutoStart(enabled bool) {
	var err error
	if enabled {
		err = autostartEnable()
	} else {
		err = autostartDisable()
	}
	if err != nil {
		slog.Warn("autostart sync failed", "enabled", enabled, "err", err)
	}
}
