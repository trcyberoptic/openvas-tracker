-- sql/migrations/018_create_scan_hosts.up.sql
-- Tracks which hosts were present in each scan import, so auto-resolve
-- only affects tickets for hosts that were actually in scope.
CREATE TABLE scan_hosts (
    scan_id CHAR(36) NOT NULL,
    host    VARCHAR(255) NOT NULL,
    PRIMARY KEY (scan_id, host),
    FOREIGN KEY (scan_id) REFERENCES scans(id) ON DELETE CASCADE
);
