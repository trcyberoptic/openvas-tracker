-- Stable per-check identifier: OpenVAS NVT OID, ZAP pluginid.
-- Ticket identity used to rest on (host, CVE) or (host, title), and both of
-- those move: Greenbone dropped the DISPUTED CVE refs from 41 NVTs in the
-- 2026-08-08 feed, which split 136 tickets and resurrected 17 false positives.
-- VT titles get renamed just as freely. The OID survives both.
ALTER TABLE vulnerabilities ADD COLUMN oid VARCHAR(120) NULL AFTER cwe_id;
CREATE INDEX idx_vulns_host_oid ON vulnerabilities (affected_host, oid);
