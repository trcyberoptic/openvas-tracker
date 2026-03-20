-- sql/migrations/010_create_assets.up.sql
CREATE TABLE assets (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    hostname        TEXT,
    ip_address      TEXT NOT NULL,
    mac_address     TEXT,
    os              TEXT,
    os_version      TEXT,
    open_ports      JSONB DEFAULT '[]',
    services        JSONB DEFAULT '[]',
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target_id       UUID REFERENCES targets(id) ON DELETE SET NULL,
    last_seen       TIMESTAMPTZ NOT NULL DEFAULT now(),
    first_seen      TIMESTAMPTZ NOT NULL DEFAULT now(),
    vuln_count      INTEGER NOT NULL DEFAULT 0,
    risk_score      DECIMAL(5,2) DEFAULT 0,
    metadata        JSONB DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_assets_ip ON assets (ip_address);
CREATE INDEX idx_assets_user ON assets (user_id);
CREATE UNIQUE INDEX idx_assets_user_ip ON assets (user_id, ip_address);
