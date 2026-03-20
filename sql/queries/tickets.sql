-- sql/queries/tickets.sql

-- name: CreateTicket :exec
INSERT INTO tickets (id, title, description, priority, vulnerability_id, assigned_to, created_by, due_date)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetTicket :one
SELECT * FROM tickets WHERE id = ?;

-- name: ListTickets :many
SELECT * FROM tickets WHERE created_by = ? OR assigned_to = ?
ORDER BY
    CASE priority WHEN 'critical' THEN 1 WHEN 'high' THEN 2 WHEN 'medium' THEN 3 WHEN 'low' THEN 4 END,
    created_at DESC
LIMIT ? OFFSET ?;

-- name: UpdateTicketStatus :exec
UPDATE tickets SET
    status = ?,
    resolved_at = CASE WHEN ? = 'resolved' THEN CURRENT_TIMESTAMP ELSE resolved_at END,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: AssignTicket :exec
UPDATE tickets SET assigned_to = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?;

-- name: AddTicketComment :exec
INSERT INTO ticket_comments (id, ticket_id, user_id, content)
VALUES (?, ?, ?, ?);

-- name: ListTicketComments :many
SELECT * FROM ticket_comments WHERE ticket_id = ? ORDER BY created_at;

-- name: CountTicketsByStatus :many
SELECT status, count(*) as count FROM tickets
WHERE created_by = ? OR assigned_to = ?
GROUP BY status;

-- name: DeleteTicket :exec
DELETE FROM tickets WHERE id = ?;
