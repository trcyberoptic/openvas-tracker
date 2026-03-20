// internal/handler/tickets.go
package handler

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/cyberoptic/openvas-tracker/internal/database/queries"
	"github.com/cyberoptic/openvas-tracker/internal/middleware"
	"github.com/cyberoptic/openvas-tracker/internal/service"
)

type TicketHandler struct {
	tickets *service.TicketService
}

func NewTicketHandler(tickets *service.TicketService) *TicketHandler {
	return &TicketHandler{tickets: tickets}
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
	userID := middleware.GetUserID(c)
	tickets, err := h.tickets.List(c.Request().Context(), userID, 100, 0)
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

type addCommentRequest struct {
	Content string `json:"content" validate:"required"`
}

func (h *TicketHandler) AddComment(c echo.Context) error {
	id := c.Param("id")
	var req addCommentRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	userID := middleware.GetUserID(c)
	comment, err := h.tickets.AddComment(c.Request().Context(), id, userID, req.Content)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to add comment")
	}
	return c.JSON(http.StatusCreated, comment)
}

func (h *TicketHandler) RegisterRoutes(g *echo.Group) {
	g.POST("", h.Create)
	g.GET("", h.List)
	g.GET("/:id", h.Get)
	g.POST("/:id/comments", h.AddComment)
}
