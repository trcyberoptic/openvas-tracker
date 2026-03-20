-- sql/queries/notifications.sql

-- name: CreateNotification :exec
INSERT INTO notifications (id, user_id, type, title, message, data) VALUES (?, ?, ?, ?, ?, ?);

-- name: GetNotification :one
SELECT * FROM notifications WHERE id = ?;

-- name: ListNotifications :many
SELECT * FROM notifications WHERE user_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?;

-- name: CountUnread :one
SELECT count(*) FROM notifications WHERE user_id = ? AND read = 0;

-- name: MarkRead :exec
UPDATE notifications SET read = 1 WHERE id = ? AND user_id = ?;

-- name: MarkAllRead :exec
UPDATE notifications SET read = 1 WHERE user_id = ?;
