// internal/service/ticket.go
package service

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"github.com/cyberoptic/openvas-tracker/internal/database/queries"
)

type TicketService struct {
	q *queries.Queries
}

func NewTicketService(db *sql.DB) *TicketService {
	return &TicketService{q: queries.New(db)}
}

func (s *TicketService) Create(ctx context.Context, params queries.CreateTicketParams) (queries.Ticket, error) {
	if params.ID == "" {
		params.ID = uuid.New().String()
	}
	return s.q.CreateTicket(ctx, params)
}

func (s *TicketService) Get(ctx context.Context, id string) (queries.Ticket, error) {
	return s.q.GetTicket(ctx, id)
}

func (s *TicketService) List(ctx context.Context, userID string, limit, offset int32) ([]queries.Ticket, error) {
	return s.q.ListTickets(ctx, queries.ListTicketsParams{CreatedBy: userID, Limit: limit, Offset: offset})
}

func (s *TicketService) UpdateStatus(ctx context.Context, id string, status string) (queries.Ticket, error) {
	return s.q.UpdateTicketStatus(ctx, queries.UpdateTicketStatusParams{ID: id, Status: queries.TicketStatus(status)})
}

func (s *TicketService) Assign(ctx context.Context, id, assigneeID string) (queries.Ticket, error) {
	return s.q.AssignTicket(ctx, queries.AssignTicketParams{ID: id, AssignedTo: &assigneeID})
}

func (s *TicketService) AddComment(ctx context.Context, ticketID, userID string, content string) (queries.TicketComment, error) {
	return s.q.AddTicketComment(ctx, queries.AddTicketCommentParams{
		ID: uuid.New().String(), TicketID: ticketID, UserID: userID, Content: content,
	})
}

func (s *TicketService) ListComments(ctx context.Context, ticketID string) ([]queries.TicketComment, error) {
	return s.q.ListTicketComments(ctx, ticketID)
}
