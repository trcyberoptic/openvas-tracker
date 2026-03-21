ALTER TABLE tickets MODIFY COLUMN status ENUM('open', 'fixed', 'risk_accepted', 'false_positive') NOT NULL DEFAULT 'open';
ALTER TABLE tickets ADD COLUMN risk_accepted_until DATE NULL AFTER resolved_at;
