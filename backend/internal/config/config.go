package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	developmentEnv         = "development"
	defaultDevJWTSecret    = "dev-only-insecure-jwt-secret"
	legacyDefaultJWTSecret = "change-me"
)

type Config struct {
	HTTPAddr                    string
	AppEnv                      string
	DatabaseURL                 string
	JWTSecret                   string
	JWTTTL                      time.Duration
	InstanceSweeperPollInterval string
	DockerSocketPath            string
	PublicBaseURL               string
	AttachmentStorageDir        string
	SubmissionRateLimitWindow   time.Duration
	SubmissionRateLimitMax      int
}

func Load() Config {
	return Config{
		HTTPAddr:                    getEnv("HTTP_ADDR", ":8080"),
		AppEnv:                      getEnv("APP_ENV", developmentEnv),
		DatabaseURL:                 getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/ctf?sslmode=disable"),
		JWTSecret:                   getEnv("JWT_SECRET", defaultDevJWTSecret),
		JWTTTL:                      getDurationEnv("JWT_TTL", 24*time.Hour),
		InstanceSweeperPollInterval: getEnv("INSTANCE_SWEEPER_POLL_INTERVAL", "30s"),
		DockerSocketPath:            getEnv("DOCKER_SOCKET_PATH", "/var/run/docker.sock"),
		PublicBaseURL:               getEnv("PUBLIC_BASE_URL", "http://localhost:8080"),
		AttachmentStorageDir:        getEnv("ATTACHMENT_STORAGE_DIR", "/tmp/ctf-attachments"),
		SubmissionRateLimitWindow:   getDurationEnv("SUBMISSION_RATE_LIMIT_WINDOW", time.Minute),
		SubmissionRateLimitMax:      getIntEnv("SUBMISSION_RATE_LIMIT_MAX", 10),
	}
}

func (c Config) Validate() error {
	if c.IsDevelopment() {
		return nil
	}

	secret := strings.TrimSpace(c.JWTSecret)
	if secret == "" {
		return fmt.Errorf("JWT_SECRET must be set when APP_ENV=%s", normalizeAppEnv(c.AppEnv))
	}
	if secret == defaultDevJWTSecret || secret == legacyDefaultJWTSecret {
		return fmt.Errorf("JWT_SECRET must not use a development default when APP_ENV=%s", normalizeAppEnv(c.AppEnv))
	}
	return nil
}

func (c Config) IsDevelopment() bool {
	return normalizeAppEnv(c.AppEnv) == developmentEnv
}

func normalizeAppEnv(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return developmentEnv
	}
	return normalized
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getIntEnv(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
