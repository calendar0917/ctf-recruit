package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"ctf/backend/internal/admin"
)

type AdminRepository struct {
	db *sql.DB
}

func NewAdminRepository(db *sql.DB) *AdminRepository {
	return &AdminRepository{db: db}
}

func (r *AdminRepository) ListChallenges(ctx context.Context) ([]admin.ChallengeSummary, error) {
	const query = `
SELECT c.id, c.slug, c.title, cat.slug, c.points, c.visible, c.dynamic_enabled
FROM challenges c
JOIN categories cat ON cat.id = c.category_id
ORDER BY cat.sort_order ASC, c.sort_order ASC, c.id ASC
`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list admin challenges: %w", err)
	}
	defer rows.Close()

	items := make([]admin.ChallengeSummary, 0)
	for rows.Next() {
		var item admin.ChallengeSummary
		if err := rows.Scan(&item.ID, &item.Slug, &item.Title, &item.Category, &item.Points, &item.Visible, &item.DynamicEnabled); err != nil {
			return nil, fmt.Errorf("scan admin challenge: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate admin challenges: %w", err)
	}
	return items, nil
}

func (r *AdminRepository) CreateChallenge(ctx context.Context, input admin.UpsertChallengeInput) (admin.ChallengeSummary, error) {
	const query = `
INSERT INTO challenges (
    contest_id,
    category_id,
    slug,
    title,
    description,
    points,
    difficulty,
    flag_type,
    flag_value,
    dynamic_enabled,
    visible,
    sort_order
)
SELECT c.id, cat.id, $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
FROM contests c, categories cat
WHERE c.slug = 'recruit-2025' AND cat.slug = $11
RETURNING id
`
	var id int64
	err := r.db.QueryRowContext(ctx, query,
		input.Slug,
		input.Title,
		input.Description,
		input.Points,
		input.Difficulty,
		input.FlagType,
		input.FlagValue,
		input.DynamicEnabled,
		input.Visible,
		input.SortOrder,
		input.CategorySlug,
	).Scan(&id)
	if err != nil {
		return admin.ChallengeSummary{}, fmt.Errorf("create challenge: %w", err)
	}
	return admin.ChallengeSummary{
		ID:             id,
		Slug:           input.Slug,
		Title:          input.Title,
		Category:       input.CategorySlug,
		Points:         input.Points,
		Visible:        input.Visible,
		DynamicEnabled: input.DynamicEnabled,
	}, nil
}

func (r *AdminRepository) UpdateChallenge(ctx context.Context, challengeID int64, input admin.UpsertChallengeInput) (admin.ChallengeSummary, error) {
	const query = `
UPDATE challenges c
SET
    category_id = cat.id,
    slug = $2,
    title = $3,
    description = $4,
    points = $5,
    difficulty = $6,
    flag_type = $7,
    flag_value = $8,
    dynamic_enabled = $9,
    visible = $10,
    sort_order = $11,
    updated_at = NOW()
FROM categories cat
WHERE c.id = $1 AND cat.slug = $12
RETURNING c.id
`
	var id int64
	err := r.db.QueryRowContext(ctx, query,
		challengeID,
		input.Slug,
		input.Title,
		input.Description,
		input.Points,
		input.Difficulty,
		input.FlagType,
		input.FlagValue,
		input.DynamicEnabled,
		input.Visible,
		input.SortOrder,
		input.CategorySlug,
	).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return admin.ChallengeSummary{}, admin.ErrResourceNotFound
		}
		return admin.ChallengeSummary{}, fmt.Errorf("update challenge: %w", err)
	}
	return admin.ChallengeSummary{
		ID:             id,
		Slug:           input.Slug,
		Title:          input.Title,
		Category:       input.CategorySlug,
		Points:         input.Points,
		Visible:        input.Visible,
		DynamicEnabled: input.DynamicEnabled,
	}, nil
}

func (r *AdminRepository) ListAnnouncements(ctx context.Context) ([]admin.Announcement, error) {
	const query = `
SELECT id, title, content, pinned, published, published_at
FROM announcements
ORDER BY pinned DESC, created_at DESC
`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list admin announcements: %w", err)
	}
	defer rows.Close()

	items := make([]admin.Announcement, 0)
	for rows.Next() {
		var (
			item        admin.Announcement
			publishedAt sql.NullTime
		)
		if err := rows.Scan(&item.ID, &item.Title, &item.Content, &item.Pinned, &item.Published, &publishedAt); err != nil {
			return nil, fmt.Errorf("scan admin announcement: %w", err)
		}
		if publishedAt.Valid {
			t := publishedAt.Time
			item.PublishedAt = &t
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate admin announcements: %w", err)
	}
	return items, nil
}

