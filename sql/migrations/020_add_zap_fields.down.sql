-- sql/migrations/020_add_zap_fields.down.sql
-- Rollback for ZAP-specific fields.
ALTER TABLE vulnerabilities
  DROP COLUMN url,
  DROP COLUMN parameter,
  DROP COLUMN evidence,
  DROP COLUMN confidence;

ALTER TABLE scans MODIFY COLUMN scan_type ENUM('nmap', 'openvas', 'custom') NOT NULL;
