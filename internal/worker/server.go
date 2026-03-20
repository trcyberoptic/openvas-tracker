// internal/worker/server.go
package worker

import (
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/cyberoptic/vulntrack/internal/config"
	"github.com/cyberoptic/vulntrack/internal/scanner"
)

const (
	TaskScanNmap    = "scan:nmap"
	TaskScanOpenVAS = "scan:openvas"
	TaskReport      = "report:generate"
	TaskEnrich      = "vuln:enrich"
)

func NewServer(cfg *config.Config, pool *pgxpool.Pool) *asynq.Server {
	return asynq.NewServer(
		asynq.RedisClientOpt{Addr: cfg.Redis.Addr, Password: cfg.Redis.Password, DB: cfg.Redis.DB},
		asynq.Config{
			Concurrency: 10,
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
		},
	)
}

func NewMux(pool *pgxpool.Pool, nmapScanner *scanner.NmapScanner, openvasScanner *scanner.OpenVASScanner) *asynq.ServeMux {
	mux := asynq.NewServeMux()
	scanHandler := NewScanHandler(pool, nmapScanner, openvasScanner)
	mux.HandleFunc(TaskScanNmap, scanHandler.HandleNmapScan)
	mux.HandleFunc(TaskScanOpenVAS, scanHandler.HandleOpenVASScan)
	return mux
}
