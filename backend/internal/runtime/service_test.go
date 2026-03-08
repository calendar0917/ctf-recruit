package runtime

import (
	"context"
	"fmt"
	"testing"
	"time"
)

type fakeManager struct {
	startCalls       int
	stopCalls        int
	containers       map[string]ManagedContainer
	missingIDs       map[string]bool
	stoppedIDs       []string
	listManagedError error
	existsError      error
}

func (m *fakeManager) Start(_ context.Context, req StartRequest) (StartedContainer, error) {
	m.startCalls++
	containerID := fmt.Sprintf("container-%d", m.startCalls)
	if m.containers == nil {
		m.containers = make(map[string]ManagedContainer)
	}
	m.containers[containerID] = ManagedContainer{ContainerID: containerID, ChallengeID: req.ChallengeID, UserID: req.UserID}
	return StartedContainer{
		ContainerID:   containerID,
		ContainerName: fmt.Sprintf("demo-%d", m.startCalls),
		HostIP:        "127.0.0.1",
		HostPort:      18080 + m.startCalls,
	}, nil
}

func (m *fakeManager) Stop(_ context.Context, containerID string) error {
	m.stopCalls++
	m.stoppedIDs = append(m.stoppedIDs, containerID)
	if m.containers != nil {
		delete(m.containers, containerID)
	}
	return nil
}

func (m *fakeManager) Exists(_ context.Context, containerID string) (bool, error) {
	if m.existsError != nil {
		return false, m.existsError
	}
	if m.missingIDs != nil && m.missingIDs[containerID] {
		return false, nil
	}
	_, ok := m.containers[containerID]
	return ok, nil
}

func (m *fakeManager) ListManagedContainers(_ context.Context) ([]ManagedContainer, error) {
	if m.listManagedError != nil {
		return nil, m.listManagedError
	}
	items := make([]ManagedContainer, 0, len(m.containers))
	for _, item := range m.containers {
		items = append(items, item)
	}
	return items, nil
}

type fakeRepository struct {
	challenge RuntimeConfigRecord
	active    map[string]InstanceRecord
	nextID    int64
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{
		challenge: RuntimeConfigRecord{
			ID: 101,
			Challenge: ChallengeConfig{
				ID:              "1",
				Slug:            "web-welcome",
				Title:           "Welcome Panel",
				Category:        "web",
				Points:          100,
				Dynamic:         true,
				ImageName:       "ctf/web-welcome:dev",
				ExposedProtocol: "http",
				ContainerPort:   80,
				TTL:             30 * time.Minute,
				MaxRenewCount:   2,
				MemoryLimitMB:   256,
				CPUMilli:        500,
			},
		},
		active: make(map[string]InstanceRecord),
		nextID: 1,
	}
}

func (r *fakeRepository) ListChallenges(context.Context) ([]ChallengeSummary, error) {
	cfg := r.challenge.Challenge
	return []ChallengeSummary{{
		ID:       cfg.ID,
		Slug:     cfg.Slug,
		Title:    cfg.Title,
		Category: cfg.Category,
		Points:   cfg.Points,
		Dynamic:  cfg.Dynamic,
	}}, nil
}

func (r *fakeRepository) GetChallengeConfig(_ context.Context, challengeRef string) (RuntimeConfigRecord, error) {
	if challengeRef == r.challenge.Challenge.ID || challengeRef == r.challenge.Challenge.Slug {
		return r.challenge, nil
	}
	return RuntimeConfigRecord{}, ErrRepositoryNotFound
}

func (r *fakeRepository) GetActiveInstance(_ context.Context, userID int64, challengeID string) (InstanceRecord, error) {
	key := fmt.Sprintf("%d:%s", userID, challengeID)
	item, ok := r.active[key]
	if !ok {
		return InstanceRecord{}, ErrRepositoryNotFound
	}
	return item, nil
}

func (r *fakeRepository) CreateInstance(_ context.Context, runtimeConfigID int64, instance Instance) (InstanceRecord, error) {
	key := fmt.Sprintf("%d:%s", instance.UserID, instance.ChallengeID)
	record := InstanceRecord{
		ID:              r.nextID,
		RuntimeConfigID: runtimeConfigID,
		Instance:        instance,
	}
	r.nextID++
	r.active[key] = record
	return record, nil
}

