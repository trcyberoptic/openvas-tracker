ALTER TABLE tickets MODIFY COLUMN status ENUM('open', 'fixed', 'risk_accepted') NOT NULL DEFAULT 'open';
