// internal/service/team.go
package service

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"

	"github.com/cyberoptic/openvas-tracker/internal/database/queries"
)

type TeamService struct {
	q *queries.Queries
}

func NewTeamService(db *sql.DB) *TeamService {
	return &TeamService{q: queries.New(db)}
}

func (s *TeamService) Create(ctx context.Context, name, description string, creatorID string) (queries.Team, error) {
	team, err := s.q.CreateTeam(ctx, queries.CreateTeamParams{
		ID: uuid.New().String(), Name: name, Description: &description, CreatorID: creatorID,
	})
	if err != nil {
		return queries.Team{}, err
	}
	// Add creator as owner
	s.q.AddTeamMember(ctx, queries.AddTeamMemberParams{
		TeamID: team.ID, UserID: creatorID, Role: "owner",
	})
	return team, nil
}

func (s *TeamService) ListByUser(ctx context.Context, userID string) ([]queries.Team, error) {
	return s.q.ListTeamsByUser(ctx, userID)
}

func (s *TeamService) Get(ctx context.Context, id string) (queries.Team, error) {
	return s.q.GetTeam(ctx, id)
}

func (s *TeamService) ListMembers(ctx context.Context, teamID string) ([]queries.ListTeamMembersRow, error) {
	return s.q.ListTeamMembers(ctx, teamID)
}

func (s *TeamService) AddMember(ctx context.Context, teamID, userID string, role string) error {
	return s.q.AddTeamMember(ctx, queries.AddTeamMemberParams{
		TeamID: teamID, UserID: userID, Role: queries.TeamMemberRole(role),
	})
}

func (s *TeamService) RemoveMember(ctx context.Context, teamID, userID string) error {
	return s.q.RemoveTeamMember(ctx, queries.RemoveTeamMemberParams{TeamID: teamID, UserID: userID})
}

func (s *TeamService) Invite(ctx context.Context, teamID string, email string, invitedBy string) (queries.Invitation, error) {
	return s.q.CreateInvitation(ctx, queries.CreateInvitationParams{
		ID: uuid.New().String(), TeamID: teamID, Email: email, InvitedBy: invitedBy,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	})
}

func (s *TeamService) Delete(ctx context.Context, id string) error {
	return s.q.DeleteTeam(ctx, id)
}
