package config

import "os"

type Config struct {
	HTTPAddr                    string
	AppEnv                      string
	DatabaseURL                 string
	JWTSecret                   string
	InstanceSweeperPollInterval string
	DockerSocketPath            string
	PublicBaseURL               string
}

func Load() Config {
	return Config{
		HTTPAddr:                    getEnv("HTTP_ADDR", ":8080"),
		AppEnv:                      getEnv("APP_ENV", "development"),
		DatabaseURL:                 getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/ctf?sslmode=disable"),
		JWTSecret:                   getEnv("JWT_SECRET", "change-me"),
		InstanceSweeperPollInterval: getEnv("INSTANCE_SWEEPER_POLL_INTERVAL", "30s"),
		DockerSocketPath:            getEnv("DOCKER_SOCKET_PATH", "/var/run/docker.sock"),
		PublicBaseURL:               getEnv("PUBLIC_BASE_URL", "http://localhost:8080"),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
