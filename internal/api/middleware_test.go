package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func dummyHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	})
}

func TestTokenAuth_ValidToken(t *testing.T) {
	h := TokenAuth("secret")(dummyHandler())
	req := httptest.NewRequest("GET", "/api/v1/tunnels", nil)
	req.Header.Set("Authorization", "Bearer secret")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestTokenAuth_MissingToken(t *testing.T) {
	h := TokenAuth("secret")(dummyHandler())
	req := httptest.NewRequest("GET", "/api/v1/tunnels", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestTokenAuth_WrongToken(t *testing.T) {
	h := TokenAuth("secret")(dummyHandler())
	req := httptest.NewRequest("GET", "/api/v1/tunnels", nil)
	req.Header.Set("Authorization", "Bearer wrong")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestTokenAuth_EmptyConfigToken(t *testing.T) {
	h := TokenAuth("")(dummyHandler())
	req := httptest.NewRequest("GET", "/api/v1/tunnels", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 (auth skipped)", w.Code)
	}
}

func TestCORS_Preflight(t *testing.T) {
	h := CORS(dummyHandler())
	req := httptest.NewRequest("OPTIONS", "/api/v1/tunnels", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("missing CORS Allow-Origin")
	}
	if !strings.Contains(w.Header().Get("Access-Control-Allow-Methods"), "PATCH") {
		t.Error("missing PATCH in Allow-Methods")
	}
}

func TestCORS_NormalRequest(t *testing.T) {
	h := CORS(dummyHandler())
	req := httptest.NewRequest("GET", "/api/v1/tunnels", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("missing CORS header on normal request")
	}
}

func TestSecurityHeaders(t *testing.T) {
	h := SecurityHeaders(dummyHandler())
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("missing X-Content-Type-Options")
	}
	if w.Header().Get("X-Frame-Options") != "DENY" {
		t.Error("missing X-Frame-Options")
	}
}
