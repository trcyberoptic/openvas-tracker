-- sql/queries/notifications.sql
-- name: CreateNotification :one
INSERT INTO notifications (user_id, type, title, message, data) VALUES ($1, $2, $3, $4, $5) RETURNING *;

-- name: ListNotifications :many
SELECT * FROM notifications WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3;

-- name: CountUnread :one
SELECT count(*) FROM notifications WHERE user_id = $1 AND NOT read;

-- name: MarkRead :exec
UPDATE notifications SET read = true WHERE id = $1 AND user_id = $2;

-- name: MarkAllRead :exec
UPDATE notifications SET read = true WHERE user_id = $1;
