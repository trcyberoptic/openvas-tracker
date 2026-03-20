-- sql/migrations/010_create_assets.up.sql
CREATE TABLE assets (
    id          CHAR(36) PRIMARY KEY,
    hostname    VARCHAR(255),
    ip_address  VARCHAR(45) NOT NULL,
    mac_address VARCHAR(17),
    os          VARCHAR(255),
    os_version  VARCHAR(100),
    open_ports  JSON,
    services    JSON,
    user_id     CHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target_id   CHAR(36) REFERENCES targets(id) ON DELETE SET NULL,
    last_seen   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    first_seen  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    vuln_count  INT NOT NULL DEFAULT 0,
    risk_score  DECIMAL(5,2) DEFAULT 0,
    metadata    JSON,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY idx_assets_user_ip (user_id, ip_address)
);

CREATE INDEX idx_assets_ip ON assets (ip_address);
CREATE INDEX idx_assets_user ON assets (user_id);
