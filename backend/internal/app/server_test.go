package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ctf/backend/internal/config"
	"ctf/backend/internal/runtime"
)

type testManager struct{}

func (m *testManager) Start(_ context.Context, req runtime.StartRequest) (runtime.StartedContainer, error) {
	return runtime.StartedContainer{
		ContainerID:   "test-container",
		ContainerName: "test-name",
		HostIP:        "127.0.0.1",
		HostPort:      18081,
	}, nil
}

func (m *testManager) Stop(_ context.Context, _ string) error {
	return nil
}

type testRepo struct {
	challenge runtime.RuntimeConfigRecord
	instance  *runtime.InstanceRecord
}

func newTestServer(t *testing.T) *Server {
	t.Helper()
	repo := &testRepo{
		challenge: runtime.RuntimeConfigRecord{
			ID: 11,
			Challenge: runtime.ChallengeConfig{
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
				MemoryLimitMB:   256,
				CPUMilli:        500,
			},
		},
	}
	service := runtime.NewService("http://localhost:8080", &testManager{}, repo)
	return NewServerWithRuntime(config.Load(), service)
}

func (r *testRepo) ListChallenges(context.Context) ([]runtime.ChallengeSummary, error) {
	cfg := r.challenge.Challenge
	return []runtime.ChallengeSummary{{
		ID:       cfg.ID,
		Slug:     cfg.Slug,
		Title:    cfg.Title,
		Category: cfg.Category,
		Points:   cfg.Points,
		Dynamic:  cfg.Dynamic,
	}}, nil
}

func (r *testRepo) GetChallengeConfig(_ context.Context, challengeRef string) (runtime.RuntimeConfigRecord, error) {
	if challengeRef == r.challenge.Challenge.ID || challengeRef == r.challenge.Challenge.Slug {
		return r.challenge, nil
	}
	return runtime.RuntimeConfigRecord{}, runtime.ErrRepositoryNotFound
}

func (r *testRepo) GetActiveInstance(_ context.Context, userID int64, challengeID string) (runtime.InstanceRecord, error) {
	if r.instance == nil || r.instance.Instance.UserID != userID || r.instance.Instance.ChallengeID != challengeID {
		return runtime.InstanceRecord{}, runtime.ErrRepositoryNotFound
	}
	return *r.instance, nil
}

func (r *testRepo) CreateInstance(_ context.Context, runtimeConfigID int64, instance runtime.Instance) (runtime.InstanceRecord, error) {
	record := runtime.InstanceRecord{ID: 1, RuntimeConfigID: runtimeConfigID, Instance: instance}
	r.instance = &record
	return record, nil
}

func (r *testRepo) TerminateInstance(_ context.Context, instanceID int64, terminatedAt time.Time) error {
	if r.instance == nil || r.instance.ID != instanceID {
		return runtime.ErrRepositoryNotFound
	}
	r.instance = nil
	_ = terminatedAt
	return nil
}

func (r *testRepo) ListExpiredInstances(_ context.Context, now time.Time) ([]runtime.InstanceRecord, error) {
	if r.instance == nil || r.instance.Instance.ExpiresAt.After(now) {
		return nil, nil
	}
	return []runtime.InstanceRecord{*r.instance}, nil
}

func TestHealthEndpoint(t *testing.T) {
	server := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	res := httptest.NewRecorder()

	server.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if payload["status"] != "ok" {
		t.Fatalf("expected status ok, got %#v", payload["status"])
	}
}

func TestCreateInstanceRequiresUserHeader(t *testing.T) {
	server := newTestServer(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/challenges/1/instances/me", nil)
	res := httptest.NewRecorder()

	server.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.Code)
	}
}
