-- sql/queries/targets.sql

-- name: CreateTarget :one
INSERT INTO targets (host, ip_address, hostname, os_guess, group_id, user_id, metadata)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetTarget :one
SELECT * FROM targets WHERE id = $1 AND user_id = $2;

-- name: ListTargets :many
SELECT * FROM targets WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3;

-- name: ListTargetsByGroup :many
SELECT * FROM targets WHERE group_id = $1 AND user_id = $2 ORDER BY created_at DESC;

-- name: UpdateTarget :one
UPDATE targets SET
    host = COALESCE(sqlc.narg('host'), host),
    ip_address = COALESCE(sqlc.narg('ip_address'), ip_address),
    hostname = COALESCE(sqlc.narg('hostname'), hostname),
    os_guess = COALESCE(sqlc.narg('os_guess'), os_guess),
    group_id = COALESCE(sqlc.narg('group_id'), group_id),
    metadata = COALESCE(sqlc.narg('metadata'), metadata),
    updated_at = now()
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: DeleteTarget :exec
DELETE FROM targets WHERE id = $1 AND user_id = $2;

-- name: CountTargets :one
SELECT count(*) FROM targets WHERE user_id = $1;

-- name: CreateTargetGroup :one
INSERT INTO target_groups (name, description, user_id)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListTargetGroups :many
SELECT * FROM target_groups WHERE user_id = $1 ORDER BY name;

-- name: DeleteTargetGroup :exec
DELETE FROM target_groups WHERE id = $1 AND user_id = $2;
