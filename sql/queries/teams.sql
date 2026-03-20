-- sql/queries/teams.sql
-- name: CreateTeam :one
INSERT INTO teams (name, description, creator_id) VALUES ($1, $2, $3) RETURNING *;

-- name: GetTeam :one
SELECT * FROM teams WHERE id = $1;

-- name: ListTeamsByUser :many
SELECT t.* FROM teams t JOIN team_members tm ON t.id = tm.team_id WHERE tm.user_id = $1 ORDER BY t.name;

-- name: AddTeamMember :exec
INSERT INTO team_members (team_id, user_id, role) VALUES ($1, $2, $3);

-- name: RemoveTeamMember :exec
DELETE FROM team_members WHERE team_id = $1 AND user_id = $2;

-- name: ListTeamMembers :many
SELECT u.id, u.email, u.username, u.role, tm.role as team_role, tm.joined_at
FROM users u JOIN team_members tm ON u.id = tm.user_id
WHERE tm.team_id = $1;

-- name: CreateInvitation :one
INSERT INTO invitations (team_id, email, invited_by, expires_at) VALUES ($1, $2, $3, $4) RETURNING *;

-- name: DeleteTeam :exec
DELETE FROM teams WHERE id = $1;
