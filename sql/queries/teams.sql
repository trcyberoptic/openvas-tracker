-- sql/queries/teams.sql

-- name: CreateTeam :exec
INSERT INTO teams (id, name, description, creator_id) VALUES (?, ?, ?, ?);

-- name: GetTeam :one
SELECT * FROM teams WHERE id = ?;

-- name: ListTeamsByUser :many
SELECT t.* FROM teams t JOIN team_members tm ON t.id = tm.team_id WHERE tm.user_id = ? ORDER BY t.name;

-- name: AddTeamMember :exec
INSERT INTO team_members (team_id, user_id, role) VALUES (?, ?, ?);

-- name: RemoveTeamMember :exec
DELETE FROM team_members WHERE team_id = ? AND user_id = ?;

-- name: ListTeamMembers :many
SELECT u.id, u.email, u.username, u.role, tm.role as team_role, tm.joined_at
FROM users u JOIN team_members tm ON u.id = tm.user_id
WHERE tm.team_id = ?;

-- name: CreateInvitation :exec
INSERT INTO invitations (id, team_id, email, invited_by, expires_at) VALUES (?, ?, ?, ?, ?);

-- name: GetInvitation :one
SELECT * FROM invitations WHERE id = ?;

-- name: DeleteTeam :exec
DELETE FROM teams WHERE id = ?;
