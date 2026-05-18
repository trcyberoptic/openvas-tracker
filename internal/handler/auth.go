package handler

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/cyberoptic/openvas-tracker/internal/auth"
	"github.com/cyberoptic/openvas-tracker/internal/config"
	"github.com/cyberoptic/openvas-tracker/internal/database/queries"
	"github.com/cyberoptic/openvas-tracker/internal/service"
)

type AuthHandler struct {
	users     *service.UserService
	ldap      *service.LDAPService
	cfg       *config.Config
	jwtSecret string
	jwtExpiry time.Duration
	q         *queries.Queries
}

func NewAuthHandler(users *service.UserService, ldap *service.LDAPService, cfg *config.Config, q *queries.Queries, jwtSecret string, jwtExpiry time.Duration) *AuthHandler {
	return &AuthHandler{users: users, ldap: ldap, cfg: cfg, q: q, jwtSecret: jwtSecret, jwtExpiry: jwtExpiry}
}

type loginRequest struct {
	Username string `json:"username" validate:"required"`
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

func (h *AuthHandler) Login(c echo.Context) error {
	var req loginRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	// Try admin login first
	if req.Username == "admin" && h.cfg.Admin.Password != "" {
		reqPasswordHash := sha256.Sum256([]byte(req.Password))
		adminPasswordHash := sha256.Sum256([]byte(h.cfg.Admin.Password))

		if subtle.ConstantTimeCompare(reqPasswordHash[:], adminPasswordHash[:]) == 1 {
			return h.loginAsAdmin(c)
		}
	}

	// Try LDAP if configured
	ldapCfg := h.currentLDAPConfig()
	if ldapCfg.Enabled() {
		ldapUser, err := h.ldap.Authenticate(ldapCfg, req.Username, req.Password)
		if err == nil {
			return h.loginAsLDAP(c, ldapUser)
		}
	}

	// Fallback: try DB user (for backwards compat with existing accounts)
	user, err := h.users.Authenticate(c.Request().Context(), req.Username, req.Password)
	if err == nil {
		token, _ := auth.GenerateToken(user.ID, string(user.Role), h.jwtSecret, h.jwtExpiry)
		return c.JSON(http.StatusOK, authResponse{
			Token: token,
			User:  userDTO{ID: user.ID, Email: user.Email, Username: user.Username, Role: string(user.Role)},
		})
	}

	return echo.NewHTTPError(http.StatusUnauthorized, "invalid credentials")
}

func (h *AuthHandler) loginAsAdmin(c echo.Context) error {
	// Ensure admin user exists in DB
	ctx := c.Request().Context()
	user, err := h.users.GetByUsername(ctx, "admin")
	if err != nil {
		hash, _ := auth.HashPassword(h.cfg.Admin.Password)
		user, err = h.ensureUser(ctx, "admin", "admin@local", hash)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to create admin user")
		}
	}

	token, _ := auth.GenerateToken(user.ID, "admin", h.jwtSecret, h.jwtExpiry)
	return c.JSON(http.StatusOK, authResponse{
		Token: token,
		User:  userDTO{ID: user.ID, Email: user.Email, Username: user.Username, Role: "admin"},
	})
}

func (h *AuthHandler) loginAsLDAP(c echo.Context, ldapUser *service.LDAPUser) error {
	ctx := c.Request().Context()
	user, err := h.users.GetByUsername(ctx, ldapUser.Username)
	if err != nil {
		// Auto-create on first login
		hash, _ := auth.HashPassword(uuid.New().String()) // random password, never used
		email := ldapUser.Email
		if email == "" {
			email = ldapUser.Username + "@ldap"
		}
		user, err = h.ensureUser(ctx, ldapUser.Username, email, hash)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to create LDAP user")
		}
	}

	token, _ := auth.GenerateToken(user.ID, "user", h.jwtSecret, h.jwtExpiry)
	return c.JSON(http.StatusOK, authResponse{
		Token: token,
		User:  userDTO{ID: user.ID, Email: user.Email, Username: user.Username, Role: "user"},
	})
}

func (h *AuthHandler) ensureUser(ctx context.Context, username, email, passwordHash string) (queries.User, error) {
	user, err := h.q.CreateUser(ctx, queries.CreateUserParams{
		ID: uuid.New().String(), Email: email, Username: username,
		Password: passwordHash, Role: queries.UserRoleAdmin,
	})
	if err != nil {
		// Duplicate — fetch existing
		return h.users.GetByUsername(ctx, username)
	}
	return user, nil
}

func (h *AuthHandler) currentLDAPConfig() config.LDAPConfig {
	// Re-read .env for live LDAP config changes
	cfg, err := config.Load()
	if err != nil {
		return h.cfg.LDAP
	}
	return cfg.LDAP
}

func (h *AuthHandler) RegisterRoutes(g *echo.Group) {
	g.POST("/login", h.Login)
}
