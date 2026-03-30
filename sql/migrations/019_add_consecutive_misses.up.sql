-- sql/migrations/019_add_consecutive_misses.up.sql
-- Adds consecutive_misses counter for flapping protection.
-- Tickets accumulate misses before auto-resolve instead of resolving immediately.
ALTER TABLE tickets ADD COLUMN consecutive_misses INT NOT NULL DEFAULT 0 AFTER risk_accepted_until;
