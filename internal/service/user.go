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
	user, err := s.q.CreateUser(ctx, queries.CreateUserParams{
		ID:       uuid.New().String(),
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

func (s *UserService) GetByID(ctx context.Context, id string) (queries.User, error) {
	user, err := s.q.GetUserByID(ctx, id)
	if err != nil {
		return queries.User{}, ErrUserNotFound
	}
	return user, nil
}
