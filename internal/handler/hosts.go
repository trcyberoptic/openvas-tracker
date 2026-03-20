package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/cyberoptic/openvas-tracker/internal/database/queries"
)

type HostHandler struct {
	q *queries.Queries
}

func NewHostHandler(q *queries.Queries) *HostHandler {
	return &HostHandler{q: q}
}

func (h *HostHandler) List(c echo.Context) error {
	hosts, err := h.q.ListHostSummaries(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list hosts")
	}
	return c.JSON(http.StatusOK, hosts)
}

func (h *HostHandler) RegisterRoutes(g *echo.Group) {
	g.GET("", h.List)
}
