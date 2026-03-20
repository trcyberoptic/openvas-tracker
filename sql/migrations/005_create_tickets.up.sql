-- sql/migrations/005_create_tickets.up.sql
CREATE TABLE tickets (
    id                CHAR(36) PRIMARY KEY,
    title             VARCHAR(500) NOT NULL,
    description       TEXT,
    status            ENUM('open', 'in_progress', 'review', 'resolved', 'closed') NOT NULL DEFAULT 'open',
    priority          ENUM('critical', 'high', 'medium', 'low') NOT NULL DEFAULT 'medium',
    vulnerability_id  CHAR(36) REFERENCES vulnerabilities(id) ON DELETE SET NULL,
    assigned_to       CHAR(36) REFERENCES users(id) ON DELETE SET NULL,
    created_by        CHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    due_date          TIMESTAMP NULL,
    resolved_at       TIMESTAMP NULL,
    created_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE ticket_comments (
    id          CHAR(36) PRIMARY KEY,
    ticket_id   CHAR(36) NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    user_id     CHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content     TEXT NOT NULL,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_tickets_assigned ON tickets (assigned_to);
CREATE INDEX idx_tickets_status ON tickets (status);
CREATE INDEX idx_tickets_vuln ON tickets (vulnerability_id);
