-- GVM task UUID of an OpenVAS import, so the scan diff only compares runs of the same task.
-- NULL for ZAP, manual and pre-027 imports.
ALTER TABLE scans ADD COLUMN gvm_task_id CHAR(36) NULL;
