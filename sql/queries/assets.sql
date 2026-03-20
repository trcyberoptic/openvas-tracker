-- sql/queries/assets.sql
-- name: UpsertAsset :one
INSERT INTO assets (hostname, ip_address, mac_address, os, os_version, open_ports, services, user_id, target_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (user_id, ip_address) DO UPDATE SET
    hostname = COALESCE(EXCLUDED.hostname, assets.hostname),
    os = COALESCE(EXCLUDED.os, assets.os),
    open_ports = EXCLUDED.open_ports,
    services = EXCLUDED.services,
    last_seen = now(),
    updated_at = now()
RETURNING *;

-- name: GetAsset :one
SELECT * FROM assets WHERE id = $1 AND user_id = $2;

-- name: ListAssets :many
SELECT * FROM assets WHERE user_id = $1 ORDER BY last_seen DESC LIMIT $2 OFFSET $3;

-- name: UpdateAssetRisk :exec
UPDATE assets SET vuln_count = $2, risk_score = $3, updated_at = now() WHERE id = $1;

-- name: DeleteAsset :exec
DELETE FROM assets WHERE id = $1 AND user_id = $2;
