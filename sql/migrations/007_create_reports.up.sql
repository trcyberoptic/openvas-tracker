-- sql/migrations/007_create_reports.up.sql
CREATE TABLE reports (
    id          CHAR(36) PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,
    report_type ENUM('technical', 'executive', 'compliance', 'comparison', 'trend') NOT NULL,
    format      ENUM('html', 'pdf', 'excel', 'markdown') NOT NULL DEFAULT 'html',
    status      ENUM('pending', 'generating', 'completed', 'failed') NOT NULL DEFAULT 'pending',
    scan_ids    JSON,
    user_id     CHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    file_path   TEXT,
    file_data   LONGBLOB,
    metadata    JSON,
    generated_at TIMESTAMP NULL,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_reports_user ON reports (user_id);
CREATE INDEX idx_reports_status ON reports (status);
