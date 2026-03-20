-- sql/migrations/005_create_tickets.down.sql
DROP TABLE IF EXISTS ticket_comments;
DROP TABLE IF EXISTS tickets;
DROP TYPE IF EXISTS ticket_priority;
DROP TYPE IF EXISTS ticket_status;
