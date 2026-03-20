// internal/handler/targets.go
package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/cyberoptic/openvas-tracker/internal/database/queries"
	"github.com/cyberoptic/openvas-tracker/internal/middleware"
	"github.com/cyberoptic/openvas-tracker/internal/service"
)

type TargetHandler struct {
	targets *service.TargetService
}

func NewTargetHandler(targets *service.TargetService) *TargetHandler {
	return &TargetHandler{targets: targets}
}

type createTargetRequest struct {
	Host      string  `json:"host" validate:"required"`
	IPAddress *string `json:"ip_address"`
	Hostname  *string `json:"hostname"`
	GroupID   *string `json:"group_id"`
}

func (h *TargetHandler) Create(c echo.Context) error {
	var req createTargetRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	userID := middleware.GetUserID(c)
	params := queries.CreateTargetParams{
		Host:   req.Host,
		UserID: userID,
	}
	if req.IPAddress != nil {
		params.IpAddress = req.IPAddress
	}
	if req.Hostname != nil {
		params.Hostname = req.Hostname
	}
	if req.GroupID != nil {
		params.GroupID = req.GroupID
	}

	target, err := h.targets.Create(c.Request().Context(), params)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create target")
	}
	return c.JSON(http.StatusCreated, target)
}

func (h *TargetHandler) List(c echo.Context) error {
	userID := middleware.GetUserID(c)
	targets, err := h.targets.List(c.Request().Context(), userID, 100, 0)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list targets")
	}
	return c.JSON(http.StatusOK, targets)
}

func (h *TargetHandler) Get(c echo.Context) error {
	id := c.Param("id")
	userID := middleware.GetUserID(c)
	target, err := h.targets.Get(c.Request().Context(), id, userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "target not found")
	}
	return c.JSON(http.StatusOK, target)
}

func (h *TargetHandler) Delete(c echo.Context) error {
	id := c.Param("id")
	userID := middleware.GetUserID(c)
	if err := h.targets.Delete(c.Request().Context(), id, userID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete target")
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *TargetHandler) RegisterRoutes(g *echo.Group) {
	g.POST("", h.Create)
	g.GET("", h.List)
	g.GET("/:id", h.Get)
	g.DELETE("/:id", h.Delete)
}
