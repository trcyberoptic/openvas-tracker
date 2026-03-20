-- sql/queries/reports.sql

-- name: CreateReport :exec
INSERT INTO reports (id, name, report_type, format, status, scan_ids, user_id, metadata)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetReport :one
SELECT * FROM reports WHERE id = ?;

-- name: ListReports :many
SELECT * FROM reports WHERE user_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?;

-- name: UpdateReportStatus :exec
UPDATE reports SET
    status = ?,
    file_data = ?,
    generated_at = CASE WHEN ? = 'completed' THEN CURRENT_TIMESTAMP ELSE generated_at END
WHERE id = ?;
