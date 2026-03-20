// internal/handler/schedules.go
package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/cyberoptic/vulntrack/internal/database/queries"
	"github.com/cyberoptic/vulntrack/internal/middleware"
	"github.com/cyberoptic/vulntrack/internal/service"
)

type ScheduleHandler struct {
	schedules *service.ScheduleService
}

func NewScheduleHandler(s *service.ScheduleService) *ScheduleHandler {
	return &ScheduleHandler{schedules: s}
}

type createScheduleRequest struct {
	Name     string `json:"name" validate:"required"`
	CronExpr string `json:"cron_expr" validate:"required"`
	ScanType string `json:"scan_type" validate:"required,oneof=nmap openvas"`
	TargetID string `json:"target_id" validate:"required,uuid"`
}

func (h *ScheduleHandler) Create(c echo.Context) error {
	var req createScheduleRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	userID := middleware.GetUserID(c)
	tid, _ := uuid.Parse(req.TargetID)
	sched, err := h.schedules.Create(c.Request().Context(), queries.CreateScheduleParams{
		Name: req.Name, CronExpr: req.CronExpr,
		ScanType: queries.ScanType(req.ScanType), TargetID: &tid, UserID: userID,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create schedule")
	}
	return c.JSON(http.StatusCreated, sched)
}

func (h *ScheduleHandler) List(c echo.Context) error {
	userID := middleware.GetUserID(c)
	scheds, err := h.schedules.List(c.Request().Context(), userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list schedules")
	}
	return c.JSON(http.StatusOK, scheds)
}

func (h *ScheduleHandler) Toggle(c echo.Context) error {
	id, _ := uuid.Parse(c.Param("id"))
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	if err := h.schedules.Toggle(c.Request().Context(), id, req.Enabled); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to toggle schedule")
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *ScheduleHandler) Delete(c echo.Context) error {
	id, _ := uuid.Parse(c.Param("id"))
	userID := middleware.GetUserID(c)
	if err := h.schedules.Delete(c.Request().Context(), id, userID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete schedule")
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *ScheduleHandler) RegisterRoutes(g *echo.Group) {
	g.POST("", h.Create)
	g.GET("", h.List)
	g.PATCH("/:id/toggle", h.Toggle)
	g.DELETE("/:id", h.Delete)
}
