// internal/handler/search.go
package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/cyberoptic/openvas-tracker/internal/service"
)

type SearchHandler struct {
	search *service.SearchService
}

func NewSearchHandler(s *service.SearchService) *SearchHandler {
	return &SearchHandler{search: s}
}

func (h *SearchHandler) Search(c echo.Context) error {
	q := c.QueryParam("q")
	if q == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "query parameter 'q' is required")
	}
	results, err := h.search.Search(c.Request().Context(), q, 50)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "search failed")
	}
	return c.JSON(http.StatusOK, results)
}

func (h *SearchHandler) RegisterRoutes(g *echo.Group) {
	g.GET("", h.Search)
}
