package instance

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

type sweeperFixedClock struct {
	now time.Time
}

func (c sweeperFixedClock) Now() time.Time { return c.now }

type sweeperRuntimeStub struct {
	stopErrByContainer map[string]error
	stopCalls          []string
}

func (s *sweeperRuntimeStub) Start(_ context.Context, _ RuntimeStartSpec) (*RuntimeStartResult, error) {
	return nil, errors.New("not implemented")
}

func (s *sweeperRuntimeStub) Stop(_ context.Context, containerID string) error {
	containerID = strings.TrimSpace(containerID)
	s.stopCalls = append(s.stopCalls, containerID)
	if err, ok := s.stopErrByContainer[containerID]; ok {
		return err
	}
	return nil
}

type sweeperRepo struct {
	mu        sync.Mutex
	instances map[uuid.UUID]ChallengeInstance
	order     []uuid.UUID
}

func newSweeperRepo() *sweeperRepo {
	return &sweeperRepo{instances: map[uuid.UUID]ChallengeInstance{}, order: []uuid.UUID{}}
}

func (r *sweeperRepo) StartInstance(_ context.Context, _, _ uuid.UUID, _ time.Time, _, _ time.Duration) (*ChallengeInstance, *time.Time, error) {
	return nil, nil, errors.New("not implemented")
}

func (r *sweeperRepo) GetByIDForUser(_ context.Context, id, userID uuid.UUID) (*ChallengeInstance, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.instances[id]
	if !ok || item.UserID != userID {
		return nil, nil
	}
	copy := item
	return &copy, nil
}

func (r *sweeperRepo) GetByID(_ context.Context, id uuid.UUID) (*ChallengeInstance, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.instances[id]
	if !ok {
		return nil, nil
	}
	copy := item
	return &copy, nil
}

func (r *sweeperRepo) FindActiveForUser(_ context.Context, userID uuid.UUID) (*ChallengeInstance, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := len(r.order) - 1; i >= 0; i-- {
		item := r.instances[r.order[i]]
		if item.UserID == userID && (item.Status == StatusStarting || item.Status == StatusRunning || item.Status == StatusStopping) {
			copy := item
			return &copy, nil
		}
	}
	return nil, nil
}

func (r *sweeperRepo) FindLatestForUser(_ context.Context, userID uuid.UUID) (*ChallengeInstance, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := len(r.order) - 1; i >= 0; i-- {
		item := r.instances[r.order[i]]
		if item.UserID != userID {
			continue
		}
		copy := item
		return &copy, nil
	}
	return nil, nil
}

func (r *sweeperRepo) ListExpirableBefore(_ context.Context, now time.Time, limit int) ([]ChallengeInstance, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if limit <= 0 {
		limit = 20
	}
	items := make([]ChallengeInstance, 0, limit)
	for _, id := range r.order {
		item := r.instances[id]
		if item.Status != StatusRunning || item.ExpiresAt == nil || item.ExpiresAt.After(now.UTC()) {
			continue
		}
		items = append(items, item)
		if len(items) >= limit {
			break
		}
	}
	return items, nil
}

func (r *sweeperRepo) TransitionStatus(_ context.Context, id, userID uuid.UUID, to Status, now time.Time, ttl, cooldown time.Duration, containerID *string) (*ChallengeInstance, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	item, ok := r.instances[id]
	if !ok || item.UserID != userID {
		return nil, ErrInstanceNotFound
	}
	if !CanTransition(item.Status, to) {
		return nil, ErrInvalidTransition
	}

	item.Status = to
	item.UpdatedAt = now.UTC()
	if containerID != nil {
		item.ContainerID = containerID
	}
	if to == StatusRunning {
		start := now.UTC()
		exp := start.Add(ttl)
		item.StartedAt = &start
		item.ExpiresAt = &exp
		item.CooldownUntil = nil
	}
	if to == StatusStopped || to == StatusExpired || to == StatusFailed || to == StatusCooldown {
		cd := now.UTC().Add(cooldown)
		item.CooldownUntil = &cd
	}

	r.instances[id] = item
	copy := item
	return &copy, nil
}

