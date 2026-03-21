package handler

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/cyberoptic/openvas-tracker/internal/database/queries"
	"github.com/cyberoptic/openvas-tracker/internal/middleware"
	"github.com/cyberoptic/openvas-tracker/internal/service"
)

type TicketHandler struct {
	tickets *service.TicketService
	q       *queries.Queries
}

func NewTicketHandler(tickets *service.TicketService, q *queries.Queries) *TicketHandler {
	return &TicketHandler{tickets: tickets, q: q}
}

type createTicketRequest struct {
	Title           string  `json:"title" validate:"required"`
	Description     string  `json:"description"`
	Priority        string  `json:"priority" validate:"required,oneof=critical high medium low"`
	VulnerabilityID *string `json:"vulnerability_id"`
	AssignedTo      *string `json:"assigned_to"`
	DueDate         *string `json:"due_date"`
}

func (h *TicketHandler) Create(c echo.Context) error {
	var req createTicketRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	if err := c.Validate(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	userID := middleware.GetUserID(c)

	params := queries.CreateTicketParams{
		Title:       req.Title,
		Description: &req.Description,
		Priority:    queries.TicketPriority(req.Priority),
		CreatedBy:   userID,
	}
	if req.VulnerabilityID != nil {
		params.VulnerabilityID = req.VulnerabilityID
	}
	if req.AssignedTo != nil {
		params.AssignedTo = req.AssignedTo
	}
	if req.DueDate != nil {
		t, _ := time.Parse(time.RFC3339, *req.DueDate)
		params.DueDate = &t
	}

	ticket, err := h.tickets.Create(c.Request().Context(), params)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create ticket")
	}
	return c.JSON(http.StatusCreated, ticket)
}

func (h *TicketHandler) List(c echo.Context) error {
	limit, offset := paginate(c)
	tickets, err := h.tickets.List(c.Request().Context(), limit, offset)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list tickets")
	}
	return c.JSON(http.StatusOK, tickets)
}

func (h *TicketHandler) Get(c echo.Context) error {
	id := c.Param("id")
	ticket, err := h.tickets.Get(c.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "ticket not found")
	}
	return c.JSON(http.StatusOK, ticket)
}

type updateStatusRequest struct {
	Status            string  `json:"status" validate:"required,oneof=open fixed risk_accepted false_positive"`
	RiskAcceptedUntil *string `json:"risk_accepted_until"` // date string YYYY-MM-DD
}

func (h *TicketHandler) UpdateStatus(c echo.Context) error {
	id := c.Param("id")
	var req updateStatusRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	if err := c.Validate(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// Get current ticket for activity logging
	old, err := h.tickets.Get(c.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "ticket not found")
	}

	ticket, err := h.tickets.UpdateStatus(c.Request().Context(), id, req.Status)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update status")
	}

	// Set risk_accepted_until if accepting risk
	if req.Status == "risk_accepted" && req.RiskAcceptedUntil != nil {
		t, err := time.Parse("2006-01-02", *req.RiskAcceptedUntil)
		if err == nil {
			h.q.SetRiskAcceptedUntil(c.Request().Context(), id, &t)
		}
	}
	// Clear risk_accepted_until if moving away from risk_accepted
	if req.Status != "risk_accepted" {
		h.q.SetRiskAcceptedUntil(c.Request().Context(), id, nil)
	}

	// Log activity
	userID := middleware.GetUserID(c)
	oldStatus := string(old.Status)
	h.q.LogTicketActivity(c.Request().Context(), queries.LogTicketActivityParams{
		ID: uuid.New().String(), TicketID: id, Action: "status_changed",
		OldValue: &oldStatus, NewValue: &req.Status, ChangedBy: userID,
	})

	// Re-fetch to include updated risk_accepted_until
	ticket, _ = h.tickets.Get(c.Request().Context(), id)
	return c.JSON(http.StatusOK, ticket)
}

type assignTicketRequest struct {
	AssignedTo *string `json:"assigned_to"`
}

