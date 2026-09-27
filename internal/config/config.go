package config

import "github.com/ilyakaznacheev/cleanenv"

type Config struct {
	ServerPort  string `env:"SERVER_PORT" envDefault:"8080"`
	ServerEnv   string `env:"SERVER_ENV" envDefault:"development"`
	DBHost      string `env:"DB_HOST" envDefault:"localhost"`
	DBPort      string `env:"DB_PORT" envDefault:"5432"`
	DBUser      string `env:"DB_USER" envDefault:"postgres"`
	DBPassword  string `env:"DB_PASSWORD" envDefault:"1221"`
	DBName      string `env:"DB_NAME" envDefault:"bito_db"`
	DBSSLMode   string `env:"DB_SSL_MODE" envDefault:"disable"`
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