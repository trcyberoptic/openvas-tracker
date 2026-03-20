-- sql/migrations/004_create_vulnerabilities.up.sql
CREATE TYPE severity_level AS ENUM ('critical', 'high', 'medium', 'low', 'info');
CREATE TYPE vuln_status AS ENUM ('open', 'confirmed', 'mitigated', 'resolved', 'false_positive', 'accepted');

CREATE TABLE vulnerabilities (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    scan_id         UUID NOT NULL REFERENCES scans(id) ON DELETE CASCADE,
    target_id       UUID REFERENCES targets(id) ON DELETE SET NULL,
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title           TEXT NOT NULL,
    description     TEXT,
    severity        severity_level NOT NULL DEFAULT 'info',
    status          vuln_status NOT NULL DEFAULT 'open',
    cvss_score      DECIMAL(3,1),
    cve_id          TEXT,
    cwe_id          TEXT,
    affected_host   TEXT,
    affected_port   INTEGER,
    protocol        TEXT,
    service         TEXT,
    solution        TEXT,
    references      JSONB DEFAULT '[]',
    enrichment_data JSONB DEFAULT '{}',
    risk_score      DECIMAL(5,2),
    discovered_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_vulns_scan ON vulnerabilities (scan_id);
CREATE INDEX idx_vulns_severity ON vulnerabilities (severity);
CREATE INDEX idx_vulns_status ON vulnerabilities (status);
CREATE INDEX idx_vulns_cve ON vulnerabilities (cve_id);
CREATE INDEX idx_vulns_user ON vulnerabilities (user_id);
CREATE INDEX idx_vulns_target ON vulnerabilities (target_id);
