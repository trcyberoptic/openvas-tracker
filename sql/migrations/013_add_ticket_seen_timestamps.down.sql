DROP TABLE IF EXISTS ticket_activity;
ALTER TABLE tickets DROP COLUMN last_seen_at;
ALTER TABLE tickets DROP COLUMN first_seen_at;