func (r *fakeRepository) RenewInstance(_ context.Context, instanceID int64, expiresAt time.Time) (InstanceRecord, error) {
	for key, item := range r.active {
		if item.ID == instanceID {
			item.Instance.RenewCount++
			item.Instance.ExpiresAt = expiresAt
			r.active[key] = item
			return item, nil
		}
	}
	return InstanceRecord{}, ErrRepositoryNotFound
}

func (r *fakeRepository) TerminateInstance(_ context.Context, instanceID int64, terminatedAt time.Time) error {
	for key, item := range r.active {
		if item.ID == instanceID {
			item.Instance.Status = "terminated"
			item.Instance.TerminatedAt = &terminatedAt
			delete(r.active, key)
			return nil
		}
	}
	return ErrRepositoryNotFound
}

func (r *fakeRepository) ListExpiredInstances(_ context.Context, now time.Time) ([]InstanceRecord, error) {
	items := make([]InstanceRecord, 0)
	for _, item := range r.active {
		if !item.Instance.ExpiresAt.After(now) {
			items = append(items, item)
		}
	}
	return items, nil
}

func (r *fakeRepository) ListActiveInstances(context.Context) ([]InstanceRecord, error) {
	items := make([]InstanceRecord, 0, len(r.active))
	for _, item := range r.active {
		items = append(items, item)
	}
	return items, nil
}

func TestStartInstanceIsIdempotentPerUserAndChallenge(t *testing.T) {
	manager := &fakeManager{}
	repo := newFakeRepository()
	service := NewService("http://localhost:8080", manager, repo)

	first, created, err := service.StartInstance(context.Background(), 42, "1")
	if err != nil {
		t.Fatalf("start instance: %v", err)
	}
	if !created {
		t.Fatalf("expected first call to create an instance")
	}

	second, created, err := service.StartInstance(context.Background(), 42, "web-welcome")
	if err != nil {
		t.Fatalf("start instance again: %v", err)
	}
	if created {
		t.Fatalf("expected second call to reuse existing instance")
	}
	if manager.startCalls != 1 {
		t.Fatalf("expected one runtime start call, got %d", manager.startCalls)
	}
	if first.ContainerID != second.ContainerID {
		t.Fatalf("expected same instance to be returned")
	}
}

func TestRenewInstanceExtendsExpiryAndCountsRenewals(t *testing.T) {
	manager := &fakeManager{}
	repo := newFakeRepository()
	service := NewService("http://localhost:8080", manager, repo)
	baseTime := time.Date(2025, time.March, 8, 9, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return baseTime }

	instance, created, err := service.StartInstance(context.Background(), 7, "1")
	if err != nil {
		t.Fatalf("start instance: %v", err)
	}
	if !created {
		t.Fatalf("expected instance to be created")
	}

	renewed, err := service.RenewInstance(context.Background(), 7, "web-welcome")
	if err != nil {
		t.Fatalf("renew instance: %v", err)
	}
	if renewed.RenewCount != 1 {
		t.Fatalf("expected renew count 1, got %d", renewed.RenewCount)
	}
	wantExpiry := instance.ExpiresAt.Add(repo.challenge.Challenge.TTL)
	if !renewed.ExpiresAt.Equal(wantExpiry) {
		t.Fatalf("expected expiry %s, got %s", wantExpiry, renewed.ExpiresAt)
	}
}

func TestRenewInstanceFailsWhenRenewLimitReached(t *testing.T) {
	manager := &fakeManager{}
	repo := newFakeRepository()
	repo.challenge.Challenge.MaxRenewCount = 0
	service := NewService("http://localhost:8080", manager, repo)
	service.now = func() time.Time { return time.Date(2025, time.March, 8, 9, 0, 0, 0, time.UTC) }

	if _, _, err := service.StartInstance(context.Background(), 7, "1"); err != nil {
		t.Fatalf("start instance: %v", err)
	}
	if _, err := service.RenewInstance(context.Background(), 7, "1"); err != ErrInstanceRenewLimitReached {
		t.Fatalf("expected renew limit error, got %v", err)
	}
}

