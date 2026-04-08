package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func newTestAuthServer(username, password string) *Server {
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	s := &Server{
		sessions: NewSessionStore(),
	}
	s.webAuth = &webAuthConfig{
		Username:     username,
		PasswordHash: string(hash),
	}
	return s
}

func TestHandleLogin_Success(t *testing.T) {
	s := newTestAuthServer("admin", "secret123")
	body := `{"username":"admin","password":"secret123"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.handleLogin(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	cookies := w.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == "session" {
			found = true
			if !c.HttpOnly {
				t.Error("cookie should be HttpOnly")
			}
			if c.SameSite != http.SameSiteStrictMode {
				t.Error("cookie should be SameSite=Strict")
			}
		}
	}
	if !found {
		t.Error("expected session cookie in response")
	}
}

func TestHandleLogin_WrongPassword(t *testing.T) {
	s := newTestAuthServer("admin", "secret123")
	body := `{"username":"admin","password":"wrong"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.handleLogin(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestHandleLogin_WrongUsername(t *testing.T) {
	s := newTestAuthServer("admin", "secret123")
	body := `{"username":"root","password":"secret123"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.handleLogin(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestHandleLogout(t *testing.T) {
	s := newTestAuthServer("admin", "secret123")
	sess := s.sessions.Create(24 * time.Hour)
	req := httptest.NewRequest("POST", "/api/v1/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: sess.Token})
	w := httptest.NewRecorder()
	s.handleLogout(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	if s.sessions.Valid(sess.Token) {
		t.Error("session should be deleted after logout")
	}
}

func TestHandleAuthCheck_Authenticated(t *testing.T) {
	s := newTestAuthServer("admin", "secret123")
	sess := s.sessions.Create(24 * time.Hour)
	req := httptest.NewRequest("GET", "/api/v1/auth/check", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: sess.Token})
	w := httptest.NewRecorder()
	s.handleAuthCheck(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"authenticated":true`) {
		t.Errorf("body = %s, want authenticated:true", w.Body.String())
	}
}

func TestHandleAuthCheck_NotAuthenticated(t *testing.T) {
	s := newTestAuthServer("admin", "secret123")
	req := httptest.NewRequest("GET", "/api/v1/auth/check", nil)
	w := httptest.NewRecorder()
	s.handleAuthCheck(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"authenticated":false`) {
		t.Errorf("body = %s, want authenticated:false", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"required":true`) {
		t.Errorf("body = %s, want required:true", w.Body.String())
	}
}
