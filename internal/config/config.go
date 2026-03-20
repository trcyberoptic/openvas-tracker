package config

import (
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	JWT      JWTConfig
	Scanner  ScannerConfig
}

type ServerConfig struct {
	Host string
	Port int
}

type DatabaseConfig struct {
	URL      string
	MaxConns int
	MinConns int
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type JWTConfig struct {
	Secret      string
	ExpireHours int
}

type ScannerConfig struct {
	NmapPath    string
	OpenVASPath string
}

func Load() (*Config, error) {
	v := viper.New()

	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("database.url", "postgres://vulntrack:vulntrack@localhost:5432/vulntrack?sslmode=disable")
	v.SetDefault("database.maxconns", 25)
	v.SetDefault("database.minconns", 5)
	v.SetDefault("redis.addr", "localhost:6379")
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)
	v.SetDefault("jwt.secret", "change-me-in-production")
	v.SetDefault("jwt.expirehours", 24)
	v.SetDefault("scanner.nmappath", "nmap")
	v.SetDefault("scanner.openvaspath", "gvm-cli")

	v.SetEnvPrefix("VT")
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
