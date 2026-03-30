-- sql/migrations/020_add_zap_fields.up.sql
-- Adds ZAP-specific fields to vulnerabilities and extends scan_type to support OWASP ZAP.
ALTER TABLE vulnerabilities
  ADD COLUMN url VARCHAR(2048) DEFAULT '' AFTER hostname,
  ADD COLUMN parameter VARCHAR(255) DEFAULT '' AFTER url,
  ADD COLUMN evidence TEXT AFTER solution,
  ADD COLUMN confidence VARCHAR(20) DEFAULT '' AFTER evidence;

ALTER TABLE scans MODIFY COLUMN scan_type ENUM('nmap', 'openvas', 'zap', 'custom') NOT NULL;
