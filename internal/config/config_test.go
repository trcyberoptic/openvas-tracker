package config

import (
	"os"
	"strings"
	"testing"

	"github.com/joho/godotenv"
)

func TestLoad_Defaults(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("expected default port 8080, got %d", cfg.Server.Port)
	}
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("expected default host 0.0.0.0, got %s", cfg.Server.Host)
	}
}

func TestLoad_EnvOverride(t *testing.T) {
	os.Setenv("OT_SERVER_PORT", "9090")
	defer os.Unsetenv("OT_SERVER_PORT")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if cfg.Server.Port != 9090 {
		t.Errorf("expected port 9090, got %d", cfg.Server.Port)
	}
}

func validConfig() *Config {
	return &Config{
		Database: DatabaseConfig{DSN: "otracker:s3cret@tcp(localhost:3306)/openvas-tracker?parseTime=true"},
		JWT:      JWTConfig{Secret: strings.Repeat("a", 64)},
		Import:   ImportConfig{APIKey: strings.Repeat("b", 64)},
		Admin:    AdminConfig{Password: "admin-pw"},
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Config)
		want   []string // substrings of the error; nil = config is valid
	}{
		{"valid", func(*Config) {}, nil},
		{"import disabled", func(c *Config) { c.Import.APIKey = "" }, nil},
		{"admin login disabled", func(c *Config) { c.Admin.Password = "" }, nil},
		{"docker-compose dev values", func(c *Config) {
			c.JWT.Secret = "local-dev-secret-change-in-prod!!"
			c.Import.APIKey = "local-dev-import-key-min-32-characters"
			c.Admin.Password = "admin"
		}, nil},
		{"JWT secret unset", func(c *Config) { c.JWT.Secret = "change-me-in-production" }, []string{"OT_JWT_SECRET must be set"}},
		{"JWT secret too short", func(c *Config) { c.JWT.Secret = "short" }, []string{"OT_JWT_SECRET must be set"}},
		// 32 chars long, so only the placeholder check catches it
		{"JWT secret placeholder", func(c *Config) { c.JWT.Secret = "CHANGEME-use-openssl-rand-hex-32" }, []string{"still set in OT_JWT_SECRET"}},
		{"import key too short", func(c *Config) { c.Import.APIKey = "short" }, []string{"OT_IMPORT_APIKEY must be at least 32 characters"}},
		{"import key placeholder", func(c *Config) { c.Import.APIKey = "CHANGEME-use-openssl-rand-hex-32-min-32-chars" }, []string{"still set in OT_IMPORT_APIKEY"}},
		{"admin password placeholder", func(c *Config) { c.Admin.Password = "CHANGEME" }, []string{"still set in OT_ADMIN_PASSWORD"}},
		{"DSN placeholder", func(c *Config) {
			c.Database.DSN = "otracker:CHANGEME@tcp(localhost:3306)/openvas-tracker?parseTime=true"
		}, []string{"still set in OT_DATABASE_DSN"}},
		{"all problems reported at once", func(c *Config) {
			c.Database.DSN = "otracker:CHANGEME@tcp(localhost:3306)/openvas-tracker?parseTime=true"
			c.JWT.Secret = "short"
		}, []string{"still set in OT_DATABASE_DSN", "OT_JWT_SECRET must be set"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			tt.mutate(cfg)
			err := cfg.Validate()
			if tt.want == nil {
				if err != nil {
					t.Fatalf("Validate() = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("Validate() = nil, want error containing %q", tt.want)
			}
			for _, w := range tt.want {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("Validate() = %q, want it to contain %q", err, w)
				}
			}
		})
	}
}

// The env file the .deb installs must not start the service until every
// placeholder is replaced, so the shipped example has to fail validation.
func TestValidate_RejectsShippedEnvExample(t *testing.T) {
	vars, err := godotenv.Read("../../deploy/openvas-tracker.env.example")
	if err != nil {
		t.Fatalf("read env.example: %v", err)
	}
	for k, v := range vars {
		t.Setenv(k, v)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	err = cfg.Validate()
	if err == nil {
		t.Fatal("Validate() accepted the shipped env.example")
	}
	for _, key := range []string{"OT_DATABASE_DSN", "OT_JWT_SECRET", "OT_IMPORT_APIKEY", "OT_ADMIN_PASSWORD"} {
		if !strings.Contains(err.Error(), key) {
			t.Errorf("Validate() = %q, want it to name %s", err, key)
		}
	}
}
