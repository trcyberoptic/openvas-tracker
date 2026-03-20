// internal/service/notification.go
package service

import (
	"context"
	"database/sql"

	"github.com/cyberoptic/openvas-tracker/internal/database/queries"
)

type NotificationService struct {
	q *queries.Queries
}

func NewNotificationService(db *sql.DB) *NotificationService {
	return &NotificationService{q: queries.New(db)}
}

func (s *NotificationService) Create(ctx context.Context, params queries.CreateNotificationParams) (queries.Notification, error) {
	return s.q.CreateNotification(ctx, params)
}

func (s *NotificationService) List(ctx context.Context, userID string, limit, offset int32) ([]queries.Notification, error) {
	return s.q.ListNotifications(ctx, queries.ListNotificationsParams{UserID: userID, Limit: limit, Offset: offset})
}

func (s *NotificationService) CountUnread(ctx context.Context, userID string) (int64, error) {
	return s.q.CountUnread(ctx, userID)
}

func (s *NotificationService) MarkRead(ctx context.Context, id, userID string) error {
	return s.q.MarkRead(ctx, queries.MarkReadParams{ID: id, UserID: userID})
}

func (s *NotificationService) MarkAllRead(ctx context.Context, userID string) error {
	return s.q.MarkAllRead(ctx, userID)
}
