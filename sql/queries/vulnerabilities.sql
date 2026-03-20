-- sql/queries/vulnerabilities.sql

-- name: CreateVulnerability :one
INSERT INTO vulnerabilities (
    scan_id, target_id, user_id, title, description, severity,
    cvss_score, cve_id, cwe_id, affected_host, affected_port,
    protocol, service, solution, references
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
RETURNING *;

-- name: GetVulnerability :one
SELECT * FROM vulnerabilities WHERE id = $1;

-- name: ListVulnerabilities :many
SELECT * FROM vulnerabilities WHERE user_id = $1
ORDER BY
    CASE severity
        WHEN 'critical' THEN 1
        WHEN 'high' THEN 2
        WHEN 'medium' THEN 3
        WHEN 'low' THEN 4
        WHEN 'info' THEN 5
    END
LIMIT $2 OFFSET $3;

-- name: ListVulnsByScan :many
SELECT * FROM vulnerabilities WHERE scan_id = $1 ORDER BY severity, cvss_score DESC;

-- name: UpdateVulnStatus :one
UPDATE vulnerabilities SET
    status = $2,
    resolved_at = CASE WHEN $2 = 'resolved' THEN now() ELSE resolved_at END,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: UpdateVulnEnrichment :exec
UPDATE vulnerabilities SET
    enrichment_data = $2,
    risk_score = $3,
    updated_at = now()
WHERE id = $1;

-- name: CountVulnsBySeverity :many
SELECT severity, count(*) as count FROM vulnerabilities
WHERE user_id = $1 AND status NOT IN ('resolved', 'false_positive')
GROUP BY severity;

-- name: DeleteVulnerability :exec
DELETE FROM vulnerabilities WHERE id = $1 AND user_id = $2;
