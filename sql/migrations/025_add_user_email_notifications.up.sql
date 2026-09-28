-- Per-user opt-out for assignment mails. Default on.
ALTER TABLE users ADD COLUMN email_notifications BOOLEAN NOT NULL DEFAULT TRUE;
