-- GVM report UUID of an OpenVAS import. UNIQUE so the same report is never imported twice,
-- NULL for ZAP, manual and pre-026 imports (MariaDB allows any number of NULLs).
ALTER TABLE scans ADD COLUMN gvm_report_id CHAR(36) NULL, ADD UNIQUE KEY uq_scans_gvm_report_id (gvm_report_id);
