-- sql/migrations/005_create_tickets.up.sql
CREATE TYPE ticket_status AS ENUM ('open', 'in_progress', 'review', 'resolved', 'closed');
CREATE TYPE ticket_priority AS ENUM ('critical', 'high', 'medium', 'low');

CREATE TABLE tickets (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title             TEXT NOT NULL,
    description       TEXT,
    status            ticket_status NOT NULL DEFAULT 'open',
    priority          ticket_priority NOT NULL DEFAULT 'medium',
    vulnerability_id  UUID REFERENCES vulnerabilities(id) ON DELETE SET NULL,
    assigned_to       UUID REFERENCES users(id) ON DELETE SET NULL,
    created_by        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    due_date          TIMESTAMPTZ,
    resolved_at       TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE ticket_comments (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    ticket_id   UUID NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content     TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_tickets_assigned ON tickets (assigned_to);
CREATE INDEX idx_tickets_status ON tickets (status);
CREATE INDEX idx_tickets_vuln ON tickets (vulnerability_id);
