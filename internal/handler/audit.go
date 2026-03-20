// internal/handler/audit.go
package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/cyberoptic/openvas-tracker/internal/middleware"
	"github.com/cyberoptic/openvas-tracker/internal/service"
)

type AuditHandler struct {
	audit *service.AuditService
}

func NewAuditHandler(a *service.AuditService) *AuditHandler {
	return &AuditHandler{audit: a}
}

func (h *AuditHandler) List(c echo.Context) error {
	role := middleware.GetUserRole(c)
	if role != "admin" {
		return echo.NewHTTPError(http.StatusForbidden, "admin only")
	}
	logs, err := h.audit.List(c.Request().Context(), 100, 0)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list audit logs")
	}
	return c.JSON(http.StatusOK, logs)
}

func (h *AuditHandler) RegisterRoutes(g *echo.Group) {
	g.GET("", h.List)
}
