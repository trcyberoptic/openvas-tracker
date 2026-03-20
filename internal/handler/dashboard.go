package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/cyberoptic/openvas-tracker/internal/database/queries"
	"github.com/cyberoptic/openvas-tracker/internal/middleware"
	"github.com/cyberoptic/openvas-tracker/internal/service"
)

type DashboardHandler struct {
	vulns   *service.VulnerabilityService
	tickets *service.TicketService
	q       *queries.Queries
}

func NewDashboardHandler(vulns *service.VulnerabilityService, tickets *service.TicketService, q *queries.Queries) *DashboardHandler {
	return &DashboardHandler{vulns: vulns, tickets: tickets, q: q}
}

type dashboardResponse struct {
	VulnsBySeverity    []severityCount `json:"vulns_by_severity"`
	MyTickets          int64           `json:"my_tickets"`
	UnassignedTickets  int64           `json:"unassigned_tickets"`
	OpenTicketsTotal   int64           `json:"open_tickets_total"`
	ResolvedTickets    int64           `json:"resolved_tickets"`
}

type severityCount struct {
	Severity string `json:"severity"`
	Count    int64  `json:"count"`
}

func (h *DashboardHandler) Get(c echo.Context) error {
	ctx := c.Request().Context()
	userID := middleware.GetUserID(c)

	vulnCounts, err := h.vulns.CountBySeverity(ctx)
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

	stats, err := h.q.DashboardTicketStats(ctx, userID)
	if err != nil {
		stats = queries.DashboardTicketStatsRow{}
	}

	return c.JSON(http.StatusOK, dashboardResponse{
		VulnsBySeverity:   sevCounts,
		MyTickets:         stats.MyTickets,
		UnassignedTickets: stats.UnassignedTickets,
		OpenTicketsTotal:  stats.OpenTicketsTotal,
		ResolvedTickets:   stats.ResolvedTickets,
	})
}

func (h *DashboardHandler) RegisterRoutes(g *echo.Group) {
	g.GET("", h.Get)
}
