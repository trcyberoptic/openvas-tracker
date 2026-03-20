package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestAPIKeyAuth_ValidKey(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("X-API-Key", "a]valid-key-that-is-at-least-32-chars!!")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := APIKeyAuth("a]valid-key-that-is-at-least-32-chars!!")(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
	if err := handler(c); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestAPIKeyAuth_InvalidKey(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("X-API-Key", "wrong-key-wrong-key-wrong-key-wrong")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := APIKeyAuth("a]valid-key-that-is-at-least-32-chars!!")(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
	err := handler(c)
	he, ok := err.(*echo.HTTPError)
	if !ok || he.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %v", err)
	}
}

func TestAPIKeyAuth_MissingHeader(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := APIKeyAuth("a]valid-key-that-is-at-least-32-chars!!")(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
	err := handler(c)
	he, ok := err.(*echo.HTTPError)
	if !ok || he.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %v", err)
	}
}

func TestAPIKeyAuth_EmptyConfigKey(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("X-API-Key", "anything")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := APIKeyAuth("")(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
	err := handler(c)
	he, ok := err.(*echo.HTTPError)
	if !ok || he.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %v", err)
	}
}
