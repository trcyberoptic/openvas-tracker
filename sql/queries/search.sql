-- sql/queries/search.sql
-- name: SearchAll :many
SELECT 'vulnerability' as type, id, title as name, COALESCE(description, '') as detail
FROM vulnerabilities WHERE title ILIKE '%' || $1 || '%' OR description ILIKE '%' || $1 || '%'
UNION ALL
SELECT 'target', id, host, COALESCE(hostname, '')
FROM targets WHERE host ILIKE '%' || $1 || '%' OR hostname ILIKE '%' || $1 || '%'
UNION ALL
SELECT 'ticket', id, title, COALESCE(description, '')
FROM tickets WHERE title ILIKE '%' || $1 || '%'
LIMIT $2;
