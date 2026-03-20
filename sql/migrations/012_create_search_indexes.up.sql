-- sql/migrations/012_create_search_indexes.up.sql
ALTER TABLE vulnerabilities ADD FULLTEXT INDEX idx_vulns_fulltext (title, description);
ALTER TABLE targets ADD FULLTEXT INDEX idx_targets_fulltext (host);
ALTER TABLE tickets ADD FULLTEXT INDEX idx_tickets_fulltext (title);
ALTER TABLE assets ADD FULLTEXT INDEX idx_assets_fulltext (hostname);
