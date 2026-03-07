package instance

import (
	"context"
	apperrors "ctf-recruit/backend/internal/errors"
	"ctf-recruit/backend/internal/modules/challenge"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrActiveInstanceExists = errors.New("active instance exists")
	ErrCooldownActive       = errors.New("cooldown active")
	ErrInvalidTransition    = errors.New("invalid status transition")
	ErrInstanceNotFound     = errors.New("instance not found")
)

const (
	defaultTTL      = time.Hour
	defaultCooldown = time.Minute
)

type Clock interface {
	Now() time.Time
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now().UTC() }

type Service struct {
	repo            Repository
	challengeReader ChallengeReader
	runtime         RuntimeController
	ttl             time.Duration
	cooldown        time.Duration
	clock           Clock
}

type ChallengeReader interface {
	GetForSubmission(ctx context.Context, id string, publishedOnly bool) (*challenge.Challenge, error)
}

type RuntimeStartSpec struct {
	Image       string
	Command     []string
	ExposedPort *int
	Labels      map[string]string
}

type RuntimeAccessInfo struct {
	Host             string
	Port             int
	ConnectionString string
}

type RuntimeStartResult struct {
	ContainerID string
	AccessInfo  *RuntimeAccessInfo
}

type RuntimeController interface {
	Start(ctx context.Context, spec RuntimeStartSpec) (*RuntimeStartResult, error)
	Stop(ctx context.Context, containerID string) error
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo, ttl: defaultTTL, cooldown: defaultCooldown, clock: realClock{}}
}

func NewServiceWithRuntime(repo Repository, challengeReader ChallengeReader, runtime RuntimeController) *Service {
	return &Service{
		repo:            repo,
		challengeReader: challengeReader,
		runtime:         runtime,
		ttl:             defaultTTL,
		cooldown:        defaultCooldown,
		clock:           realClock{},
	}
}

func NewServiceWithOptions(repo Repository, ttl, cooldown time.Duration, clock Clock) *Service {
	if ttl <= 0 {
		ttl = defaultTTL
	}
	if cooldown <= 0 {
		cooldown = defaultCooldown
	}
	if clock == nil {
		clock = realClock{}
	}
	return &Service{repo: repo, ttl: ttl, cooldown: cooldown, clock: clock}
}

