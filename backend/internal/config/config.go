package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port                string
	DatabaseURL         string
	JWTSecret           string
	JWTTTL              time.Duration
	WorkerPollInterval  time.Duration
	WorkerMaxConcurrency int
}

func Load() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	databaseURL := os.Getenv("DATABASE_URL")
	jwtSecret := os.Getenv("JWT_SECRET")

	jwtTTLRaw := os.Getenv("JWT_TTL")
	if jwtTTLRaw == "" {
		jwtTTLRaw = "24h"
	}

	jwtTTL, err := time.ParseDuration(jwtTTLRaw)
	if err != nil {
		jwtTTL = 24 * time.Hour
	}

	workerPollIntervalRaw := os.Getenv("WORKER_POLL_INTERVAL")
	if workerPollIntervalRaw == "" {
		workerPollIntervalRaw = "2s"
	}
	workerPollInterval, err := time.ParseDuration(workerPollIntervalRaw)
	if err != nil {
		workerPollInterval = 2 * time.Second
	}

	workerMaxConcurrency := 2
	if workerMaxConcurrencyRaw := os.Getenv("WORKER_MAX_CONCURRENCY"); workerMaxConcurrencyRaw != "" {
		if parsed, parseErr := strconv.Atoi(workerMaxConcurrencyRaw); parseErr == nil && parsed > 0 {
			workerMaxConcurrency = parsed
		}
	}

	return Config{
		Port:                 port,
		DatabaseURL:          databaseURL,
		JWTSecret:            jwtSecret,
		JWTTTL:               jwtTTL,
		WorkerPollInterval:   workerPollInterval,
		WorkerMaxConcurrency: workerMaxConcurrency,
	}
}

func (c Config) Validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if c.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}
	if c.WorkerPollInterval <= 0 {
		return fmt.Errorf("WORKER_POLL_INTERVAL must be greater than 0")
	}
	if c.WorkerMaxConcurrency <= 0 {
		return fmt.Errorf("WORKER_MAX_CONCURRENCY must be greater than 0")
	}
	return nil
}
