// internal/handler/dashboard.go
package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/cyberoptic/openvas-tracker/internal/middleware"
	"github.com/cyberoptic/openvas-tracker/internal/service"
)

type DashboardHandler struct {
	vulns   *service.VulnerabilityService
	tickets *service.TicketService
}

func NewDashboardHandler(vulns *service.VulnerabilityService, tickets *service.TicketService) *DashboardHandler {
	return &DashboardHandler{vulns: vulns, tickets: tickets}
}

type dashboardResponse struct {
	VulnsBySeverity []severityCount `json:"vulns_by_severity"`
	TicketsByStatus []statusCount   `json:"tickets_by_status"`
	RecentVulns     int             `json:"recent_vulns"`
}

type severityCount struct {
	Severity string `json:"severity"`
	Count    int64  `json:"count"`
}

type statusCount struct {
	Status string `json:"status"`
	Count  int64  `json:"count"`
}

func (h *DashboardHandler) Get(c echo.Context) error {
	ctx := c.Request().Context()
	userID := middleware.GetUserID(c)

	vulnCounts, err := h.vulns.CountBySeverity(ctx, userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to load dashboard")
	}

	var sevCounts []severityCount
	for _, vc := range vulnCounts {
		sevCounts = append(sevCounts, severityCount{
			Severity: string(vc.Severity),
			Count:    vc.Count,
		})
	}

	return c.JSON(http.StatusOK, dashboardResponse{
		VulnsBySeverity: sevCounts,
	})
}

func (h *DashboardHandler) RegisterRoutes(g *echo.Group) {
	g.GET("", h.Get)
}
