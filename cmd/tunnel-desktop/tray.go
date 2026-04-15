package main

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"fyne.io/systray"
	wailsrt "github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/maxzhang666/ops-tunnel/internal/config"
	"github.com/maxzhang666/ops-tunnel/internal/engine"
)

func initTray(app *App, eng engine.Engine, bus engine.EventBus, cfg *config.Config) func() {
	start, end := systray.RunWithExternalLoop(func() {
		onTrayReady(app, eng, bus, cfg)
	}, func() {})
	runOnMainThread(start)
	return end
}

func onTrayReady(app *App, eng engine.Engine, bus engine.EventBus, cfg *config.Config) {
	systray.SetIcon(iconGray) // initial icon before first status check
	systray.SetTitle("OpsTunnel")
	systray.SetTooltip(T("tray.tooltip"))

	buildTrayMenu(app, eng, cfg)

	ch, cancel := bus.Subscribe(64)
	defer cancel()
	for ev := range ch {
		if ev.Type == engine.EventTunnelStateChanged {
			refreshTrayStatus(eng, cfg)
		}
	}
}

// trayTunnelItem holds references to a single tunnel's menu hierarchy.
type trayTunnelItem struct {
	id      string
	menu    *systray.MenuItem
	start   *systray.MenuItem
	stop    *systray.MenuItem
	restart *systray.MenuItem
	copies  []*trayCopyItem
}

type trayCopyItem struct {
	item *systray.MenuItem
	addr string
}

var trayItems []trayTunnelItem

func buildTrayMenu(app *App, eng engine.Engine, cfg *config.Config) {
	statuses := eng.ListStatus()
	running, total := countRunning(statuses), len(statuses)

	applyTrayIcon(computeIconState(statuses))
	systray.SetTooltip(fmt.Sprintf("OpsTunnel — %d/%d Running", running, total))

	mTitle := systray.AddMenuItem(fmt.Sprintf("OpsTunnel — %d/%d Running", running, total), "")
	mTitle.Disable()

	systray.AddSeparator()

	mStartAll := systray.AddMenuItem(T("tray.startAll"), "")
	mStopAll := systray.AddMenuItem(T("tray.stopAll"), "")
	go func() {
		for {
			select {
			case <-mStartAll.ClickedCh:
				for _, t := range cfg.Tunnels {
					go eng.StartTunnel(context.Background(), t.ID)
				}
			case <-mStopAll.ClickedCh:
				for _, t := range cfg.Tunnels {
					go eng.StopTunnel(context.Background(), t.ID)
				}
			}
		}
	}()

	systray.AddSeparator()

	trayItems = make([]trayTunnelItem, 0, len(cfg.Tunnels))
	for _, t := range cfg.Tunnels {
		status, _ := eng.GetStatus(t.ID)
		ti := trayTunnelItem{id: t.ID}

		ti.menu = systray.AddMenuItem(tunnelLabel(t, status), "")
		ti.start = ti.menu.AddSubMenuItem(T("tray.start"), "")
		ti.stop = ti.menu.AddSubMenuItem(T("tray.stop"), "")
		ti.restart = ti.menu.AddSubMenuItem(T("tray.restart"), "")

		// Copy-address items for each mapping
		for _, m := range t.Mappings {
			addr := fmt.Sprintf("%s:%d", m.Listen.Host, m.Listen.Port)
			ci := &trayCopyItem{
				item: ti.menu.AddSubMenuItem(fmt.Sprintf(T("tray.copy"), addr), ""),
				addr: addr,
			}
			ti.copies = append(ti.copies, ci)
			go func() {
				for range ci.item.ClickedCh {
					copyToClipboard(app, ci.addr)
				}
			}()
		}

		applyTunnelVisibility(&ti, status.State)

		tunnelID := t.ID
		go func() {
			for {
				select {
				case <-ti.start.ClickedCh:
					go eng.StartTunnel(context.Background(), tunnelID)
				case <-ti.stop.ClickedCh:
					go eng.StopTunnel(context.Background(), tunnelID)
				case <-ti.restart.ClickedCh:
					go eng.RestartTunnel(context.Background(), tunnelID)
				}
			}
		}()

		trayItems = append(trayItems, ti)
	}

	systray.AddSeparator()

	mShow := systray.AddMenuItem(T("tray.show"), "")
	go func() {
		for range mShow.ClickedCh {
			app.ShowWindow()
		}
	}()

	systray.AddSeparator()

	mQuit := systray.AddMenuItem(T("tray.quit"), "")
	go func() {
		<-mQuit.ClickedCh
		slog.Info("quit requested from tray")
		systray.Quit()
	}()
}

