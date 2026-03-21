// internal/handler/ws.go
package handler

import (
	"net/http"

	gws "github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"

	"github.com/cyberoptic/openvas-tracker/internal/auth"
	"github.com/cyberoptic/openvas-tracker/internal/websocket"
)

var upgrader = gws.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true
		}
		return origin == "http://"+r.Host || origin == "https://"+r.Host
	},
}

type WSHandler struct {
	hub       *websocket.Hub
	jwtSecret string
}

func NewWSHandler(hub *websocket.Hub, jwtSecret string) *WSHandler {
	return &WSHandler{hub: hub, jwtSecret: jwtSecret}
}

func (h *WSHandler) Handle(c echo.Context) error {
	token := c.QueryParam("token")
	if token == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "missing token")
	}
	claims, err := auth.ValidateToken(token, h.jwtSecret)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
	}

	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}

	userID := claims.UserID
	client := websocket.NewClient(conn)
	h.hub.Register(userID, client)

	go client.WritePump()
	go client.ReadPump(func() {
		h.hub.Unregister(userID, client)
	})

	return nil
}