func (r *sweeperRepo) seedRunningExpired(userID, challengeID uuid.UUID, containerID string, expiresAt time.Time) ChallengeInstance {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := expiresAt.Add(-time.Hour)
	item := ChallengeInstance{
		ID:          uuid.New(),
		UserID:      userID,
		ChallengeID: challengeID,
		Status:      StatusRunning,
		ContainerID: &containerID,
		StartedAt:   &now,
		ExpiresAt:   ptrTime(expiresAt),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	r.instances[item.ID] = item
	r.order = append(r.order, item.ID)
	return item
}

func TestSweeperExpiresRunningInstanceAndSetsCooldown(t *testing.T) {
	repo := newSweeperRepo()
	runtime := &sweeperRuntimeStub{stopErrByContainer: map[string]error{}}
	now := time.Date(2026, 2, 16, 10, 0, 0, 0, time.UTC)
	userID := uuid.New()
	seed := repo.seedRunningExpired(userID, uuid.New(), "container-expire-1", now.Add(-time.Second))

	sweeper := NewSweeper(repo, runtime).WithClock(sweeperFixedClock{now: now})
	processed, err := sweeper.ProcessOnce(context.Background())
	if err != nil {
		t.Fatalf("process once failed: %v", err)
	}
	if processed != 1 {
		t.Fatalf("expected processed=1, got %d", processed)
	}

	got, err := repo.GetByIDForUser(context.Background(), seed.ID, userID)
	if err != nil {
		t.Fatalf("get instance failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected instance to exist")
	}
	if got.Status != StatusExpired {
		t.Fatalf("expected expired status, got %s", got.Status)
	}
	if got.CooldownUntil == nil || !got.CooldownUntil.Equal(now.Add(defaultCooldown)) {
		t.Fatalf("expected cooldownUntil %s, got %#v", now.Add(defaultCooldown).Format(time.RFC3339), got.CooldownUntil)
	}

	active, err := repo.FindActiveForUser(context.Background(), userID)
	if err != nil {
		t.Fatalf("find active failed: %v", err)
	}
	if active != nil {
		t.Fatalf("expected no active/running instance after expiration, got %s", active.Status)
	}
}

func TestSweeperStopFailureLogsContextAndIsRetriable(t *testing.T) {
	repo := newSweeperRepo()
	containerID := "container-expire-fail"
	runtime := &sweeperRuntimeStub{stopErrByContainer: map[string]error{containerID: errors.New("docker rm failed")}}
	now := time.Date(2026, 2, 16, 10, 5, 0, 0, time.UTC)
	seed := repo.seedRunningExpired(uuid.New(), uuid.New(), containerID, now.Add(-2*time.Second))

	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, nil))
	sweeper := NewSweeper(repo, runtime).WithClock(sweeperFixedClock{now: now}).WithLogger(logger)

	processed, err := sweeper.ProcessOnce(context.Background())
	if err != nil {
		t.Fatalf("process once failed: %v", err)
	}
	if processed != 0 {
		t.Fatalf("expected processed=0 on stop failure, got %d", processed)
	}

	failedLog := logBuf.String()
	for _, expect := range []string{"instance sweeper stop failed", "instanceId=" + seed.ID.String(), "containerId=" + containerID} {
		if !strings.Contains(failedLog, expect) {
			t.Fatalf("expected log to contain %q, got %q", expect, failedLog)
		}
	}

	itemAfterFail, err := repo.GetByIDForUser(context.Background(), seed.ID, seed.UserID)
	if err != nil {
		t.Fatalf("get instance failed: %v", err)
	}
	if itemAfterFail == nil || itemAfterFail.Status != StatusRunning {
		t.Fatalf("expected instance to remain running after stop failure, got %#v", itemAfterFail)
	}

	delete(runtime.stopErrByContainer, containerID)
	processed, err = sweeper.ProcessOnce(context.Background())
	if err != nil {
		t.Fatalf("process retry failed: %v", err)
	}
	if processed != 1 {
		t.Fatalf("expected processed=1 on retry, got %d", processed)
	}

	itemAfterRetry, err := repo.GetByIDForUser(context.Background(), seed.ID, seed.UserID)
	if err != nil {
		t.Fatalf("get instance retry failed: %v", err)
	}
	if itemAfterRetry == nil || itemAfterRetry.Status != StatusExpired {
		t.Fatalf("expected expired after retry, got %#v", itemAfterRetry)
	}
}
