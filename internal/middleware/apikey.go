package middleware

import (
	"crypto/subtle"
	"net/http"

	"github.com/labstack/echo/v4"
)

func APIKeyAuth(key string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if key == "" {
				return echo.NewHTTPError(http.StatusServiceUnavailable, "import API key not configured")
			}
			provided := c.Request().Header.Get("X-API-Key")
			if provided == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing API key")
			}
			if subtle.ConstantTimeCompare([]byte(provided), []byte(key)) != 1 {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid API key")
			}
			return next(c)
		}
	}
}
