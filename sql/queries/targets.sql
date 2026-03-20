-- sql/queries/targets.sql

-- name: CreateTarget :exec
INSERT INTO targets (id, host, ip_address, hostname, os_guess, group_id, user_id, metadata)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetTarget :one
SELECT * FROM targets WHERE id = ? AND user_id = ?;

-- name: ListTargets :many
SELECT * FROM targets WHERE user_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?;

-- name: ListTargetsByGroup :many
SELECT * FROM targets WHERE group_id = ? AND user_id = ? ORDER BY created_at DESC;

-- name: DeleteTarget :exec
DELETE FROM targets WHERE id = ? AND user_id = ?;

-- name: CountTargets :one
SELECT count(*) FROM targets WHERE user_id = ?;

-- name: CreateTargetGroup :exec
INSERT INTO target_groups (id, name, description, user_id)
VALUES (?, ?, ?, ?);

-- name: GetTargetGroup :one
SELECT * FROM target_groups WHERE id = ?;

-- name: ListTargetGroups :many
SELECT * FROM target_groups WHERE user_id = ? ORDER BY name;

-- name: DeleteTargetGroup :exec
DELETE FROM target_groups WHERE id = ? AND user_id = ?;
