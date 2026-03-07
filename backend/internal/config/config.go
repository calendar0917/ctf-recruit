package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port                        string
	DatabaseURL                 string
	JWTSecret                   string
	JWTTTL                      time.Duration
	InstanceAccessHost          string
	WorkerPollInterval          time.Duration
	InstanceSweeperPollInterval time.Duration
	WorkerMaxConcurrency        int
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

	instanceSweeperPollIntervalRaw := os.Getenv("INSTANCE_SWEEPER_POLL_INTERVAL")
	if instanceSweeperPollIntervalRaw == "" {
		instanceSweeperPollIntervalRaw = "5s"
	}
	instanceSweeperPollInterval, err := time.ParseDuration(instanceSweeperPollIntervalRaw)
	if err != nil {
		instanceSweeperPollInterval = 5 * time.Second
	}

	workerMaxConcurrency := 2
	if workerMaxConcurrencyRaw := os.Getenv("WORKER_MAX_CONCURRENCY"); workerMaxConcurrencyRaw != "" {
		if parsed, parseErr := strconv.Atoi(workerMaxConcurrencyRaw); parseErr == nil && parsed > 0 {
			workerMaxConcurrency = parsed
		}
	}

	instanceAccessHost := os.Getenv("INSTANCE_ACCESS_HOST")
	if instanceAccessHost == "" {
		instanceAccessHost = "localhost"
	}

	return Config{
		Port:                        port,
		DatabaseURL:                 databaseURL,
		JWTSecret:                   jwtSecret,
		JWTTTL:                      jwtTTL,
		InstanceAccessHost:          instanceAccessHost,
		WorkerPollInterval:          workerPollInterval,
		InstanceSweeperPollInterval: instanceSweeperPollInterval,
		WorkerMaxConcurrency:        workerMaxConcurrency,
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
	if c.InstanceSweeperPollInterval <= 0 {
		return fmt.Errorf("INSTANCE_SWEEPER_POLL_INTERVAL must be greater than 0")
	}
	if c.WorkerMaxConcurrency <= 0 {
		return fmt.Errorf("WORKER_MAX_CONCURRENCY must be greater than 0")
	}
	return nil
}
