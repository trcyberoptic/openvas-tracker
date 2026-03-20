// internal/worker/scan_task.go
package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"

	"github.com/cyberoptic/openvas-tracker/internal/database/queries"
	"github.com/cyberoptic/openvas-tracker/internal/scanner"
)

type ScanPayload struct {
	ScanID  string   `json:"scan_id"`
	Target  string   `json:"target"`
	Options []string `json:"options"`
}

type ScanHandler struct {
	q       *queries.Queries
	nmap    *scanner.NmapScanner
	openvas *scanner.OpenVASScanner
}

func NewScanHandler(db *sql.DB, nmap *scanner.NmapScanner, openvas *scanner.OpenVASScanner) *ScanHandler {
	return &ScanHandler{
		q:       queries.New(db),
		nmap:    nmap,
		openvas: openvas,
	}
}

func (h *ScanHandler) HandleNmapScan(ctx context.Context, t *asynq.Task) error {
	var payload ScanPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	now := time.Now()
	h.q.UpdateScanStatus(ctx, queries.UpdateScanStatusParams{
		ID:        payload.ScanID,
		Status:    queries.ScanStatusRunning,
		StartedAt: &now,
	})

	result, err := h.nmap.Scan(ctx, payload.Target, payload.Options...)
	if err != nil {
		errMsg := err.Error()
		h.q.UpdateScanStatus(ctx, queries.UpdateScanStatusParams{
			ID:           payload.ScanID,
			Status:       queries.ScanStatusFailed,
			ErrorMessage: &errMsg,
		})
		return err
	}

	rawJSON, _ := json.Marshal(result)
	rawStr := string(rawJSON)
	completed := time.Now()
	h.q.UpdateScanStatus(ctx, queries.UpdateScanStatusParams{
		ID:          payload.ScanID,
		Status:      queries.ScanStatusCompleted,
		CompletedAt: &completed,
		RawOutput:   &rawStr,
	})

	// TODO: Parse results into vulnerabilities table
	return nil
}

func (h *ScanHandler) HandleOpenVASScan(ctx context.Context, t *asynq.Task) error {
	var payload ScanPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	now := time.Now()
	h.q.UpdateScanStatus(ctx, queries.UpdateScanStatusParams{
		ID:        payload.ScanID,
		Status:    queries.ScanStatusRunning,
		StartedAt: &now,
	})

	results, err := h.openvas.Scan(ctx, payload.Target)
	if err != nil {
		errMsg := err.Error()
		h.q.UpdateScanStatus(ctx, queries.UpdateScanStatusParams{
			ID:           payload.ScanID,
			Status:       queries.ScanStatusFailed,
			ErrorMessage: &errMsg,
		})
		return err
	}

	rawJSON, _ := json.Marshal(results)
	rawStr := string(rawJSON)
	completed := time.Now()
	h.q.UpdateScanStatus(ctx, queries.UpdateScanStatusParams{
		ID:          payload.ScanID,
		Status:      queries.ScanStatusCompleted,
		CompletedAt: &completed,
		RawOutput:   &rawStr,
	})

	return nil
}

func NewScanTask(taskType string, payload ScanPayload) (*asynq.Task, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(taskType, data, asynq.MaxRetry(3), asynq.Queue("default")), nil
}
