// cmd/openvas-tracker/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"

	"github.com/cyberoptic/openvas-tracker/internal/config"
	"github.com/cyberoptic/openvas-tracker/internal/database"
	"github.com/cyberoptic/openvas-tracker/internal/database/queries"
	"github.com/cyberoptic/openvas-tracker/sql/migrations"
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

	// Enforce JWT secret
	if cfg.JWT.Secret == "change-me-in-production" || len(cfg.JWT.Secret) < 32 {
		log.Fatal("OT_JWT_SECRET must be set to a random string of at least 32 characters")
	}

	db, err := database.NewPool(database.PoolConfig{
		DSN:      cfg.Database.DSN,
		MaxConns: cfg.Database.MaxConns,
		MinConns: cfg.Database.MinConns,
	})
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()

	// Auto-apply pending database migrations
	if err := database.AutoMigrate(db, migrations.FS); err != nil {
		log.Fatalf("migrations: %v", err)
	}

	// Services
	userSvc := service.NewUserService(db)
	ldapSvc := service.NewLDAPService()
	envPath := ".env"
	if p := os.Getenv("OT_ENV_FILE"); p != "" {
		envPath = p
	} else if _, err := os.Stat("/etc/openvas-tracker/env"); err == nil {
		envPath = "/etc/openvas-tracker/env"
	}
	envSvc := service.NewEnvFileService(envPath)
	targetSvc := service.NewTargetService(db)
	vulnSvc := service.NewVulnerabilityService(db)
	ticketSvc := service.NewTicketService(db)
	reportSvc := service.NewReportService(db, vulnSvc)
	teamSvc := service.NewTeamService(db)
	notifSvc := service.NewNotificationService(db)
	assetSvc := service.NewAssetService(db)
	auditSvc := service.NewAuditService(db)
	searchSvc := service.NewSearchService(db)

	// WebSocket hub
	hub := websocket.NewHub()

	// Echo
	e := echo.New()
	e.HideBanner = true
	e.Validator = &customValidator{validator: validator.New()}

	// Global middleware
	e.Use(echomw.Logger())
	e.Use(echomw.Recover())
	e.Use(mw.SecurityHeaders())
	// Global body limit — import endpoint uses a skipper to allow larger uploads
	e.Use(echomw.BodyLimitWithConfig(echomw.BodyLimitConfig{
		Limit:   "5M",
		Skipper: func(c echo.Context) bool { return strings.HasPrefix(c.Path(), "/api/import") },
	}))
	rl := mw.NewRateLimiter(500, time.Minute)
	e.Use(rl.Middleware())

	// Health
	e.GET("/api/health", func(c echo.Context) error {
		status := "ok"
		checks := map[string]string{"database": "ok"}
		code := http.StatusOK

		if err := db.PingContext(c.Request().Context()); err != nil {
			checks["database"] = err.Error()
			status = "degraded"
			code = http.StatusServiceUnavailable
		}

		return c.JSON(code, map[string]interface{}{"status": status, "checks": checks})
	})

	// Auth routes (public) with dedicated rate limiter
	authLimiter := mw.NewRateLimiter(60, time.Minute)
	jwtExpiry := time.Duration(cfg.JWT.ExpireHours) * time.Hour
	q := queries.New(db)
	authH := handler.NewAuthHandler(userSvc, ldapSvc, cfg, q, cfg.JWT.Secret, jwtExpiry)
	authH.RegisterRoutes(e.Group("/api/auth", authLimiter.Middleware()))

	// Protected routes
	p := e.Group("/api", mw.JWTAuth(cfg.JWT.Secret))

	handler.NewTargetHandler(targetSvc).RegisterRoutes(p.Group("/targets"))
	handler.NewScanHandler(q, vulnSvc).RegisterRoutes(p.Group("/scans"))

	handler.NewHostHandler(q).RegisterRoutes(p.Group("/hosts"))
	handler.NewVulnHandler(vulnSvc, q).RegisterRoutes(p.Group("/vulnerabilities"))

	// Ticket routes — status/assign require admin or analyst role
	ticketH := handler.NewTicketHandler(ticketSvc, q)
	ticketG := p.Group("/tickets")
	ticketH.RegisterRoutes(ticketG)

	handler.NewReportHandler(reportSvc).RegisterRoutes(p.Group("/reports"))
	handler.NewDashboardHandler(vulnSvc, ticketSvc, q).RegisterRoutes(p.Group("/dashboard"))
	handler.NewTeamHandler(teamSvc).RegisterRoutes(p.Group("/teams"))
	handler.NewNotificationHandler(notifSvc).RegisterRoutes(p.Group("/notifications"))
	handler.NewAssetHandler(assetSvc).RegisterRoutes(p.Group("/assets"))
	handler.NewAuditHandler(auditSvc).RegisterRoutes(p.Group("/audit"))
	handler.NewSearchHandler(searchSvc).RegisterRoutes(p.Group("/search"))
	handler.NewSettingsHandler(cfg, q, envSvc, ldapSvc).RegisterRoutes(p.Group("/settings", mw.RequireRole("admin")))

	// WebSocket
	wsH := handler.NewWSHandler(hub, cfg.JWT.Secret)
	e.GET("/ws", wsH.Handle)

	// OpenVAS import webhook (API-Key auth, outside JWT group)
	if cfg.Import.APIKey != "" {
		if len(cfg.Import.APIKey) < 32 {
			log.Fatal("OT_IMPORT_APIKEY must be at least 32 characters")
		}
		importSvc := service.NewImportService(db)
		importG := e.Group("/api/import", mw.APIKeyAuth(cfg.Import.APIKey), echomw.BodyLimit("50M"))
		handler.NewImportHandler(importSvc).RegisterRoutes(importG)
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

	// Graceful shutdown — handle both SIGINT and SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	log.Println("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	e.Shutdown(ctx)
}
