// cmd/vulntrack/main.go
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
	"github.com/hibiken/asynq"
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"

	"github.com/cyberoptic/vulntrack/internal/config"
	"github.com/cyberoptic/vulntrack/internal/database"
	"github.com/cyberoptic/vulntrack/internal/database/queries"
	"github.com/cyberoptic/vulntrack/internal/handler"
	mw "github.com/cyberoptic/vulntrack/internal/middleware"
	"github.com/cyberoptic/vulntrack/internal/scanner"
	"github.com/cyberoptic/vulntrack/internal/service"
	"github.com/cyberoptic/vulntrack/internal/websocket"
	"github.com/cyberoptic/vulntrack/internal/worker"
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

	pool, err := database.NewPool(cfg.Database.URL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	// Asynq client (enqueue jobs)
	asynqClient := asynq.NewClient(asynq.RedisClientOpt{
		Addr: cfg.Redis.Addr, Password: cfg.Redis.Password, DB: cfg.Redis.DB,
	})
	defer asynqClient.Close()

	// Services
	userSvc := service.NewUserService(pool)
	targetSvc := service.NewTargetService(pool)
	vulnSvc := service.NewVulnerabilityService(pool)
	ticketSvc := service.NewTicketService(pool)
	reportSvc := service.NewReportService(pool, vulnSvc)
	teamSvc := service.NewTeamService(pool)
	notifSvc := service.NewNotificationService(pool)
	scheduleSvc := service.NewScheduleService(pool)
	assetSvc := service.NewAssetService(pool)
	auditSvc := service.NewAuditService(pool)
	searchSvc := service.NewSearchService(pool)

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

	q := queries.New(pool)
	handler.NewScanHandler(q, asynqClient).RegisterRoutes(p.Group("/scans"))

	handler.NewVulnHandler(vulnSvc).RegisterRoutes(p.Group("/vulnerabilities"))
	handler.NewTicketHandler(ticketSvc).RegisterRoutes(p.Group("/tickets"))
	handler.NewReportHandler(reportSvc).RegisterRoutes(p.Group("/reports"))
	handler.NewDashboardHandler(vulnSvc, ticketSvc).RegisterRoutes(p.Group("/dashboard"))
	handler.NewTeamHandler(teamSvc).RegisterRoutes(p.Group("/teams"))
	handler.NewNotificationHandler(notifSvc).RegisterRoutes(p.Group("/notifications"))
	handler.NewScheduleHandler(scheduleSvc).RegisterRoutes(p.Group("/schedules"))
	handler.NewAssetHandler(assetSvc).RegisterRoutes(p.Group("/assets"))
	handler.NewAuditHandler(auditSvc).RegisterRoutes(p.Group("/audit"))
	handler.NewSearchHandler(searchSvc).RegisterRoutes(p.Group("/search"))

	// WebSocket
	wsH := handler.NewWSHandler(hub, cfg.JWT.Secret)
	e.GET("/ws", wsH.Handle)

	// Start Asynq worker in background
	nmapScanner := scanner.NewNmapScanner(cfg.Scanner.NmapPath)
	openvasScanner := scanner.NewOpenVASScanner(cfg.Scanner.OpenVASPath)
	workerSrv := worker.NewServer(cfg, pool)
	workerMux := worker.NewMux(pool, nmapScanner, openvasScanner)
	go func() {
		if err := workerSrv.Run(workerMux); err != nil {
			log.Printf("worker error: %v", err)
		}
	}()

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

	workerSrv.Shutdown()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	e.Shutdown(ctx)
}
