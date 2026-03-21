package handler

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os/exec"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/cyberoptic/openvas-tracker/internal/auth"
	"github.com/cyberoptic/openvas-tracker/internal/database/queries"
	"github.com/cyberoptic/openvas-tracker/internal/scanner"
)

type ImportHandler struct {
	q            *queries.Queries
	systemUserID string
	once         sync.Once
	onceErr      error
}

func NewImportHandler(db *sql.DB) *ImportHandler {
	return &ImportHandler{q: queries.New(db)}
}

func (h *ImportHandler) HandleOpenVAS(c echo.Context) error {
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "failed to read request body")
	}

	results, err := scanner.ParseOpenVASXML(strings.NewReader(string(body)))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "failed to parse OpenVAS XML")
	}
	if len(results) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "empty report — no results found")
	}

	if err := h.resolveSystemUser(c.Request().Context()); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to resolve system user")
	}

	ctx := c.Request().Context()
	now := time.Now()
	scanID := uuid.New().String()

	scan, err := h.q.CreateScan(ctx, queries.CreateScanParams{
		ID:       scanID,
		Name:     fmt.Sprintf("OpenVAS Import %s", now.Format("2006-01-02 15:04:05")),
		ScanType: queries.ScanTypeOpenvas,
		Status:   queries.ScanStatusCompleted,
		UserID:   h.systemUserID,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create scan record")
	}

	rawXML := string(body)
	if _, err := h.q.UpdateScanStatus(ctx, queries.UpdateScanStatusParams{
		ID:          scan.ID,
		Status:      queries.ScanStatusCompleted,
		StartedAt:   &now,
		CompletedAt: &now,
		RawOutput:   &rawXML,
	}); err != nil {
		log.Printf("import: failed to update scan status: %v", err)
	}

	imported := 0
	skipped := 0
	ticketsCreated := 0
	ticketsReopened := 0

	for _, r := range results {
		port, proto := parsePort(r.Port)
		severity := mapSeverity(r.Severity, r.CVSSScore)

		// Skip info-level findings with CVSS 0 — not actionable
		if severity == "info" && r.CVSSScore == 0 {
			skipped++
			continue
		}

		var cveID *string
		if r.CVE != "" && r.CVE != "NOCVE" {
			cveID = &r.CVE
		}
		var desc *string
		if r.Description != "" {
			desc = &r.Description
		}
		var sol *string
		if r.Solution != "" {
			sol = &r.Solution
		}
		var host *string
		if r.Host != "" {
			host = &r.Host
		}
		var cvss *float64
		if r.CVSSScore > 0 {
			cvss = &r.CVSSScore
		}

		vulnID := uuid.New().String()
		_, err := h.q.CreateVulnerability(ctx, queries.CreateVulnerabilityParams{
			ID:             vulnID,
			ScanID:         scan.ID,
			UserID:         h.systemUserID,
			Title:          r.Title,
			Description:    desc,
			Severity:       queries.SeverityLevel(severity),
			CvssScore:      cvss,
			CveID:          cveID,
			AffectedHost:   host,
			AffectedPort:   port,
			Protocol:       proto,
			Solution:       sol,
			VulnReferences: []byte("[]"),
		})
		if err != nil {
			log.Printf("import: failed to create vuln %q for host %s: %v", r.Title, r.Host, err)
			skipped++
			continue
		}
		imported++

		// Auto-ticket logic
		created, reopened := h.processTicket(ctx, r, vulnID, severity, now)
		if created {
			ticketsCreated++
		}
		if reopened {
			ticketsReopened++
		}
	}

	// Case 4: reopen expired risk_accepted tickets
	h.reopenExpiredRiskAccepted(ctx)

	// Case 5: auto-resolve open tickets whose findings are NOT in this scan
	autoResolved := h.autoResolveStale(ctx, scan.ID)

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"scan_id":                  scan.ID,
		"vulnerabilities_imported": imported,
		"vulnerabilities_skipped":  skipped,
		"tickets_created":          ticketsCreated,
		"tickets_reopened":         ticketsReopened,
		"tickets_auto_resolved":    autoResolved,
	})
}

func (h *ImportHandler) processTicket(ctx context.Context, r scanner.OpenVASResult, vulnID, severity string, now time.Time) (created, reopened bool) {
	if r.Host == "" {
		return false, false
	}

	existing, err := h.q.FindTicketByFingerprint(ctx, r.Host, r.CVE, r.Title)
	if err != nil {
		// No existing ticket → create new one (Case 1)
		return h.createTicket(ctx, r, vulnID, severity, now), false
	}

	oldStatus := string(existing.Status)

	switch existing.Status {
	case queries.TicketStatusFalsePositive:
		// Never reopen false positives — skip silently
		return false, false

	case queries.TicketStatusFixed, queries.TicketStatusRiskAccepted:
		// Case 2: reopen
		err := h.q.ReopenTicket(ctx, queries.ReopenTicketParams{
			ID: existing.ID, VulnerabilityID: vulnID,
		})
		if err != nil {
			return false, false
		}
		newStatus := "open"
		note := fmt.Sprintf("Finding reappeared in scan — reopened. CVE: %s, Host: %s", r.CVE, r.Host)
		h.logActivity(ctx, existing.ID, "status_changed", &oldStatus, &newStatus, "Automatic", &note)
		return false, true

	default:
		// Case 3: ticket still open — touch last_seen_at
		h.q.TouchTicket(ctx, queries.TouchTicketParams{
			ID: existing.ID, VulnerabilityID: vulnID,
		})
		note := fmt.Sprintf("Finding still present in scan. CVE: %s, Host: %s", r.CVE, r.Host)
		h.logActivity(ctx, existing.ID, "still_present", nil, nil, "Automatic", &note)
		return false, false
	}
}

