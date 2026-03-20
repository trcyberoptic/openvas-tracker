ALTER TABLE tickets ADD COLUMN first_seen_at TIMESTAMP NULL AFTER resolved_at;
ALTER TABLE tickets ADD COLUMN last_seen_at TIMESTAMP NULL AFTER first_seen_at;

CREATE TABLE ticket_activity (
    id          CHAR(36) PRIMARY KEY,
    ticket_id   CHAR(36) NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    action      VARCHAR(50) NOT NULL,
    old_value   VARCHAR(100),
    new_value   VARCHAR(100),
    changed_by  VARCHAR(100) NOT NULL,
    note        TEXT,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_ticket_activity_ticket ON ticket_activity (ticket_id);
