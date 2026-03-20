// internal/middleware/audit.go
package middleware

import (
	"time"

	"github.com/labstack/echo/v4"
)

type AuditLogger interface {
	Log(userID, action, resource, ip, userAgent string)
}

func AuditLog(logger AuditLogger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)
			userID := ""
			if uid := GetUserID(c); uid.String() != "00000000-0000-0000-0000-000000000000" {
				userID = uid.String()
			}
			logger.Log(
				userID,
				c.Request().Method+" "+c.Path(),
				c.Path(),
				c.RealIP(),
				c.Request().UserAgent(),
			)
			_ = start // available for timing if needed
			return err
		}
	}
}
