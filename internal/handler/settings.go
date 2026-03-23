package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/cyberoptic/openvas-tracker/internal/config"
	"github.com/cyberoptic/openvas-tracker/internal/database/queries"
	"github.com/cyberoptic/openvas-tracker/internal/service"
)

type SettingsHandler struct {
	cfg        *config.Config
	q          *queries.Queries
	envSvc     *service.EnvFileService
	ldapSvc    *service.LDAPService
}

func NewSettingsHandler(cfg *config.Config, q *queries.Queries, envSvc *service.EnvFileService, ldapSvc *service.LDAPService) *SettingsHandler {
	return &SettingsHandler{cfg: cfg, q: q, envSvc: envSvc, ldapSvc: ldapSvc}
}

func (h *SettingsHandler) GetSetup(c echo.Context) error {
	masked := ""
	if len(h.cfg.Import.APIKey) >= 8 {
		masked = h.cfg.Import.APIKey[:8] + "..." + h.cfg.Import.APIKey[len(h.cfg.Import.APIKey)-4:]
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"api_key_masked": masked,
		"server_port":    h.cfg.Server.Port,
		"webhook_url":    fmt.Sprintf("/api/import/openvas?api_key=<YOUR_API_KEY>"),
		"curl_example": fmt.Sprintf(
			"curl -X POST http://<tracker-host>:%d/api/import/openvas \\\n  -H 'X-API-Key: <YOUR_API_KEY>' \\\n  -H 'Content-Type: application/xml' \\\n  --data-binary @scan-report.xml",
			h.cfg.Server.Port,
		),
		"ldap_enabled": h.cfg.LDAP.Enabled(),
	})
}

func (h *SettingsHandler) ListUsers(c echo.Context) error {
	type userRef struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		Email    string `json:"email"`
		Source   string `json:"source"` // "local" or "ldap"
	}

	var result []userRef

	// Local DB users
	users, err := h.q.ListUsers(c.Request().Context(), queries.ListUsersParams{Limit: 100, Offset: 0})
	if err == nil {
		for _, u := range users {
			if u.Username == "openvas-import" {
				continue
			}
			result = append(result, userRef{ID: u.ID, Username: u.Username, Email: u.Email, Source: "local"})
		}
	}

	// LDAP group members (if configured) — auto-create DB users so they
	// have a real UUID that satisfies the tickets.assigned_to FK.
	ldapCfg := h.currentLDAPConfig()
	if ldapCfg.Enabled() {
		members, err := h.ldapSvc.ListGroupMembers(ldapCfg)
		if err == nil {
			localNames := make(map[string]bool)
			for _, u := range result {
				localNames[u.Username] = true
			}
			for _, m := range members {
				if localNames[m.Username] || m.Email == "" {
					continue
				}
				// Auto-create a DB user so the UUID works for ticket assignment
				user, err := h.q.CreateUser(c.Request().Context(), queries.CreateUserParams{
					ID: uuid.New().String(), Email: m.Email, Username: m.Username,
					Password: "-", Role: queries.UserRoleViewer,
				})
				if err != nil {
					// Already exists (race) — fetch
					user, err = h.q.GetUserByUsername(c.Request().Context(), m.Username)
					if err != nil {
						continue
					}
				}
				result = append(result, userRef{ID: user.ID, Username: user.Username, Email: user.Email, Source: "ldap"})
			}
		}
	}

	return c.JSON(http.StatusOK, result)
}

// GetEnvConfig returns the current .env file contents (sensitive values masked).
func (h *SettingsHandler) GetEnvConfig(c echo.Context) error {
	vals, err := h.envSvc.Read()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to read config")
	}

	// Mask sensitive values
	sensitive := map[string]bool{
		"OT_JWT_SECRET": true, "OT_IMPORT_APIKEY": true,
		"OT_ADMIN_PASSWORD": true, "OT_LDAP_BIND_PASSWORD": true, "OT_DATABASE_DSN": true,
	}
	masked := make(map[string]string)
	for k, v := range vals {
		if sensitive[k] && len(v) > 4 {
			masked[k] = v[:4] + "********"
		} else {
			masked[k] = v
		}
	}

	return c.JSON(http.StatusOK, masked)
}

