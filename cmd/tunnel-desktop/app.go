package main

import (
	"context"

	"github.com/maxzhang666/ops-tunnel/internal/config"
	"github.com/maxzhang666/ops-tunnel/internal/engine"
	wailsrt "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App holds the Wails application state.
type App struct {
	ctx      context.Context
	config   *config.Config
	store    config.Store
	eng      engine.Engine
	bus      engine.EventBus
	trayEnd  func()
	quitting bool
}

func NewApp(cfg *config.Config, store config.Store, eng engine.Engine, bus engine.EventBus) *App {
	return &App{
		config: cfg,
		store:  store,
		eng:    eng,
		bus:    bus,
	}
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	a.trayEnd = initTray(a, a.eng, a.bus, a.config)
}

// BeforeClose intercepts window close and delegates to the frontend dialog.
func (a *App) BeforeClose(ctx context.Context) bool {
	if a.quitting {
		return false
	}
	if a.config.Desktop.CloseAction == "minimize" {
		wailsrt.WindowHide(a.ctx)
		return true
	}
	// Emit event for frontend to show the styled close dialog
	running := 0
	for _, s := range a.eng.ListStatus() {
		if s.State == engine.StateRunning || s.State == engine.StateDegraded {
			running++
		}
	}
	wailsrt.EventsEmit(a.ctx, "app:close-requested", map[string]any{
		"action":  a.config.Desktop.CloseAction,
		"running": running,
	})
	return true // always prevent default close, frontend decides
}

// DoMinimize hides the window (called from frontend).
func (a *App) DoMinimize() {
	wailsrt.WindowHide(a.ctx)
}

// DoQuit exits the application (called from frontend).
func (a *App) DoQuit() {
	a.quitting = true
	wailsrt.Quit(a.ctx)
}

func (a *App) ShowWindow() {
	wailsrt.WindowShow(a.ctx)
	wailsrt.WindowSetAlwaysOnTop(a.ctx, true)
	wailsrt.WindowSetAlwaysOnTop(a.ctx, false)
}
