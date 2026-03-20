package handler

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/cyberoptic/openvas-tracker/internal/middleware"
)

type SettingsHandler struct {
	apiKey     string
	serverPort int
}

func NewSettingsHandler(apiKey string, serverPort int) *SettingsHandler {
	return &SettingsHandler{apiKey: apiKey, serverPort: serverPort}
}

func (h *SettingsHandler) GetSetup(c echo.Context) error {
	role := middleware.GetUserRole(c)
	if role != "admin" {
		return echo.NewHTTPError(http.StatusForbidden, "admin only")
	}

	masked := ""
	if len(h.apiKey) >= 8 {
		masked = h.apiKey[:8] + "..." + h.apiKey[len(h.apiKey)-4:]
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"api_key":        h.apiKey,
		"api_key_masked": masked,
		"server_port":    h.serverPort,
		"webhook_url":    fmt.Sprintf("/api/import/openvas?api_key=%s", h.apiKey),
		"curl_example": fmt.Sprintf(
			"curl -X POST http://<tracker-host>:%d/api/import/openvas \\\n  -H 'X-API-Key: %s' \\\n  -H 'Content-Type: application/xml' \\\n  --data-binary @scan-report.xml",
			h.serverPort, h.apiKey,
		),
	})
}

func (h *SettingsHandler) RegisterRoutes(g *echo.Group) {
	g.GET("/setup", h.GetSetup)
}
