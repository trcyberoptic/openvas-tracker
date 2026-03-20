-- sql/migrations/003_create_scans.up.sql
CREATE TABLE scans (
    id              CHAR(36) PRIMARY KEY,
    name            VARCHAR(255) NOT NULL,
    scan_type       ENUM('nmap', 'openvas', 'custom') NOT NULL,
    status          ENUM('pending', 'running', 'completed', 'failed', 'cancelled') NOT NULL DEFAULT 'pending',
    target_id       CHAR(36) REFERENCES targets(id) ON DELETE SET NULL,
    target_group_id CHAR(36) REFERENCES target_groups(id) ON DELETE SET NULL,
    user_id         CHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    options         JSON,
    raw_output      LONGTEXT,
    started_at      TIMESTAMP NULL,
    completed_at    TIMESTAMP NULL,
    error_message   TEXT,
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE INDEX idx_scans_user ON scans (user_id);
CREATE INDEX idx_scans_status ON scans (status);
CREATE INDEX idx_scans_target ON scans (target_id);
