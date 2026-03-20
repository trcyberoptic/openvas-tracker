-- sql/migrations/008_create_notifications.up.sql
CREATE TABLE notifications (
    id         CHAR(36) PRIMARY KEY,
    user_id    CHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type       ENUM('scan_complete', 'vuln_found', 'ticket_assigned', 'team_invite', 'report_ready', 'system') NOT NULL,
    title      VARCHAR(500) NOT NULL,
    message    TEXT,
    read       TINYINT(1) NOT NULL DEFAULT 0,
    data       JSON,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_notifications_user ON notifications (user_id);
CREATE INDEX idx_notifications_unread ON notifications (user_id, read);
