-- sql/queries/tickets.sql

-- name: CreateTicket :one
INSERT INTO tickets (title, description, priority, vulnerability_id, assigned_to, created_by, due_date)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetTicket :one
SELECT * FROM tickets WHERE id = $1;

-- name: ListTickets :many
SELECT * FROM tickets WHERE created_by = $1 OR assigned_to = $1
ORDER BY
    CASE priority WHEN 'critical' THEN 1 WHEN 'high' THEN 2 WHEN 'medium' THEN 3 WHEN 'low' THEN 4 END,
    created_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdateTicketStatus :one
UPDATE tickets SET
    status = $2,
    resolved_at = CASE WHEN $2 = 'resolved' THEN now() ELSE resolved_at END,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: AssignTicket :one
UPDATE tickets SET assigned_to = $2, updated_at = now() WHERE id = $1 RETURNING *;

-- name: AddTicketComment :one
INSERT INTO ticket_comments (ticket_id, user_id, content)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListTicketComments :many
SELECT * FROM ticket_comments WHERE ticket_id = $1 ORDER BY created_at;

-- name: CountTicketsByStatus :many
SELECT status, count(*) as count FROM tickets
WHERE created_by = $1 OR assigned_to = $1
GROUP BY status;

-- name: DeleteTicket :exec
DELETE FROM tickets WHERE id = $1;
