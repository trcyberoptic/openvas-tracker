// internal/middleware/rbac_test.go
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestRequireRole_Allowed(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(contextKeyRole, "admin")

	handler := RequireRole("admin", "analyst")(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	if err := handler(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRequireRole_Denied(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(contextKeyRole, "viewer")

	handler := RequireRole("admin")(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	err := handler(c)
	if err == nil {
		t.Fatal("expected error for unauthorized role")
	}
	he, ok := err.(*echo.HTTPError)
	if !ok || he.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %v", err)
	}
}