func (s *Service) Start(ctx context.Context, userID string, req StartInstanceRequest) (*InstanceResponse, error) {
	uid, err := parseUserID(userID)
	if err != nil {
		return nil, err
	}

	challengeID := strings.TrimSpace(req.ChallengeID)
	if challengeID == "" {
		return nil, apperrors.BadRequest("INSTANCE_VALIDATION_ERROR", "challengeId is required")
	}

	cid, err := uuid.Parse(challengeID)
	if err != nil {
		return nil, apperrors.BadRequest("INSTANCE_VALIDATION_ERROR", "challengeId must be a valid UUID")
	}

	now := s.clock.Now().UTC()
	instance, retryAt, err := s.repo.StartInstance(ctx, uid, cid, now, s.ttl, s.cooldown)
	if err != nil {
		switch {
		case errors.Is(err, ErrActiveInstanceExists):
			details := s.activeInstanceConflictDetails(ctx, uid)
			if details == nil {
				return nil, apperrors.Conflict("INSTANCE_ACTIVE_EXISTS", "An active instance already exists")
			}
			return nil, &apperrors.AppError{
				Status:  409,
				Code:    "INSTANCE_ACTIVE_EXISTS",
				Message: "An active instance already exists",
				Details: details,
			}
		case errors.Is(err, ErrCooldownActive):
			details := map[string]any{}
			if retryAt != nil {
				details["retryAt"] = retryAt.UTC().Format(time.RFC3339)
			}
			return nil, &apperrors.AppError{Status: 409, Code: "INSTANCE_COOLDOWN_ACTIVE", Message: "Instance is in cooldown", Details: details}
		default:
			return nil, apperrors.Internal("INSTANCE_START_FAILED", "Failed to start instance", fmt.Errorf("start instance: %w", err))
		}
	}

	resp := mapInstanceResponse(instance)

	if s.runtime == nil || s.challengeReader == nil {
		return &resp, nil
	}

	runtimeSpec, appErr := s.buildRuntimeStartSpec(ctx, challengeID, instance, uid)
	if appErr != nil {
		_, _ = s.repo.TransitionStatus(ctx, instance.ID, uid, StatusFailed, now, s.ttl, s.cooldown, nil)
		return nil, appErr
	}

	runtimeResult, runErr := s.runtime.Start(ctx, runtimeSpec)
	if runErr != nil {
		_, _ = s.repo.TransitionStatus(ctx, instance.ID, uid, StatusFailed, now, s.ttl, s.cooldown, nil)
		return nil, apperrors.Internal("INSTANCE_RUNTIME_START_FAILED", "Failed to start challenge runtime", runErr)
	}

	containerID := strings.TrimSpace(runtimeResult.ContainerID)
	if containerID == "" {
		_, _ = s.repo.TransitionStatus(ctx, instance.ID, uid, StatusFailed, now, s.ttl, s.cooldown, nil)
		return nil, apperrors.Internal("INSTANCE_RUNTIME_START_FAILED", "Failed to start challenge runtime", errors.New("runtime returned empty container id"))
	}

	updated, transitionErr := s.repo.TransitionStatus(ctx, instance.ID, uid, StatusRunning, now, s.ttl, s.cooldown, &containerID)
	if transitionErr != nil {
		_ = s.runtime.Stop(ctx, containerID)
		return nil, apperrors.Internal("INSTANCE_TRANSITION_FAILED", "Failed to transition instance state", fmt.Errorf("transition to running: %w", transitionErr))
	}

	resp = mapInstanceResponse(updated)
	if runtimeResult.AccessInfo != nil {
		resp.AccessInfo = &AccessInfo{
			Host:             runtimeResult.AccessInfo.Host,
			Port:             runtimeResult.AccessInfo.Port,
			ConnectionString: runtimeResult.AccessInfo.ConnectionString,
		}
	}
	return &resp, nil
}

func (s *Service) Stop(ctx context.Context, userID string, req StopInstanceRequest) (*InstanceResponse, error) {
	uid, err := parseUserID(userID)
	if err != nil {
		return nil, err
	}

	instance, appErr := s.resolveStopTarget(ctx, uid, req)
	if appErr != nil {
		return nil, appErr
	}
	if instance == nil {
		return nil, apperrors.NotFound("INSTANCE_NOT_FOUND", "Instance not found")
	}

	if instance.UserID != uid {
		return nil, apperrors.Forbidden("INSTANCE_FORBIDDEN", "Instance does not belong to current user")
	}

	if instance.Status != StatusRunning {
		return nil, apperrors.Conflict("INSTANCE_NOT_RUNNING", "Instance is not running")
	}

	now := s.clock.Now().UTC()
	stopping, err := s.repo.TransitionStatus(ctx, instance.ID, uid, StatusStopping, now, s.ttl, s.cooldown, instance.ContainerID)
	if err != nil {
		switch {
		case errors.Is(err, ErrInstanceNotFound):
			return nil, apperrors.NotFound("INSTANCE_NOT_FOUND", "Instance not found")
		case errors.Is(err, ErrInvalidTransition):
			return nil, apperrors.Conflict("INSTANCE_INVALID_TRANSITION", "Invalid instance state transition")
		default:
			return nil, apperrors.Internal("INSTANCE_TRANSITION_FAILED", "Failed to transition instance state", fmt.Errorf("transition to stopping: %w", err))
		}
	}

	if s.runtime != nil && stopping.ContainerID != nil && strings.TrimSpace(*stopping.ContainerID) != "" {
		if err := s.runtime.Stop(ctx, *stopping.ContainerID); err != nil {
			_, _ = s.repo.TransitionStatus(ctx, instance.ID, uid, StatusFailed, now, s.ttl, s.cooldown, stopping.ContainerID)
			return nil, apperrors.Internal("INSTANCE_RUNTIME_STOP_FAILED", "Failed to stop challenge runtime", err)
		}
	}

	stopped, err := s.repo.TransitionStatus(ctx, instance.ID, uid, StatusStopped, now, s.ttl, s.cooldown, stopping.ContainerID)
	if err != nil {
		switch {
		case errors.Is(err, ErrInstanceNotFound):
			return nil, apperrors.NotFound("INSTANCE_NOT_FOUND", "Instance not found")
		case errors.Is(err, ErrInvalidTransition):
			return nil, apperrors.Conflict("INSTANCE_INVALID_TRANSITION", "Invalid instance state transition")
		default:
			return nil, apperrors.Internal("INSTANCE_TRANSITION_FAILED", "Failed to transition instance state", fmt.Errorf("transition to stopped: %w", err))
		}
	}

	resp := mapInstanceResponse(stopped)
	return &resp, nil
}

