-- sql/migrations/012_create_search_indexes.up.sql
CREATE INDEX idx_vulns_title_trgm ON vulnerabilities USING gin (title gin_trgm_ops);
CREATE INDEX idx_vulns_desc_trgm ON vulnerabilities USING gin (description gin_trgm_ops);
CREATE INDEX idx_targets_host_trgm ON targets USING gin (host gin_trgm_ops);
CREATE INDEX idx_tickets_title_trgm ON tickets USING gin (title gin_trgm_ops);
CREATE INDEX idx_assets_hostname_trgm ON assets USING gin (hostname gin_trgm_ops);