func TestSweepExpiredStopsContainers(t *testing.T) {
	manager := &fakeManager{}
	repo := newFakeRepository()
	service := NewService("http://localhost:8080", manager, repo)
	baseTime := time.Date(2025, time.March, 8, 9, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return baseTime }

	instance, created, err := service.StartInstance(context.Background(), 7, "1")
	if err != nil {
		t.Fatalf("start instance: %v", err)
	}
	if !created {
		t.Fatalf("expected instance to be created")
	}

	service.now = func() time.Time { return instance.ExpiresAt.Add(time.Second) }
	terminated, err := service.SweepExpired(context.Background())
	if err != nil {
		t.Fatalf("sweep expired: %v", err)
	}
	if terminated != 1 {
		t.Fatalf("expected one terminated instance, got %d", terminated)
	}
	if manager.stopCalls != 1 {
		t.Fatalf("expected one runtime stop call, got %d", manager.stopCalls)
	}
	if _, err := service.GetInstance(context.Background(), 7, "1"); err != ErrInstanceNotFound {
		t.Fatalf("expected instance to be removed after sweep, got %v", err)
	}
}

func TestDeleteInstanceStopsContainerAndRemovesRecord(t *testing.T) {
	manager := &fakeManager{}
	repo := newFakeRepository()
	service := NewService("http://localhost:8080", manager, repo)
	service.now = func() time.Time { return time.Date(2025, time.March, 8, 9, 0, 0, 0, time.UTC) }

	instance, created, err := service.StartInstance(context.Background(), 7, "1")
	if err != nil {
		t.Fatalf("start instance: %v", err)
	}
	if !created {
		t.Fatalf("expected instance to be created")
	}

	deleted, err := service.DeleteInstance(context.Background(), 7, "web-welcome")
	if err != nil {
		t.Fatalf("delete instance: %v", err)
	}
	if deleted.Status != "terminated" {
		t.Fatalf("expected terminated status, got %s", deleted.Status)
	}
	if deleted.TerminatedAt == nil {
		t.Fatalf("expected terminated timestamp to be set")
	}
	if manager.stopCalls != 1 {
		t.Fatalf("expected one stop call, got %d", manager.stopCalls)
	}
	if deleted.ContainerID != instance.ContainerID {
		t.Fatalf("expected same container id, got %s", deleted.ContainerID)
	}
}

func TestReconcileTerminatesMissingDatabaseRecords(t *testing.T) {
	manager := &fakeManager{missingIDs: map[string]bool{"container-1": true}}
	repo := newFakeRepository()
	service := NewService("http://localhost:8080", manager, repo)
	service.now = func() time.Time { return time.Date(2025, time.March, 8, 9, 0, 0, 0, time.UTC) }

	if _, _, err := service.StartInstance(context.Background(), 7, "1"); err != nil {
		t.Fatalf("start instance: %v", err)
	}

	report, err := service.Reconcile(context.Background())
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if report.TerminatedRecords != 1 {
		t.Fatalf("expected one terminated record, got %d", report.TerminatedRecords)
	}
	if report.RemovedContainers != 0 {
		t.Fatalf("expected zero removed containers, got %d", report.RemovedContainers)
	}
	if _, err := service.GetInstance(context.Background(), 7, "1"); err != ErrInstanceNotFound {
		t.Fatalf("expected missing instance after reconcile, got %v", err)
	}
}

func TestReconcileStopsOrphanManagedContainers(t *testing.T) {
	manager := &fakeManager{
		containers: map[string]ManagedContainer{
			"orphan-1": {ContainerID: "orphan-1", ChallengeID: "1", UserID: 77},
		},
	}
	repo := newFakeRepository()
	service := NewService("http://localhost:8080", manager, repo)

	report, err := service.Reconcile(context.Background())
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if report.TerminatedRecords != 0 {
		t.Fatalf("expected zero terminated records, got %d", report.TerminatedRecords)
	}
	if report.RemovedContainers != 1 {
		t.Fatalf("expected one removed container, got %d", report.RemovedContainers)
	}
	if manager.stopCalls != 1 {
		t.Fatalf("expected one stop call, got %d", manager.stopCalls)
	}
	if len(manager.stoppedIDs) != 1 || manager.stoppedIDs[0] != "orphan-1" {
		t.Fatalf("expected orphan container to be stopped, got %#v", manager.stoppedIDs)
	}
}
