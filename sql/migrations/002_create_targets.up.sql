-- sql/migrations/002_create_targets.up.sql
CREATE TABLE target_groups (
    id          CHAR(36) PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,
    description TEXT,
    user_id     CHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE targets (
    id          CHAR(36) PRIMARY KEY,
    host        VARCHAR(255) NOT NULL,
    ip_address  VARCHAR(45),
    hostname    VARCHAR(255),
    os_guess    VARCHAR(255),
    group_id    CHAR(36) REFERENCES target_groups(id) ON DELETE SET NULL,
    user_id     CHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    metadata    JSON,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE INDEX idx_targets_user ON targets (user_id);
CREATE INDEX idx_targets_group ON targets (group_id);
CREATE INDEX idx_targets_host ON targets (host);