func (r *AdminRepository) CreateAnnouncement(ctx context.Context, actorUserID int64, input admin.CreateAnnouncementInput) (admin.Announcement, error) {
	const query = `
INSERT INTO announcements (contest_id, title, content, pinned, published, published_at, created_by)
SELECT c.id, $1, $2, $3, $4, CASE WHEN $4 THEN NOW() ELSE NULL END, $5
FROM contests c
WHERE c.slug = 'recruit-2025'
RETURNING id, published_at
`
	var (
		id          int64
		publishedAt sql.NullTime
	)
	err := r.db.QueryRowContext(ctx, query,
		input.Title,
		input.Content,
		input.Pinned,
		input.Published,
		actorUserID,
	).Scan(&id, &publishedAt)
	if err != nil {
		return admin.Announcement{}, fmt.Errorf("create announcement: %w", err)
	}
	result := admin.Announcement{
		ID:        id,
		Title:     input.Title,
		Content:   input.Content,
		Pinned:    input.Pinned,
		Published: input.Published,
	}
	if publishedAt.Valid {
		t := publishedAt.Time
		result.PublishedAt = &t
	}
	return result, nil
}

func (r *AdminRepository) ListSubmissions(ctx context.Context) ([]admin.SubmissionRecord, error) {
	const query = `
SELECT s.id, c.id, c.slug, u.username, s.is_correct, s.submitted_at, s.source_ip
FROM submissions s
JOIN challenges c ON c.id = s.challenge_id
JOIN users u ON u.id = s.user_id
ORDER BY s.submitted_at DESC, s.id DESC
`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list submissions: %w", err)
	}
	defer rows.Close()

	items := make([]admin.SubmissionRecord, 0)
	for rows.Next() {
		var item admin.SubmissionRecord
		if err := rows.Scan(&item.ID, &item.ChallengeID, &item.ChallengeSlug, &item.Username, &item.Correct, &item.SubmittedAt, &item.SourceIP); err != nil {
			return nil, fmt.Errorf("scan submission: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate submissions: %w", err)
	}
	return items, nil
}

func (r *AdminRepository) ListInstances(ctx context.Context) ([]admin.InstanceRecord, error) {
	const query = `
SELECT ci.id, c.id, c.slug, u.username, ci.status, ci.host_port, ci.expires_at, ci.terminated_at, ci.docker_container_id
FROM challenge_instances ci
JOIN challenges c ON c.id = ci.challenge_id
JOIN users u ON u.id = ci.user_id
ORDER BY ci.created_at DESC, ci.id DESC
`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list instances: %w", err)
	}
	defer rows.Close()

	items := make([]admin.InstanceRecord, 0)
	for rows.Next() {
		var (
			item         admin.InstanceRecord
			terminatedAt sql.NullTime
		)
		if err := rows.Scan(&item.ID, &item.ChallengeID, &item.ChallengeSlug, &item.Username, &item.Status, &item.HostPort, &item.ExpiresAt, &terminatedAt, &item.ContainerID); err != nil {
			return nil, fmt.Errorf("scan instance: %w", err)
		}
		if terminatedAt.Valid {
			t := terminatedAt.Time
			item.TerminatedAt = &t
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate instances: %w", err)
	}
	return items, nil
}

func (r *AdminRepository) TerminateInstance(ctx context.Context, instanceID int64, terminatedAt time.Time) (admin.InstanceRecord, error) {
	const query = `
UPDATE challenge_instances ci
SET status = 'terminated', terminated_at = $2, updated_at = NOW()
FROM challenges c, users u
WHERE ci.id = $1 AND c.id = ci.challenge_id AND u.id = ci.user_id
RETURNING ci.id, c.id, c.slug, u.username, ci.status, ci.host_port, ci.expires_at, ci.terminated_at, ci.docker_container_id
`
	var (
		item       admin.InstanceRecord
		terminated sql.NullTime
	)
	err := r.db.QueryRowContext(ctx, query, instanceID, terminatedAt).Scan(
		&item.ID,
		&item.ChallengeID,
		&item.ChallengeSlug,
		&item.Username,
		&item.Status,
		&item.HostPort,
		&item.ExpiresAt,
		&terminated,
		&item.ContainerID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return admin.InstanceRecord{}, admin.ErrResourceNotFound
		}
		return admin.InstanceRecord{}, fmt.Errorf("terminate instance: %w", err)
	}
	if terminated.Valid {
		t := terminated.Time
		item.TerminatedAt = &t
	}
	return item, nil
}
