DROP INDEX idx_vulns_host_oid ON vulnerabilities;
ALTER TABLE vulnerabilities DROP COLUMN oid;
