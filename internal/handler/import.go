package handler

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
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

	scanID := uuid.New().String()
	now := time.Now()
	scan, err := h.q.CreateScan(c.Request().Context(), queries.CreateScanParams{
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
	h.q.UpdateScanStatus(c.Request().Context(), queries.UpdateScanStatusParams{
		ID:          scan.ID,
		Status:      queries.ScanStatusCompleted,
		StartedAt:   &now,
		CompletedAt: &now,
		RawOutput:   &rawXML,
	})

	imported := 0
	for _, r := range results {
		port, proto := parsePort(r.Port)
		severity := mapSeverity(r.Severity, r.CVSSScore)

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

		_, err := h.q.CreateVulnerability(c.Request().Context(), queries.CreateVulnerabilityParams{
			ID:             uuid.New().String(),
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
			continue
		}
		imported++
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"scan_id":                  scan.ID,
		"vulnerabilities_imported": imported,
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

func (h *ImportHandler) RegisterRoutes(g *echo.Group) {
	g.POST("/openvas", h.HandleOpenVAS)
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
