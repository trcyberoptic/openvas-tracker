// internal/service/audit.go
package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/cyberoptic/vulntrack/internal/database/queries"
)

type AuditService struct {
	q *queries.Queries
}

func NewAuditService(pool *pgxpool.Pool) *AuditService {
	return &AuditService{q: queries.New(pool)}
}

func (s *AuditService) Log(userID, action, resource, ip, userAgent string) {
	uid, _ := uuid.Parse(userID)
	s.q.CreateAuditLog(context.Background(), queries.CreateAuditLogParams{
		UserID: &uid, Action: action, Resource: resource,
		IPAddress: &ip, UserAgent: &userAgent,
	})
}

func (s *AuditService) List(ctx context.Context, limit, offset int32) ([]queries.AuditLog, error) {
	return s.q.ListAuditLogs(ctx, queries.ListAuditLogsParams{Limit: limit, Offset: offset})
}

func (s *AuditService) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]queries.AuditLog, error) {
	return s.q.ListAuditLogsByUser(ctx, queries.ListAuditLogsByUserParams{UserID: &userID, Limit: limit, Offset: offset})
}
