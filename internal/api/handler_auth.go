package api

import (
	"crypto/subtle"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// webAuthConfig holds loaded admin credentials for runtime use.
type webAuthConfig struct {
	Username     string
	PasswordHash string
}

type loginRequest struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	RememberMe bool   `json:"rememberMe"`
}

type authCheckResponse struct {
	Authenticated bool `json:"authenticated"`
	Required      bool `json:"required"`
}

const (
	sessionTTL         = 24 * time.Hour
	sessionTTLRemember = 30 * 24 * time.Hour
)

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decodeBody(r, &req); err != nil {
		writeBodyError(w, err)
		return
	}

	if subtle.ConstantTimeCompare([]byte(req.Username), []byte(s.webAuth.Username)) != 1 {
		bcrypt.CompareHashAndPassword([]byte(s.webAuth.PasswordHash), []byte(req.Password))
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "invalid credentials"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(s.webAuth.PasswordHash), []byte(req.Password)); err != nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "invalid credentials"})
		return
	}

	ttl := sessionTTL
	maxAge := int(sessionTTL.Seconds())
	if req.RememberMe {
		ttl = sessionTTLRemember
		maxAge = int(sessionTTLRemember.Seconds())
	}

	sess := s.sessions.Create(ttl)
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    sess.Token,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie("session"); err == nil {
		s.sessions.Delete(c.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleAuthCheck(w http.ResponseWriter, r *http.Request) {
	resp := authCheckResponse{Required: s.webAuth != nil}
	if s.webAuth != nil {
		if c, err := r.Cookie("session"); err == nil && s.sessions.Valid(c.Value) {
			resp.Authenticated = true
		}
		if !resp.Authenticated && s.cfg.Token != "" {
			auth := r.Header.Get("Authorization")
			if len(auth) > 7 && auth[:7] == "Bearer " && auth[7:] == s.cfg.Token {
				resp.Authenticated = true
			}
		}
	}
	writeJSON(w, http.StatusOK, resp)
}
