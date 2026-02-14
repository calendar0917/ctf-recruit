package scoreboard

import (
	"context"
	apperrors "ctf-recruit/backend/internal/errors"
	"errors"
	"testing"
	"time"
)

type mockRepository struct {
	items []ScoreboardAggregate
	err   error
}

func (m mockRepository) ListAggregates(_ context.Context) ([]ScoreboardAggregate, error) {
	if m.err != nil {
		return nil, m.err
	}
	copied := make([]ScoreboardAggregate, 0, len(m.items))
	copied = append(copied, m.items...)
	return copied, nil
}

func TestServiceListRanksByPointsThenTieBreakers(t *testing.T) {
	t0 := time.Date(2026, 2, 13, 10, 0, 0, 0, time.UTC)
	t1 := time.Date(2026, 2, 13, 12, 0, 0, 0, time.UTC)

	svc := NewService(mockRepository{items: []ScoreboardAggregate{
		{UserID: "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", DisplayName: "B", TotalPoints: 300, SolvedCount: 3, LastAcceptedAt: t0},
		{UserID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", DisplayName: "A", TotalPoints: 300, SolvedCount: 3, LastAcceptedAt: t0},
		{UserID: "cccccccc-cccc-cccc-cccc-cccccccccccc", DisplayName: "C", TotalPoints: 300, SolvedCount: 3, LastAcceptedAt: t1},
		{UserID: "dddddddd-dddd-dddd-dddd-dddddddddddd", DisplayName: "D", TotalPoints: 250, SolvedCount: 2, LastAcceptedAt: t0},
	}})

	resp, err := svc.List(context.Background(), 10, 0)
	if err != nil {
		t.Fatalf("expected list success, got error: %v", err)
	}
	if len(resp.Items) != 4 {
		t.Fatalf("expected 4 items, got %d", len(resp.Items))
	}

	if resp.Items[0].UserID != "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" {
		t.Fatalf("expected first item user a..., got %s", resp.Items[0].UserID)
	}
	if resp.Items[1].UserID != "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb" {
		t.Fatalf("expected second item user b..., got %s", resp.Items[1].UserID)
	}
	if resp.Items[2].UserID != "cccccccc-cccc-cccc-cccc-cccccccccccc" {
		t.Fatalf("expected third item user c..., got %s", resp.Items[2].UserID)
	}
	if resp.Items[3].UserID != "dddddddd-dddd-dddd-dddd-dddddddddddd" {
		t.Fatalf("expected fourth item user d..., got %s", resp.Items[3].UserID)
	}

	if resp.Items[0].Rank != 1 || resp.Items[1].Rank != 2 || resp.Items[2].Rank != 3 || resp.Items[3].Rank != 4 {
		t.Fatalf("expected sequential ranks 1..4, got %d, %d, %d, %d", resp.Items[0].Rank, resp.Items[1].Rank, resp.Items[2].Rank, resp.Items[3].Rank)
	}
}

func TestServiceListPaginationPreservesGlobalRank(t *testing.T) {
	t0 := time.Date(2026, 2, 13, 10, 0, 0, 0, time.UTC)

	svc := NewService(mockRepository{items: []ScoreboardAggregate{
		{UserID: "u1", DisplayName: "U1", TotalPoints: 300, SolvedCount: 3, LastAcceptedAt: t0},
		{UserID: "u2", DisplayName: "U2", TotalPoints: 200, SolvedCount: 2, LastAcceptedAt: t0},
		{UserID: "u3", DisplayName: "U3", TotalPoints: 100, SolvedCount: 1, LastAcceptedAt: t0},
	}})

	resp, err := svc.List(context.Background(), 2, 1)
	if err != nil {
		t.Fatalf("expected list success, got error: %v", err)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(resp.Items))
	}
	if resp.Items[0].UserID != "u2" || resp.Items[0].Rank != 2 {
		t.Fatalf("expected first paginated item u2 rank 2, got %s rank %d", resp.Items[0].UserID, resp.Items[0].Rank)
	}
	if resp.Items[1].UserID != "u3" || resp.Items[1].Rank != 3 {
		t.Fatalf("expected second paginated item u3 rank 3, got %s rank %d", resp.Items[1].UserID, resp.Items[1].Rank)
	}
}

func TestServiceListValidatesOffset(t *testing.T) {
	svc := NewService(mockRepository{})

	_, err := svc.List(context.Background(), 10, -1)
	if err == nil {
		t.Fatal("expected validation error")
	}

	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected app error, got %T", err)
	}
	if appErr.Code != "SCOREBOARD_VALIDATION_ERROR" {
		t.Fatalf("expected SCOREBOARD_VALIDATION_ERROR, got %s", appErr.Code)
	}
}
