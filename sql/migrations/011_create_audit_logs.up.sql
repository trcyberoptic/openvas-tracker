-- sql/migrations/011_create_audit_logs.up.sql
CREATE TABLE audit_logs (
    id          CHAR(36) PRIMARY KEY,
    user_id     CHAR(36) REFERENCES users(id) ON DELETE SET NULL,
    action      VARCHAR(255) NOT NULL,
    resource    VARCHAR(255) NOT NULL,
    resource_id CHAR(36),
    details     JSON,
    ip_address  VARCHAR(45),
    user_agent  TEXT,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_audit_user ON audit_logs (user_id);
CREATE INDEX idx_audit_resource ON audit_logs (resource, resource_id);
CREATE INDEX idx_audit_created ON audit_logs (created_at);
