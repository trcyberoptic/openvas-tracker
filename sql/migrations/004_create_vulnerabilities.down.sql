-- sql/migrations/004_create_vulnerabilities.down.sql
DROP TABLE IF EXISTS vulnerabilities;
DROP TYPE IF EXISTS vuln_status;
DROP TYPE IF EXISTS severity_level;