func (s *Service) Me(ctx context.Context, userID string) (*MyInstanceResponse, error) {
	uid, err := parseUserID(userID)
	if err != nil {
		return nil, err
	}

	instance, err := s.repo.FindActiveForUser(ctx, uid)
	if err != nil {
		return nil, apperrors.Internal("INSTANCE_FETCH_FAILED", "Failed to fetch active instance", err)
	}

	if instance == nil {
		latest, latestErr := s.repo.FindLatestForUser(ctx, uid)
		if latestErr != nil {
			return nil, apperrors.Internal("INSTANCE_FETCH_FAILED", "Failed to fetch active instance", latestErr)
		}

		var cooldown *InstanceCooldownResponse
		now := s.clock.Now().UTC()
		if latest != nil && latest.CooldownUntil != nil && latest.CooldownUntil.After(now) {
			cooldown = &InstanceCooldownResponse{RetryAt: latest.CooldownUntil.UTC().Format(time.RFC3339)}
		}

		return &MyInstanceResponse{Instance: nil, Cooldown: cooldown}, nil
	}

	resp := mapInstanceResponse(instance)
	return &MyInstanceResponse{Instance: &resp}, nil
}

func (s *Service) Transition(ctx context.Context, userID, instanceID string, req TransitionRequest) (*InstanceResponse, error) {
	uid, err := parseUserID(userID)
	if err != nil {
		return nil, err
	}

	instanceID = strings.TrimSpace(instanceID)
	id, err := uuid.Parse(instanceID)
	if err != nil {
		return nil, apperrors.BadRequest("INSTANCE_VALIDATION_ERROR", "instanceId must be a valid UUID")
	}

	if !isKnownStatus(req.Status) {
		return nil, apperrors.BadRequest("INSTANCE_VALIDATION_ERROR", "invalid instance status")
	}

	now := s.clock.Now().UTC()
	instance, err := s.repo.TransitionStatus(ctx, id, uid, req.Status, now, s.ttl, s.cooldown, req.ContainerID)
	if err != nil {
		switch {
		case errors.Is(err, ErrInstanceNotFound):
			return nil, apperrors.NotFound("INSTANCE_NOT_FOUND", "Instance not found")
		case errors.Is(err, ErrInvalidTransition):
			return nil, apperrors.Conflict("INSTANCE_INVALID_TRANSITION", "Invalid instance state transition")
		default:
			return nil, apperrors.Internal("INSTANCE_TRANSITION_FAILED", "Failed to transition instance state", fmt.Errorf("transition instance: %w", err))
		}
	}

	resp := mapInstanceResponse(instance)
	return &resp, nil
}

func parseUserID(userID string) (uuid.UUID, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return uuid.Nil, apperrors.Unauthorized("AUTH_INVALID_TOKEN", "Invalid token")
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return uuid.Nil, apperrors.Unauthorized("AUTH_INVALID_TOKEN", "Invalid token")
	}
	return uid, nil
}

func (s *Service) activeInstanceConflictDetails(ctx context.Context, userID uuid.UUID) map[string]any {
	active, err := s.repo.FindActiveForUser(ctx, userID)
	if err != nil || active == nil {
		return nil
	}

	details := map[string]any{
		"activeInstanceId":  active.ID.String(),
		"activeUserId":      active.UserID.String(),
		"activeChallengeId": active.ChallengeID.String(),
		"activeStatus":      active.Status,
	}

	if active.StartedAt != nil {
		details["activeStartedAt"] = active.StartedAt.UTC().Format(time.RFC3339)
	}
	if active.ExpiresAt != nil {
		details["activeExpiresAt"] = active.ExpiresAt.UTC().Format(time.RFC3339)
	}

	return details
}

