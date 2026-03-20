-- sql/migrations/002_create_targets.up.sql
CREATE TABLE target_groups (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name        TEXT NOT NULL,
    description TEXT,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE targets (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    host        TEXT NOT NULL,
    ip_address  TEXT,
    hostname    TEXT,
    os_guess    TEXT,
    group_id    UUID REFERENCES target_groups(id) ON DELETE SET NULL,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    metadata    JSONB DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_targets_user ON targets (user_id);
CREATE INDEX idx_targets_group ON targets (group_id);
CREATE INDEX idx_targets_host ON targets (host);
