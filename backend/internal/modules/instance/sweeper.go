package instance

import (
	"context"
	"errors"
	"log/slog"
	"strings"
)

const defaultSweepBatchSize = 20

type Sweeper struct {
	repo      Repository
	runtime   RuntimeController
	clock     Clock
	logger    *slog.Logger
	batchSize int
}

func NewSweeper(repo Repository, runtime RuntimeController) *Sweeper {
	return &Sweeper{
		repo:      repo,
		runtime:   runtime,
		clock:     realClock{},
		logger:    slog.Default(),
		batchSize: defaultSweepBatchSize,
	}
}

func (s *Sweeper) WithClock(clock Clock) *Sweeper {
	if clock != nil {
		s.clock = clock
	}
	return s
}

func (s *Sweeper) WithLogger(logger *slog.Logger) *Sweeper {
	if logger != nil {
		s.logger = logger
	}
	return s
}

func (s *Sweeper) WithBatchSize(size int) *Sweeper {
	if size > 0 {
		s.batchSize = size
	}
	return s
}

func (s *Sweeper) ProcessOnce(ctx context.Context) (int, error) {
	now := s.clock.Now().UTC()
	items, err := s.repo.ListExpirableBefore(ctx, now, s.batchSize)
	if err != nil {
		return 0, err
	}

	processed := 0
	for _, item := range items {
		logger := s.logger.With(
			"instanceId", item.ID.String(),
			"userId", item.UserID.String(),
			"challengeId", item.ChallengeID.String(),
		)
		if item.ContainerID != nil {
			logger = logger.With("containerId", strings.TrimSpace(*item.ContainerID))
		}

		if s.runtime != nil && item.ContainerID != nil {
			containerID := strings.TrimSpace(*item.ContainerID)
			if containerID != "" {
				if stopErr := s.runtime.Stop(ctx, containerID); stopErr != nil {
					logger.Error("instance sweeper stop failed", "error", stopErr)
					continue
				}
			}
		}

		if _, transErr := s.repo.TransitionStatus(ctx, item.ID, item.UserID, StatusExpired, now, defaultTTL, defaultCooldown, item.ContainerID); transErr != nil {
			if errors.Is(transErr, ErrInvalidTransition) || errors.Is(transErr, ErrInstanceNotFound) {
				logger.Info("instance sweeper skipped transition due to concurrent state change", "error", transErr)
				continue
			}
			logger.Error("instance sweeper transition failed", "error", transErr)
			continue
		}

		processed++
		logger.Info("instance expired by sweeper")
	}

	return processed, nil
}
