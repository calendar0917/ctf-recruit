package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"ctf/backend/internal/game"
)

type GameRepository struct {
	db *sql.DB
}

func NewGameRepository(db *sql.DB) *GameRepository {
	return &GameRepository{db: db}
}

func (r *GameRepository) ListAnnouncements(ctx context.Context) ([]game.Announcement, error) {
	const query = `
SELECT id, title, content, pinned, published_at
FROM announcements
WHERE published = TRUE
ORDER BY pinned DESC, published_at DESC NULLS LAST, created_at DESC
`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list announcements: %w", err)
	}
	defer rows.Close()

	items := make([]game.Announcement, 0)
	for rows.Next() {
		var (
			item        game.Announcement
			publishedAt sql.NullTime
		)
		if err := rows.Scan(&item.ID, &item.Title, &item.Content, &item.Pinned, &publishedAt); err != nil {
			return nil, fmt.Errorf("scan announcement: %w", err)
		}
		if publishedAt.Valid {
			t := publishedAt.Time
			item.PublishedAt = &t
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate announcements: %w", err)
	}
	return items, nil
}

func (r *GameRepository) GetChallenge(ctx context.Context, challengeRef string) (game.Challenge, string, error) {
	const challengeQuery = `
SELECT c.id, c.slug, c.title, cat.slug, c.points, c.difficulty, c.description, c.flag_type, c.dynamic_enabled, c.flag_value
FROM challenges c
JOIN categories cat ON cat.id = c.category_id
WHERE c.visible = TRUE AND (c.id::text = $1 OR lower(c.slug) = lower($1))
LIMIT 1
`

	var (
		challenge game.Challenge
		flagValue string
	)
	err := r.db.QueryRowContext(ctx, challengeQuery, challengeRef).Scan(
		&challenge.ID,
		&challenge.Slug,
		&challenge.Title,
		&challenge.Category,
		&challenge.Points,
		&challenge.Difficulty,
		&challenge.Description,
		&challenge.FlagType,
		&challenge.Dynamic,
		&flagValue,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return game.Challenge{}, "", game.ErrChallengeNotFound
		}
		return game.Challenge{}, "", fmt.Errorf("get challenge: %w", err)
	}

	attachments, err := r.listChallengeAttachments(ctx, challenge.ID)
	if err != nil {
		return game.Challenge{}, "", err
	}
	challenge.Attachments = attachments

	return challenge, flagValue, nil
}

func (r *GameRepository) GetChallengeAttachment(ctx context.Context, challengeRef string, attachmentID int64) (game.Attachment, string, error) {
	challenge, _, err := r.GetChallenge(ctx, challengeRef)
	if err != nil {
		return game.Attachment{}, "", err
	}

	const query = `
SELECT id, filename, storage_path, content_type, size_bytes
FROM challenge_attachments
WHERE challenge_id = $1 AND id = $2
LIMIT 1
`
	var (
		attachment  game.Attachment
		storagePath string
	)
	if err := r.db.QueryRowContext(ctx, query, challenge.ID, attachmentID).Scan(
		&attachment.ID,
		&attachment.Filename,
		&storagePath,
		&attachment.ContentType,
		&attachment.SizeBytes,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return game.Attachment{}, "", game.ErrAttachmentNotFound
		}
		return game.Attachment{}, "", fmt.Errorf("get challenge attachment: %w", err)
	}
	return attachment, storagePath, nil
}

func (r *GameRepository) CreateSubmission(ctx context.Context, challengeID int64, userID int64, submittedFlag string, correct bool, sourceIP string) (int64, time.Time, error) {
	const query = `
INSERT INTO submissions (challenge_id, user_id, submitted_flag, is_correct, source_ip)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, submitted_at
`

	var (
		id          int64
		submittedAt time.Time
	)
	err := r.db.QueryRowContext(ctx, query, challengeID, userID, submittedFlag, correct, sourceIP).Scan(&id, &submittedAt)
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("create submission: %w", err)
	}
	return id, submittedAt, nil
}

func (r *GameRepository) HasSolved(ctx context.Context, challengeID int64, userID int64) (bool, error) {
	const query = `SELECT EXISTS (SELECT 1 FROM solves WHERE challenge_id = $1 AND user_id = $2)`
	var solved bool
	if err := r.db.QueryRowContext(ctx, query, challengeID, userID).Scan(&solved); err != nil {
		return false, fmt.Errorf("check solved state: %w", err)
	}
	return solved, nil
}

func (r *GameRepository) CreateSolve(ctx context.Context, challengeID int64, userID int64, submissionID int64, points int) (time.Time, error) {
	const query = `
INSERT INTO solves (challenge_id, user_id, submission_id, awarded_points)
VALUES ($1, $2, $3, $4)
ON CONFLICT (challenge_id, user_id) DO NOTHING
RETURNING solved_at
`

	var solvedAt time.Time
	err := r.db.QueryRowContext(ctx, query, challengeID, userID, submissionID, points).Scan(&solvedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return time.Time{}, nil
		}
		return time.Time{}, fmt.Errorf("create solve: %w", err)
	}
	return solvedAt, nil
}

