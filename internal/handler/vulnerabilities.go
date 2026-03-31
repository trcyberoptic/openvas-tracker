// internal/handler/vulnerabilities.go
package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/cyberoptic/openvas-tracker/internal/database/queries"
	"github.com/cyberoptic/openvas-tracker/internal/service"
)

type VulnHandler struct {
	vulns *service.VulnerabilityService
	q     *queries.Queries
}

func NewVulnHandler(vulns *service.VulnerabilityService, q ...*queries.Queries) *VulnHandler {
	h := &VulnHandler{vulns: vulns}
	if len(q) > 0 {
		h.q = q[0]
	}
	return h
}

func (h *VulnHandler) List(c echo.Context) error {
	limit, offset := paginate(c)
	vulns, err := h.vulns.List(c.Request().Context(), limit, offset)
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

func (h *VulnHandler) AffectedURLs(c echo.Context) error {
	id := c.Param("id")
	if h.q == nil {
		return c.JSON(http.StatusOK, []queries.AffectedURL{})
	}
	urls, err := h.q.AffectedURLsByPeer(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusOK, []queries.AffectedURL{})
	}
	return c.JSON(http.StatusOK, urls)
}

func (h *VulnHandler) RegisterRoutes(g *echo.Group) {
	g.GET("", h.List)
	g.GET("/:id", h.Get)
	g.GET("/:id/affected-urls", h.AffectedURLs)
	g.PATCH("/:id/status", h.UpdateStatus)
}
