-- sql/migrations/012_create_search_indexes.down.sql
DROP INDEX IF EXISTS idx_vulns_title_trgm;
DROP INDEX IF EXISTS idx_vulns_desc_trgm;
DROP INDEX IF EXISTS idx_targets_host_trgm;
DROP INDEX IF EXISTS idx_tickets_title_trgm;
DROP INDEX IF EXISTS idx_assets_hostname_trgm;
