// internal/service/target.go
package service

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"github.com/cyberoptic/openvas-tracker/internal/database/queries"
)

type TargetService struct {
	q *queries.Queries
}

func NewTargetService(db *sql.DB) *TargetService {
	return &TargetService{q: queries.New(db)}
}

func (s *TargetService) Create(ctx context.Context, params queries.CreateTargetParams) (queries.Target, error) {
	if params.ID == "" {
		params.ID = uuid.New().String()
	}
	return s.q.CreateTarget(ctx, params)
}

func (s *TargetService) Get(ctx context.Context, id, userID string) (queries.Target, error) {
	return s.q.GetTarget(ctx, queries.GetTargetParams{ID: id, UserID: userID})
}

func (s *TargetService) List(ctx context.Context, userID string, limit, offset int32) ([]queries.Target, error) {
	return s.q.ListTargets(ctx, queries.ListTargetsParams{UserID: userID, Limit: limit, Offset: offset})
}

func (s *TargetService) Delete(ctx context.Context, id, userID string) error {
	return s.q.DeleteTarget(ctx, queries.DeleteTargetParams{ID: id, UserID: userID})
}

func (s *TargetService) CreateGroup(ctx context.Context, name, description string, userID string) (queries.TargetGroup, error) {
	return s.q.CreateTargetGroup(ctx, queries.CreateTargetGroupParams{
		ID: uuid.New().String(), Name: name, Description: &description, UserID: userID,
	})
}

func (s *TargetService) ListGroups(ctx context.Context, userID string) ([]queries.TargetGroup, error) {
	return s.q.ListTargetGroups(ctx, userID)
}
