// cmd/openvas-tracker/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"

	"github.com/cyberoptic/openvas-tracker/internal/config"
	"github.com/cyberoptic/openvas-tracker/internal/database"
	"github.com/cyberoptic/openvas-tracker/internal/database/queries"
	"github.com/cyberoptic/openvas-tracker/internal/handler"
	mw "github.com/cyberoptic/openvas-tracker/internal/middleware"
	"github.com/cyberoptic/openvas-tracker/internal/service"
	"github.com/cyberoptic/openvas-tracker/internal/websocket"
)

type customValidator struct {
	validator *validator.Validate
}

func (cv *customValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	db, err := database.NewPool(cfg.Database.DSN)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()

	// Services
	userSvc := service.NewUserService(db)
	targetSvc := service.NewTargetService(db)
	vulnSvc := service.NewVulnerabilityService(db)
	ticketSvc := service.NewTicketService(db)
	reportSvc := service.NewReportService(db, vulnSvc)
	teamSvc := service.NewTeamService(db)
	notifSvc := service.NewNotificationService(db)
	assetSvc := service.NewAssetService(db)
	auditSvc := service.NewAuditService(db)
	searchSvc := service.NewSearchService(db)

	// WebSocket hub (no background goroutine needed — hub is mutex-based)
	hub := websocket.NewHub()

	// Echo
	e := echo.New()
	e.HideBanner = true
	e.Validator = &customValidator{validator: validator.New()}

	// Global middleware
	e.Use(echomw.Logger())
	e.Use(echomw.Recover())
	e.Use(mw.SecurityHeaders())
	rl := mw.NewRateLimiter(100, time.Minute)
	e.Use(rl.Middleware())

	// Health
	e.GET("/api/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	// Auth routes (public)
	jwtExpiry := time.Duration(cfg.JWT.ExpireHours) * time.Hour
	authH := handler.NewAuthHandler(userSvc, cfg.JWT.Secret, jwtExpiry)
	authH.RegisterRoutes(e.Group("/api/auth"))

	// Protected routes
	p := e.Group("/api", mw.JWTAuth(cfg.JWT.Secret))

	handler.NewTargetHandler(targetSvc).RegisterRoutes(p.Group("/targets"))

	q := queries.New(db)
	handler.NewScanHandler(q, vulnSvc).RegisterRoutes(p.Group("/scans"))

	handler.NewHostHandler(q).RegisterRoutes(p.Group("/hosts"))
	handler.NewVulnHandler(vulnSvc).RegisterRoutes(p.Group("/vulnerabilities"))
	handler.NewTicketHandler(ticketSvc, q).RegisterRoutes(p.Group("/tickets"))
	handler.NewReportHandler(reportSvc).RegisterRoutes(p.Group("/reports"))
	handler.NewDashboardHandler(vulnSvc, ticketSvc, q).RegisterRoutes(p.Group("/dashboard"))
	handler.NewTeamHandler(teamSvc).RegisterRoutes(p.Group("/teams"))
	handler.NewNotificationHandler(notifSvc).RegisterRoutes(p.Group("/notifications"))
	handler.NewAssetHandler(assetSvc).RegisterRoutes(p.Group("/assets"))
	handler.NewAuditHandler(auditSvc).RegisterRoutes(p.Group("/audit"))
	handler.NewSearchHandler(searchSvc).RegisterRoutes(p.Group("/search"))

	// WebSocket
	wsH := handler.NewWSHandler(hub, cfg.JWT.Secret)
	e.GET("/ws", wsH.Handle)

	// OpenVAS import webhook (API-Key auth, outside JWT group)
	if cfg.Import.APIKey != "" {
		if len(cfg.Import.APIKey) < 32 {
			log.Fatal("OT_IMPORT_APIKEY must be at least 32 characters")
		}
		importG := e.Group("/api/import", mw.APIKeyAuth(cfg.Import.APIKey), echomw.BodyLimit("10M"))
		handler.NewImportHandler(db).RegisterRoutes(importG)
	}

	// Embedded frontend (catch-all, must be last)
	serveFrontend(e)

	// Start HTTP server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	go func() {
		if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	log.Println("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	e.Shutdown(ctx)
}
