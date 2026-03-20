-- sql/queries/scans.sql

-- name: CreateScan :exec
INSERT INTO scans (id, name, scan_type, status, target_id, target_group_id, user_id, options)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetScan :one
SELECT * FROM scans WHERE id = ?;

-- name: ListScans :many
SELECT * FROM scans WHERE user_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?;

-- name: UpdateScanStatus :exec
UPDATE scans SET
    status = ?,
    started_at = COALESCE(?, started_at),
    completed_at = COALESCE(?, completed_at),
    error_message = COALESCE(?, error_message),
    raw_output = COALESCE(?, raw_output),
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: DeleteScan :exec
DELETE FROM scans WHERE id = ? AND user_id = ?;
