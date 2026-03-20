-- sql/migrations/009_create_schedules.up.sql
CREATE TABLE schedules (
    id              CHAR(36) PRIMARY KEY,
    name            VARCHAR(255) NOT NULL,
    cron_expr       VARCHAR(100) NOT NULL,
    scan_type       ENUM('nmap', 'openvas', 'custom') NOT NULL,
    target_id       CHAR(36) REFERENCES targets(id) ON DELETE CASCADE,
    target_group_id CHAR(36) REFERENCES target_groups(id) ON DELETE CASCADE,
    user_id         CHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    options         JSON,
    enabled         TINYINT(1) NOT NULL DEFAULT 1,
    last_run        TIMESTAMP NULL,
    next_run        TIMESTAMP NULL,
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE INDEX idx_schedules_user ON schedules (user_id);
CREATE INDEX idx_schedules_next ON schedules (next_run, enabled);
