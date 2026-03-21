// internal/handler/notifications.go
package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/cyberoptic/openvas-tracker/internal/middleware"
	"github.com/cyberoptic/openvas-tracker/internal/service"
)

type NotificationHandler struct {
	notifications *service.NotificationService
}

func NewNotificationHandler(n *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{notifications: n}
}

func (h *NotificationHandler) List(c echo.Context) error {
	userID := middleware.GetUserID(c)
	limit, offset := paginate(c)
	notifs, err := h.notifications.List(c.Request().Context(), userID, limit, offset)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list notifications")
	}
	return c.JSON(http.StatusOK, notifs)
}

func (h *NotificationHandler) Unread(c echo.Context) error {
	userID := middleware.GetUserID(c)
	count, err := h.notifications.CountUnread(c.Request().Context(), userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to count unread")
	}
	return c.JSON(http.StatusOK, map[string]int64{"unread": count})
}

func (h *NotificationHandler) MarkRead(c echo.Context) error {
	id := c.Param("id")
	userID := middleware.GetUserID(c)
	if err := h.notifications.MarkRead(c.Request().Context(), id, userID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to mark read")
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *NotificationHandler) MarkAllRead(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if err := h.notifications.MarkAllRead(c.Request().Context(), userID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to mark all read")
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *NotificationHandler) RegisterRoutes(g *echo.Group) {
	g.GET("", h.List)
	g.GET("/unread", h.Unread)
	g.PUT("/:id/read", h.MarkRead)
	g.PUT("/read-all", h.MarkAllRead)
}
