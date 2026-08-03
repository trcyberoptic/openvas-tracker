UPDATE tickets SET status = 'open' WHERE status = 'pending_resolution';
ALTER TABLE tickets MODIFY COLUMN status ENUM('open', 'fixed', 'risk_accepted', 'false_positive') NOT NULL DEFAULT 'open';
