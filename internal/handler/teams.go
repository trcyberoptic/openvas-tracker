// internal/handler/teams.go
package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/cyberoptic/openvas-tracker/internal/middleware"
	"github.com/cyberoptic/openvas-tracker/internal/service"
)

type TeamHandler struct {
	teams *service.TeamService
}

func NewTeamHandler(teams *service.TeamService) *TeamHandler {
	return &TeamHandler{teams: teams}
}

type createTeamRequest struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
}

func (h *TeamHandler) Create(c echo.Context) error {
	var req createTeamRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	userID := middleware.GetUserID(c)
	team, err := h.teams.Create(c.Request().Context(), req.Name, req.Description, userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create team")
	}
	return c.JSON(http.StatusCreated, team)
}

func (h *TeamHandler) List(c echo.Context) error {
	userID := middleware.GetUserID(c)
	teams, err := h.teams.ListByUser(c.Request().Context(), userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list teams")
	}
	return c.JSON(http.StatusOK, teams)
}

func (h *TeamHandler) Get(c echo.Context) error {
	id := c.Param("id")
	team, err := h.teams.Get(c.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "team not found")
	}
	return c.JSON(http.StatusOK, team)
}

func (h *TeamHandler) Members(c echo.Context) error {
	id := c.Param("id")
	members, err := h.teams.ListMembers(c.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list members")
	}
	return c.JSON(http.StatusOK, members)
}

type addMemberRequest struct {
	UserID string `json:"user_id" validate:"required,uuid"`
	Role   string `json:"role" validate:"required,oneof=admin member"`
}

func (h *TeamHandler) AddMember(c echo.Context) error {
	id := c.Param("id")
	var req addMemberRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	if err := h.teams.AddMember(c.Request().Context(), id, req.UserID, req.Role); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to add member")
	}
	return c.NoContent(http.StatusNoContent)
}

type inviteRequest struct {
	Email string `json:"email" validate:"required,email"`
}

func (h *TeamHandler) Invite(c echo.Context) error {
	id := c.Param("id")
	var req inviteRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	userID := middleware.GetUserID(c)
	inv, err := h.teams.Invite(c.Request().Context(), id, req.Email, userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create invitation")
	}
	return c.JSON(http.StatusCreated, inv)
}

func (h *TeamHandler) Delete(c echo.Context) error {
	id := c.Param("id")
	if err := h.teams.Delete(c.Request().Context(), id); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete team")
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *TeamHandler) RegisterRoutes(g *echo.Group) {
	g.POST("", h.Create)
	g.GET("", h.List)
	g.GET("/:id", h.Get)
	g.GET("/:id/members", h.Members)
	g.POST("/:id/members", h.AddMember)
	g.POST("/:id/invite", h.Invite)
	g.DELETE("/:id", h.Delete)
}
