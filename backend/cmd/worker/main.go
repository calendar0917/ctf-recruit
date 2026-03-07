package main

import (
	"context"
	"ctf-recruit/backend/internal/config"
	"ctf-recruit/backend/internal/modules/instance"
	"ctf-recruit/backend/internal/modules/judge"
	"ctf-recruit/backend/internal/platform"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	appCtx, err := platform.NewApp(cfg)
	if err != nil {
		slog.Error("failed to initialize app", "error", err)
		os.Exit(1)
	}

	repo := judge.NewRepository(appCtx.DB)
	queue := judge.NewQueue(repo)
	worker := judge.NewWorker(queue, judge.NewMockExecutor())
	instanceRepo := instance.NewRepository(appCtx.DB)
	instanceSweeper := instance.NewSweeper(instanceRepo, instance.NewDockerController(appCtx.Cfg.InstanceAccessHost))

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	ticker := time.NewTicker(cfg.WorkerPollInterval)
	defer ticker.Stop()
	instanceSweepTicker := time.NewTicker(cfg.InstanceSweeperPollInterval)
	defer instanceSweepTicker.Stop()

	slog.Info("worker started",
		"judgePollInterval", cfg.WorkerPollInterval.String(),
		"instanceSweeperPollInterval", cfg.InstanceSweeperPollInterval.String(),
		"maxConcurrency", cfg.WorkerMaxConcurrency,
	)

	for {
		select {
		case <-ctx.Done():
			slog.Info("worker stopped")
			return
		case <-ticker.C:
			processed, err := worker.ProcessOnce(context.Background(), cfg.WorkerMaxConcurrency)
			if err != nil {
				slog.Error("judge worker cycle failed", "error", err)
				continue
			}
			if processed > 0 {
				slog.Info("judge worker processed jobs", "count", processed)
			}
		case <-instanceSweepTicker.C:
			expired, err := instanceSweeper.ProcessOnce(context.Background())
			if err != nil {
				slog.Error("instance sweeper cycle failed", "error", err)
				continue
			}
			if expired > 0 {
				slog.Info("instance sweeper expired instances", "count", expired)
			}
		}
	}
}
