ALTER TABLE tickets MODIFY COLUMN status ENUM('open', 'in_progress', 'review', 'resolved', 'closed') NOT NULL DEFAULT 'open';
