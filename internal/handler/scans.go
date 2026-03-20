// internal/handler/scans.go
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/hibiken/asynq"
	"github.com/labstack/echo/v4"

	"github.com/cyberoptic/openvas-tracker/internal/database/queries"
	"github.com/cyberoptic/openvas-tracker/internal/middleware"
	"github.com/cyberoptic/openvas-tracker/internal/worker"
)

type ScanHandler struct {
	q      *queries.Queries
	client *asynq.Client
}

func NewScanHandler(q *queries.Queries, client *asynq.Client) *ScanHandler {
	return &ScanHandler{q: q, client: client}
}

type launchScanRequest struct {
	Name     string   `json:"name" validate:"required"`
	ScanType string   `json:"scan_type" validate:"required,oneof=nmap openvas"`
	TargetID string   `json:"target_id" validate:"required,uuid"`
	Options  []string `json:"options"`
}

func (h *ScanHandler) Launch(c echo.Context) error {
	var req launchScanRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	userID := middleware.GetUserID(c)
	targetID := req.TargetID

	optJSON, _ := json.Marshal(req.Options)

	scan, err := h.q.CreateScan(c.Request().Context(), queries.CreateScanParams{
		Name:     req.Name,
		ScanType: queries.ScanType(req.ScanType),
		Status:   queries.ScanStatusPending,
		TargetID: &targetID,
		UserID:   userID,
		Options:  optJSON,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create scan")
	}

	// Get target host for the scanner
	target, err := h.q.GetTarget(c.Request().Context(), queries.GetTargetParams{ID: targetID, UserID: userID})
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "target not found")
	}

	taskType := worker.TaskScanNmap
	if req.ScanType == "openvas" {
		taskType = worker.TaskScanOpenVAS
	}

	task, err := worker.NewScanTask(taskType, worker.ScanPayload{
		ScanID:  scan.ID,
		Target:  target.Host,
		Options: req.Options,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create task")
	}

	if _, err := h.client.Enqueue(task); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to enqueue scan")
	}

	return c.JSON(http.StatusAccepted, scan)
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
	g.POST("", h.Launch)
	g.GET("", h.List)
	g.GET("/:id", h.Get)
}
