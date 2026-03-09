package challengeimport

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"ctf/backend/internal/game"
)

type Importer struct {
	db *sql.DB
}

func New(db *sql.DB) *Importer {
	return &Importer{db: db}
}

type ImportResult struct {
	Path          string
	Slug          string
	ChallengeID   int64
	RuntimeSynced bool
}

type ChallengeSpec struct {
	Meta    ChallengeMeta
	Flag    ChallengeFlag
	Content ChallengeContent
	Runtime *ChallengeRuntime
}

type ChallengeMeta struct {
	Slug       string
	Title      string
	Category   string
	Points     int
	Difficulty string
	Dynamic    bool
	Visible    bool
	SortOrder  int
}

type ChallengeFlag struct {
	Type  string
	Value string
}

type ChallengeContent struct {
	Description string
	Author      string
}

type ChallengeRuntime struct {
	ImageName          string
	Mode               string
	ExposedProtocol    string
	ContainerPort      int
	TTL                time.Duration
	MemoryLimitMB      int
	CPUMilli           int
	MaxRenewCount      int
	MaxActiveInstances int
	UserCooldown       time.Duration
	Env                map[string]string
	Command            []string
	Enabled            bool
}

func (i *Importer) ImportFile(ctx context.Context, contestSlug, path string) (ImportResult, error) {
	spec, err := LoadSpecFile(path)
	if err != nil {
		return ImportResult{}, err
	}
	return i.ImportSpec(ctx, contestSlug, path, spec)
}

