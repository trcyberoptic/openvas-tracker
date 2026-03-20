-- sql/migrations/007_create_reports.up.sql
CREATE TYPE report_type AS ENUM ('technical', 'executive', 'compliance', 'comparison', 'trend');
CREATE TYPE report_format AS ENUM ('html', 'pdf', 'excel', 'markdown');
CREATE TYPE report_status AS ENUM ('pending', 'generating', 'completed', 'failed');

CREATE TABLE reports (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name        TEXT NOT NULL,
    report_type report_type NOT NULL,
    format      report_format NOT NULL DEFAULT 'html',
    status      report_status NOT NULL DEFAULT 'pending',
    scan_ids    UUID[] DEFAULT '{}',
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    file_path   TEXT,
    file_data   BYTEA,
    metadata    JSONB DEFAULT '{}',
    generated_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_reports_user ON reports (user_id);
CREATE INDEX idx_reports_status ON reports (status);
