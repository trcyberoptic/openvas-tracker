-- sql/queries/assets.sql

-- name: UpsertAsset :exec
INSERT INTO assets (id, hostname, ip_address, mac_address, os, os_version, open_ports, services, user_id, target_id)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
    hostname = COALESCE(VALUES(hostname), hostname),
    os = COALESCE(VALUES(os), os),
    open_ports = VALUES(open_ports),
    services = VALUES(services),
    last_seen = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP;

-- name: GetAsset :one
SELECT * FROM assets WHERE id = ? AND user_id = ?;

-- name: GetAssetByUserIP :one
SELECT * FROM assets WHERE user_id = ? AND ip_address = ?;

-- name: ListAssets :many
SELECT * FROM assets WHERE user_id = ? ORDER BY last_seen DESC LIMIT ? OFFSET ?;

-- name: UpdateAssetRisk :exec
UPDATE assets SET vuln_count = ?, risk_score = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?;

-- name: DeleteAsset :exec
DELETE FROM assets WHERE id = ? AND user_id = ?;
