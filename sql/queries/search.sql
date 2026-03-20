-- sql/queries/search.sql

-- name: SearchAll :many
SELECT 'vulnerability' as type, id, title as name, COALESCE(description, '') as detail
FROM vulnerabilities WHERE title LIKE CONCAT('%', ?, '%') OR description LIKE CONCAT('%', ?, '%')
UNION ALL
SELECT 'target', id, host, COALESCE(hostname, '')
FROM targets WHERE host LIKE CONCAT('%', ?, '%') OR hostname LIKE CONCAT('%', ?, '%')
UNION ALL
SELECT 'ticket', id, title, COALESCE(description, '')
FROM tickets WHERE title LIKE CONCAT('%', ?, '%')
LIMIT ?;
