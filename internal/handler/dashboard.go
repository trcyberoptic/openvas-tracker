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
	VulnsBySeverity        []severityCount  `json:"vulns_by_severity"`
	TicketsByScanType      []scanTypeCount  `json:"tickets_by_scan_type"`
	MyTickets              int64            `json:"my_tickets"`
	UnassignedTickets      int64            `json:"unassigned_tickets"`
	OpenTicketsTotal       int64            `json:"open_tickets_total"`
	PendingResolutionTotal int64            `json:"pending_resolution_total"`
	ResolvedTickets        int64            `json:"resolved_tickets"`
}

type scanTypeCount struct {
	ScanType string `json:"scan_type"`
	Count    int64  `json:"count"`
}

type severityCount struct {
	Severity string `json:"severity"`
	Count    int64  `json:"count"`
}

func (h *DashboardHandler) Get(c echo.Context) error {
	ctx := c.Request().Context()
	userID := middleware.GetUserID(c)

	priorityCounts, err := h.q.OpenTicketsByPriority(ctx)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to load dashboard")
	}

	var sevCounts []severityCount
	for _, pc := range priorityCounts {
		sevCounts = append(sevCounts, severityCount{
			Severity: pc.Priority,
			Count:    pc.Count,
		})
	}

	stats, err := h.q.DashboardTicketStats(ctx, userID)
	if err != nil {
		stats = queries.DashboardTicketStatsRow{}
	}

	scanTypeCounts, _ := h.q.OpenTicketsByScanType(ctx)
	var stCounts []scanTypeCount
	for _, st := range scanTypeCounts {
		stCounts = append(stCounts, scanTypeCount{ScanType: st.ScanType, Count: st.Count})
	}

	return c.JSON(http.StatusOK, dashboardResponse{
		VulnsBySeverity:        sevCounts,
		TicketsByScanType:      stCounts,
		MyTickets:              stats.MyTickets,
		UnassignedTickets:      stats.UnassignedTickets,
		OpenTicketsTotal:       stats.OpenTicketsTotal,
		PendingResolutionTotal: stats.PendingResolutionTotal,
		ResolvedTickets:        stats.ResolvedTickets,
	})
}

func (h *DashboardHandler) Trend(c echo.Context) error {
	trend, err := h.q.VulnTrend(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to load trend")
	}
	return c.JSON(http.StatusOK, trend)
}

func (h *DashboardHandler) RegisterRoutes(g *echo.Group) {
	g.GET("", h.Get)
	g.GET("/trend", h.Trend)
}
