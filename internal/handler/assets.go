// internal/handler/assets.go
package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/cyberoptic/vulntrack/internal/middleware"
	"github.com/cyberoptic/vulntrack/internal/service"
)

type AssetHandler struct {
	assets *service.AssetService
}

func NewAssetHandler(a *service.AssetService) *AssetHandler {
	return &AssetHandler{assets: a}
}

func (h *AssetHandler) List(c echo.Context) error {
	userID := middleware.GetUserID(c)
	assets, err := h.assets.List(c.Request().Context(), userID, 100, 0)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list assets")
	}
	return c.JSON(http.StatusOK, assets)
}

func (h *AssetHandler) Get(c echo.Context) error {
	id, _ := uuid.Parse(c.Param("id"))
	userID := middleware.GetUserID(c)
	asset, err := h.assets.Get(c.Request().Context(), id, userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "asset not found")
	}
	return c.JSON(http.StatusOK, asset)
}

func (h *AssetHandler) Delete(c echo.Context) error {
	id, _ := uuid.Parse(c.Param("id"))
	userID := middleware.GetUserID(c)
	if err := h.assets.Delete(c.Request().Context(), id, userID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete asset")
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *AssetHandler) RegisterRoutes(g *echo.Group) {
	g.GET("", h.List)
	g.GET("/:id", h.Get)
	g.DELETE("/:id", h.Delete)
}
