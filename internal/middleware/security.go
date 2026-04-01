package middleware

import (
	"net/url"
	"os"

	"github.com/labstack/echo/v4"
)

func SecurityHeaders() echo.MiddlewareFunc {
	// Build CSP once at startup
	scriptSrc := "'self'"
	connectSrc := "'self' ws: wss:"
	if raw := os.Getenv("OT_BUGREPORT_URL"); raw != "" {
		if u, err := url.Parse(raw); err == nil {
			origin := u.Scheme + "://" + u.Host
			scriptSrc += " " + origin
			connectSrc += " " + origin
		}
	}
	csp := "default-src 'self'; script-src " + scriptSrc +
		"; style-src 'self' 'unsafe-inline'" +
		"; connect-src " + connectSrc +
		"; img-src 'self' data:; font-src 'self' " + scriptSrc +
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