type updateEnvRequest struct {
	Key   string `json:"key" validate:"required"`
	Value string `json:"value"`
}

// UpdateEnvConfig updates a single key in the .env file.
func (h *SettingsHandler) UpdateEnvConfig(c echo.Context) error {
	var req updateEnvRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	if err := h.envSvc.Update(req.Key, req.Value); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update config")
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok", "note": "restart required for changes to take effect"})
}

type updateEnvBatchRequest struct {
	Values map[string]string `json:"values"`
}

// UpdateEnvBatch updates multiple keys in the .env file at once.
func (h *SettingsHandler) UpdateEnvBatch(c echo.Context) error {
	var req updateEnvBatchRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	if err := h.envSvc.UpdateMultiple(req.Values); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update config")
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok", "note": "restart required for changes to take effect"})
}

// TestLDAP tests the current LDAP configuration.
func (h *SettingsHandler) TestLDAP(c echo.Context) error {
	ldapCfg := h.currentLDAPConfig()
	if !ldapCfg.Enabled() {
		return c.JSON(http.StatusOK, map[string]string{"status": "not_configured"})
	}

	if err := h.ldapSvc.TestConnection(ldapCfg); err != nil {
		return c.JSON(http.StatusOK, map[string]string{"status": "error", "error": err.Error()})
	}

	members, err := h.ldapSvc.ListGroupMembers(ldapCfg)
	memberCount := 0
	if err == nil {
		memberCount = len(members)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":        "ok",
		"group_members": memberCount,
	})
}

func (h *SettingsHandler) currentLDAPConfig() config.LDAPConfig {
	cfg, err := config.Load()
	if err != nil {
		return h.cfg.LDAP
	}
	return cfg.LDAP
}

func (h *SettingsHandler) ListRiskRules(c echo.Context) error {
	rules, err := h.q.ListRiskAcceptRules(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list rules")
	}
	if rules == nil {
		rules = []queries.RiskAcceptRule{}
	}
	return c.JSON(http.StatusOK, rules)
}

func (h *SettingsHandler) DeleteRiskRule(c echo.Context) error {
	id := c.Param("id")
	if err := h.q.DeleteRiskAcceptRule(c.Request().Context(), id); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete rule")
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}

// ApplyRiskRules re-applies all active rules to existing open tickets.
func (h *SettingsHandler) ApplyRiskRules(c echo.Context) error {
	ctx := c.Request().Context()
	rules, err := h.q.ListRiskAcceptRules(ctx)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list rules")
	}

	total := 0
	for _, rule := range rules {
		if rule.ExpiresAt != nil && rule.ExpiresAt.Before(time.Now()) {
			continue
		}
		affected, err := h.q.ApplyRuleToExistingTickets(ctx, rule.Fingerprint, rule.HostPattern, rule.ExpiresAt)
		if err != nil {
			continue
		}
		for _, tid := range affected {
			newStatus := "risk_accepted"
			note := fmt.Sprintf("Risk accepted via rule refresh: %s", rule.Reason)
			h.q.LogTicketActivity(ctx, queries.LogTicketActivityParams{
				ID: uuid.New().String(), TicketID: tid, Action: "status_changed",
				OldValue: strPtr("open"), NewValue: &newStatus, ChangedBy: "Automatic", Note: &note,
			})
		}
		total += len(affected)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":           "ok",
		"tickets_affected": total,
	})
}

func strPtr(s string) *string { return &s }

func (h *SettingsHandler) RegisterRoutes(g *echo.Group) {
	g.GET("/setup", h.GetSetup)
	g.GET("/users", h.ListUsers)
	g.GET("/env", h.GetEnvConfig)
	g.PUT("/env", h.UpdateEnvConfig)
	g.PUT("/env/batch", h.UpdateEnvBatch)
	g.POST("/ldap/test", h.TestLDAP)
	g.GET("/risk-rules", h.ListRiskRules)
	g.DELETE("/risk-rules/:id", h.DeleteRiskRule)
	g.POST("/risk-rules/apply", h.ApplyRiskRules)
}
