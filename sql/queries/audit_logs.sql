-- sql/queries/audit_logs.sql

-- name: CreateAuditLog :exec
INSERT INTO audit_logs (id, user_id, action, resource, resource_id, details, ip_address, user_agent)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: ListAuditLogs :many
SELECT * FROM audit_logs ORDER BY created_at DESC LIMIT ? OFFSET ?;

-- name: ListAuditLogsByUser :many
SELECT * FROM audit_logs WHERE user_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?;
