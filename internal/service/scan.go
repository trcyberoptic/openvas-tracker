// internal/service/scan.go
package service

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"github.com/cyberoptic/openvas-tracker/internal/database/queries"
	"github.com/cyberoptic/openvas-tracker/internal/worker"
)

type ScanService struct {
	q      *queries.Queries
	client *asynq.Client
}

func NewScanService(db *sql.DB, client *asynq.Client) *ScanService {
	return &ScanService{q: queries.New(db), client: client}
}

func (s *ScanService) Create(ctx context.Context, params queries.CreateScanParams) (queries.Scan, error) {
	if params.ID == "" {
		params.ID = uuid.New().String()
	}
	return s.q.CreateScan(ctx, params)
}

func (s *ScanService) Get(ctx context.Context, id string) (queries.Scan, error) {
	return s.q.GetScan(ctx, id)
}

func (s *ScanService) List(ctx context.Context, userID string, limit, offset int32) ([]queries.Scan, error) {
	return s.q.ListScans(ctx, queries.ListScansParams{UserID: userID, Limit: limit, Offset: offset})
}

func (s *ScanService) Launch(ctx context.Context, scan queries.Scan, target string, options []string) error {
	taskType := worker.TaskScanNmap
	if scan.ScanType == queries.ScanTypeOpenvas {
		taskType = worker.TaskScanOpenVAS
	}
	task, err := worker.NewScanTask(taskType, worker.ScanPayload{
		ScanID:  scan.ID,
		Target:  target,
		Options: options,
	})
	if err != nil {
		return err
	}
	_, err = s.client.Enqueue(task)
	return err
}

func (s *ScanService) Delete(ctx context.Context, id, userID string) error {
	return s.q.DeleteScan(ctx, queries.DeleteScanParams{ID: id, UserID: userID})
}

// marshalOptions serialises a slice of strings to JSON bytes.
func marshalOptions(options []string) []byte {
	data, _ := json.Marshal(options)
	return data
}
