package config

import (
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	ServerPort string `env:"SERVER_PORT" envDefault:"8080"`
	ServerEnv  string `env:"SERVER_ENV" envDefault:"development"`

	DBHost     string `env:"DB_HOST" envDefault:"localhost"`
	DBPort     string `env:"DB_PORT" envDefault:"5432"`
	DBUser     string `env:"DB_USER" envDefault:"postgres"`
	DBPassword string `env:"DB_PASSWORD" envDefault:"1221"`
	DBName     string `env:"DB_NAME" envDefault:"bito_db"`
	DBSSLMode  string `env:"DB_SSL_MODE" envDefault:"disable"`

	RedisHost     string `env:"REDIS_HOST" envDefault:"localhost"`
	RedisPort     string `env:"REDIS_PORT" envDefault:"6379"`
	RedisPassword string `env:"REDIS_PASSWORD" envDefault:""`
	RedisDB       int    `env:"REDIS_DB" envDefault:"0"`

	SMTPHost  string `env:"SMTP_HOST" envDefault:"localhost"`
	SMTPPort  string `env:"SMTP_PORT" envDefault:"1025"`
	SMTPFrom  string `env:"SMTP_FROM" envDefault:"no-reply@bito.local"`

	JWTSecret     string        `env:"JWT_SECRET" envDefault:"secretkeysecretkeysecretkey"`
	JWTAccessTTL  time.Duration `env:"JWT_ACCESS_EXPIRY" envDefault:"15m"`
	JWTRefreshTTL time.Duration `env:"JWT_REFRESH_EXPIRY" envDefault:"168h"`
}

func LoadConfig() (*Config, error) {
	cfg := &Config{}
	err := cleanenv.ReadConfig(".env", cfg)
	if err != nil {
		err = cleanenv.ReadEnv(cfg)
		if err != nil {
			return nil, err
		}
	}
	return cfg, nil
}
