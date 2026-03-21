package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	Import   ImportConfig
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
	}, nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
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
