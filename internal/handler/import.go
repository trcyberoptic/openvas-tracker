package handler

import (
	"context"
	"log"
	"net/http"
	"os/exec"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/cyberoptic/openvas-tracker/internal/scanner"
	"github.com/cyberoptic/openvas-tracker/internal/service"
)

type ImportHandler struct {
	importSvc *service.ImportService
}

func NewImportHandler(importSvc *service.ImportService) *ImportHandler {
	return &ImportHandler{importSvc: importSvc}
}

func (h *ImportHandler) HandleOpenVAS(c echo.Context) error {
	results, err := scanner.ParseOpenVASXML(c.Request().Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "failed to parse OpenVAS XML")
	}
	if len(results) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "empty report — no results found")
	}

	res, err := h.importSvc.Import(c.Request().Context(), results)
	if err != nil {
		log.Printf("import error: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "import failed")
	}

	return c.JSON(http.StatusCreated, res)
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

func (h *ImportHandler) RegisterRoutes(g *echo.Group) {
	g.POST("/openvas", h.HandleOpenVAS)
	g.GET("/openvas", h.TriggerFetch)
}
