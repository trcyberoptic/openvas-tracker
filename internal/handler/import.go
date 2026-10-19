package handler

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/exec"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/cyberoptic/openvas-tracker/internal/database/queries"
	"github.com/cyberoptic/openvas-tracker/internal/scanner"
	"github.com/cyberoptic/openvas-tracker/internal/service"
)

type ImportHandler struct {
	importSvc *service.ImportService
	q         *queries.Queries
}

func NewImportHandler(importSvc *service.ImportService, q *queries.Queries) *ImportHandler {
	return &ImportHandler{importSvc: importSvc, q: q}
}

func (h *ImportHandler) HandleOpenVAS(c echo.Context) error {
	results, meta, err := scanner.ParseOpenVASXML(c.Request().Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "failed to parse OpenVAS XML")
	}
	if len(results) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "empty report — no results found")
	}

	res, err := h.importSvc.Import(c.Request().Context(), results, "openvas", meta)
	if errors.Is(err, service.ErrReportAlreadyImported) {
		return echo.NewHTTPError(http.StatusConflict, "report already imported")
	}
	if err != nil {
		log.Printf("import error: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "import failed")
	}

	// Backfill hostnames via PTR for any IPs missing them
	go func() {
		if n, err := h.importSvc.BackfillHostnames(context.Background()); err != nil {
			log.Printf("hostname backfill error: %v", err)
		} else if n > 0 {
			log.Printf("hostname backfill: resolved %d hosts", n)
		}
	}()

	return c.JSON(http.StatusCreated, res)
}

// shouldImportReport decides whether the fetch script should import a GVM report: not when a
// scan already holds it, and not when it ended at or before the newest imported OpenVAS report
// (stale, e.g. a long-idle task's last report on the first run after an upgrade).
func shouldImportReport(known bool, cutoff *time.Time, scanEnd time.Time) (bool, string) {
	if known {
		return false, "already imported"
	}
	if cutoff != nil && !scanEnd.After(*cutoff) {
		return false, "ended before the newest imported report"
	}
	return true, ""
}

// CheckOpenVASReport answers the fetch script's "import this GVM report?" before it downloads
// the (possibly huge) report. scan_start/scan_end come from GMP get_tasks <last_report>.
func (h *ImportHandler) CheckOpenVASReport(c echo.Context) error {
	id, err := uuid.Parse(c.QueryParam("report_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "report_id must be a UUID")
	}
	scanEnd, err := time.Parse(time.RFC3339, c.QueryParam("scan_end"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "scan_end must be RFC3339")
	}
	var scanStart *time.Time
	if t, err := time.Parse(time.RFC3339, c.QueryParam("scan_start")); err == nil {
		scanStart = &t
	}

	ctx := c.Request().Context()
	known, err := h.q.OpenVASReportImported(ctx, id.String(), scanStart)
	if err != nil {
		log.Printf("openvas check: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "query failed")
	}
	cutoff, err := h.q.LatestOpenVASScanEnd(ctx)
	if err != nil {
		log.Printf("openvas check: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "query failed")
	}
	ok, reason := shouldImportReport(known, cutoff, scanEnd)
	return c.JSON(http.StatusOK, map[string]any{"import": ok, "reason": reason})
}

func (h *ImportHandler) TriggerFetch(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 120*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sudo", "/usr/local/bin/openvas-tracker-fetch-latest")
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("fetch-latest error: %v: %s", err, out)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": string(out)})
	}
	return c.JSON(http.StatusOK, map[string]string{"output": string(out)})
}

func (h *ImportHandler) HandleZAP(c echo.Context) error {
	results, err := scanner.ParseZAPJSON(c.Request().Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "failed to parse ZAP JSON")
	}
	if len(results) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "empty report — no results found")
	}

	res, err := h.importSvc.Import(c.Request().Context(), results, "zap", nil)
	if err != nil {
		log.Printf("import error: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "import failed")
	}

	go func() {
		if n, err := h.importSvc.BackfillHostnames(context.Background()); err != nil {
			log.Printf("hostname backfill error: %v", err)
		} else if n > 0 {
			log.Printf("hostname backfill: resolved %d hosts", n)
		}
	}()

	return c.JSON(http.StatusCreated, res)
}

// HandleFeeds ingests a GMP <get_feeds/> response and upserts the feed versions.
func (h *ImportHandler) HandleFeeds(c echo.Context) error {
	feeds, err := scanner.ParseFeeds(c.Request().Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "failed to parse feeds XML")
	}
	updated := 0
	for _, f := range feeds {
		if err := h.q.UpsertFeedStatus(c.Request().Context(), queries.UpsertFeedStatusParams{
			FeedType: f.Type, FeedName: f.Name, Version: f.Version,
		}); err != nil {
			log.Printf("feed upsert error (%s): %v", f.Type, err)
			continue
		}
		updated++
	}
	return c.JSON(http.StatusOK, map[string]int{"feeds_updated": updated})
}

func (h *ImportHandler) RegisterRoutes(g *echo.Group) {
	g.POST("/openvas", h.HandleOpenVAS)
	g.GET("/openvas", h.TriggerFetch)
	g.GET("/openvas/check", h.CheckOpenVASReport)
	g.POST("/zap", h.HandleZAP)
	g.POST("/feeds", h.HandleFeeds)
}