func mapInstanceResponse(instance *ChallengeInstance) InstanceResponse {
	resp := InstanceResponse{
		ID:          instance.ID.String(),
		UserID:      instance.UserID.String(),
		ChallengeID: instance.ChallengeID.String(),
		Status:      instance.Status,
		ContainerID: instance.ContainerID,
	}
	if instance.StartedAt != nil {
		v := instance.StartedAt.UTC().Format(time.RFC3339)
		resp.StartedAt = &v
	}
	if instance.ExpiresAt != nil {
		v := instance.ExpiresAt.UTC().Format(time.RFC3339)
		resp.ExpiresAt = &v
	}
	if instance.CooldownUntil != nil {
		v := instance.CooldownUntil.UTC().Format(time.RFC3339)
		resp.CooldownUntil = &v
	}
	return resp
}

func (s *Service) buildRuntimeStartSpec(ctx context.Context, challengeID string, instance *ChallengeInstance, userID uuid.UUID) (RuntimeStartSpec, *apperrors.AppError) {
	ch, err := s.challengeReader.GetForSubmission(ctx, challengeID, true)
	if err != nil {
		var appErr *apperrors.AppError
		if errors.As(err, &appErr) && appErr.Code == "CHALLENGE_NOT_FOUND" {
			return RuntimeStartSpec{}, apperrors.NotFound("INSTANCE_CHALLENGE_NOT_FOUND", "Challenge not found")
		}
		return RuntimeStartSpec{}, apperrors.Internal("INSTANCE_CHALLENGE_FETCH_FAILED", "Failed to load challenge runtime config", err)
	}
	if ch == nil {
		return RuntimeStartSpec{}, apperrors.NotFound("INSTANCE_CHALLENGE_NOT_FOUND", "Challenge not found")
	}

	if ch.RuntimeImage == nil || strings.TrimSpace(*ch.RuntimeImage) == "" {
		return RuntimeStartSpec{}, apperrors.BadRequest("INSTANCE_CHALLENGE_RUNTIME_MISSING", "Challenge runtime image is not configured")
	}

	command := []string{}
	if ch.RuntimeCommand != nil {
		command = strings.Fields(strings.TrimSpace(*ch.RuntimeCommand))
	}

	labels := map[string]string{
		"ctf-recruit.instance-id":  instance.ID.String(),
		"ctf-recruit.user-id":      userID.String(),
		"ctf-recruit.challenge-id": instance.ChallengeID.String(),
	}

	return RuntimeStartSpec{
		Image:       strings.TrimSpace(*ch.RuntimeImage),
		Command:     command,
		ExposedPort: ch.RuntimeExposedPort,
		Labels:      labels,
	}, nil
}

func (s *Service) resolveStopTarget(ctx context.Context, userID uuid.UUID, req StopInstanceRequest) (*ChallengeInstance, *apperrors.AppError) {
	if req.InstanceID != nil && strings.TrimSpace(*req.InstanceID) != "" {
		id, err := uuid.Parse(strings.TrimSpace(*req.InstanceID))
		if err != nil {
			return nil, apperrors.BadRequest("INSTANCE_VALIDATION_ERROR", "instanceId must be a valid UUID")
		}
		instance, repoErr := s.repo.GetByID(ctx, id)
		if repoErr != nil {
			return nil, apperrors.Internal("INSTANCE_FETCH_FAILED", "Failed to fetch instance", repoErr)
		}
		return instance, nil
	}

	instance, err := s.repo.FindActiveForUser(ctx, userID)
	if err != nil {
		return nil, apperrors.Internal("INSTANCE_FETCH_FAILED", "Failed to fetch active instance", err)
	}
	return instance, nil
}

func isKnownStatus(status Status) bool {
	switch status {
	case StatusStarting, StatusRunning, StatusStopping, StatusStopped, StatusExpired, StatusFailed, StatusCooldown:
		return true
	default:
		return false
	}
}