func (i *Importer) ImportSpec(ctx context.Context, contestSlug, path string, spec ChallengeSpec) (ImportResult, error) {
	if strings.TrimSpace(contestSlug) == "" {
		return ImportResult{}, errors.New("contest slug is required")
	}
	normalized, err := NormalizeSpec(spec)
	if err != nil {
		return ImportResult{}, err
	}

	tx, err := i.db.BeginTx(ctx, nil)
	if err != nil {
		return ImportResult{}, fmt.Errorf("begin import tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	challengeID, err := upsertChallenge(ctx, tx, strings.TrimSpace(contestSlug), normalized)
	if err != nil {
		return ImportResult{}, err
	}

	runtimeSynced := false
	if normalized.Runtime != nil {
		if err := upsertRuntimeConfig(ctx, tx, challengeID, *normalized.Runtime); err != nil {
			return ImportResult{}, err
		}
		runtimeSynced = true
	}

	if err := tx.Commit(); err != nil {
		return ImportResult{}, fmt.Errorf("commit import challenge: %w", err)
	}

	return ImportResult{
		Path:          path,
		Slug:          normalized.Meta.Slug,
		ChallengeID:   challengeID,
		RuntimeSynced: runtimeSynced,
	}, nil
}

func DiscoverSpecFiles(root string) ([]string, error) {
	paths := make([]string, 0)
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if d.Name() == "challenge.yaml" {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk challenge root: %w", err)
	}
	sort.Strings(paths)
	return paths, nil
}

func LoadSpecFile(path string) (ChallengeSpec, error) {
	file, err := os.Open(path)
	if err != nil {
		return ChallengeSpec{}, fmt.Errorf("open challenge spec: %w", err)
	}
	defer file.Close()

	return parseSpec(bufio.NewScanner(file))
}

func NormalizeSpec(spec ChallengeSpec) (ChallengeSpec, error) {
	normalized := spec
	meta := normalized.Meta
	meta.Slug = strings.TrimSpace(meta.Slug)
	meta.Title = strings.TrimSpace(meta.Title)
	meta.Category = strings.TrimSpace(meta.Category)
	meta.Difficulty = strings.ToLower(strings.TrimSpace(meta.Difficulty))
	if meta.Difficulty == "" {
		meta.Difficulty = "normal"
	}
	if meta.Points <= 0 {
		return ChallengeSpec{}, errors.New("meta.points must be greater than 0")
	}
	if meta.SortOrder == 0 {
		meta.SortOrder = 10
	}
	if meta.Slug == "" {
		return ChallengeSpec{}, errors.New("meta.slug is required")
	}
	if meta.Title == "" {
		return ChallengeSpec{}, errors.New("meta.title is required")
	}
	if meta.Category == "" {
		return ChallengeSpec{}, errors.New("meta.category is required")
	}
	normalized.Meta = meta

	flagValue := strings.TrimSpace(normalized.Flag.Value)
	if flagValue == "" {
		return ChallengeSpec{}, errors.New("flag.value is required")
	}
	flagType, err := game.ValidateFlagTypeConfig(normalized.Flag.Type, flagValue)
	if err != nil {
		return ChallengeSpec{}, fmt.Errorf("flag config invalid: %w", err)
	}
	normalized.Flag.Type = flagType
	normalized.Flag.Value = flagValue

	normalized.Content.Description = strings.TrimSpace(normalized.Content.Description)
	normalized.Content.Author = strings.TrimSpace(normalized.Content.Author)

	if normalized.Runtime == nil {
		return normalized, nil
	}
	if !normalized.Meta.Dynamic {
		return ChallengeSpec{}, errors.New("runtime section requires meta.dynamic to be true")
	}

	runtimeCfg := *normalized.Runtime
	runtimeCfg.ImageName = strings.TrimSpace(runtimeCfg.ImageName)
	runtimeCfg.Mode = strings.ToLower(strings.TrimSpace(runtimeCfg.Mode))
	if runtimeCfg.Mode == "" {
		runtimeCfg.Mode = "per-user"
	}
	if runtimeCfg.Mode != "per-user" {
		return ChallengeSpec{}, fmt.Errorf("runtime.mode %q is not supported; only per-user is allowed", normalized.Runtime.Mode)
	}

	runtimeCfg.ExposedProtocol = strings.ToLower(strings.TrimSpace(runtimeCfg.ExposedProtocol))
	if runtimeCfg.ExposedProtocol == "" {
		runtimeCfg.ExposedProtocol = "http"
	}
	switch runtimeCfg.ExposedProtocol {
	case "http", "https", "tcp", "udp":
	default:
		return ChallengeSpec{}, fmt.Errorf("runtime.expose %q is not supported", normalized.Runtime.ExposedProtocol)
	}

	if runtimeCfg.ImageName == "" {
		return ChallengeSpec{}, errors.New("runtime.image is required when runtime section is present")
	}
	if runtimeCfg.ContainerPort <= 0 {
		return ChallengeSpec{}, errors.New("runtime.container_port must be greater than 0")
	}
	if runtimeCfg.TTL <= 0 {
		return ChallengeSpec{}, errors.New("runtime.ttl must be greater than 0")
	}
	if runtimeCfg.MemoryLimitMB <= 0 {
		runtimeCfg.MemoryLimitMB = 256
	}
	if runtimeCfg.CPUMilli <= 0 {
		runtimeCfg.CPUMilli = 500
	}
	if runtimeCfg.MaxRenewCount < 0 {
		return ChallengeSpec{}, errors.New("runtime.max_renew_count cannot be negative")
	}
	if runtimeCfg.MaxActiveInstances < 0 {
		return ChallengeSpec{}, errors.New("runtime.max_active_instances cannot be negative")
	}
	if runtimeCfg.UserCooldown < 0 {
		return ChallengeSpec{}, errors.New("runtime.user_cooldown cannot be negative")
	}
	if runtimeCfg.Env == nil {
		runtimeCfg.Env = map[string]string{}
	}
	if runtimeCfg.Command == nil {
		runtimeCfg.Command = []string{}
	}
	runtimeCfg.Enabled = true
	normalized.Runtime = &runtimeCfg
	return normalized, nil
}

func parseSpec(scanner *bufio.Scanner) (ChallengeSpec, error) {
	var (
		spec        ChallengeSpec
		section     string
		nested      string
		lineNumber  int
		runtimeEnv  = make(map[string]string)
		runtimeCmd  []string
		runtimeSeen bool
	)

	for scanner.Scan() {
		lineNumber++
		raw := scanner.Text()
		trimmed := strings.TrimSpace(stripComment(raw))
		if trimmed == "" {
			continue
		}

		indent := len(raw) - len(strings.TrimLeft(raw, " "))
		if indent%2 != 0 {
			return ChallengeSpec{}, fmt.Errorf("line %d: indentation must use multiples of 2 spaces", lineNumber)
		}

		if indent == 0 {
			if !strings.HasSuffix(trimmed, ":") {
				return ChallengeSpec{}, fmt.Errorf("line %d: expected top-level section", lineNumber)
			}
			section = strings.TrimSuffix(trimmed, ":")
			nested = ""
			if section == "runtime" && spec.Runtime == nil {
				spec.Runtime = &ChallengeRuntime{Enabled: true}
				runtimeSeen = true
			}
			continue
		}

		if indent == 2 {
			if strings.HasSuffix(trimmed, ":") {
				nested = strings.TrimSuffix(trimmed, ":")
				if section == "runtime" {
					switch nested {
					case "env":
						if spec.Runtime == nil {
							spec.Runtime = &ChallengeRuntime{Enabled: true}
							runtimeSeen = true
						}
						spec.Runtime.Env = runtimeEnv
						continue
					case "command":
						if spec.Runtime == nil {
							spec.Runtime = &ChallengeRuntime{Enabled: true}
							runtimeSeen = true
						}
						spec.Runtime.Command = runtimeCmd
						continue
					}
				}
				return ChallengeSpec{}, fmt.Errorf("line %d: unsupported nested section %q", lineNumber, nested)
			}

			key, value, err := splitKeyValue(trimmed)
			if err != nil {
				return ChallengeSpec{}, fmt.Errorf("line %d: %w", lineNumber, err)
			}
			nested = ""
			if err := assignScalar(&spec, section, key, value); err != nil {
				return ChallengeSpec{}, fmt.Errorf("line %d: %w", lineNumber, err)
			}
			continue
		}

		if indent == 4 {
			if section != "runtime" {
				return ChallengeSpec{}, fmt.Errorf("line %d: nested values are only supported under runtime", lineNumber)
			}
			switch nested {
			case "command":
				value := strings.TrimSpace(trimmed)
				if !strings.HasPrefix(value, "- ") {
					return ChallengeSpec{}, fmt.Errorf("line %d: command items must use '- value'", lineNumber)
				}
				runtimeCmd = append(runtimeCmd, strings.TrimSpace(value[2:]))
				if spec.Runtime != nil {
					spec.Runtime.Command = runtimeCmd
				}
				continue
			case "env":
				key, value, err := splitKeyValue(trimmed)
				if err != nil {
					return ChallengeSpec{}, fmt.Errorf("line %d: %w", lineNumber, err)
				}
				runtimeEnv[key] = normalizeScalar(value)
				if spec.Runtime != nil {
					spec.Runtime.Env = runtimeEnv
				}
				continue
			default:
				return ChallengeSpec{}, fmt.Errorf("line %d: unsupported nested runtime section %q", lineNumber, nested)
			}
		}

		return ChallengeSpec{}, fmt.Errorf("line %d: nesting deeper than 4 spaces is not supported", lineNumber)
	}
	if err := scanner.Err(); err != nil {
		return ChallengeSpec{}, fmt.Errorf("scan challenge spec: %w", err)
	}
	if runtimeSeen && spec.Runtime == nil {
		spec.Runtime = &ChallengeRuntime{Enabled: true}
	}
	return NormalizeSpec(spec)
}

func upsertChallenge(ctx context.Context, tx *sql.Tx, contestSlug string, spec ChallengeSpec) (int64, error) {
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
    sort_order,
    updated_at
)
SELECT c.id, cat.id, $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW()
FROM contests c
JOIN categories cat ON cat.slug = $11
WHERE c.slug = $12
ON CONFLICT (slug) DO UPDATE SET
    category_id = EXCLUDED.category_id,
    title = EXCLUDED.title,
    description = EXCLUDED.description,
    points = EXCLUDED.points,
    difficulty = EXCLUDED.difficulty,
    flag_type = EXCLUDED.flag_type,
    flag_value = EXCLUDED.flag_value,
    dynamic_enabled = EXCLUDED.dynamic_enabled,
    visible = EXCLUDED.visible,
    sort_order = EXCLUDED.sort_order,
    updated_at = NOW()
RETURNING id
`
	var challengeID int64
	err := tx.QueryRowContext(ctx, query,
		spec.Meta.Slug,
		spec.Meta.Title,
		spec.Content.Description,
		spec.Meta.Points,
		spec.Meta.Difficulty,
		spec.Flag.Type,
		spec.Flag.Value,
		spec.Meta.Dynamic,
		spec.Meta.Visible,
		spec.Meta.SortOrder,
		spec.Meta.Category,
		contestSlug,
	).Scan(&challengeID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, fmt.Errorf("contest %q or category %q not found", contestSlug, spec.Meta.Category)
		}
		return 0, fmt.Errorf("upsert challenge %q: %w", spec.Meta.Slug, err)
	}
	return challengeID, nil
}

func upsertRuntimeConfig(ctx context.Context, tx *sql.Tx, challengeID int64, cfg ChallengeRuntime) error {
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
		int(cfg.TTL/time.Second),
		cfg.MaxRenewCount,
		cfg.MemoryLimitMB,
		cfg.CPUMilli,
		cfg.MaxActiveInstances,
		int(cfg.UserCooldown/time.Second),
		envJSON,
		commandJSON,
		cfg.Enabled,
	); err != nil {
		return fmt.Errorf("upsert runtime config for challenge %d: %w", challengeID, err)
	}
	return nil
}

func assignScalar(spec *ChallengeSpec, section, key, value string) error {
	value = normalizeScalar(value)
	switch section {
	case "meta":
		switch key {
		case "slug":
			spec.Meta.Slug = value
		case "title":
			spec.Meta.Title = value
		case "category":
			spec.Meta.Category = value
		case "points":
			parsed, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("meta.points must be numeric")
			}
			spec.Meta.Points = parsed
		case "difficulty":
			spec.Meta.Difficulty = value
		case "dynamic":
			parsed, err := parseBool(value)
			if err != nil {
				return fmt.Errorf("meta.dynamic must be boolean")
			}
			spec.Meta.Dynamic = parsed
		case "visible":
			parsed, err := parseBool(value)
			if err != nil {
				return fmt.Errorf("meta.visible must be boolean")
			}
			spec.Meta.Visible = parsed
		case "sort_order":
			parsed, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("meta.sort_order must be numeric")
			}
			spec.Meta.SortOrder = parsed
		default:
			return fmt.Errorf("unsupported meta key %q", key)
		}
	case "flag":
		switch key {
		case "type":
			spec.Flag.Type = value
		case "value":
			spec.Flag.Value = value
		default:
			return fmt.Errorf("unsupported flag key %q", key)
		}
	case "content":
		switch key {
		case "description":
			spec.Content.Description = value
		case "author":
			spec.Content.Author = value
		default:
			return fmt.Errorf("unsupported content key %q", key)
		}
	case "runtime":
		if spec.Runtime == nil {
			spec.Runtime = &ChallengeRuntime{Enabled: true}
		}
		switch key {
		case "image":
			spec.Runtime.ImageName = value
		case "mode":
			spec.Runtime.Mode = value
		case "expose":
			spec.Runtime.ExposedProtocol = value
		case "container_port":
			parsed, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("runtime.container_port must be numeric")
			}
			spec.Runtime.ContainerPort = parsed
		case "ttl":
			parsed, err := time.ParseDuration(value)
			if err != nil {
				return fmt.Errorf("runtime.ttl must be a valid duration")
			}
			spec.Runtime.TTL = parsed
		case "memory_limit_mb":
			parsed, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("runtime.memory_limit_mb must be numeric")
			}
			spec.Runtime.MemoryLimitMB = parsed
		case "cpu_limit_millicores":
			parsed, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("runtime.cpu_limit_millicores must be numeric")
			}
			spec.Runtime.CPUMilli = parsed
		case "max_renew_count":
			parsed, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("runtime.max_renew_count must be numeric")
			}
			spec.Runtime.MaxRenewCount = parsed
		case "max_active_instances":
			parsed, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("runtime.max_active_instances must be numeric")
			}
			spec.Runtime.MaxActiveInstances = parsed
		case "user_cooldown":
			parsed, err := time.ParseDuration(value)
			if err != nil {
				return fmt.Errorf("runtime.user_cooldown must be a valid duration")
			}
			spec.Runtime.UserCooldown = parsed
		case "renew_allowed":
			parsed, err := parseBool(value)
			if err != nil {
				return fmt.Errorf("runtime.renew_allowed must be boolean")
			}
			if parsed {
				if spec.Runtime.MaxRenewCount == 0 {
					spec.Runtime.MaxRenewCount = 1
				}
			} else {
				spec.Runtime.MaxRenewCount = 0
			}
		default:
			return fmt.Errorf("unsupported runtime key %q", key)
		}
	default:
		return fmt.Errorf("unsupported section %q", section)
	}
	return nil
}

func splitKeyValue(line string) (string, string, error) {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return "", "", errors.New("expected key: value")
	}
	key := strings.TrimSpace(parts[0])
	if key == "" {
		return "", "", errors.New("key cannot be empty")
	}
	return key, strings.TrimSpace(parts[1]), nil
}

func normalizeScalar(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, `"`)
	value = strings.Trim(value, `'`)
	return value
}

func parseBool(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "yes", "on":
		return true, nil
	case "false", "no", "off":
		return false, nil
	default:
		return false, errors.New("invalid boolean")
	}
}

func stripComment(line string) string {
	inSingle := false
	inDouble := false
	for i, r := range line {
		switch r {
		case '\'':
			if !inDouble {
				inSingle = !inSingle
			}
		case '"':
			if !inSingle {
				inDouble = !inDouble
			}
		case '#':
			if !inSingle && !inDouble {
				return line[:i]
			}
		}
	}
	return line
}
