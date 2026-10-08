package handler

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/cyberoptic/openvas-tracker/internal/config"
)

func TestLogin_InvalidBody(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`not-json`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	h := &AuthHandler{jwtSecret: "test", cfg: &config.Config{}}
	err := h.Login(c)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	he, ok := err.(*echo.HTTPError)
	if !ok {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if he.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", he.Code)
	}
}

func postLogin(body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "203.0.113.7:4444"
	return req
}

func TestLogin_AdminNeverFallsThrough(t *testing.T) {
	// users and ldap are nil on purpose: a fallthrough to LDAP or the DB user
	// table (where a stale bcrypt hash of an old admin password lives) would nil-deref.
	for name, cfg := range map[string]*config.Config{
		"wrong password": {Admin: config.AdminConfig{Password: "correct-horse"}},
		"admin disabled": {},
	} {
		h := &AuthHandler{jwtSecret: "test", cfg: cfg}
		rec := httptest.NewRecorder()
		err := h.Login(newTestEcho().NewContext(postLogin(`{"username":"admin","password":"stale-old-password"}`), rec))
		he, ok := err.(*echo.HTTPError)
		if !ok || he.Code != http.StatusUnauthorized {
			t.Errorf("%s: err = %v, want 401", name, err)
		}
	}
}

func TestLogin_FailedAttemptIsLogged(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	h := &AuthHandler{jwtSecret: "test", cfg: &config.Config{Admin: config.AdminConfig{Password: "correct-horse"}}}
	rec := httptest.NewRecorder()
	_ = h.Login(newTestEcho().NewContext(postLogin(`{"username":"admin","password":"nope"}`), rec))

	for _, want := range []string{"login failed", `"admin"`, "203.0.113.7"} {
		if !strings.Contains(buf.String(), want) {
			t.Errorf("log %q lacks %q", buf.String(), want)
		}
	}
}
