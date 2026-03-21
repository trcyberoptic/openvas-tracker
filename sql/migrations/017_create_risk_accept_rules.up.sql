CREATE TABLE IF NOT EXISTS risk_accept_rules (
    id CHAR(36) PRIMARY KEY,
    fingerprint VARCHAR(500) NOT NULL,
    host_pattern VARCHAR(255) DEFAULT '*',
    reason TEXT NOT NULL,
    expires_at DATE,
    created_by VARCHAR(100) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_rules_fingerprint (fingerprint)
);
