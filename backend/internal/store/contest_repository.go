package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"ctf/backend/internal/contest"
)

type ContestRepository struct {
	db *sql.DB
}

func NewContestRepository(db *sql.DB) *ContestRepository {
	return &ContestRepository{db: db}
}

func (r *ContestRepository) Current(ctx context.Context) (contest.Contest, error) {
	const query = `
SELECT id, slug, title, description, status, starts_at, ends_at
FROM contests
ORDER BY id ASC
LIMIT 1
`
	return r.queryContest(ctx, query)
}

func (r *ContestRepository) Update(ctx context.Context, input contest.UpdateInput) (contest.Contest, error) {
	status := contest.NormalizeStatus(input.Status)
	startsAt, err := parseOptionalTime(input.StartsAt)
	if err != nil {
		return contest.Contest{}, fmt.Errorf("parse starts_at: %w", err)
	}
	endsAt, err := parseOptionalTime(input.EndsAt)
	if err != nil {
		return contest.Contest{}, fmt.Errorf("parse ends_at: %w", err)
	}
	if startsAt != nil && endsAt != nil && startsAt.(time.Time).After(endsAt.(time.Time)) {
		return contest.Contest{}, fmt.Errorf("starts_at must not be after ends_at")
	}

	const query = `
UPDATE contests
SET status = $1, starts_at = $2, ends_at = $3, updated_at = NOW()
WHERE id = (
	SELECT id FROM contests ORDER BY id ASC LIMIT 1
)
RETURNING id, slug, title, description, status, starts_at, ends_at
`
	return r.queryContest(ctx, query, status, startsAt, endsAt)
}

func (r *ContestRepository) queryContest(ctx context.Context, query string, args ...any) (contest.Contest, error) {
	var current contest.Contest
	var startsAt sql.NullTime
	var endsAt sql.NullTime
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&current.ID,
		&current.Slug,
		&current.Title,
		&current.Description,
		&current.Status,
		&startsAt,
		&endsAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return contest.Contest{}, contest.ErrContestNotFound
		}
		return contest.Contest{}, fmt.Errorf("query contest: %w", err)
	}
	if startsAt.Valid {
		t := startsAt.Time.UTC()
		current.StartsAt = &t
	}
	if endsAt.Valid {
		t := endsAt.Time.UTC()
		current.EndsAt = &t
	}
	current.Status = contest.NormalizeStatus(current.Status)
	return current, nil
}

func parseOptionalTime(value string) (any, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return nil, err
	}
	return parsed.UTC(), nil
}
