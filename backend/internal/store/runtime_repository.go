package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"ctf/backend/internal/runtime"
)

type RuntimeRepository struct {
	db *sql.DB
}

func NewRuntimeRepository(db *sql.DB) *RuntimeRepository {
	return &RuntimeRepository{db: db}
}

func (r *RuntimeRepository) ListChallenges(ctx context.Context) ([]runtime.ChallengeSummary, error) {
	const query = `
SELECT c.id::text, c.slug, c.title, cat.slug, c.points, c.dynamic_enabled
FROM challenges c
JOIN categories cat ON cat.id = c.category_id
ORDER BY c.id ASC
`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list challenges: %w", err)
	}
	defer rows.Close()

	items := make([]runtime.ChallengeSummary, 0)
	for rows.Next() {
		var item runtime.ChallengeSummary
		if err := rows.Scan(&item.ID, &item.Slug, &item.Title, &item.Category, &item.Points, &item.Dynamic); err != nil {
			return nil, fmt.Errorf("scan challenge summary: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate challenge summaries: %w", err)
	}
	return items, nil
}

func (r *RuntimeRepository) GetChallengeConfig(ctx context.Context, challengeRef string) (runtime.RuntimeConfigRecord, error) {
	const query = `
SELECT
    c.id,
    c.slug,
    c.title,
    cat.slug,
    c.points,
    c.dynamic_enabled,
    rc.id,
    rc.image_name,
    rc.exposed_protocol,
    rc.container_port,
    rc.default_ttl_seconds,
    rc.memory_limit_mb,
    rc.cpu_limit_millicores,
    COALESCE(rc.env_json, '{}'::jsonb),
    COALESCE(rc.command_json, '[]'::jsonb)
FROM challenges c
JOIN categories cat ON cat.id = c.category_id
LEFT JOIN challenge_runtime_configs rc ON rc.challenge_id = c.id AND rc.enabled = TRUE
WHERE c.id::text = $1 OR lower(c.slug) = lower($1)
LIMIT 1
`

	var (
		challengeID     int64
		slug            string
		title           string
		category        string
		points          int
		dynamicEnabled  bool
		runtimeConfigID sql.NullInt64
		imageName       sql.NullString
		exposedProtocol sql.NullString
		containerPort   sql.NullInt32
		defaultTTL      sql.NullInt32
		memoryLimitMB   sql.NullInt32
		cpuLimitMilli   sql.NullInt32
		envJSON         []byte
		commandJSON     []byte
	)

	err := r.db.QueryRowContext(ctx, query, challengeRef).Scan(
		&challengeID,
		&slug,
		&title,
		&category,
		&points,
		&dynamicEnabled,
		&runtimeConfigID,
		&imageName,
		&exposedProtocol,
		&containerPort,
		&defaultTTL,
		&memoryLimitMB,
		&cpuLimitMilli,
		&envJSON,
		&commandJSON,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return runtime.RuntimeConfigRecord{}, runtime.ErrRepositoryNotFound
		}
		return runtime.RuntimeConfigRecord{}, fmt.Errorf("get challenge config: %w", err)
	}

	cfg := runtime.ChallengeConfig{
		ID:       fmt.Sprintf("%d", challengeID),
		Slug:     slug,
		Title:    title,
		Category: category,
		Points:   points,
		Dynamic:  dynamicEnabled,
	}

	if runtimeConfigID.Valid {
		cfg.ImageName = imageName.String
		cfg.ExposedProtocol = exposedProtocol.String
		cfg.ContainerPort = int(containerPort.Int32)
		cfg.TTL = time.Duration(defaultTTL.Int32) * time.Second
		cfg.MemoryLimitMB = int(memoryLimitMB.Int32)
		cfg.CPUMilli = int(cpuLimitMilli.Int32)
	}

	if len(envJSON) > 0 {
		cfg.Env = make(map[string]string)
		if err := json.Unmarshal(envJSON, &cfg.Env); err != nil {
			return runtime.RuntimeConfigRecord{}, fmt.Errorf("decode runtime env: %w", err)
		}
	}
	if len(commandJSON) > 0 {
		if err := json.Unmarshal(commandJSON, &cfg.Command); err != nil {
			return runtime.RuntimeConfigRecord{}, fmt.Errorf("decode runtime command: %w", err)
		}
	}

	return runtime.RuntimeConfigRecord{
		ID:        runtimeConfigID.Int64,
		Challenge: cfg,
	}, nil
}

