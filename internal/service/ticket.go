// internal/service/ticket.go
package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/cyberoptic/vulntrack/internal/database/queries"
)

type TicketService struct {
	q *queries.Queries
}

func NewTicketService(pool *pgxpool.Pool) *TicketService {
	return &TicketService{q: queries.New(pool)}
}

func (s *TicketService) Create(ctx context.Context, params queries.CreateTicketParams) (queries.Ticket, error) {
	return s.q.CreateTicket(ctx, params)
}

func (s *TicketService) Get(ctx context.Context, id uuid.UUID) (queries.Ticket, error) {
	return s.q.GetTicket(ctx, id)
}

func (s *TicketService) List(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]queries.Ticket, error) {
	return s.q.ListTickets(ctx, queries.ListTicketsParams{CreatedBy: userID, Limit: limit, Offset: offset})
}

func (s *TicketService) UpdateStatus(ctx context.Context, id uuid.UUID, status string) (queries.Ticket, error) {
	return s.q.UpdateTicketStatus(ctx, queries.UpdateTicketStatusParams{ID: id, Status: queries.TicketStatus(status)})
}

func (s *TicketService) Assign(ctx context.Context, id, assigneeID uuid.UUID) (queries.Ticket, error) {
	return s.q.AssignTicket(ctx, queries.AssignTicketParams{ID: id, AssignedTo: &assigneeID})
}

func (s *TicketService) AddComment(ctx context.Context, ticketID, userID uuid.UUID, content string) (queries.TicketComment, error) {
	return s.q.AddTicketComment(ctx, queries.AddTicketCommentParams{TicketID: ticketID, UserID: userID, Content: content})
}

func (s *TicketService) ListComments(ctx context.Context, ticketID uuid.UUID) ([]queries.TicketComment, error) {
	return s.q.ListTicketComments(ctx, ticketID)
}
