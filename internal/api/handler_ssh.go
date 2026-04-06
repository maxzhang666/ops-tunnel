package api

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/maxzhang666/ops-tunnel/internal/config"
	tunnelssh "github.com/maxzhang666/ops-tunnel/internal/ssh"
	"github.com/rs/xid"
)

func (s *Server) listSSHConnections(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	conns := make([]config.SSHConnection, len(s.data.SSHConnections))
	for i, c := range s.data.SSHConnections {
		conns[i] = config.RedactSSHConnection(c)
	}
	s.mu.RUnlock()

	writeJSON(w, http.StatusOK, conns)
}

func (s *Server) createSSHConnection(w http.ResponseWriter, r *http.Request) {
	var conn config.SSHConnection
	if err := decodeBody(r, &conn); err != nil {
		writeBodyError(w, err)
		return
	}

	conn.ID = xid.New().String()
	config.ApplySSHConnectionDefaults(&conn)

	s.mu.Lock()
	defer s.mu.Unlock()

	s.data.SSHConnections = append(s.data.SSHConnections, conn)

	vr, err := s.saveConfig(r.Context())
	if err != nil {
		s.data.SSHConnections = s.data.SSHConnections[:len(s.data.SSHConnections)-1]
		slog.Error("failed to save config", "err", err)
		writeInternalError(w)
		return
	}
	if vr.HasErrors() {
		s.data.SSHConnections = s.data.SSHConnections[:len(s.data.SSHConnections)-1]
		writeValidationError(w, vr.Errors)
		return
	}

	writeData(w, http.StatusCreated, config.RedactSSHConnection(conn), vr.Warnings)
}

func (s *Server) getSSHConnection(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, c := range s.data.SSHConnections {
		if c.ID == id {
			writeJSON(w, http.StatusOK, config.RedactSSHConnection(c))
			return
		}
	}
	writeNotFound(w, "ssh-connection", id)
}

func (s *Server) updateSSHConnection(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var conn config.SSHConnection
	if err := decodeBody(r, &conn); err != nil {
		writeBodyError(w, err)
		return
	}
	conn.ID = id
	config.ApplySSHConnectionDefaults(&conn)

	s.mu.Lock()
	defer s.mu.Unlock()

	idx := -1
	for i, c := range s.data.SSHConnections {
		if c.ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		writeNotFound(w, "ssh-connection", id)
		return
	}

	old := s.data.SSHConnections[idx]
	s.data.SSHConnections[idx] = conn

	vr, err := s.saveConfig(r.Context())
	if err != nil {
		s.data.SSHConnections[idx] = old
		slog.Error("failed to save config", "err", err)
		writeInternalError(w)
		return
	}
	if vr.HasErrors() {
		s.data.SSHConnections[idx] = old
		writeValidationError(w, vr.Errors)
		return
	}

	writeData(w, http.StatusOK, config.RedactSSHConnection(conn), vr.Warnings)
}

func (s *Server) deleteSSHConnection(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	s.mu.Lock()
	defer s.mu.Unlock()

	idx := -1
	for i, c := range s.data.SSHConnections {
		if c.ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		writeNotFound(w, "ssh-connection", id)
		return
	}

	refs := config.FindSSHConnectionReferences(s.data, id)
	if len(refs) > 0 {
		details := make([]config.ValidationError, len(refs))
		for i, name := range refs {
			details[i] = config.ValidationError{Field: "tunnel", Message: "referenced by tunnel '" + name + "'"}
		}
		writeConflict(w, details)
		return
	}

	s.data.SSHConnections = append(s.data.SSHConnections[:idx], s.data.SSHConnections[idx+1:]...)

	if _, err := s.saveConfig(r.Context()); err != nil {
		slog.Error("failed to save config", "err", err)
		writeInternalError(w)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) testSSHConnection(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	s.mu.RLock()
	var conn *config.SSHConnection
	for i, c := range s.data.SSHConnections {
		if c.ID == id {
			conn = &s.data.SSHConnections[i]
			break
		}
	}
	s.mu.RUnlock()

	if conn == nil {
		writeNotFound(w, "ssh-connection", id)
		return
	}

	s.writeTestResult(w, r, *conn)
}

func (s *Server) testSSHConnectionDirect(w http.ResponseWriter, r *http.Request) {
	var conn config.SSHConnection
	if err := decodeBody(r, &conn); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	s.writeTestResult(w, r, conn)
}

func (s *Server) writeTestResult(w http.ResponseWriter, r *http.Request, conn config.SSHConnection) {
	result := tunnelssh.TestConnection(r.Context(), conn, s.hostKeys)
	if result.OK {
		writeJSON(w, http.StatusOK, map[string]any{
			"status":    "ok",
			"message":   "connected successfully",
			"latencyMs": result.LatencyMs,
		})
	} else {
		writeJSON(w, http.StatusOK, map[string]any{
			"status":  "error",
			"message": result.Error,
		})
	}
}

type sshConnectionPatch struct {
	Name                *string                     `json:"name,omitempty"`
	Endpoint            *config.Endpoint            `json:"endpoint,omitempty"`
	Auth                *config.Auth                `json:"auth,omitempty"`
	HostKeyVerification *config.HostKeyVerification `json:"hostKeyVerification,omitempty"`
	DialTimeoutMs       *int                        `json:"dialTimeoutMs,omitempty"`
	KeepAlive           *config.KeepAlive           `json:"keepAlive,omitempty"`
}

func (s *Server) patchSSHConnection(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var patch sshConnectionPatch
	if err := decodeBody(r, &patch); err != nil {
		writeBodyError(w, err)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	idx := -1
	for i, c := range s.data.SSHConnections {
		if c.ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		writeNotFound(w, "ssh-connection", id)
		return
	}

	old := s.data.SSHConnections[idx]
	conn := old

	if patch.Name != nil {
		conn.Name = *patch.Name
	}
	if patch.Endpoint != nil {
		conn.Endpoint = *patch.Endpoint
	}
	if patch.Auth != nil {
		conn.Auth = *patch.Auth
	}
	if patch.HostKeyVerification != nil {
		conn.HostKeyVerification = *patch.HostKeyVerification
	}
	if patch.DialTimeoutMs != nil {
		conn.DialTimeoutMs = *patch.DialTimeoutMs
	}
	if patch.KeepAlive != nil {
		conn.KeepAlive = *patch.KeepAlive
	}

	config.ApplySSHConnectionDefaults(&conn)
	s.data.SSHConnections[idx] = conn

	vr, err := s.saveConfig(r.Context())
	if err != nil {
		s.data.SSHConnections[idx] = old
		slog.Error("failed to save config", "err", err)
		writeInternalError(w)
		return
	}
	if vr.HasErrors() {
		s.data.SSHConnections[idx] = old
		writeValidationError(w, vr.Errors)
		return
	}

	writeData(w, http.StatusOK, config.RedactSSHConnection(conn), vr.Warnings)
}
