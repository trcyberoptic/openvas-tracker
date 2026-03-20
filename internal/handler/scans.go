package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/cyberoptic/openvas-tracker/internal/database/queries"
	"github.com/cyberoptic/openvas-tracker/internal/middleware"
)

type ScanHandler struct {
	q *queries.Queries
}

func NewScanHandler(q *queries.Queries) *ScanHandler {
	return &ScanHandler{q: q}
}

func (h *ScanHandler) List(c echo.Context) error {
	userID := middleware.GetUserID(c)
	scans, err := h.q.ListScans(c.Request().Context(), queries.ListScansParams{
		UserID: userID, Limit: 50, Offset: 0,
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

func (h *ScanHandler) RegisterRoutes(g *echo.Group) {
	g.GET("", h.List)
	g.GET("/:id", h.Get)
}
