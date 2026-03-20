-- sql/queries/vulnerabilities.sql

-- name: CreateVulnerability :exec
INSERT INTO vulnerabilities (
    id, scan_id, target_id, user_id, title, description, severity,
    cvss_score, cve_id, cwe_id, affected_host, affected_port,
    protocol, service, solution, vuln_references
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetVulnerability :one
SELECT * FROM vulnerabilities WHERE id = ?;

-- name: ListVulnerabilities :many
SELECT * FROM vulnerabilities WHERE user_id = ?
ORDER BY
    CASE severity
        WHEN 'critical' THEN 1
        WHEN 'high' THEN 2
        WHEN 'medium' THEN 3
        WHEN 'low' THEN 4
        WHEN 'info' THEN 5
    END
LIMIT ? OFFSET ?;

-- name: ListVulnsByScan :many
SELECT * FROM vulnerabilities WHERE scan_id = ? ORDER BY severity, cvss_score DESC;

-- name: UpdateVulnStatus :exec
UPDATE vulnerabilities SET
    status = ?,
    resolved_at = CASE WHEN ? = 'resolved' THEN CURRENT_TIMESTAMP ELSE resolved_at END,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: UpdateVulnEnrichment :exec
UPDATE vulnerabilities SET
    enrichment_data = ?,
    risk_score = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: CountVulnsBySeverity :many
SELECT severity, count(*) as count FROM vulnerabilities
WHERE user_id = ? AND status NOT IN ('resolved', 'false_positive')
GROUP BY severity;

-- name: DeleteVulnerability :exec
DELETE FROM vulnerabilities WHERE id = ? AND user_id = ?;
