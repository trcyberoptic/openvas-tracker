-- sql/migrations/009_create_schedules.up.sql
CREATE TABLE schedules (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name        TEXT NOT NULL,
    cron_expr   TEXT NOT NULL,
    scan_type   scan_type NOT NULL,
    target_id   UUID REFERENCES targets(id) ON DELETE CASCADE,
    target_group_id UUID REFERENCES target_groups(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    options     JSONB DEFAULT '{}',
    enabled     BOOLEAN NOT NULL DEFAULT true,
    last_run    TIMESTAMPTZ,
    next_run    TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_schedules_user ON schedules (user_id);
CREATE INDEX idx_schedules_next ON schedules (next_run) WHERE enabled;
