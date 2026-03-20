-- sql/migrations/004_create_vulnerabilities.up.sql
CREATE TABLE vulnerabilities (
    id              CHAR(36) PRIMARY KEY,
    scan_id         CHAR(36) NOT NULL REFERENCES scans(id) ON DELETE CASCADE,
    target_id       CHAR(36) REFERENCES targets(id) ON DELETE SET NULL,
    user_id         CHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title           VARCHAR(500) NOT NULL,
    description     TEXT,
    severity        ENUM('critical', 'high', 'medium', 'low', 'info') NOT NULL DEFAULT 'info',
    status          ENUM('open', 'confirmed', 'mitigated', 'resolved', 'false_positive', 'accepted') NOT NULL DEFAULT 'open',
    cvss_score      DECIMAL(3,1),
    cve_id          VARCHAR(50),
    cwe_id          VARCHAR(50),
    affected_host   VARCHAR(255),
    affected_port   INT,
    protocol        VARCHAR(20),
    service         VARCHAR(100),
    solution        TEXT,
    vuln_references JSON,
    enrichment_data JSON,
    risk_score      DECIMAL(5,2),
    discovered_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    resolved_at     TIMESTAMP NULL,
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE INDEX idx_vulns_scan ON vulnerabilities (scan_id);
CREATE INDEX idx_vulns_severity ON vulnerabilities (severity);
CREATE INDEX idx_vulns_status ON vulnerabilities (status);
CREATE INDEX idx_vulns_cve ON vulnerabilities (cve_id);
CREATE INDEX idx_vulns_user ON vulnerabilities (user_id);
CREATE INDEX idx_vulns_target ON vulnerabilities (target_id);
