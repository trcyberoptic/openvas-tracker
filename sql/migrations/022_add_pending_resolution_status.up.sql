-- The code has written status 'pending_resolution' since the flapping-protection
-- feature, but no migration ever added it to the ENUM — under strict mode fresh
-- installs fail with "Data truncated" on the first scan miss.
ALTER TABLE tickets MODIFY COLUMN status ENUM('open', 'pending_resolution', 'fixed', 'risk_accepted', 'false_positive') NOT NULL DEFAULT 'open';
