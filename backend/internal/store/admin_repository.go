package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"ctf/backend/internal/admin"
)

type AdminRepository struct {
	db *sql.DB
}

func NewAdminRepository(db *sql.DB) *AdminRepository {
	return &AdminRepository{db: db}
}

func (r *AdminRepository) ListChallenges(ctx context.Context, actor admin.Actor) ([]admin.ChallengeSummary, error) {
	query := `
SELECT c.id, c.slug, c.title, cat.slug, c.points, c.visible, c.dynamic_enabled
FROM challenges c
JOIN categories cat ON cat.id = c.category_id
`
	args := make([]any, 0, 1)
	if actor.RestrictToOwnedChallenges() {
		query += `
JOIN challenge_authors ca ON ca.challenge_id = c.id
WHERE ca.user_id = $1
`
		args = append(args, actor.UserID)
	}
	query += "ORDER BY cat.sort_order ASC, c.sort_order ASC, c.id ASC"

	rows, err := r.db.QueryContext(ctx, query, args...)
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

func (r *AdminRepository) GetChallenge(ctx context.Context, actor admin.Actor, challengeID int64) (admin.ChallengeDetail, error) {
	challengeQuery := `
SELECT c.id, c.slug, c.title, cat.slug, c.description, c.points, c.difficulty, c.flag_type, c.flag_value, c.visible, c.dynamic_enabled, c.sort_order
FROM challenges c
JOIN categories cat ON cat.id = c.category_id
`
	args := []any{challengeID}
	if actor.RestrictToOwnedChallenges() {
		challengeQuery += `JOIN challenge_authors ca ON ca.challenge_id = c.id
WHERE c.id = $1 AND ca.user_id = $2
LIMIT 1
`
		args = append(args, actor.UserID)
	} else {
		challengeQuery += `WHERE c.id = $1
LIMIT 1
`
	}

	var detail admin.ChallengeDetail
	if err := r.db.QueryRowContext(ctx, challengeQuery, args...).Scan(
		&detail.ID,
		&detail.Slug,
		&detail.Title,
		&detail.Category,
		&detail.Description,
		&detail.Points,
		&detail.Difficulty,
		&detail.FlagType,
		&detail.FlagValue,
		&detail.Visible,
		&detail.DynamicEnabled,
		&detail.SortOrder,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return admin.ChallengeDetail{}, admin.ErrResourceNotFound
		}
		return admin.ChallengeDetail{}, fmt.Errorf("get admin challenge: %w", err)
	}

	authors, err := r.listChallengeAuthors(ctx, challengeID)
	if err != nil {
		return admin.ChallengeDetail{}, err
	}
	detail.Authors = authors

	attachments, err := r.listChallengeAttachments(ctx, challengeID)
	if err != nil {
		return admin.ChallengeDetail{}, err
	}
	detail.Attachments = attachments

	runtimeConfig, err := r.getChallengeRuntimeConfig(ctx, challengeID)
	if err != nil {
		return admin.ChallengeDetail{}, err
	}
	detail.RuntimeConfig = runtimeConfig
	return detail, nil
}

