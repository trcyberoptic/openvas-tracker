-- sql/migrations/006_create_teams.up.sql
CREATE TABLE teams (
    id          CHAR(36) PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,
    description TEXT,
    creator_id  CHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE team_members (
    team_id   CHAR(36) NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id   CHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role      ENUM('owner', 'admin', 'member') NOT NULL DEFAULT 'member',
    joined_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (team_id, user_id)
);

CREATE TABLE invitations (
    id          CHAR(36) PRIMARY KEY,
    team_id     CHAR(36) NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    email       VARCHAR(255) NOT NULL,
    invited_by  CHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    accepted    TINYINT(1) NOT NULL DEFAULT 0,
    expires_at  TIMESTAMP NOT NULL,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
