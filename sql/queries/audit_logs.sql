-- sql/queries/audit_logs.sql
-- name: CreateAuditLog :exec
INSERT INTO audit_logs (user_id, action, resource, resource_id, details, ip_address, user_agent)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: ListAuditLogs :many
SELECT * FROM audit_logs ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- name: ListAuditLogsByUser :many
SELECT * FROM audit_logs WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3;
