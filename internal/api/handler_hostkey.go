package api

import (
	"crypto/subtle"
	"net/http"

	"github.com/maxzhang666/ops-tunnel/internal/config"
	tunnelssh "github.com/maxzhang666/ops-tunnel/internal/ssh"
)

type knownHostsResponse struct {
	Hosts []tunnelssh.KnownHost `json:"hosts"`
}

type trustHostKeyRequest struct {
	HostPort    string `json:"hostPort"`
	Fingerprint string `json:"fingerprint"`
}

// listKnownHosts returns every trusted host key with its fingerprint.
func (s *Server) listKnownHosts(w http.ResponseWriter, r *http.Request) {
	hosts := s.hostKeys.List()
	if hosts == nil {
		hosts = []tunnelssh.KnownHost{}
	}
	writeJSON(w, http.StatusOK, knownHostsResponse{Hosts: hosts})
}

// trustHostKey promotes a pending host key to trusted. The request carries the
// fingerprint the user confirmed, never key material: the key itself comes from
// what the server observed on the wire, so a client cannot inject a key.
func (s *Server) trustHostKey(w http.ResponseWriter, r *http.Request) {
	var req trustHostKeyRequest
	if err := decodeBody(r, &req); err != nil {
		writeBodyError(w, err)
		return
	}

	var errs []config.ValidationError
	if req.HostPort == "" {
		errs = append(errs, config.ValidationError{Field: "hostPort", Message: "must not be empty"})
	}
	if req.Fingerprint == "" {
		errs = append(errs, config.ValidationError{Field: "fingerprint", Message: "must not be empty"})
	}
	if len(errs) > 0 {
		writeValidationError(w, errs)
		return
	}

	pending, ok := s.hostKeys.Pending(req.HostPort)
	if !ok {
		writeConflictCode(w, "no_pending_host_key")
		return
	}

	fp, keyType, err := tunnelssh.Fingerprint(pending)
	if err != nil {
		writeInternalError(w)
		return
	}
	if subtle.ConstantTimeCompare([]byte(fp), []byte(req.Fingerprint)) != 1 {
		writeConflictCode(w, "fingerprint_mismatch")
		return
	}

	if err := s.hostKeys.Add(req.HostPort, pending); err != nil {
		writeInternalError(w)
		return
	}
	s.hostKeys.ClearPending(req.HostPort)

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":          true,
		"fingerprint": fp,
		"keyType":     keyType,
	})
}

// revokeHostKey removes a trusted host key. Idempotent.
func (s *Server) revokeHostKey(w http.ResponseWriter, r *http.Request) {
	hostPort := r.URL.Query().Get("hostPort")
	if hostPort == "" {
		writeValidationError(w, []config.ValidationError{
			{Field: "hostPort", Message: "must not be empty"},
		})
		return
	}

	if err := s.hostKeys.Delete(hostPort); err != nil {
		writeInternalError(w)
		return
	}
	s.hostKeys.ClearPending(hostPort)

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
