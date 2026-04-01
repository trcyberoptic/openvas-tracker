package middleware

import (
	"net/url"
	"os"

	"github.com/labstack/echo/v4"
)

func SecurityHeaders() echo.MiddlewareFunc {
	// Build CSP once at startup; if OT_BUGREPORT_URL is set, allow its origin
	extra := ""
	if raw := os.Getenv("OT_BUGREPORT_URL"); raw != "" {
		if u, err := url.Parse(raw); err == nil {
			extra = " " + u.Scheme + "://" + u.Host
		}
	}
	csp := "default-src 'self'" +
		"; script-src 'self'" + extra +
		"; style-src 'self' 'unsafe-inline'" + extra +
		"; connect-src 'self' ws: wss:" + extra +
		"; img-src 'self' data:" +
		"; font-src 'self'" + extra +
		"; frame-ancestors 'none'; base-uri 'self'; form-action 'self'"

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			h := c.Response().Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("X-XSS-Protection", "1; mode=block")
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			h.Set("Content-Security-Policy", csp)
			h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=()")
			return next(c)
		}
	}
}
