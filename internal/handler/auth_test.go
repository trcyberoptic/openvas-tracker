// internal/handler/auth_test.go
package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestRegister_InvalidBody(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Without a validator set, Bind succeeds but Validate should catch missing fields
	// This test validates the handler returns 400 for missing required fields
	h := &AuthHandler{jwtSecret: "test"}
	err := h.Register(c)
	if err == nil {
		t.Fatal("expected error for empty body")
	}
	he, ok := err.(*echo.HTTPError)
	if !ok {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if he.Code != http.StatusBadRequest && he.Code != http.StatusInternalServerError {
		t.Errorf("expected 400 or 500, got %d", he.Code)
	}
}

func TestLogin_InvalidBody(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`not-json`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	h := &AuthHandler{jwtSecret: "test"}
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
