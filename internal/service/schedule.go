// internal/service/schedule.go
package service

import (
	"context"
	"database/sql"

	"github.com/cyberoptic/openvas-tracker/internal/database/queries"
)

type ScheduleService struct {
	q *queries.Queries
}

func NewScheduleService(db *sql.DB) *ScheduleService {
	return &ScheduleService{q: queries.New(db)}
}

func (s *ScheduleService) Create(ctx context.Context, params queries.CreateScheduleParams) (queries.Schedule, error) {
	return s.q.CreateSchedule(ctx, params)
}

func (s *ScheduleService) List(ctx context.Context, userID string) ([]queries.Schedule, error) {
	return s.q.ListSchedules(ctx, userID)
}

func (s *ScheduleService) Get(ctx context.Context, id string) (queries.Schedule, error) {
	return s.q.GetSchedule(ctx, id)
}

func (s *ScheduleService) Toggle(ctx context.Context, id string, enabled bool) error {
	return s.q.ToggleSchedule(ctx, queries.ToggleScheduleParams{ID: id, Enabled: enabled})
}

func (s *ScheduleService) Delete(ctx context.Context, id, userID string) error {
	return s.q.DeleteSchedule(ctx, queries.DeleteScheduleParams{ID: id, UserID: userID})
}
