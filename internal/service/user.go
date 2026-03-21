// internal/service/user.go
package service

import (
	"context"
	"database/sql"
	"errors"

	"github.com/cyberoptic/openvas-tracker/internal/auth"
	"github.com/cyberoptic/openvas-tracker/internal/database/queries"
	"github.com/google/uuid"
)

var (
	ErrUserNotFound    = errors.New("user not found")
	ErrDuplicateUser   = errors.New("user already exists")
	ErrInvalidPassword = errors.New("invalid password")
)

type UserService struct {
	q  *queries.Queries
	db *sql.DB
}

func NewUserService(db *sql.DB) *UserService {
	return &UserService{
		q:  queries.New(db),
		db: db,
	}
}

func (s *UserService) Register(ctx context.Context, email, username, password string) (queries.User, error) {
	hash, err := auth.HashPassword(password)
	if err != nil {
		return queries.User{}, err
	}

	// First user gets admin role, all others get viewer
	role := queries.UserRole("viewer")
	count, err := s.q.CountUsers(ctx)
	if err == nil && count == 0 {
		role = queries.UserRole("admin")
	}

	user, err := s.q.CreateUser(ctx, queries.CreateUserParams{
		ID:       uuid.New().String(),
		Email:    email,
		Username: username,
		Password: hash,
		Role:     role,
	})
	if err != nil {
		return queries.User{}, ErrDuplicateUser
	}
	return user, nil
}

func (s *UserService) Authenticate(ctx context.Context, email, password string) (queries.User, error) {
	user, err := s.q.GetUserByEmail(ctx, email)
	if err != nil {
		return queries.User{}, ErrUserNotFound
	}
	if !user.IsActive {
		return queries.User{}, ErrUserNotFound
	}
	if !auth.CheckPassword(password, user.Password) {
		return queries.User{}, ErrInvalidPassword
	}
	return user, nil
}

func (s *UserService) GetByID(ctx context.Context, id string) (queries.User, error) {
	user, err := s.q.GetUserByID(ctx, id)
	if err != nil {
		return queries.User{}, ErrUserNotFound
	}
	return user, nil
}
