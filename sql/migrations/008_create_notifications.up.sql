-- sql/migrations/008_create_notifications.up.sql
CREATE TYPE notification_type AS ENUM ('scan_complete', 'vuln_found', 'ticket_assigned', 'team_invite', 'report_ready', 'system');

CREATE TABLE notifications (
    id        UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type      notification_type NOT NULL,
    title     TEXT NOT NULL,
    message   TEXT,
    read      BOOLEAN NOT NULL DEFAULT false,
    data      JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_notifications_user ON notifications (user_id);
CREATE INDEX idx_notifications_unread ON notifications (user_id) WHERE NOT read;
