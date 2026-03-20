-- Combined init script for Docker: runs all .up.sql migrations in order.
-- This file is auto-generated for docker-entrypoint-initdb.d usage.

SOURCE /docker-entrypoint-initdb.d/migrations/001_create_users.up.sql;
SOURCE /docker-entrypoint-initdb.d/migrations/002_create_targets.up.sql;
SOURCE /docker-entrypoint-initdb.d/migrations/003_create_scans.up.sql;
SOURCE /docker-entrypoint-initdb.d/migrations/004_create_vulnerabilities.up.sql;
SOURCE /docker-entrypoint-initdb.d/migrations/005_create_tickets.up.sql;
SOURCE /docker-entrypoint-initdb.d/migrations/006_create_teams.up.sql;
SOURCE /docker-entrypoint-initdb.d/migrations/007_create_reports.up.sql;
SOURCE /docker-entrypoint-initdb.d/migrations/008_create_notifications.up.sql;
SOURCE /docker-entrypoint-initdb.d/migrations/009_create_schedules.up.sql;
SOURCE /docker-entrypoint-initdb.d/migrations/010_create_assets.up.sql;
SOURCE /docker-entrypoint-initdb.d/migrations/011_create_audit_logs.up.sql;
SOURCE /docker-entrypoint-initdb.d/migrations/012_create_search_indexes.up.sql;
SOURCE /docker-entrypoint-initdb.d/migrations/013_add_ticket_seen_timestamps.up.sql;
