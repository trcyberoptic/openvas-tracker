package config

import (
	"strings"

	"github.com/spf13/viper"
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
	v := viper.New()

	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("database.dsn", "openvas-tracker:openvas-tracker@tcp(localhost:3306)/openvas-tracker?parseTime=true")
	v.SetDefault("database.maxconns", 25)
	v.SetDefault("database.minconns", 5)
	v.SetDefault("jwt.secret", "change-me-in-production")
	v.SetDefault("jwt.expirehours", 24)
	v.SetDefault("import.apikey", "")

	v.SetEnvPrefix("OT")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	v.SetConfigName(".env")
	v.SetConfigType("env")
	v.AddConfigPath(".")
	_ = v.ReadInConfig()

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
