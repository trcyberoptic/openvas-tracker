package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// envPlaceholder marks the values in deploy/openvas-tracker.env.example that
// must be replaced before the service may start.
const envPlaceholder = "CHANGEME"

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	Import   ImportConfig
	Admin    AdminConfig
	LDAP     LDAPConfig
}

type ServerConfig struct {
	Host string
	Port int
}

type DatabaseConfig struct {
	DSN      string
	MaxConns int
	MinConns int
}

type JWTConfig struct {
	Secret      string
	ExpireHours int
}

type ImportConfig struct {
	APIKey string
}

type AdminConfig struct {
	Password string
}

type LDAPConfig struct {
	URL                string
	BaseDN             string
	BindDN             string
	BindPassword       string
	GroupDN            string
	UserFilter         string // e.g. (sAMAccountName=%s)
	InsecureSkipVerify bool
}

func (l LDAPConfig) Enabled() bool {
	return l.URL != "" && l.BaseDN != ""
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	return &Config{
		Server: ServerConfig{
			Host: env("OT_SERVER_HOST", "0.0.0.0"),
			Port: envInt("OT_SERVER_PORT", 8080),
		},
		Database: DatabaseConfig{
			DSN:      env("OT_DATABASE_DSN", "openvas-tracker:openvas-tracker@tcp(localhost:3306)/openvas-tracker?parseTime=true"),
			MaxConns: envInt("OT_DATABASE_MAXCONNS", 25),
			MinConns: envInt("OT_DATABASE_MINCONNS", 5),
		},
		JWT: JWTConfig{
			Secret:      env("OT_JWT_SECRET", "change-me-in-production"),
			ExpireHours: envInt("OT_JWT_EXPIREHOURS", 24),
		},
		Import: ImportConfig{
			APIKey: env("OT_IMPORT_APIKEY", ""),
		},
		Admin: AdminConfig{
			Password: env("OT_ADMIN_PASSWORD", ""),
		},
		LDAP: LDAPConfig{
			URL:                env("OT_LDAP_URL", ""),
			BaseDN:             env("OT_LDAP_BASE_DN", ""),
			BindDN:             env("OT_LDAP_BIND_DN", ""),
			BindPassword:       env("OT_LDAP_BIND_PASSWORD", ""),
			GroupDN:            env("OT_LDAP_GROUP_DN", ""),
			UserFilter:         env("OT_LDAP_USER_FILTER", "(sAMAccountName=%s)"),
			InsecureSkipVerify: envBool("OT_LDAP_INSECURE_SKIP_VERIFY", false),
		},
	}, nil
}

// Validate reports settings the service must not start with. It is called once
// at startup, not from Load, which also runs on every login.
func (c *Config) Validate() error {
	var errs []error

	var placeholders []string
	for _, v := range []struct{ key, value string }{
		{"OT_DATABASE_DSN", c.Database.DSN},
		{"OT_JWT_SECRET", c.JWT.Secret},
		{"OT_IMPORT_APIKEY", c.Import.APIKey},
		{"OT_ADMIN_PASSWORD", c.Admin.Password},
	} {
		if strings.Contains(v.value, envPlaceholder) {
			placeholders = append(placeholders, v.key)
		}
	}
	if len(placeholders) > 0 {
		errs = append(errs, fmt.Errorf("%s placeholder from env.example still set in %s — replace it (generate secrets with: openssl rand -hex 32)",
			envPlaceholder, strings.Join(placeholders, ", ")))
	}

	if c.JWT.Secret == "change-me-in-production" || len(c.JWT.Secret) < 32 {
		errs = append(errs, errors.New("OT_JWT_SECRET must be set to a random string of at least 32 characters"))
	}
	// An empty import key is valid: it disables the /api/import routes.
	if c.Import.APIKey != "" && len(c.Import.APIKey) < 32 {
		errs = append(errs, errors.New("OT_IMPORT_APIKEY must be at least 32 characters"))
	}
	return errors.Join(errs...)
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
