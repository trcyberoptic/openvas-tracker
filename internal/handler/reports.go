// internal/handler/reports.go
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/cyberoptic/openvas-tracker/internal/database/queries"
	"github.com/cyberoptic/openvas-tracker/internal/middleware"
	"github.com/cyberoptic/openvas-tracker/internal/service"
)

type ReportHandler struct {
	reports *service.ReportService
}

func NewReportHandler(reports *service.ReportService) *ReportHandler {
	return &ReportHandler{reports: reports}
}

type generateReportRequest struct {
	Name       string   `json:"name" validate:"required"`
	ReportType string   `json:"report_type" validate:"required,oneof=technical executive compliance comparison trend"`
	Format     string   `json:"format" validate:"required,oneof=html pdf excel markdown"`
	ScanIDs    []string `json:"scan_ids" validate:"required,min=1"`
}

func (h *ReportHandler) Generate(c echo.Context) error {
	var req generateReportRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	userID := middleware.GetUserID(c)

	scanIDsJSON, _ := json.Marshal(req.ScanIDs)

	rpt, err := h.reports.Create(c.Request().Context(), queries.CreateReportParams{
		Name:       req.Name,
		ReportType: queries.ReportType(req.ReportType),
		Format:     queries.ReportFormat(req.Format),
		Status:     "generating",
		ScanIds:    scanIDsJSON,
		UserID:     userID,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create report")
	}

	data, err := h.reports.Generate(c.Request().Context(), rpt.ID, req.ScanIDs, req.Format, userID)
	if err != nil {
		h.reports.UpdateStatus(c.Request().Context(), rpt.ID, "failed", nil)
		return echo.NewHTTPError(http.StatusInternalServerError, "report generation failed: "+err.Error())
	}

	h.reports.UpdateStatus(c.Request().Context(), rpt.ID, "completed", data)

	contentType := map[string]string{
		"html":     "text/html",
		"pdf":      "application/pdf",
		"excel":    "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"markdown": "text/markdown",
	}
	return c.Blob(http.StatusOK, contentType[req.Format], data)
}

func (h *ReportHandler) List(c echo.Context) error {
	userID := middleware.GetUserID(c)
	reports, err := h.reports.List(c.Request().Context(), userID, 50, 0)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list reports")
	}
	return c.JSON(http.StatusOK, reports)
}

func (h *ReportHandler) Get(c echo.Context) error {
	id := c.Param("id")
	rpt, err := h.reports.Get(c.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "report not found")
	}
	return c.JSON(http.StatusOK, rpt)
}

func (h *ReportHandler) RegisterRoutes(g *echo.Group) {
	g.POST("", h.Generate)
	g.GET("", h.List)
	g.GET("/:id", h.Get)
}