// refreshTrayStatus updates icon, tooltip, and per-tunnel labels on state change.
func refreshTrayStatus(eng engine.Engine, cfg *config.Config) {
	statuses := eng.ListStatus()
	running, total := countRunning(statuses), len(statuses)

	applyTrayIcon(computeIconState(statuses))
	systray.SetTooltip(fmt.Sprintf("OpsTunnel — %d/%d Running", running, total))

	for i, ti := range trayItems {
		if i >= len(cfg.Tunnels) {
			break
		}
		t := cfg.Tunnels[i]
		status, _ := eng.GetStatus(ti.id)
		ti.menu.SetTitle(tunnelLabel(t, status))
		applyTunnelVisibility(&trayItems[i], status.State)
	}
}

func tunnelLabel(t config.Tunnel, status engine.TunnelStatus) string {
	icon := "○"
	switch status.State {
	case engine.StateRunning:
		icon = "●"
	case engine.StateError, engine.StateDegraded:
		icon = "✕"
	}
	return fmt.Sprintf("%s %s (%s)", icon, t.Name, t.Mode)
}

func applyTunnelVisibility(ti *trayTunnelItem, state engine.TunnelState) {
	if state == engine.StateRunning || state == engine.StateDegraded {
		ti.start.Hide()
		ti.stop.Show()
		ti.restart.Show()
		for _, ci := range ti.copies {
			ci.item.Show()
		}
	} else {
		ti.start.Show()
		ti.stop.Hide()
		ti.restart.Hide()
		for _, ci := range ti.copies {
			ci.item.Hide()
		}
	}
}

func copyToClipboard(app *App, text string) {
	if app.ctx != nil {
		wailsrt.ClipboardSetText(app.ctx, text)
	}
}

// trayIconState represents the desired tray icon appearance.
type trayIconState int

const (
	trayStateGray   trayIconState = iota // all stopped
	trayStateYellow                      // running normally
	trayStateBlink                       // error / degraded / retrying
)

func computeIconState(statuses []engine.TunnelStatus) trayIconState {
	hasRunning, hasError := false, false
	for _, s := range statuses {
		switch s.State {
		case engine.StateRunning:
			hasRunning = true
		case engine.StateError, engine.StateDegraded:
			hasError = true
		}
	}
	if hasError {
		return trayStateBlink
	}
	if hasRunning {
		return trayStateYellow
	}
	return trayStateGray
}

// blinker manages the tray icon blink animation.
var blinker struct {
	mu      sync.Mutex
	active  bool
	stopCh  chan struct{}
}

func applyTrayIcon(state trayIconState) {
	blinker.mu.Lock()
	defer blinker.mu.Unlock()

	// Stop any existing blink.
	if blinker.active {
		close(blinker.stopCh)
		blinker.active = false
	}

	switch state {
	case trayStateBlink:
		blinker.stopCh = make(chan struct{})
		blinker.active = true
		go blinkLoop(blinker.stopCh)
	case trayStateYellow:
		systray.SetIcon(iconYellow)
	default:
		systray.SetIcon(iconGray)
	}
}

func blinkLoop(stop <-chan struct{}) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	visible := true
	systray.SetIcon(iconYellow)
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			if visible {
				systray.SetIcon(iconEmpty)
			} else {
				systray.SetIcon(iconYellow)
			}
			visible = !visible
		}
	}
}

func countRunning(statuses []engine.TunnelStatus) int {
	n := 0
	for _, s := range statuses {
		if s.State == engine.StateRunning {
			n++
		}
	}
	return n
}
