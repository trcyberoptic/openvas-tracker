-- sql/migrations/019_add_consecutive_misses.down.sql
ALTER TABLE tickets DROP COLUMN consecutive_misses;
