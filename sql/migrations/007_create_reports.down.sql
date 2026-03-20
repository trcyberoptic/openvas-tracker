-- sql/migrations/007_create_reports.down.sql
DROP TABLE IF EXISTS reports;
DROP TYPE IF EXISTS report_status;
DROP TYPE IF EXISTS report_format;
DROP TYPE IF EXISTS report_type;
