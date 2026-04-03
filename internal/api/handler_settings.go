package api

import (
	"log/slog"
	"net/http"

	"github.com/maxzhang666/ops-tunnel/internal/config"
	"github.com/maxzhang666/ops-tunnel/internal/engine"
)

type settingsResponse struct {
	General    config.GeneralConfig    `json:"general"`
	Appearance config.AppearanceConfig `json:"appearance"`
	Desktop    config.DesktopConfig    `json:"desktop"`
}

func (s *Server) getSettings(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	resp := settingsResponse{
		General:    s.data.General,
		Appearance: s.data.Appearance,
		Desktop:    s.data.Desktop,
	}
	s.mu.RUnlock()
	writeJSON(w, http.StatusOK, resp)
}

type settingsPatch struct {
	General    *generalPatch    `json:"general,omitempty"`
	Appearance *appearancePatch `json:"appearance,omitempty"`
	Desktop    *desktopPatch    `json:"desktop,omitempty"`
}

type generalPatch struct {
	LogLevel  *string `json:"logLevel,omitempty"`
	Language  *string `json:"language,omitempty"`
	AutoStart *bool   `json:"autoStart,omitempty"`
}

type appearancePatch struct {
	Theme *string `json:"theme,omitempty"`
}

type desktopPatch struct {
	CloseAction *string `json:"closeAction,omitempty"`
}

var validLogLevels = map[string]slog.Level{
	"debug": slog.LevelDebug,
	"info":  slog.LevelInfo,
	"warn":  slog.LevelWarn,
	"error": slog.LevelError,
}

func (s *Server) patchSettings(w http.ResponseWriter, r *http.Request) {
	var patch settingsPatch
	if err := decodeBody(r, &patch); err != nil {
		writeBodyError(w, err)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	old := settingsResponse{
		General:    s.data.General,
		Appearance: s.data.Appearance,
		Desktop:    s.data.Desktop,
	}

	if patch.General != nil {
		if patch.General.LogLevel != nil {
			if _, ok := validLogLevels[*patch.General.LogLevel]; !ok {
				writeValidationError(w, []config.ValidationError{{Field: "general.logLevel", Message: "must be debug, info, warn, or error"}})
				return
			}
			s.data.General.LogLevel = *patch.General.LogLevel
		}
		if patch.General.Language != nil {
			s.data.General.Language = *patch.General.Language
		}
		if patch.General.AutoStart != nil {
			s.data.General.AutoStart = *patch.General.AutoStart
		}
	}

	if patch.Appearance != nil {
		if patch.Appearance.Theme != nil {
			switch *patch.Appearance.Theme {
			case "light", "dark", "system":
				s.data.Appearance.Theme = *patch.Appearance.Theme
			default:
				writeValidationError(w, []config.ValidationError{{Field: "appearance.theme", Message: "must be light, dark, or system"}})
				return
			}
		}
	}

	if patch.Desktop != nil {
		if patch.Desktop.CloseAction != nil {
			switch *patch.Desktop.CloseAction {
			case "minimize", "quit", "ask":
				s.data.Desktop.CloseAction = *patch.Desktop.CloseAction
			default:
				writeValidationError(w, []config.ValidationError{{Field: "desktop.closeAction", Message: "must be minimize, quit, or ask"}})
				return
			}
		}
	}

	vr, err := s.saveConfig(r.Context())
	if err != nil {
		s.data.General = old.General
		s.data.Appearance = old.Appearance
		s.data.Desktop = old.Desktop
		slog.Error("failed to save config", "err", err)
		writeInternalError(w)
		return
	}
	if vr.HasErrors() {
		s.data.General = old.General
		s.data.Appearance = old.Appearance
		s.data.Desktop = old.Desktop
		writeValidationError(w, vr.Errors)
		return
	}

	if patch.General != nil && patch.General.LogLevel != nil {
		if s.cfg.LogLevelVar != nil {
			s.cfg.LogLevelVar.Set(validLogLevels[*patch.General.LogLevel])
		}
	}

	s.bus.Publish(engine.Event{
		Type:    engine.EventSettingsChanged,
		Level:   "info",
		Message: "settings updated",
	})

	writeJSON(w, http.StatusOK, settingsResponse{
		General:    s.data.General,
		Appearance: s.data.Appearance,
		Desktop:    s.data.Desktop,
	})
}
