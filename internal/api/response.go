package api

import (
	"encoding/json"
	"net/http"

	"github.com/maxzhang666/ops-tunnel/internal/config"
)

// ErrorResponse is the standard error envelope.
type ErrorResponse struct {
	Error   string                   `json:"error"`
	Details []config.ValidationError `json:"details,omitempty"`
}

// DataResponse wraps a successful response, optionally with warnings.
type DataResponse struct {
	Data     any      `json:"data"`
	Warnings []string `json:"warnings,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeData(w http.ResponseWriter, status int, data any, warnings []string) {
	if len(warnings) > 0 {
		writeJSON(w, status, DataResponse{Data: data, Warnings: warnings})
	} else {
		writeJSON(w, status, data)
	}
}

func writeValidationError(w http.ResponseWriter, errs []config.ValidationError) {
	writeJSON(w, http.StatusBadRequest, ErrorResponse{
		Error:   "validation_failed",
		Details: errs,
	})
}

func writeNotFound(w http.ResponseWriter, resource, id string) {
	writeJSON(w, http.StatusNotFound, ErrorResponse{
		Error:   "not_found",
		Details: []config.ValidationError{{Field: resource, Message: "'" + id + "' not found"}},
	})
}

func writeConflict(w http.ResponseWriter, details []config.ValidationError) {
	writeJSON(w, http.StatusConflict, ErrorResponse{
		Error:   "conflict",
		Details: details,
	})
}

func writeInternalError(w http.ResponseWriter) {
	writeJSON(w, http.StatusInternalServerError, ErrorResponse{
		Error: "internal_error",
	})
}

func decodeBody(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}
