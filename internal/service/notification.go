// internal/service/notification.go
package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/cyberoptic/vulntrack/internal/database/queries"
)

type NotificationService struct {
	q *queries.Queries
}

func NewNotificationService(pool *pgxpool.Pool) *NotificationService {
	return &NotificationService{q: queries.New(pool)}
}

func (s *NotificationService) Create(ctx context.Context, params queries.CreateNotificationParams) (queries.Notification, error) {
	return s.q.CreateNotification(ctx, params)
}

func (s *NotificationService) List(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]queries.Notification, error) {
	return s.q.ListNotifications(ctx, queries.ListNotificationsParams{UserID: userID, Limit: limit, Offset: offset})
}

func (s *NotificationService) CountUnread(ctx context.Context, userID uuid.UUID) (int64, error) {
	return s.q.CountUnread(ctx, userID)
}

func (s *NotificationService) MarkRead(ctx context.Context, id, userID uuid.UUID) error {
	return s.q.MarkRead(ctx, queries.MarkReadParams{ID: id, UserID: userID})
}

func (s *NotificationService) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	return s.q.MarkAllRead(ctx, userID)
}
