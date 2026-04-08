package api

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

func (s *Server) startTunnel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.eng.StartTunnel(r.Context(), id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeNotFound(w, "tunnel", id)
			return
		}
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) stopTunnel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.eng.StopTunnel(r.Context(), id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeNotFound(w, "tunnel", id)
			return
		}
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) restartTunnel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.eng.RestartTunnel(r.Context(), id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeNotFound(w, "tunnel", id)
			return
		}
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) getTunnelStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	st, ok := s.eng.GetStatus(id)
	if !ok {
		writeNotFound(w, "tunnel", id)
		return
	}
	writeJSON(w, http.StatusOK, st)
}
