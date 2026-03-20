// internal/service/schedule.go
package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/cyberoptic/vulntrack/internal/database/queries"
)

type ScheduleService struct {
	q *queries.Queries
}

func NewScheduleService(pool *pgxpool.Pool) *ScheduleService {
	return &ScheduleService{q: queries.New(pool)}
}

func (s *ScheduleService) Create(ctx context.Context, params queries.CreateScheduleParams) (queries.Schedule, error) {
	return s.q.CreateSchedule(ctx, params)
}

func (s *ScheduleService) List(ctx context.Context, userID uuid.UUID) ([]queries.Schedule, error) {
	return s.q.ListSchedules(ctx, userID)
}

func (s *ScheduleService) Get(ctx context.Context, id uuid.UUID) (queries.Schedule, error) {
	return s.q.GetSchedule(ctx, id)
}

func (s *ScheduleService) Toggle(ctx context.Context, id uuid.UUID, enabled bool) error {
	return s.q.ToggleSchedule(ctx, queries.ToggleScheduleParams{ID: id, Enabled: enabled})
}

func (s *ScheduleService) Delete(ctx context.Context, id, userID uuid.UUID) error {
	return s.q.DeleteSchedule(ctx, queries.DeleteScheduleParams{ID: id, UserID: userID})
}
