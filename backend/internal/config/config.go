package config

import (
	"os"
	"strconv"
	"time"
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
		AppEnv:                      getEnv("APP_ENV", "development"),
		DatabaseURL:                 getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/ctf?sslmode=disable"),
		JWTSecret:                   getEnv("JWT_SECRET", "change-me"),
		JWTTTL:                      getDurationEnv("JWT_TTL", 24*time.Hour),
		InstanceSweeperPollInterval: getEnv("INSTANCE_SWEEPER_POLL_INTERVAL", "30s"),
		DockerSocketPath:            getEnv("DOCKER_SOCKET_PATH", "/var/run/docker.sock"),
		PublicBaseURL:               getEnv("PUBLIC_BASE_URL", "http://localhost:8080"),
		AttachmentStorageDir:        getEnv("ATTACHMENT_STORAGE_DIR", "/tmp/ctf-attachments"),
		SubmissionRateLimitWindow:   getDurationEnv("SUBMISSION_RATE_LIMIT_WINDOW", time.Minute),
		SubmissionRateLimitMax:      getIntEnv("SUBMISSION_RATE_LIMIT_MAX", 10),
	}
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