func (r *GameRepository) ListUserSubmissions(ctx context.Context, userID int64) ([]game.UserSubmission, error) {
	const query = `
SELECT s.id, c.id, c.slug, c.title, cat.slug, s.is_correct, s.submitted_at, s.source_ip
FROM submissions s
JOIN challenges c ON c.id = s.challenge_id
JOIN categories cat ON cat.id = c.category_id
WHERE s.user_id = $1
ORDER BY s.submitted_at DESC, s.id DESC
`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list user submissions: %w", err)
	}
	defer rows.Close()

	items := make([]game.UserSubmission, 0)
	for rows.Next() {
		var item game.UserSubmission
		if err := rows.Scan(
			&item.ID,
			&item.ChallengeID,
			&item.ChallengeSlug,
			&item.ChallengeTitle,
			&item.Category,
			&item.Correct,
			&item.SubmittedAt,
			&item.SourceIP,
		); err != nil {
			return nil, fmt.Errorf("scan user submission: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate user submissions: %w", err)
	}
	return items, nil
}

func (r *GameRepository) ListUserSolves(ctx context.Context, userID int64) ([]game.UserSolve, error) {
	const query = `
SELECT s.id, c.id, c.slug, c.title, cat.slug, s.submission_id, s.awarded_points, s.solved_at
FROM solves s
JOIN challenges c ON c.id = s.challenge_id
JOIN categories cat ON cat.id = c.category_id
WHERE s.user_id = $1
ORDER BY s.solved_at DESC, s.id DESC
`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list user solves: %w", err)
	}
	defer rows.Close()

	items := make([]game.UserSolve, 0)
	for rows.Next() {
		var item game.UserSolve
		if err := rows.Scan(
			&item.ID,
			&item.ChallengeID,
			&item.ChallengeSlug,
			&item.ChallengeTitle,
			&item.Category,
			&item.SubmissionID,
			&item.AwardedPoints,
			&item.SolvedAt,
		); err != nil {
			return nil, fmt.Errorf("scan user solve: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate user solves: %w", err)
	}
	return items, nil
}

func (r *GameRepository) ListScoreboard(ctx context.Context) ([]game.ScoreboardEntry, error) {
	const query = `
SELECT
    u.id,
    u.username,
    u.display_name,
    COALESCE(SUM(s.awarded_points), 0) AS score,
    MAX(s.solved_at) AS last_solve_at
FROM users u
JOIN roles r ON r.id = u.role_id
LEFT JOIN solves s ON s.user_id = u.id
WHERE r.name = 'player' AND u.status = 'active'
GROUP BY u.id, u.username, u.display_name
ORDER BY score DESC, last_solve_at ASC NULLS LAST, u.id ASC
`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list scoreboard: %w", err)
	}
	defer rows.Close()

	items := make([]game.ScoreboardEntry, 0)
	for rows.Next() {
		var (
			item        game.ScoreboardEntry
			lastSolveAt sql.NullTime
		)
		if err := rows.Scan(&item.UserID, &item.Username, &item.DisplayName, &item.Score, &lastSolveAt); err != nil {
			return nil, fmt.Errorf("scan scoreboard entry: %w", err)
		}
		if lastSolveAt.Valid {
			t := lastSolveAt.Time
			item.LastSolveAt = &t
		}
		item.Solves, err = r.listScoreboardSolves(ctx, item.UserID)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate scoreboard entries: %w", err)
	}
	return items, nil
}

func (r *GameRepository) listChallengeAttachments(ctx context.Context, challengeID int64) ([]game.Attachment, error) {
	const query = `
SELECT id, filename, content_type, size_bytes
FROM challenge_attachments
WHERE challenge_id = $1
ORDER BY id ASC
`
	rows, err := r.db.QueryContext(ctx, query, challengeID)
	if err != nil {
		return nil, fmt.Errorf("list attachments: %w", err)
	}
	defer rows.Close()

	attachments := make([]game.Attachment, 0)
	for rows.Next() {
		var attachment game.Attachment
		if err := rows.Scan(&attachment.ID, &attachment.Filename, &attachment.ContentType, &attachment.SizeBytes); err != nil {
			return nil, fmt.Errorf("scan attachment: %w", err)
		}
		attachments = append(attachments, attachment)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate attachments: %w", err)
	}
	return attachments, nil
}

func (r *GameRepository) listScoreboardSolves(ctx context.Context, userID int64) ([]game.ScoreboardSolve, error) {
	const query = `
SELECT c.id, c.slug, c.title, cat.slug, c.difficulty, s.awarded_points, s.solved_at
FROM solves s
JOIN challenges c ON c.id = s.challenge_id
JOIN categories cat ON cat.id = c.category_id
WHERE s.user_id = $1
ORDER BY s.solved_at ASC, s.id ASC
`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list scoreboard solves: %w", err)
	}
	defer rows.Close()

	items := make([]game.ScoreboardSolve, 0)
	for rows.Next() {
		var item game.ScoreboardSolve
		if err := rows.Scan(&item.ChallengeID, &item.ChallengeSlug, &item.ChallengeTitle, &item.Category, &item.Difficulty, &item.AwardedPoints, &item.SolvedAt); err != nil {
			return nil, fmt.Errorf("scan scoreboard solve: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate scoreboard solves: %w", err)
	}
	return items, nil
}
