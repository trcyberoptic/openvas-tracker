// internal/middleware/auth.go
package middleware

import (
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/cyberoptic/vulntrack/internal/auth"
)

const (
	contextKeyUserID = "user_id"
	contextKeyRole   = "user_role"
)

func JWTAuth(secret string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			header := c.Request().Header.Get("Authorization")
			if header == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing authorization header")
			}

			parts := strings.SplitN(header, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid authorization format")
			}

			claims, err := auth.ValidateToken(parts[1], secret)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid or expired token")
			}

			c.Set(contextKeyUserID, claims.UserID)
			c.Set(contextKeyRole, claims.Role)
			return next(c)
		}
	}
}

func GetUserID(c echo.Context) uuid.UUID {
	id, _ := c.Get(contextKeyUserID).(uuid.UUID)
	return id
}

func GetUserRole(c echo.Context) string {
	role, _ := c.Get(contextKeyRole).(string)
	return role
}
