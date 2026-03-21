package handler

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/cyberoptic/openvas-tracker/internal/database/queries"
)

type SettingsHandler struct {
	apiKey     string
	serverPort int
	q          *queries.Queries
}

func NewSettingsHandler(apiKey string, serverPort int, q *queries.Queries) *SettingsHandler {
	return &SettingsHandler{apiKey: apiKey, serverPort: serverPort, q: q}
}

func (h *SettingsHandler) GetSetup(c echo.Context) error {
	masked := ""
	if len(h.apiKey) >= 8 {
		masked = h.apiKey[:8] + "..." + h.apiKey[len(h.apiKey)-4:]
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"api_key_masked": masked,
		"server_port":    h.serverPort,
		"webhook_url":    fmt.Sprintf("/api/import/openvas?api_key=<YOUR_API_KEY>"),
		"curl_example": fmt.Sprintf(
			"curl -X POST http://<tracker-host>:%d/api/import/openvas \\\n  -H 'X-API-Key: <YOUR_API_KEY>' \\\n  -H 'Content-Type: application/xml' \\\n  --data-binary @scan-report.xml",
			h.serverPort,
		),
	})
}

func (h *SettingsHandler) ListUsers(c echo.Context) error {
	users, err := h.q.ListUsers(c.Request().Context(), queries.ListUsersParams{Limit: 100, Offset: 0})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list users")
	}
	type userRef struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		Email    string `json:"email"`
	}
	var result []userRef
	for _, u := range users {
		result = append(result, userRef{ID: u.ID, Username: u.Username, Email: u.Email})
	}
	return c.JSON(http.StatusOK, result)
}

func (h *SettingsHandler) RegisterRoutes(g *echo.Group) {
	g.GET("/setup", h.GetSetup)
	g.GET("/users", h.ListUsers)
}
