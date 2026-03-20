// internal/handler/auth.go
package handler

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/cyberoptic/vulntrack/internal/auth"
	"github.com/cyberoptic/vulntrack/internal/service"
)

type AuthHandler struct {
	users     *service.UserService
	jwtSecret string
	jwtExpiry time.Duration
}

func NewAuthHandler(users *service.UserService, jwtSecret string, jwtExpiry time.Duration) *AuthHandler {
	return &AuthHandler{users: users, jwtSecret: jwtSecret, jwtExpiry: jwtExpiry}
}

type registerRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Username string `json:"username" validate:"required,min=3,max=50"`
	Password string `json:"password" validate:"required,min=8"`
}

type loginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type authResponse struct {
	Token string  `json:"token"`
	User  userDTO `json:"user"`
}

type userDTO struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

func (h *AuthHandler) Register(c echo.Context) error {
	var req registerRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.Validate(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	user, err := h.users.Register(c.Request().Context(), req.Email, req.Username, req.Password)
	if err != nil {
		if err == service.ErrDuplicateUser {
			return echo.NewHTTPError(http.StatusConflict, "user already exists")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "registration failed")
	}

	token, err := auth.GenerateToken(user.ID, string(user.Role), h.jwtSecret, h.jwtExpiry)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "token generation failed")
	}

	return c.JSON(http.StatusCreated, authResponse{
		Token: token,
		User:  userDTO{ID: user.ID.String(), Email: user.Email, Username: user.Username, Role: string(user.Role)},
	})
}

func (h *AuthHandler) Login(c echo.Context) error {
	var req loginRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	user, err := h.users.Authenticate(c.Request().Context(), req.Email, req.Password)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid credentials")
	}

	token, err := auth.GenerateToken(user.ID, string(user.Role), h.jwtSecret, h.jwtExpiry)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "token generation failed")
	}

	return c.JSON(http.StatusOK, authResponse{
		Token: token,
		User:  userDTO{ID: user.ID.String(), Email: user.Email, Username: user.Username, Role: string(user.Role)},
	})
}

func (h *AuthHandler) RegisterRoutes(g *echo.Group) {
	g.POST("/register", h.Register)
	g.POST("/login", h.Login)
}
