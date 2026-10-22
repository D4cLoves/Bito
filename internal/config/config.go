package config

import (
	"errors"
	"os"
	"strings"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

const (
	EnvProduction    = "production"
	defaultJWTSecret = "secretkeysecretkeysecretkey"
	minJWTSecretLen  = 32
)

var (
	ErrDefaultJWTSecret = errors.New("JWT_SECRET must be set to a non-default value in production")
	ErrShortJWTSecret   = errors.New("JWT_SECRET must be at least 32 bytes in production")
)

type Config struct {
	ServerPort string `env:"SERVER_PORT" env-default:"8080"`
	ServerEnv  string `env:"SERVER_ENV" env-default:"development"`

	DBHost     string `env:"DB_HOST" env-default:"localhost"`
	DBPort     string `env:"DB_PORT" env-default:"5432"`
	DBUser     string `env:"DB_USER" env-default:"postgres"`
	DBPassword string `env:"DB_PASSWORD" env-default:"1221"`
	DBName     string `env:"DB_NAME" env-default:"bito_db"`
	DBSSLMode  string `env:"DB_SSLMODE" env-default:"disable"`

	RedisHost     string `env:"REDIS_HOST" env-default:"localhost"`
	RedisPort     string `env:"REDIS_PORT" env-default:"6379"`
	RedisPassword string `env:"REDIS_PASSWORD" env-default:""`
	RedisDB       int    `env:"REDIS_DB" env-default:"0"`

	SMTPHost     string `env:"SMTP_HOST" env-default:"localhost"`
	SMTPPort     string `env:"SMTP_PORT" env-default:"1025"`
	SMTPFrom     string `env:"SMTP_FROM" env-default:"no-reply@bito.local"`
	SMTPUser     string `env:"SMTP_USER" env-default:""`
	SMTPPassword string `env:"SMTP_PASSWORD" env-default:""`

	JWTSecret     string        `env:"JWT_SECRET" env-default:"secretkeysecretkeysecretkey"`
	JWTAccessTTL  time.Duration `env:"JWT_ACCESS_EXPIRY" env-default:"15m"`
	JWTRefreshTTL time.Duration `env:"JWT_REFRESH_EXPIRY" env-default:"168h"`

	AllowedOrigins []string `env:"ALLOWED_ORIGINS" env-default:"http://localhost:5173,http://127.0.0.1:5173"`
}

func LoadConfig() (*Config, error) {
	cfg := &Config{}

	envFiles := []string{".env", "../.env", "../../.env"}
	var loaded bool
	for _, f := range envFiles {
		if _, err := os.Stat(f); err == nil {
			if err := cleanenv.ReadConfig(f, cfg); err == nil {
				loaded = true
				break
			}
		}
	}

	if !loaded {
		if err := cleanenv.ReadEnv(cfg); err != nil {
			return nil, err
		}
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) IsProduction() bool {
	return strings.EqualFold(strings.TrimSpace(c.ServerEnv), EnvProduction)
}

func (c *Config) Validate() error {
	if !c.IsProduction() {
		return nil
	}
	if c.JWTSecret == defaultJWTSecret {
		return ErrDefaultJWTSecret
	}
	if len(c.JWTSecret) < minJWTSecretLen {
		return ErrShortJWTSecret
	}
	return nil
}
