-- sql/migrations/003_create_scans.down.sql
DROP TABLE IF EXISTS scans;
DROP TYPE IF EXISTS scan_type;
DROP TYPE IF EXISTS scan_status;
