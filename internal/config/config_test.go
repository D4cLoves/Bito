package config

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/ilyakaznacheev/cleanenv"
)

func TestDefaultSecretMatchesTag(t *testing.T) {
	t.Setenv("JWT_SECRET", "")
	os.Unsetenv("JWT_SECRET")
	cfg := &Config{}
	if err := cleanenv.ReadEnv(cfg); err != nil {
		t.Fatalf("read env: %v", err)
	}
	if cfg.JWTSecret != defaultJWTSecret {
		t.Errorf("default secret = %q, want the guarded constant %q", cfg.JWTSecret, defaultJWTSecret)
	}
}

func TestValidate_JWTSecretInProduction(t *testing.T) {
	strong := strings.Repeat("k", minJWTSecretLen)
	cases := []struct {
		name   string
		env    string
		secret string
		want   error
	}{
		{"development keeps the default", "development", defaultJWTSecret, nil},
		{"production refuses the default", "production", defaultJWTSecret, ErrDefaultJWTSecret},
		{"production refuses a short secret", "Production", "short-but-custom", ErrShortJWTSecret},
		{"production accepts 32 bytes", "production", strong, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &Config{ServerEnv: tc.env, JWTSecret: tc.secret}
			if err := cfg.Validate(); !errors.Is(err, tc.want) {
				t.Errorf("Validate() = %v, want %v", err, tc.want)
			}
		})
	}
}
