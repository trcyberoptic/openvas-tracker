// internal/service/audit.go
package service

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"github.com/cyberoptic/openvas-tracker/internal/database/queries"
)

type AuditService struct {
	q *queries.Queries
}

func NewAuditService(db *sql.DB) *AuditService {
	return &AuditService{q: queries.New(db)}
}

func (s *AuditService) Log(userID, action, resource, ip, userAgent string) {
	id := uuid.New().String()
	s.q.CreateAuditLog(context.Background(), queries.CreateAuditLogParams{
		ID: id, UserID: &userID, Action: action, Resource: resource,
		IPAddress: &ip, UserAgent: &userAgent,
	})
}

func (s *AuditService) List(ctx context.Context, limit, offset int32) ([]queries.AuditLog, error) {
	return s.q.ListAuditLogs(ctx, queries.ListAuditLogsParams{Limit: limit, Offset: offset})
}

func (s *AuditService) ListByUser(ctx context.Context, userID string, limit, offset int32) ([]queries.AuditLog, error) {
	return s.q.ListAuditLogsByUser(ctx, queries.ListAuditLogsByUserParams{UserID: &userID, Limit: limit, Offset: offset})
}
