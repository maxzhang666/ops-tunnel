package api

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/maxzhang666/ops-tunnel/internal/config"
	"github.com/rs/xid"
)

func (s *Server) listTunnels(w http.ResponseWriter, r *http.Request) {
	statusFilter := r.URL.Query().Get("status")

	s.mu.RLock()
	var tunnels []config.Tunnel
	for _, t := range s.data.Tunnels {
		if statusFilter != "" {
			st, ok := s.eng.GetStatus(t.ID)
			if !ok || string(st.State) != statusFilter {
				continue
			}
		}
		tunnels = append(tunnels, config.RedactTunnel(t))
	}
	s.mu.RUnlock()

	if tunnels == nil {
		tunnels = []config.Tunnel{}
	}
	writeJSON(w, http.StatusOK, tunnels)
}

func (s *Server) createTunnel(w http.ResponseWriter, r *http.Request) {
	var tun config.Tunnel
	if err := decodeBody(r, &tun); err != nil {
		writeBodyError(w, err)
		return
	}

	tun.ID = xid.New().String()
	for i := range tun.Mappings {
		if tun.Mappings[i].ID == "" {
			tun.Mappings[i].ID = xid.New().String()
		}
	}
	config.ApplyTunnelDefaults(&tun)

	s.mu.Lock()
	defer s.mu.Unlock()

	s.data.Tunnels = append(s.data.Tunnels, tun)

	vr, err := s.saveConfig(r.Context())
	if err != nil {
		s.data.Tunnels = s.data.Tunnels[:len(s.data.Tunnels)-1]
		slog.Error("failed to save config", "err", err)
		writeInternalError(w)
		return
	}
	if vr.HasErrors() {
		s.data.Tunnels = s.data.Tunnels[:len(s.data.Tunnels)-1]
		writeValidationError(w, vr.Errors)
		return
	}

	writeData(w, http.StatusCreated, config.RedactTunnel(tun), vr.Warnings)
}

func (s *Server) getTunnel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, t := range s.data.Tunnels {
		if t.ID == id {
			writeJSON(w, http.StatusOK, config.RedactTunnel(t))
			return
		}
	}
	writeNotFound(w, "tunnel", id)
}

func (s *Server) updateTunnel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var tun config.Tunnel
	if err := decodeBody(r, &tun); err != nil {
		writeBodyError(w, err)
		return
	}
	tun.ID = id
	for i := range tun.Mappings {
		if tun.Mappings[i].ID == "" {
			tun.Mappings[i].ID = xid.New().String()
		}
	}
	config.ApplyTunnelDefaults(&tun)

	s.mu.Lock()
	defer s.mu.Unlock()

	idx := -1
	for i, t := range s.data.Tunnels {
		if t.ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		writeNotFound(w, "tunnel", id)
		return
	}

	old := s.data.Tunnels[idx]
	s.data.Tunnels[idx] = tun

	vr, err := s.saveConfig(r.Context())
	if err != nil {
		s.data.Tunnels[idx] = old
		slog.Error("failed to save config", "err", err)
		writeInternalError(w)
		return
	}
	if vr.HasErrors() {
		s.data.Tunnels[idx] = old
		writeValidationError(w, vr.Errors)
		return
	}

	writeData(w, http.StatusOK, config.RedactTunnel(tun), vr.Warnings)
}

func (s *Server) deleteTunnel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	s.mu.Lock()
	defer s.mu.Unlock()

	idx := -1
	for i, t := range s.data.Tunnels {
		if t.ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		writeNotFound(w, "tunnel", id)
		return
	}

	s.data.Tunnels = append(s.data.Tunnels[:idx], s.data.Tunnels[idx+1:]...)

	if _, err := s.saveConfig(r.Context()); err != nil {
		slog.Error("failed to save config", "err", err)
		writeInternalError(w)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type tunnelPatch struct {
	Name     *string            `json:"name,omitempty"`
	Mode     *config.TunnelMode `json:"mode,omitempty"`
	Chain    []string           `json:"chain,omitempty"`
	Mappings []config.Mapping   `json:"mappings,omitempty"`
	Policy   *config.Policy     `json:"policy,omitempty"`
}

func (s *Server) patchTunnel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var patch tunnelPatch
	if err := decodeBody(r, &patch); err != nil {
		writeBodyError(w, err)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	idx := -1
	for i, t := range s.data.Tunnels {
		if t.ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		writeNotFound(w, "tunnel", id)
		return
	}

	old := s.data.Tunnels[idx]
	tun := old

	if patch.Name != nil {
		tun.Name = *patch.Name
	}
	if patch.Mode != nil {
		tun.Mode = *patch.Mode
	}
	if patch.Chain != nil {
		tun.Chain = patch.Chain
	}
	if patch.Mappings != nil {
		tun.Mappings = patch.Mappings
		for i := range tun.Mappings {
			if tun.Mappings[i].ID == "" {
				tun.Mappings[i].ID = xid.New().String()
			}
		}
	}
	if patch.Policy != nil {
		tun.Policy = *patch.Policy
	}

	config.ApplyTunnelDefaults(&tun)
	s.data.Tunnels[idx] = tun

	vr, err := s.saveConfig(r.Context())
	if err != nil {
		s.data.Tunnels[idx] = old
		slog.Error("failed to save config", "err", err)
		writeInternalError(w)
		return
	}
	if vr.HasErrors() {
		s.data.Tunnels[idx] = old
		writeValidationError(w, vr.Errors)
		return
	}

	writeData(w, http.StatusOK, config.RedactTunnel(tun), vr.Warnings)
}
