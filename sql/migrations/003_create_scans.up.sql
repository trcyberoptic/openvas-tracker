-- sql/migrations/003_create_scans.up.sql
CREATE TYPE scan_status AS ENUM ('pending', 'running', 'completed', 'failed', 'cancelled');
CREATE TYPE scan_type AS ENUM ('nmap', 'openvas', 'custom');

CREATE TABLE scans (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name            TEXT NOT NULL,
    scan_type       scan_type NOT NULL,
    status          scan_status NOT NULL DEFAULT 'pending',
    target_id       UUID REFERENCES targets(id) ON DELETE SET NULL,
    target_group_id UUID REFERENCES target_groups(id) ON DELETE SET NULL,
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    options         JSONB DEFAULT '{}',
    raw_output      TEXT,
    started_at      TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    error_message   TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_scans_user ON scans (user_id);
CREATE INDEX idx_scans_status ON scans (status);
CREATE INDEX idx_scans_target ON scans (target_id);
