package config

import (
	"testing"
	"time"
)

func TestConfigValidateAllowsDevelopmentDefaults(t *testing.T) {
	cfg := Config{AppEnv: "development", JWTSecret: defaultDevJWTSecret}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected development config to validate: %v", err)
	}
}

func TestConfigValidateRejectsMissingSecretOutsideDevelopment(t *testing.T) {
	cfg := Config{AppEnv: "production", JWTSecret: "   "}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected missing jwt secret to fail validation")
	}
}

func TestConfigValidateRejectsDefaultSecretsOutsideDevelopment(t *testing.T) {
	tests := []struct {
		name   string
		secret string
	}{
		{name: "current development default", secret: defaultDevJWTSecret},
		{name: "legacy default", secret: legacyDefaultJWTSecret},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{AppEnv: "production", JWTSecret: tt.secret}
			if err := cfg.Validate(); err == nil {
				t.Fatal("expected default jwt secret to fail validation")
			}
		})
	}
}

func TestConfigValidateAllowsExplicitSecretOutsideDevelopment(t *testing.T) {
	cfg := Config{AppEnv: "production", JWTSecret: "replace-with-strong-random-secret"}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected explicit jwt secret to validate: %v", err)
	}
}

func TestRateLimitWindowUsesSeconds(t *testing.T) {
	cfg := Config{}
	if got := cfg.RateLimitWindow("login", 60); got != time.Minute {
		t.Fatalf("expected 1 minute, got %s", got)
	}
	if got := cfg.RateLimitWindow("login", 0); got != 0 {
		t.Fatalf("expected zero duration, got %s", got)
	}
}

func TestConfigValidateAllowsUnsetRuntimePortRange(t *testing.T) {
	cfg := Config{AppEnv: "production", JWTSecret: "replace-with-strong-random-secret"}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected config without runtime ports to validate: %v", err)
	}
}

func TestConfigValidateRejectsInvalidRuntimePortRange(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
	}{
		{
			name: "min greater than max",
			cfg:  Config{AppEnv: "production", JWTSecret: "ok", RuntimePortMin: 20010, RuntimePortMax: 20000},
		},
		{
			name: "below 1024",
			cfg:  Config{AppEnv: "production", JWTSecret: "ok", RuntimePortMin: 80, RuntimePortMax: 100},
		},
		{
			name: "max above 65535",
			cfg:  Config{AppEnv: "production", JWTSecret: "ok", RuntimePortMin: 20000, RuntimePortMax: 70000},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.cfg.Validate(); err == nil {
				t.Fatalf("expected validation to fail")
			}
		})
	}
}

func TestConfigValidateAllowsEmptyRuntimePublicBaseURL(t *testing.T) {
	cfg := Config{AppEnv: "production", JWTSecret: "ok", RuntimePublicBaseURL: "   "}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected validation to allow empty runtime public base URL: %v", err)
	}
}
