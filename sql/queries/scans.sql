-- sql/queries/scans.sql

-- name: CreateScan :one
INSERT INTO scans (name, scan_type, status, target_id, target_group_id, user_id, options)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetScan :one
SELECT * FROM scans WHERE id = $1;

-- name: ListScans :many
SELECT * FROM scans WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3;

-- name: UpdateScanStatus :one
UPDATE scans SET
    status = $2,
    started_at = COALESCE(sqlc.narg('started_at'), started_at),
    completed_at = COALESCE(sqlc.narg('completed_at'), completed_at),
    error_message = COALESCE(sqlc.narg('error_message'), error_message),
    raw_output = COALESCE(sqlc.narg('raw_output'), raw_output),
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteScan :exec
DELETE FROM scans WHERE id = $1 AND user_id = $2;
