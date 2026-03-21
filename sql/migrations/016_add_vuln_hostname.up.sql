ALTER TABLE vulnerabilities ADD COLUMN hostname VARCHAR(255) AFTER affected_host;
