package middleware

import (
	"crypto/sha256"
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
				provided = c.QueryParam("api_key")
			}
			if provided == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing API key")
			}
			providedHash := sha256.Sum256([]byte(provided))
			keyHash := sha256.Sum256([]byte(key))
			if subtle.ConstantTimeCompare(providedHash[:], keyHash[:]) != 1 {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid API key")
			}
			return next(c)
		}
	}
}
