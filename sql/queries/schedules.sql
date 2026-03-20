-- sql/queries/schedules.sql
-- name: CreateSchedule :one
INSERT INTO schedules (name, cron_expr, scan_type, target_id, target_group_id, user_id, options)
VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING *;

-- name: ListSchedules :many
SELECT * FROM schedules WHERE user_id = $1 ORDER BY name;

-- name: GetSchedule :one
SELECT * FROM schedules WHERE id = $1;

-- name: GetDueSchedules :many
SELECT * FROM schedules WHERE enabled AND next_run <= now();

-- name: UpdateScheduleNextRun :exec
UPDATE schedules SET last_run = now(), next_run = $2, updated_at = now() WHERE id = $1;

-- name: ToggleSchedule :exec
UPDATE schedules SET enabled = $2, updated_at = now() WHERE id = $1;

-- name: DeleteSchedule :exec
DELETE FROM schedules WHERE id = $1 AND user_id = $2;