func (r *RuntimeRepository) GetActiveInstance(ctx context.Context, userID int64, challengeID string) (runtime.InstanceRecord, error) {
	const query = `
SELECT
    ci.id,
    ci.runtime_config_id,
    ci.challenge_id::text,
    ci.user_id,
    ci.status,
    ci.host_port,
    ci.started_at,
    ci.expires_at,
    ci.terminated_at,
    ci.docker_container_id,
    ci.docker_container_name,
    ci.host_ip
FROM challenge_instances ci
WHERE ci.user_id = $1 AND ci.challenge_id::text = $2 AND ci.status IN ('creating', 'running')
LIMIT 1
`

	var (
		record     runtime.InstanceRecord
		terminated sql.NullTime
	)

	err := r.db.QueryRowContext(ctx, query, userID, challengeID).Scan(
		&record.ID,
		&record.RuntimeConfigID,
		&record.Instance.ChallengeID,
		&record.Instance.UserID,
		&record.Instance.Status,
		&record.Instance.HostPort,
		&record.Instance.StartedAt,
		&record.Instance.ExpiresAt,
		&terminated,
		&record.Instance.ContainerID,
		&record.Instance.ContainerName,
		&record.Instance.HostIP,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return runtime.InstanceRecord{}, runtime.ErrRepositoryNotFound
		}
		return runtime.InstanceRecord{}, fmt.Errorf("get active instance: %w", err)
	}

	if terminated.Valid {
		t := terminated.Time
		record.Instance.TerminatedAt = &t
	}
	return record, nil
}

func (r *RuntimeRepository) CreateInstance(ctx context.Context, runtimeConfigID int64, instance runtime.Instance) (runtime.InstanceRecord, error) {
	const query = `
INSERT INTO challenge_instances (
    challenge_id,
    user_id,
    runtime_config_id,
    docker_container_id,
    docker_container_name,
    host_ip,
    host_port,
    status,
    started_at,
    expires_at
) VALUES ($1::bigint, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING id
`

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		instance.ChallengeID,
		instance.UserID,
		runtimeConfigID,
		instance.ContainerID,
		instance.ContainerName,
		instance.HostIP,
		instance.HostPort,
		instance.Status,
		instance.StartedAt,
		instance.ExpiresAt,
	).Scan(&id)
	if err != nil {
		return runtime.InstanceRecord{}, fmt.Errorf("create instance: %w", err)
	}

	return runtime.InstanceRecord{
		ID:              id,
		RuntimeConfigID: runtimeConfigID,
		Instance:        instance,
	}, nil
}

func (r *RuntimeRepository) TerminateInstance(ctx context.Context, instanceID int64, terminatedAt time.Time) error {
	const query = `
UPDATE challenge_instances
SET status = 'terminated', terminated_at = $2, updated_at = NOW()
WHERE id = $1
`

	result, err := r.db.ExecContext(ctx, query, instanceID, terminatedAt)
	if err != nil {
		return fmt.Errorf("terminate instance: %w", err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return runtime.ErrRepositoryNotFound
	}
	return nil
}

func (r *RuntimeRepository) ListExpiredInstances(ctx context.Context, now time.Time) ([]runtime.InstanceRecord, error) {
	const query = `
SELECT
    ci.id,
    ci.runtime_config_id,
    ci.challenge_id::text,
    ci.user_id,
    ci.status,
    ci.host_port,
    ci.started_at,
    ci.expires_at,
    ci.terminated_at,
    ci.docker_container_id,
    ci.docker_container_name,
    ci.host_ip
FROM challenge_instances ci
WHERE ci.status IN ('creating', 'running') AND ci.expires_at <= $1
ORDER BY ci.expires_at ASC
`

	rows, err := r.db.QueryContext(ctx, query, now)
	if err != nil {
		return nil, fmt.Errorf("list expired instances: %w", err)
	}
	defer rows.Close()

	items := make([]runtime.InstanceRecord, 0)
	for rows.Next() {
		var (
			record     runtime.InstanceRecord
			terminated sql.NullTime
		)
		if err := rows.Scan(
			&record.ID,
			&record.RuntimeConfigID,
			&record.Instance.ChallengeID,
			&record.Instance.UserID,
			&record.Instance.Status,
			&record.Instance.HostPort,
			&record.Instance.StartedAt,
			&record.Instance.ExpiresAt,
			&terminated,
			&record.Instance.ContainerID,
			&record.Instance.ContainerName,
			&record.Instance.HostIP,
		); err != nil {
			return nil, fmt.Errorf("scan expired instance: %w", err)
		}
		if terminated.Valid {
			t := terminated.Time
			record.Instance.TerminatedAt = &t
		}
		items = append(items, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate expired instances: %w", err)
	}
	return items, nil
}