func (h *TicketHandler) Assign(c echo.Context) error {
	id := c.Param("id")
	var req assignTicketRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	old, err := h.tickets.Get(c.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "ticket not found")
	}

	_, err = h.q.AssignTicket(c.Request().Context(), queries.AssignTicketParams{
		ID: id, AssignedTo: req.AssignedTo,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to assign ticket")
	}

	userID := middleware.GetUserID(c)
	oldVal := "unassigned"
	if old.AssignedTo != nil {
		oldVal = *old.AssignedTo
	}
	newVal := "unassigned"
	if req.AssignedTo != nil {
		newVal = *req.AssignedTo
	}
	h.q.LogTicketActivity(c.Request().Context(), queries.LogTicketActivityParams{
		ID: uuid.New().String(), TicketID: id, Action: "assigned",
		OldValue: &oldVal, NewValue: &newVal, ChangedBy: userID,
	})

	ticket, _ := h.tickets.Get(c.Request().Context(), id)
	return c.JSON(http.StatusOK, ticket)
}

type addCommentRequest struct {
	Content string `json:"content" validate:"required"`
}

func (h *TicketHandler) AddComment(c echo.Context) error {
	id := c.Param("id")
	var req addCommentRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	if err := c.Validate(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	userID := middleware.GetUserID(c)
	comment, err := h.tickets.AddComment(c.Request().Context(), id, userID, req.Content)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to add comment")
	}

	// Log activity
	h.q.LogTicketActivity(c.Request().Context(), queries.LogTicketActivityParams{
		ID: uuid.New().String(), TicketID: id, Action: "comment_added",
		ChangedBy: userID, Note: &req.Content,
	})

	return c.JSON(http.StatusCreated, comment)
}

func (h *TicketHandler) ListComments(c echo.Context) error {
	id := c.Param("id")
	comments, err := h.tickets.ListComments(c.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list comments")
	}
	return c.JSON(http.StatusOK, comments)
}

func (h *TicketHandler) ListActivity(c echo.Context) error {
	id := c.Param("id")
	activity, err := h.q.ListTicketActivity(c.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list activity")
	}
	return c.JSON(http.StatusOK, activity)
}

func (h *TicketHandler) AlsoAffected(c echo.Context) error {
	id := c.Param("id")
	hosts, err := h.q.AlsoAffectedHosts(c.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list affected hosts")
	}
	if hosts == nil {
		hosts = []queries.AlsoAffectedHost{}
	}
	return c.JSON(http.StatusOK, hosts)
}

type bulkUpdateRequest struct {
	TicketIDs  []string `json:"ticket_ids" validate:"required,min=1"`
	Status     *string  `json:"status"`
	AssignedTo *string  `json:"assigned_to"`
}

func (h *TicketHandler) BulkUpdate(c echo.Context) error {
	var req bulkUpdateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	if req.Status == nil && req.AssignedTo == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "must provide status or assigned_to")
	}
	if len(req.TicketIDs) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "no ticket IDs provided")
	}

	userID := middleware.GetUserID(c)
	updated := 0

	for _, tid := range req.TicketIDs {
		if req.Status != nil {
			old, err := h.tickets.Get(c.Request().Context(), tid)
			if err != nil {
				continue
			}
			if _, err := h.tickets.UpdateStatus(c.Request().Context(), tid, *req.Status); err != nil {
				continue
			}
			oldStatus := string(old.Status)
			h.q.LogTicketActivity(c.Request().Context(), queries.LogTicketActivityParams{
				ID: uuid.New().String(), TicketID: tid, Action: "status_changed",
				OldValue: &oldStatus, NewValue: req.Status, ChangedBy: userID,
			})
		}
		if req.AssignedTo != nil {
			h.q.AssignTicket(c.Request().Context(), queries.AssignTicketParams{
				ID: tid, AssignedTo: req.AssignedTo,
			})
			action := "assigned"
			h.q.LogTicketActivity(c.Request().Context(), queries.LogTicketActivityParams{
				ID: uuid.New().String(), TicketID: tid, Action: action,
				NewValue: req.AssignedTo, ChangedBy: userID,
			})
		}
		updated++
	}

	return c.JSON(http.StatusOK, map[string]int{"updated": updated})
}

func (h *TicketHandler) RegisterRoutes(g *echo.Group) {
	g.POST("", h.Create)
	g.GET("", h.List)
	g.GET("/:id", h.Get)
	g.PATCH("/:id/status", h.UpdateStatus)
	g.PATCH("/:id/assign", h.Assign)
	g.POST("/:id/comments", h.AddComment)
	g.GET("/:id/comments", h.ListComments)
	g.GET("/:id/activity", h.ListActivity)
	g.GET("/:id/also-affected", h.AlsoAffected)
	g.POST("/bulk", h.BulkUpdate)
}
