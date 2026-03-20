// internal/handler/vulnerabilities.go
package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/cyberoptic/openvas-tracker/internal/service"
)

type VulnHandler struct {
	vulns *service.VulnerabilityService
}

func NewVulnHandler(vulns *service.VulnerabilityService) *VulnHandler {
	return &VulnHandler{vulns: vulns}
}

func (h *VulnHandler) List(c echo.Context) error {
	vulns, err := h.vulns.List(c.Request().Context(), 100, 0)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list vulnerabilities")
	}
	return c.JSON(http.StatusOK, vulns)
}

func (h *VulnHandler) Get(c echo.Context) error {
	id := c.Param("id")
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
	id := c.Param("id")
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
