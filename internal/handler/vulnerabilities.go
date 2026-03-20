// internal/handler/vulnerabilities.go
package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/cyberoptic/vulntrack/internal/middleware"
	"github.com/cyberoptic/vulntrack/internal/service"
)

type VulnHandler struct {
	vulns *service.VulnerabilityService
}

func NewVulnHandler(vulns *service.VulnerabilityService) *VulnHandler {
	return &VulnHandler{vulns: vulns}
}

func (h *VulnHandler) List(c echo.Context) error {
	userID := middleware.GetUserID(c)
	vulns, err := h.vulns.List(c.Request().Context(), userID, 100, 0)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list vulnerabilities")
	}
	return c.JSON(http.StatusOK, vulns)
}

func (h *VulnHandler) Get(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid ID")
	}
	vuln, err := h.vulns.Get(c.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "vulnerability not found")
	}
	return c.JSON(http.StatusOK, vuln)
}

type updateVulnStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=open confirmed mitigated resolved false_positive accepted"`
}

func (h *VulnHandler) UpdateStatus(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid ID")
	}
	var req updateVulnStatusRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	vuln, err := h.vulns.UpdateStatus(c.Request().Context(), id, req.Status)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update status")
	}
	return c.JSON(http.StatusOK, vuln)
}

func (h *VulnHandler) RegisterRoutes(g *echo.Group) {
	g.GET("", h.List)
	g.GET("/:id", h.Get)
	g.PATCH("/:id/status", h.UpdateStatus)
}
