package handler

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"

	"github.com/cyberoptic/openvas-tracker/internal/config"
	"github.com/cyberoptic/openvas-tracker/internal/service"
)

type testValidator struct{ v *validator.Validate }

func (t *testValidator) Validate(i interface{}) error { return t.v.Struct(i) }

func newTestEcho() *echo.Echo {
	e := echo.New()
	e.Validator = &testValidator{v: validator.New()}
	return e
}

// withRole stands in for JWTAuth: it stamps the role the way the middleware does.
func withRole(role string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("user_role", role)
			return next(c)
		}
	}
}

func TestMaskEnvValues(t *testing.T) {
	in := map[string]string{
		"OT_JWT_SECRET":         "supersecretjwtkey12345",
		"OT_IMPORT_APIKEY":      "apikey-abcdefghijklmnop",
		"OT_ADMIN_PASSWORD":     "adminpw123",
		"OT_LDAP_BIND_PASSWORD": "ldappw1234",
		"OT_GMP_PASSWORD":       "gmppassword123",
		"OT_DATABASE_DSN":       "user:pw@tcp(localhost:3306)/db",
		"OT_SERVER_PORT":        "8080",
	}
	out := maskEnvValues(in)

	for _, k := range []string{"OT_JWT_SECRET", "OT_IMPORT_APIKEY", "OT_ADMIN_PASSWORD", "OT_LDAP_BIND_PASSWORD", "OT_GMP_PASSWORD", "OT_DATABASE_DSN"} {
		want := in[k][:4] + "********"
		if out[k] != want {
			t.Errorf("%s = %q, want masked %q", k, out[k], want)
		}
	}
	if out["OT_SERVER_PORT"] != "8080" {
		t.Errorf("OT_SERVER_PORT = %q, want passthrough", out["OT_SERVER_PORT"])
	}
}

func TestSettingsRoutes_ConfigEndpointsRequireAdmin(t *testing.T) {
	// Nil dependencies on purpose: the role check must reject before any handler runs.
	h := &SettingsHandler{}
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/settings/env"},
		{http.MethodPut, "/settings/env"},
		{http.MethodPut, "/settings/env/batch"},
		{http.MethodPost, "/settings/ldap/test"},
		{http.MethodGet, "/settings/mail"},
		{http.MethodPut, "/settings/mail"},
		{http.MethodPost, "/settings/mail/test"},
	} {
		e := newTestEcho()
		h.RegisterRoutes(e.Group("/settings", withRole("viewer")))
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, httptest.NewRequest(tc.method, tc.path, nil))
		if rec.Code != http.StatusForbidden {
			t.Errorf("%s %s as viewer: status %d, want 403", tc.method, tc.path, rec.Code)
		}
	}
}

func TestUpdateEnvBatch_RejectsSecretKeysWith400(t *testing.T) {
	dir := t.TempDir()
	h := &SettingsHandler{envSvc: service.NewEnvFileService(filepath.Join(dir, "env"))}
	e := newTestEcho()
	h.RegisterRoutes(e.Group("/settings", withRole("admin")))

	req := httptest.NewRequest(http.MethodPut, "/settings/env/batch", strings.NewReader(`{"values":{"OT_JWT_SECRET":"x"}}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d, want 400: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "not editable") {
		t.Errorf("body = %s, want the allowlist error", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), dir) {
		t.Errorf("body leaks the env file path: %s", rec.Body.String())
	}
}

func TestGetSetup_HidesImportKeyFromNonAdmins(t *testing.T) {
	h := &SettingsHandler{cfg: &config.Config{Import: config.ImportConfig{APIKey: "abcdefgh-import-key-0123456789"}}}
	for role, wantKey := range map[string]bool{"viewer": false, "admin": true} {
		e := newTestEcho()
		h.RegisterRoutes(e.Group("/settings", withRole(role)))
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/settings/setup", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: status %d", role, rec.Code)
		}
		if got := strings.Contains(rec.Body.String(), "api_key_masked"); got != wantKey {
			t.Errorf("%s: api_key_masked present = %v, want %v", role, got, wantKey)
		}
		// Shell.tsx loads the bug-report widget for everyone from this field.
		if !strings.Contains(rec.Body.String(), "bugreport_url") {
			t.Errorf("%s: bugreport_url missing", role)
		}
	}
}
