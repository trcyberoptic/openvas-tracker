-- sql/queries/reports.sql

-- name: CreateReport :one
INSERT INTO reports (name, report_type, format, status, scan_ids, user_id, metadata)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetReport :one
SELECT * FROM reports WHERE id = $1;

-- name: ListReports :many
SELECT * FROM reports WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3;

-- name: UpdateReportStatus :exec
UPDATE reports SET
    status = $2,
    file_data = $3,
    generated_at = CASE WHEN $2 = 'completed' THEN now() ELSE generated_at END
WHERE id = $1;
