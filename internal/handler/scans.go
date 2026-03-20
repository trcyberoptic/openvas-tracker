package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/cyberoptic/openvas-tracker/internal/database/queries"
	"github.com/cyberoptic/openvas-tracker/internal/service"
)

type ScanHandler struct {
	q     *queries.Queries
	vulns *service.VulnerabilityService
}

func NewScanHandler(q *queries.Queries, vulns *service.VulnerabilityService) *ScanHandler {
	return &ScanHandler{q: q, vulns: vulns}
}

func (h *ScanHandler) List(c echo.Context) error {
	scans, err := h.q.ListScans(c.Request().Context(), queries.ListScansParams{
		Limit: 50, Offset: 0,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list scans")
	}
	return c.JSON(http.StatusOK, scans)
}

func (h *ScanHandler) Get(c echo.Context) error {
	id := c.Param("id")
	scan, err := h.q.GetScan(c.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "scan not found")
	}
	return c.JSON(http.StatusOK, scan)
}

func (h *ScanHandler) Vulns(c echo.Context) error {
	id := c.Param("id")
	vulns, err := h.vulns.ListByScan(c.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list vulnerabilities")
	}
	return c.JSON(http.StatusOK, vulns)
}

func (h *ScanHandler) RegisterRoutes(g *echo.Group) {
	g.GET("", h.List)
	g.GET("/:id", h.Get)
	g.GET("/:id/vulnerabilities", h.Vulns)
}
