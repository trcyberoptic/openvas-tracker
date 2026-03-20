-- sql/migrations/012_create_search_indexes.down.sql
ALTER TABLE vulnerabilities DROP INDEX idx_vulns_fulltext;
ALTER TABLE targets DROP INDEX idx_targets_fulltext;
ALTER TABLE tickets DROP INDEX idx_tickets_fulltext;
ALTER TABLE assets DROP INDEX idx_assets_fulltext;