func (h *ImportHandler) createTicket(ctx context.Context, r scanner.OpenVASResult, vulnID, severity string, now time.Time) bool {
	priority := mapSeverityToPriority(severity)
	title := fmt.Sprintf("[%s] %s — %s", strings.ToUpper(severity), r.Title, r.Host)
	var desc *string
	if r.Description != "" {
		d := fmt.Sprintf("%s\n\nSolution: %s", r.Description, r.Solution)
		desc = &d
	}

	ticketID := uuid.New().String()
	_, err := h.q.CreateTicket(ctx, queries.CreateTicketParams{
		ID:              ticketID,
		Title:           title,
		Description:     desc,
		Priority:        queries.TicketPriority(priority),
		VulnerabilityID: &vulnID,
		CreatedBy:       h.systemUserID,
	})
	if err != nil {
		return false
	}

	// Set first_seen_at and last_seen_at
	h.q.TouchTicket(ctx, queries.TouchTicketParams{ID: ticketID, VulnerabilityID: vulnID})

	newStatus := "open"
	note := fmt.Sprintf("Ticket created from OpenVAS import. CVE: %s, Host: %s, CVSS: %.1f", r.CVE, r.Host, r.CVSSScore)
	h.logActivity(ctx, ticketID, "created", nil, &newStatus, "Automatic", &note)
	return true
}

func (h *ImportHandler) reopenExpiredRiskAccepted(ctx context.Context) {
	reopened, err := h.q.ReopenExpiredRiskAccepted(ctx)
	if err != nil {
		log.Printf("reopen expired risk_accepted error: %v", err)
		return
	}
	for _, t := range reopened {
		oldStatus := "risk_accepted"
		newStatus := "open"
		note := "Risk acceptance expired — ticket reopened"
		h.logActivity(ctx, t.ID, "status_changed", &oldStatus, &newStatus, "Automatic", &note)
	}
}

func (h *ImportHandler) autoResolveStale(ctx context.Context, scanID string) int {
	resolved, err := h.q.AutoResolveStaleTickets(ctx, scanID)
	if err != nil {
		log.Printf("auto-resolve error: %v", err)
		return 0
	}
	for _, t := range resolved {
		oldStatus := "open"
		newStatus := "fixed"
		note := "Finding not present in latest scan — auto-fixed"
		h.logActivity(ctx, t.ID, "status_changed", &oldStatus, &newStatus, "Automatic", &note)
	}
	return len(resolved)
}

func (h *ImportHandler) logActivity(ctx context.Context, ticketID, action string, oldVal, newVal *string, changedBy string, note *string) {
	h.q.LogTicketActivity(ctx, queries.LogTicketActivityParams{
		ID:        uuid.New().String(),
		TicketID:  ticketID,
		Action:    action,
		OldValue:  oldVal,
		NewValue:  newVal,
		ChangedBy: changedBy,
		Note:      note,
	})
}

func (h *ImportHandler) resolveSystemUser(ctx context.Context) error {
	h.once.Do(func() {
		user, err := h.q.GetUserByUsername(ctx, "openvas-import")
		if err == nil {
			h.systemUserID = user.ID
			return
		}
		randBytes := make([]byte, 32)
		rand.Read(randBytes)
		password := hex.EncodeToString(randBytes)
		hash, err := auth.HashPassword(password)
		if err != nil {
			h.onceErr = fmt.Errorf("failed to hash password: %w", err)
			return
		}
		user, err = h.q.CreateUser(ctx, queries.CreateUserParams{
			ID:       uuid.New().String(),
			Email:    "openvas-import@system.local",
			Username: "openvas-import",
			Password: hash,
			Role:     queries.UserRoleViewer,
		})
		if err != nil {
			user, err = h.q.GetUserByUsername(ctx, "openvas-import")
			if err != nil {
				h.onceErr = fmt.Errorf("failed to resolve system user: %w", err)
				return
			}
		}
		h.systemUserID = user.ID
	})
	return h.onceErr
}

func (h *ImportHandler) TriggerFetch(c echo.Context) error {
	cmd := exec.Command("sudo", "/usr/local/bin/openvas-tracker-fetch-latest")
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("fetch-latest error: %v: %s", err, out)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": string(out)})
	}
	return c.JSON(http.StatusOK, map[string]string{"output": string(out)})
}

func (h *ImportHandler) RegisterRoutes(g *echo.Group) {
	g.POST("/openvas", h.HandleOpenVAS)
	g.GET("/openvas", h.TriggerFetch)
}

func mapSeverity(threat string, cvss float64) string {
	switch strings.ToLower(threat) {
	case "high":
		if cvss >= 9.0 {
			return "critical"
		}
		return "high"
	case "medium":
		return "medium"
	case "low":
		return "low"
	default:
		return "info"
	}
}

func mapSeverityToPriority(severity string) string {
	switch severity {
	case "critical":
		return "critical"
	case "high":
		return "high"
	case "medium":
		return "medium"
	default:
		return "low"
	}
}

func parsePort(portStr string) (*int32, *string) {
	if portStr == "" {
		return nil, nil
	}
	parts := strings.SplitN(portStr, "/", 2)
	if len(parts) != 2 {
		return nil, nil
	}
	num, err := strconv.ParseInt(parts[0], 10, 32)
	if err != nil {
		return nil, nil
	}
	port := int32(num)
	proto := parts[1]
	return &port, &proto
}
