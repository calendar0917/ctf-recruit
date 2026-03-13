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
	HTTPAddr                         string
	AppEnv                           string
	DatabaseURL                      string
	JWTSecret                        string
	JWTTTL                           time.Duration
	InstanceSweeperPollInterval      string
	DockerSocketPath                 string
	PublicBaseURL                    string
	RuntimePublicBaseURL             string
	RuntimePortMin                   int
	RuntimePortMax                   int
	RuntimeBindAddr                  string
	AttachmentStorageDir             string
	RedisAddr                        string
	RedisPassword                    string
	RedisDB                          int
	RedisKeyPrefix                   string
	LoginRateLimitWindowSeconds      int
	LoginRateLimitMax                int
	RegisterRateLimitWindowSeconds   int
	RegisterRateLimitMax             int
	SubmissionRateLimitWindowSeconds int
	SubmissionRateLimitMax           int
	AdminWriteRateLimitWindowSeconds int
	AdminWriteRateLimitMax           int
}

func Load() Config {
	return Config{
		HTTPAddr:                         getEnv("HTTP_ADDR", ":8080"),
		AppEnv:                           getEnv("APP_ENV", developmentEnv),
		DatabaseURL:                      getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/ctf?sslmode=disable"),
		JWTSecret:                        getEnv("JWT_SECRET", defaultDevJWTSecret),
		JWTTTL:                           getDurationEnv("JWT_TTL", 24*time.Hour),
		InstanceSweeperPollInterval:      getEnv("INSTANCE_SWEEPER_POLL_INTERVAL", "30s"),
		DockerSocketPath:                 getEnv("DOCKER_SOCKET_PATH", "/var/run/docker.sock"),
		PublicBaseURL:                    getEnv("PUBLIC_BASE_URL", "http://localhost:8080"),
		RuntimePublicBaseURL:             getEnv("RUNTIME_PUBLIC_BASE_URL", getEnv("PUBLIC_BASE_URL", "http://localhost:8080")),
		RuntimePortMin:                   getIntEnv("RUNTIME_PORT_MIN", 0),
		RuntimePortMax:                   getIntEnv("RUNTIME_PORT_MAX", 0),
		RuntimeBindAddr:                  getEnv("RUNTIME_BIND_ADDR", "127.0.0.1"),
		AttachmentStorageDir:             getEnv("ATTACHMENT_STORAGE_DIR", "/tmp/ctf-attachments"),
		RedisAddr:                        getEnv("REDIS_ADDR", "redis:6379"),
		RedisPassword:                    getEnv("REDIS_PASSWORD", ""),
		RedisDB:                          getIntEnv("REDIS_DB", 0),
		RedisKeyPrefix:                   getEnv("REDIS_KEY_PREFIX", "ctf:"),
		LoginRateLimitWindowSeconds:      getIntEnv("LOGIN_RATE_LIMIT_WINDOW_SECONDS", 60),
		LoginRateLimitMax:                getIntEnv("LOGIN_RATE_LIMIT_MAX", 10),
		RegisterRateLimitWindowSeconds:   getIntEnv("REGISTER_RATE_LIMIT_WINDOW_SECONDS", 300),
		RegisterRateLimitMax:             getIntEnv("REGISTER_RATE_LIMIT_MAX", 5),
		SubmissionRateLimitWindowSeconds: getIntEnv("SUBMISSION_RATE_LIMIT_WINDOW_SECONDS", 60),
		SubmissionRateLimitMax:           getIntEnv("SUBMISSION_RATE_LIMIT_MAX", 10),
		AdminWriteRateLimitWindowSeconds: getIntEnv("ADMIN_WRITE_RATE_LIMIT_WINDOW_SECONDS", 60),
		AdminWriteRateLimitMax:           getIntEnv("ADMIN_WRITE_RATE_LIMIT_MAX", 30),
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
	if err := validateRuntimePortRange(c.RuntimePortMin, c.RuntimePortMax); err != nil {
		return err
	}
	return nil
}

func validateRuntimePortRange(minPort, maxPort int) error {
	if minPort == 0 && maxPort == 0 {
		return nil
	}
	if minPort <= 0 || maxPort <= 0 {
		return fmt.Errorf("RUNTIME_PORT_MIN and RUNTIME_PORT_MAX must be positive when set")
	}
	if minPort > maxPort {
		return fmt.Errorf("RUNTIME_PORT_MIN must be <= RUNTIME_PORT_MAX")
	}
	if minPort < 1024 {
		return fmt.Errorf("RUNTIME_PORT_MIN must be >= 1024")
	}
	if maxPort > 65535 {
		return fmt.Errorf("RUNTIME_PORT_MAX must be <= 65535")
	}
	return nil
}

func (c Config) IsDevelopment() bool {
	return normalizeAppEnv(c.AppEnv) == developmentEnv
}

func (c Config) RateLimitWindow(_ string, seconds int) time.Duration {
	if seconds <= 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
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
