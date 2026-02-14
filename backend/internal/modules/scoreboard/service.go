package scoreboard

import (
	"context"
	apperrors "ctf-recruit/backend/internal/errors"
	"fmt"
	"sort"
)

const (
	defaultLimit = 20
	maxLimit     = 100
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, limit, offset int) (*ScoreboardResponse, error) {
	limit = normalizeLimit(limit)
	if offset < 0 {
		return nil, apperrors.BadRequest("SCOREBOARD_VALIDATION_ERROR", "Offset must be zero or greater")
	}

	aggregates, err := s.repo.ListAggregates(ctx)
	if err != nil {
		return nil, apperrors.Internal("SCOREBOARD_LIST_FAILED", "Failed to list scoreboard", fmt.Errorf("list scoreboard aggregates: %w", err))
	}

	sort.SliceStable(aggregates, func(i, j int) bool {
		if aggregates[i].TotalPoints != aggregates[j].TotalPoints {
			return aggregates[i].TotalPoints > aggregates[j].TotalPoints
		}
		if !aggregates[i].LastAcceptedAt.Equal(aggregates[j].LastAcceptedAt) {
			return aggregates[i].LastAcceptedAt.Before(aggregates[j].LastAcceptedAt)
		}
		return aggregates[i].UserID < aggregates[j].UserID
	})

	if offset >= len(aggregates) {
		return &ScoreboardResponse{Items: []ScoreboardItem{}, Limit: limit, Offset: offset}, nil
	}

	end := offset + limit
	if end > len(aggregates) {
		end = len(aggregates)
	}

	items := make([]ScoreboardItem, 0, end-offset)
	for idx := offset; idx < end; idx++ {
		row := aggregates[idx]
		items = append(items, ScoreboardItem{
			Rank:        idx + 1,
			UserID:      row.UserID,
			DisplayName: row.DisplayName,
			TotalPoints: row.TotalPoints,
			SolvedCount: row.SolvedCount,
		})
	}

	return &ScoreboardResponse{Items: items, Limit: limit, Offset: offset}, nil
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return defaultLimit
	}
	if limit > maxLimit {
		return maxLimit
	}
	return limit
}
