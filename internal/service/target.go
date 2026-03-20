// internal/service/target.go
package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/cyberoptic/vulntrack/internal/database/queries"
)

type TargetService struct {
	q *queries.Queries
}

func NewTargetService(pool *pgxpool.Pool) *TargetService {
	return &TargetService{q: queries.New(pool)}
}

func (s *TargetService) Create(ctx context.Context, params queries.CreateTargetParams) (queries.Target, error) {
	return s.q.CreateTarget(ctx, params)
}

func (s *TargetService) Get(ctx context.Context, id, userID uuid.UUID) (queries.Target, error) {
	return s.q.GetTarget(ctx, queries.GetTargetParams{ID: id, UserID: userID})
}

func (s *TargetService) List(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]queries.Target, error) {
	return s.q.ListTargets(ctx, queries.ListTargetsParams{UserID: userID, Limit: limit, Offset: offset})
}

func (s *TargetService) Delete(ctx context.Context, id, userID uuid.UUID) error {
	return s.q.DeleteTarget(ctx, queries.DeleteTargetParams{ID: id, UserID: userID})
}

func (s *TargetService) CreateGroup(ctx context.Context, name, description string, userID uuid.UUID) (queries.TargetGroup, error) {
	return s.q.CreateTargetGroup(ctx, queries.CreateTargetGroupParams{Name: name, Description: &description, UserID: userID})
}

func (s *TargetService) ListGroups(ctx context.Context, userID uuid.UUID) ([]queries.TargetGroup, error) {
	return s.q.ListTargetGroups(ctx, userID)
}
