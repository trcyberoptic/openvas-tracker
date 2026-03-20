// internal/service/user.go
package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/cyberoptic/vulntrack/internal/auth"
	"github.com/cyberoptic/vulntrack/internal/database/queries"
)

var (
	ErrUserNotFound    = errors.New("user not found")
	ErrDuplicateUser   = errors.New("user already exists")
	ErrInvalidPassword = errors.New("invalid password")
)

type UserService struct {
	q    *queries.Queries
	pool *pgxpool.Pool
}

func NewUserService(pool *pgxpool.Pool) *UserService {
	return &UserService{
		q:    queries.New(pool),
		pool: pool,
	}
}

func (s *UserService) Register(ctx context.Context, email, username, password string) (queries.User, error) {
	hash, err := auth.HashPassword(password)
	if err != nil {
		return queries.User{}, err
	}
	user, err := s.q.CreateUser(ctx, queries.CreateUserParams{
		Email:    email,
		Username: username,
		Password: hash,
		Role:     "viewer",
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
	if !auth.CheckPassword(password, user.Password) {
		return queries.User{}, ErrInvalidPassword
	}
	return user, nil
}

func (s *UserService) GetByID(ctx context.Context, id uuid.UUID) (queries.User, error) {
	user, err := s.q.GetUserByID(ctx, id)
	if err != nil {
		return queries.User{}, ErrUserNotFound
	}
	return user, nil
}