func (r *AdminRepository) CreateChallenge(ctx context.Context, actor admin.Actor, input admin.UpsertChallengeInput) (admin.ChallengeSummary, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return admin.ChallengeSummary{}, fmt.Errorf("begin create challenge tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

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
	err = tx.QueryRowContext(ctx, query,
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

	if actor.RestrictToOwnedChallenges() {
		if err := bindChallengeAuthor(ctx, tx, id, actor.UserID); err != nil {
			return admin.ChallengeSummary{}, err
		}
	}
	if err := upsertChallengeRuntimeConfig(ctx, tx, id, input.RuntimeConfig); err != nil {
		return admin.ChallengeSummary{}, err
	}
	if err := tx.Commit(); err != nil {
		return admin.ChallengeSummary{}, fmt.Errorf("commit create challenge: %w", err)
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

func (r *AdminRepository) ListChallengeAuthors(ctx context.Context, actor admin.Actor, challengeID int64) ([]admin.ChallengeAuthor, error) {
	if actor.RestrictToOwnedChallenges() {
		allowed, err := challengeOwnedByUser(ctx, r.db, challengeID, actor.UserID)
		if err != nil {
			return nil, err
		}
		if !allowed {
			return nil, admin.ErrResourceNotFound
		}
	}
	return r.listChallengeAuthors(ctx, challengeID)
}

func (r *AdminRepository) UpdateChallengeAuthors(ctx context.Context, actor admin.Actor, challengeID int64, userIDs []int64) ([]admin.ChallengeAuthor, error) {
	if actor.Role != "admin" {
		return nil, admin.ErrResourceNotFound
	}
	if exists, err := challengeExists(ctx, r.db, challengeID); err != nil {
		return nil, err
	} else if !exists {
		return nil, admin.ErrResourceNotFound
	}
	for _, userID := range userIDs {
		if err := ensureChallengeAuthorCandidate(ctx, r.db, userID); err != nil {
			return nil, err
		}
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin update challenge authors tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()
	if err := SetChallengeAuthors(ctx, tx, challengeID, userIDs); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit update challenge authors: %w", err)
	}
	return r.listChallengeAuthors(ctx, challengeID)
}

func (r *AdminRepository) UpdateChallenge(ctx context.Context, actor admin.Actor, challengeID int64, input admin.UpsertChallengeInput) (admin.ChallengeSummary, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return admin.ChallengeSummary{}, fmt.Errorf("begin update challenge tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	query := `
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
`
	args := []any{
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
	}
	if actor.RestrictToOwnedChallenges() {
		query += `
WHERE c.id = $1 AND cat.slug = $12 AND EXISTS (
    SELECT 1 FROM challenge_authors ca WHERE ca.challenge_id = c.id AND ca.user_id = $13
)
RETURNING c.id
`
		args = append(args, actor.UserID)
	} else {
		query += `
WHERE c.id = $1 AND cat.slug = $12
RETURNING c.id
`
	}

	var id int64
	err = tx.QueryRowContext(ctx, query, args...).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return admin.ChallengeSummary{}, admin.ErrResourceNotFound
		}
		return admin.ChallengeSummary{}, fmt.Errorf("update challenge: %w", err)
	}

	if err := upsertChallengeRuntimeConfig(ctx, tx, challengeID, input.RuntimeConfig); err != nil {
		return admin.ChallengeSummary{}, err
	}
	if err := tx.Commit(); err != nil {
		return admin.ChallengeSummary{}, fmt.Errorf("commit update challenge: %w", err)
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

func (r *AdminRepository) CreateAttachment(ctx context.Context, actor admin.Actor, challengeID int64, filename, storagePath, contentType string, sizeBytes int64) (admin.Attachment, error) {
	if actor.RestrictToOwnedChallenges() {
		allowed, err := challengeOwnedByUser(ctx, r.db, challengeID, actor.UserID)
		if err != nil {
			return admin.Attachment{}, err
		}
		if !allowed {
			return admin.Attachment{}, admin.ErrResourceNotFound
		}
	}

	const query = `
INSERT INTO challenge_attachments (challenge_id, filename, storage_path, content_type, size_bytes)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, filename, content_type, size_bytes
`
	var item admin.Attachment
	if err := r.db.QueryRowContext(ctx, query, challengeID, filename, storagePath, contentType, sizeBytes).Scan(
		&item.ID,
		&item.Filename,
		&item.ContentType,
		&item.SizeBytes,
	); err != nil {
		return admin.Attachment{}, fmt.Errorf("create attachment: %w", err)
	}
	return item, nil
}

func (r *AdminRepository) GetAttachment(ctx context.Context, challengeID int64, attachmentID int64) (admin.Attachment, string, error) {
	const query = `
SELECT id, filename, storage_path, content_type, size_bytes
FROM challenge_attachments
WHERE challenge_id = $1 AND id = $2
LIMIT 1
`
	var (
		item        admin.Attachment
		storagePath string
	)
	if err := r.db.QueryRowContext(ctx, query, challengeID, attachmentID).Scan(
		&item.ID,
		&item.Filename,
		&storagePath,
		&item.ContentType,
		&item.SizeBytes,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return admin.Attachment{}, "", admin.ErrResourceNotFound
		}
		return admin.Attachment{}, "", fmt.Errorf("get attachment: %w", err)
	}
	return item, storagePath, nil
}

func (r *AdminRepository) ListUsers(ctx context.Context) ([]admin.UserRecord, error) {
	const query = `
SELECT u.id, r.name, u.username, u.email, u.display_name, u.status, u.last_login_at, u.created_at
FROM users u
JOIN roles r ON r.id = u.role_id
ORDER BY u.id ASC
`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	items := make([]admin.UserRecord, 0)
	for rows.Next() {
		var (
			item        admin.UserRecord
			lastLoginAt sql.NullTime
		)
		if err := rows.Scan(&item.ID, &item.Role, &item.Username, &item.Email, &item.DisplayName, &item.Status, &lastLoginAt, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		if lastLoginAt.Valid {
			t := lastLoginAt.Time
			item.LastLoginAt = &t
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate users: %w", err)
	}
	return items, nil
}

func (r *AdminRepository) UpdateUser(ctx context.Context, userID int64, input admin.UpdateUserInput) (admin.UserRecord, error) {
	const query = `
UPDATE users u
SET role_id = r.id, display_name = $2, status = $3, updated_at = NOW()
FROM roles r
WHERE u.id = $1 AND r.name = $4
RETURNING u.id, r.name, u.username, u.email, u.display_name, u.status, u.last_login_at, u.created_at
`
	var (
		item        admin.UserRecord
		lastLoginAt sql.NullTime
	)
	if err := r.db.QueryRowContext(ctx, query, userID, input.DisplayName, input.Status, input.Role).Scan(
		&item.ID,
		&item.Role,
		&item.Username,
		&item.Email,
		&item.DisplayName,
		&item.Status,
		&lastLoginAt,
		&item.CreatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return admin.UserRecord{}, admin.ErrResourceNotFound
		}
		return admin.UserRecord{}, fmt.Errorf("update user: %w", err)
	}
	if lastLoginAt.Valid {
		t := lastLoginAt.Time
		item.LastLoginAt = &t
	}
	return item, nil
}

func (r *AdminRepository) ListAuditLogs(ctx context.Context) ([]admin.AuditLogRecord, error) {
	const query = `
SELECT id, actor_user_id, action, resource_type, resource_id, details_json, created_at
FROM audit_logs
ORDER BY created_at DESC, id DESC
`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list audit logs: %w", err)
	}
	defer rows.Close()

	items := make([]admin.AuditLogRecord, 0)
	for rows.Next() {
		var (
			item        admin.AuditLogRecord
			actorUserID sql.NullInt64
			detailsJSON []byte
		)
		if err := rows.Scan(&item.ID, &actorUserID, &item.Action, &item.ResourceType, &item.ResourceID, &detailsJSON, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan audit log: %w", err)
		}
		if actorUserID.Valid {
			value := actorUserID.Int64
			item.ActorUserID = &value
		}
		if len(detailsJSON) > 0 {
			item.Details = make(map[string]any)
			if err := json.Unmarshal(detailsJSON, &item.Details); err != nil {
				return nil, fmt.Errorf("decode audit log details: %w", err)
			}
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate audit logs: %w", err)
	}
	return items, nil
}

func (r *AdminRepository) CreateAuditLog(ctx context.Context, actorUserID *int64, action, resourceType, resourceID string, details map[string]any) error {
	if details == nil {
		details = map[string]any{}
	}
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		return fmt.Errorf("encode audit log details: %w", err)
	}
	const query = `
INSERT INTO audit_logs (actor_user_id, action, resource_type, resource_id, details_json)
VALUES ($1, $2, $3, $4, $5)
`
	if _, err := r.db.ExecContext(ctx, query, actorUserID, action, resourceType, resourceID, detailsJSON); err != nil {
		return fmt.Errorf("create audit log: %w", err)
	}
	return nil
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

func (r *AdminRepository) DeleteAnnouncement(ctx context.Context, announcementID int64) (admin.Announcement, error) {
	const query = `
DELETE FROM announcements
WHERE id = $1
RETURNING id, title, content, pinned, published, published_at
`
	var (
		item        admin.Announcement
		publishedAt sql.NullTime
	)
	err := r.db.QueryRowContext(ctx, query, announcementID).Scan(&item.ID, &item.Title, &item.Content, &item.Pinned, &item.Published, &publishedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return admin.Announcement{}, admin.ErrResourceNotFound
		}
		return admin.Announcement{}, fmt.Errorf("delete announcement: %w", err)
	}
	if publishedAt.Valid {
		t := publishedAt.Time
		item.PublishedAt = &t
	}
	return item, nil
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

func (r *AdminRepository) GetInstance(ctx context.Context, instanceID int64) (admin.InstanceRecord, error) {
	const query = `
SELECT ci.id, c.id, c.slug, u.username, ci.status, ci.host_port, ci.expires_at, ci.terminated_at, ci.docker_container_id
FROM challenge_instances ci
JOIN challenges c ON c.id = ci.challenge_id
JOIN users u ON u.id = ci.user_id
WHERE ci.id = $1
LIMIT 1
`
	var (
		item         admin.InstanceRecord
		terminatedAt sql.NullTime
	)
	if err := r.db.QueryRowContext(ctx, query, instanceID).Scan(
		&item.ID,
		&item.ChallengeID,
		&item.ChallengeSlug,
		&item.Username,
		&item.Status,
		&item.HostPort,
		&item.ExpiresAt,
		&terminatedAt,
		&item.ContainerID,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return admin.InstanceRecord{}, admin.ErrResourceNotFound
		}
		return admin.InstanceRecord{}, fmt.Errorf("get instance: %w", err)
	}
	if terminatedAt.Valid {
		t := terminatedAt.Time
		item.TerminatedAt = &t
	}
	return item, nil
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

func (r *AdminRepository) listChallengeAuthors(ctx context.Context, challengeID int64) ([]admin.ChallengeAuthor, error) {
	const query = `
SELECT u.id, u.username, u.email, u.display_name, role.name
FROM challenge_authors ca
JOIN users u ON u.id = ca.user_id
JOIN roles role ON role.id = u.role_id
WHERE ca.challenge_id = $1
ORDER BY ca.created_at ASC, u.id ASC
`
	rows, err := r.db.QueryContext(ctx, query, challengeID)
	if err != nil {
		return nil, fmt.Errorf("list challenge authors: %w", err)
	}
	defer rows.Close()

	items := make([]admin.ChallengeAuthor, 0)
	for rows.Next() {
		var item admin.ChallengeAuthor
		if err := rows.Scan(&item.UserID, &item.Username, &item.Email, &item.DisplayName, &item.Role); err != nil {
			return nil, fmt.Errorf("scan challenge author: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate challenge authors: %w", err)
	}
	return items, nil
}

func (r *AdminRepository) listChallengeAttachments(ctx context.Context, challengeID int64) ([]admin.Attachment, error) {
	const query = `
SELECT id, filename, content_type, size_bytes
FROM challenge_attachments
WHERE challenge_id = $1
ORDER BY id ASC
`
	rows, err := r.db.QueryContext(ctx, query, challengeID)
	if err != nil {
		return nil, fmt.Errorf("list admin challenge attachments: %w", err)
	}
	defer rows.Close()

	items := make([]admin.Attachment, 0)
	for rows.Next() {
		var item admin.Attachment
		if err := rows.Scan(&item.ID, &item.Filename, &item.ContentType, &item.SizeBytes); err != nil {
			return nil, fmt.Errorf("scan admin challenge attachment: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate admin challenge attachments: %w", err)
	}
	return items, nil
}

func (r *AdminRepository) getChallengeRuntimeConfig(ctx context.Context, challengeID int64) (admin.RuntimeConfig, error) {
	const query = `
SELECT enabled, image_name, exposed_protocol, container_port, default_ttl_seconds, max_renew_count, memory_limit_mb, cpu_limit_millicores,
       max_active_instances, user_cooldown_seconds, COALESCE(env_json, '{}'::jsonb), COALESCE(command_json, '[]'::jsonb)
FROM challenge_runtime_configs
WHERE challenge_id = $1
LIMIT 1
`

	var (
		cfg                admin.RuntimeConfig
		enabled            bool
		imageName          string
		protocol           string
		containerPort      int
		defaultTTL         int
		maxRenewCount      int
		memoryLimitMB      int
		cpuLimitMilli      int
		maxActiveInstances int
		userCooldown       int
		envJSON            []byte
		commandJSON        []byte
	)
	if err := r.db.QueryRowContext(ctx, query, challengeID).Scan(
		&enabled,
		&imageName,
		&protocol,
		&containerPort,
		&defaultTTL,
		&maxRenewCount,
		&memoryLimitMB,
		&cpuLimitMilli,
		&maxActiveInstances,
		&userCooldown,
		&envJSON,
		&commandJSON,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return admin.RuntimeConfig{}, nil
		}
		return admin.RuntimeConfig{}, fmt.Errorf("get admin challenge runtime config: %w", err)
	}
	cfg.Enabled = enabled
	cfg.ImageName = imageName
	cfg.ExposedProtocol = protocol
	cfg.ContainerPort = containerPort
	cfg.DefaultTTL = defaultTTL
	cfg.MaxRenewCount = maxRenewCount
	cfg.MemoryLimitMB = memoryLimitMB
	cfg.CPUMilli = cpuLimitMilli
	cfg.MaxActiveInstances = maxActiveInstances
	cfg.UserCooldown = userCooldown
	if len(envJSON) > 0 {
		cfg.Env = make(map[string]string)
		if err := json.Unmarshal(envJSON, &cfg.Env); err != nil {
			return admin.RuntimeConfig{}, fmt.Errorf("decode admin runtime env: %w", err)
		}
	}
	if len(commandJSON) > 0 {
		if err := json.Unmarshal(commandJSON, &cfg.Command); err != nil {
			return admin.RuntimeConfig{}, fmt.Errorf("decode admin runtime command: %w", err)
		}
	}
	return cfg, nil
}

func upsertChallengeRuntimeConfig(ctx context.Context, tx *sql.Tx, challengeID int64, cfg *admin.RuntimeConfig) error {
	if cfg == nil {
		return nil
	}
	envJSON, err := json.Marshal(cfg.Env)
	if err != nil {
		return fmt.Errorf("encode runtime env: %w", err)
	}
	commandJSON, err := json.Marshal(cfg.Command)
	if err != nil {
		return fmt.Errorf("encode runtime command: %w", err)
	}

	const query = `
INSERT INTO challenge_runtime_configs (
    challenge_id,
    image_name,
    exposed_protocol,
    container_port,
    default_ttl_seconds,
    max_renew_count,
    memory_limit_mb,
    cpu_limit_millicores,
    max_active_instances,
    user_cooldown_seconds,
    env_json,
    command_json,
    enabled,
    updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NOW())
ON CONFLICT (challenge_id) DO UPDATE SET
    image_name = EXCLUDED.image_name,
    exposed_protocol = EXCLUDED.exposed_protocol,
    container_port = EXCLUDED.container_port,
    default_ttl_seconds = EXCLUDED.default_ttl_seconds,
    max_renew_count = EXCLUDED.max_renew_count,
    memory_limit_mb = EXCLUDED.memory_limit_mb,
    cpu_limit_millicores = EXCLUDED.cpu_limit_millicores,
    max_active_instances = EXCLUDED.max_active_instances,
    user_cooldown_seconds = EXCLUDED.user_cooldown_seconds,
    env_json = EXCLUDED.env_json,
    command_json = EXCLUDED.command_json,
    enabled = EXCLUDED.enabled,
    updated_at = NOW()
`
	if _, err := tx.ExecContext(ctx, query,
		challengeID,
		cfg.ImageName,
		cfg.ExposedProtocol,
		cfg.ContainerPort,
		cfg.DefaultTTL,
		cfg.MaxRenewCount,
		cfg.MemoryLimitMB,
		cfg.CPUMilli,
		cfg.MaxActiveInstances,
		cfg.UserCooldown,
		envJSON,
		commandJSON,
		cfg.Enabled,
	); err != nil {
		return fmt.Errorf("upsert runtime config: %w", err)
	}
	return nil
}

func bindChallengeAuthor(ctx context.Context, tx *sql.Tx, challengeID, userID int64) error {
	const query = `
INSERT INTO challenge_authors (challenge_id, user_id)
VALUES ($1, $2)
ON CONFLICT (challenge_id, user_id) DO NOTHING
`
	if _, err := tx.ExecContext(ctx, query, challengeID, userID); err != nil {
		return fmt.Errorf("bind challenge author: %w", err)
	}
	return nil
}

func challengeOwnedByUser(ctx context.Context, queryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, challengeID, userID int64) (bool, error) {
	const query = `
SELECT 1
FROM challenge_authors
WHERE challenge_id = $1 AND user_id = $2
LIMIT 1
`
	var found int
	if err := queryer.QueryRowContext(ctx, query, challengeID, userID).Scan(&found); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("check challenge ownership: %w", err)
	}
	return true, nil
}

func challengeExists(ctx context.Context, queryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, challengeID int64) (bool, error) {
	const query = `SELECT 1 FROM challenges WHERE id = $1 LIMIT 1`
	var found int
	if err := queryer.QueryRowContext(ctx, query, challengeID).Scan(&found); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("check challenge exists: %w", err)
	}
	return true, nil
}

func ensureChallengeAuthorCandidate(ctx context.Context, queryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, userID int64) error {
	const query = `
SELECT role.name
FROM users u
JOIN roles role ON role.id = u.role_id
WHERE u.id = $1
LIMIT 1
`
	var role string
	if err := queryer.QueryRowContext(ctx, query, userID).Scan(&role); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return admin.ErrResourceNotFound
		}
		return fmt.Errorf("load challenge author candidate: %w", err)
	}
	if role != "author" && role != "admin" {
		return fmt.Errorf("user %d must have author or admin role", userID)
	}
	return nil
}

func ResolveUserIDByAuthorRef(ctx context.Context, db *sql.DB, value string) (int64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, admin.ErrResourceNotFound
	}
	const query = `
SELECT id
FROM users
WHERE username = $1 OR email = $1
ORDER BY id ASC
LIMIT 1
`
	var userID int64
	if err := db.QueryRowContext(ctx, query, value).Scan(&userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, admin.ErrResourceNotFound
		}
		return 0, fmt.Errorf("resolve challenge author %q: %w", value, err)
	}
	return userID, nil
}

func SetChallengeAuthors(ctx context.Context, tx *sql.Tx, challengeID int64, userIDs []int64) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM challenge_authors WHERE challenge_id = $1`, challengeID); err != nil {
		return fmt.Errorf("clear challenge authors: %w", err)
	}
	for _, userID := range userIDs {
		if err := bindChallengeAuthor(ctx, tx, challengeID, userID); err != nil {
			return err
		}
	}
	return nil
}
