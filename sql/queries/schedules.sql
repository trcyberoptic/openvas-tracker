-- sql/queries/schedules.sql

-- name: CreateSchedule :exec
INSERT INTO schedules (id, name, cron_expr, scan_type, target_id, target_group_id, user_id, options)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetSchedule :one
SELECT * FROM schedules WHERE id = ?;

-- name: ListSchedules :many
SELECT * FROM schedules WHERE user_id = ? ORDER BY name;

-- name: GetDueSchedules :many
SELECT * FROM schedules WHERE enabled = 1 AND next_run <= CURRENT_TIMESTAMP;

-- name: UpdateScheduleNextRun :exec
UPDATE schedules SET last_run = CURRENT_TIMESTAMP, next_run = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?;

-- name: ToggleSchedule :exec
UPDATE schedules SET enabled = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?;

-- name: DeleteSchedule :exec
DELETE FROM schedules WHERE id = ? AND user_id = ?;
