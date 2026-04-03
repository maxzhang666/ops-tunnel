package main

import (
	"context"
	"fmt"

	"github.com/maxzhang666/ops-tunnel/internal/config"
	"github.com/maxzhang666/ops-tunnel/internal/engine"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App holds the Wails application state.
type App struct {
	ctx    context.Context
	config *config.Config
	store  config.Store
	eng    engine.Engine
}

func NewApp(cfg *config.Config, store config.Store, eng engine.Engine) *App {
	return &App{
		config: cfg,
		store:  store,
		eng:    eng,
	}
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) BeforeClose(ctx context.Context) bool {
	switch a.config.Desktop.CloseAction {
	case "minimize":
		runtime.WindowHide(a.ctx)
		return true
	case "quit":
		return a.confirmQuitIfRunning()
	default: // "ask"
		result, _ := runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
			Type:          runtime.QuestionDialog,
			Title:         "Close OpsTunnel",
			Message:       "What would you like to do?",
			Buttons:       []string{"Minimize to Tray", "Quit", "Cancel"},
			DefaultButton: "Minimize to Tray",
		})
		switch result {
		case "Minimize to Tray":
			runtime.WindowHide(a.ctx)
			return true
		case "Quit":
			return a.confirmQuitIfRunning()
		default:
			return true
		}
	}
}

func (a *App) confirmQuitIfRunning() bool {
	running := 0
	for _, s := range a.eng.ListStatus() {
		if s.State == engine.StateRunning || s.State == engine.StateDegraded {
			running++
		}
	}
	if running == 0 {
		return false
	}
	result, _ := runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:          runtime.WarningDialog,
		Title:         "Tunnels Running",
		Message:       fmt.Sprintf("%d tunnel(s) are still running. Quit anyway?", running),
		Buttons:       []string{"Quit", "Cancel"},
		DefaultButton: "Cancel",
	})
	return result != "Quit"
}

func (a *App) ShowWindow() {
	runtime.WindowShow(a.ctx)
	runtime.WindowSetAlwaysOnTop(a.ctx, true)
	runtime.WindowSetAlwaysOnTop(a.ctx, false)
}
