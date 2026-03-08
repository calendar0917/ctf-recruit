package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ctf/backend/internal/auth"
	"ctf/backend/internal/config"
	"ctf/backend/internal/runtime"
)

type testManager struct{}

func (m *testManager) Start(_ context.Context, _ runtime.StartRequest) (runtime.StartedContainer, error) {
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

type testRuntimeRepo struct {
	challenge runtime.RuntimeConfigRecord
	instance  *runtime.InstanceRecord
}

type testUserRepo struct {
	users      map[int64]auth.User
	identifier map[string]int64
	nextID     int64
}

func newTestServer(t *testing.T) *Server {
	t.Helper()

	runtimeRepo := &testRuntimeRepo{
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
	userRepo := &testUserRepo{
		users:      make(map[int64]auth.User),
		identifier: make(map[string]int64),
		nextID:     1,
	}

	cfg := config.Load()
	tokens := auth.NewTokenManager(cfg.JWTSecret, cfg.JWTTTL)
	authService := auth.NewService(userRepo, tokens)
	runtimeService := runtime.NewService("http://localhost:8080", &testManager{}, runtimeRepo)
	return NewServerForTests(cfg, authService, runtimeService)
}

func (r *testRuntimeRepo) ListChallenges(context.Context) ([]runtime.ChallengeSummary, error) {
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

func (r *testRuntimeRepo) GetChallengeConfig(_ context.Context, challengeRef string) (runtime.RuntimeConfigRecord, error) {
	if challengeRef == r.challenge.Challenge.ID || challengeRef == r.challenge.Challenge.Slug {
		return r.challenge, nil
	}
	return runtime.RuntimeConfigRecord{}, runtime.ErrRepositoryNotFound
}

func (r *testRuntimeRepo) GetActiveInstance(_ context.Context, userID int64, challengeID string) (runtime.InstanceRecord, error) {
	if r.instance == nil || r.instance.Instance.UserID != userID || r.instance.Instance.ChallengeID != challengeID {
		return runtime.InstanceRecord{}, runtime.ErrRepositoryNotFound
	}
	return *r.instance, nil
}

func (r *testRuntimeRepo) CreateInstance(_ context.Context, runtimeConfigID int64, instance runtime.Instance) (runtime.InstanceRecord, error) {
	record := runtime.InstanceRecord{ID: 1, RuntimeConfigID: runtimeConfigID, Instance: instance}
	r.instance = &record
	return record, nil
}

func (r *testRuntimeRepo) TerminateInstance(_ context.Context, instanceID int64, _ time.Time) error {
	if r.instance == nil || r.instance.ID != instanceID {
		return runtime.ErrRepositoryNotFound
	}
	r.instance = nil
	return nil
}

func (r *testRuntimeRepo) ListExpiredInstances(_ context.Context, now time.Time) ([]runtime.InstanceRecord, error) {
	if r.instance == nil || r.instance.Instance.ExpiresAt.After(now) {
		return nil, nil
	}
	return []runtime.InstanceRecord{*r.instance}, nil
}

func (r *testUserRepo) CreateUser(_ context.Context, params auth.CreateUserParams) (auth.User, error) {
	id := r.nextID
	r.nextID++
	user := auth.User{
		ID:           id,
		Role:         params.RoleName,
		Username:     params.Username,
		Email:        params.Email,
		DisplayName:  params.DisplayName,
		Status:       "active",
		PasswordHash: params.PasswordHash,
	}
	r.users[id] = user
	r.identifier[params.Username] = id
	r.identifier[params.Email] = id
	return user, nil
}

func (r *testUserRepo) GetUserByIdentifier(_ context.Context, identifier string) (auth.User, error) {
	id, ok := r.identifier[identifier]
	if !ok {
		return auth.User{}, runtime.ErrRepositoryNotFound
	}
	return r.users[id], nil
}

func (r *testUserRepo) GetUserByID(_ context.Context, userID int64) (auth.User, error) {
	user, ok := r.users[userID]
	if !ok {
		return auth.User{}, runtime.ErrRepositoryNotFound
	}
	return user, nil
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

func TestProtectedInstanceEndpointRequiresBearerToken(t *testing.T) {
	server := newTestServer(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/challenges/1/instances/me", nil)
	res := httptest.NewRecorder()

	server.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", res.Code)
	}
}

func TestRegisterThenAccessMe(t *testing.T) {
	server := newTestServer(t)

	registerBody := []byte(`{"username":"alice","email":"alice@example.com","password":"Password123!","display_name":"Alice"}`)
	registerReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(registerBody))
	registerRes := httptest.NewRecorder()
	server.Handler().ServeHTTP(registerRes, registerReq)

	if registerRes.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", registerRes.Code)
	}

	var registerPayload map[string]any
	if err := json.Unmarshal(registerRes.Body.Bytes(), &registerPayload); err != nil {
		t.Fatalf("decode register response: %v", err)
	}

	token, ok := registerPayload["token"].(string)
	if !ok || token == "" {
		t.Fatalf("expected token in register response")
	}

	meReq := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+token)
	meRes := httptest.NewRecorder()
	server.Handler().ServeHTTP(meRes, meReq)

	if meRes.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", meRes.Code)
	}
}
