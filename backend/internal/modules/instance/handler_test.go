package instance

import (
	"bytes"
	"context"
	apperrors "ctf-recruit/backend/internal/errors"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"ctf-recruit/backend/internal/middleware"
	"ctf-recruit/backend/internal/modules/auth"
	"ctf-recruit/backend/internal/modules/challenge"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type inMemoryRepo struct {
	mu        sync.Mutex
	instances map[uuid.UUID]ChallengeInstance
	order     []uuid.UUID
}

func newInMemoryRepo() *inMemoryRepo {
	return &inMemoryRepo{instances: map[uuid.UUID]ChallengeInstance{}, order: make([]uuid.UUID, 0)}
}

func (r *inMemoryRepo) StartInstance(_ context.Context, userID, challengeID uuid.UUID, now time.Time, ttl, _ time.Duration) (*ChallengeInstance, *time.Time, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, id := range r.order {
		item := r.instances[id]
		if item.UserID == userID && (item.Status == StatusStarting || item.Status == StatusRunning) {
			return nil, nil, ErrActiveInstanceExists
		}
	}

	var latest *ChallengeInstance
	for i := len(r.order) - 1; i >= 0; i-- {
		item := r.instances[r.order[i]]
		if item.UserID == userID {
			copy := item
			latest = &copy
			break
		}
	}
	if latest != nil && latest.CooldownUntil != nil && latest.CooldownUntil.After(now.UTC()) {
		retry := latest.CooldownUntil.UTC()
		return nil, &retry, ErrCooldownActive
	}

	start := now.UTC()
	expires := start.Add(ttl)
	item := ChallengeInstance{
		ID:          uuid.New(),
		UserID:      userID,
		ChallengeID: challengeID,
		Status:      StatusStarting,
		StartedAt:   &start,
		ExpiresAt:   &expires,
		CreatedAt:   now.UTC(),
		UpdatedAt:   now.UTC(),
	}
	r.instances[item.ID] = item
	r.order = append(r.order, item.ID)
	copy := item
	return &copy, nil, nil
}

func (r *inMemoryRepo) GetByIDForUser(_ context.Context, id, userID uuid.UUID) (*ChallengeInstance, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	item, ok := r.instances[id]
	if !ok || item.UserID != userID {
		return nil, nil
	}
	copy := item
	return &copy, nil
}

func (r *inMemoryRepo) GetByID(_ context.Context, id uuid.UUID) (*ChallengeInstance, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	item, ok := r.instances[id]
	if !ok {
		return nil, nil
	}
	copy := item
	return &copy, nil
}

func (r *inMemoryRepo) FindActiveForUser(_ context.Context, userID uuid.UUID) (*ChallengeInstance, error) {
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

func (r *inMemoryRepo) FindLatestForUser(_ context.Context, userID uuid.UUID) (*ChallengeInstance, error) {
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

func (r *inMemoryRepo) ListExpirableBefore(_ context.Context, now time.Time, limit int) ([]ChallengeInstance, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if limit <= 0 {
		limit = 20
	}

	result := make([]ChallengeInstance, 0, limit)
	for _, id := range r.order {
		item := r.instances[id]
		if item.Status != StatusRunning || item.ExpiresAt == nil || item.ExpiresAt.After(now.UTC()) {
			continue
		}
		result = append(result, item)
		if len(result) >= limit {
			break
		}
	}
	return result, nil
}

func (r *inMemoryRepo) TransitionStatus(_ context.Context, id, userID uuid.UUID, to Status, now time.Time, ttl, cooldown time.Duration, containerID *string) (*ChallengeInstance, error) {
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
	switch to {
	case StatusRunning:
		started := now.UTC()
		expires := started.Add(ttl)
		item.StartedAt = &started
		item.ExpiresAt = &expires
		item.CooldownUntil = nil
	case StatusStopped, StatusExpired, StatusFailed, StatusCooldown:
		cd := now.UTC().Add(cooldown)
		item.CooldownUntil = &cd
	}

	r.instances[id] = item
	copy := item
	return &copy, nil
}

func (r *inMemoryRepo) seedCooldown(userID, challengeID uuid.UUID, until time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()

	item := ChallengeInstance{
		ID:            uuid.New(),
		UserID:        userID,
		ChallengeID:   challengeID,
		Status:        StatusCooldown,
		CooldownUntil: &until,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}
	r.instances[item.ID] = item
	r.order = append(r.order, item.ID)
}

func (r *inMemoryRepo) seedRunning(userID, challengeID uuid.UUID, containerID string) ChallengeInstance {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	item := ChallengeInstance{
		ID:          uuid.New(),
		UserID:      userID,
		ChallengeID: challengeID,
		Status:      StatusRunning,
		ContainerID: &containerID,
		StartedAt:   &now,
		ExpiresAt:   ptrTime(now.Add(time.Hour)),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	r.instances[item.ID] = item
	r.order = append(r.order, item.ID)
	return item
}

func (r *inMemoryRepo) seedRunningWithTimestamps(userID, challengeID uuid.UUID, containerID string, startedAt, expiresAt time.Time) ChallengeInstance {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := startedAt.UTC()
	expires := expiresAt.UTC()
	item := ChallengeInstance{
		ID:          uuid.New(),
		UserID:      userID,
		ChallengeID: challengeID,
		Status:      StatusRunning,
		ContainerID: &containerID,
		StartedAt:   &now,
		ExpiresAt:   &expires,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	r.instances[item.ID] = item
	r.order = append(r.order, item.ID)
	return item
}

func TestInstancesConcurrentStartOnlyOneCreated(t *testing.T) {
	app, authService := setupInstanceTestApp()
	userID := uuid.New()
	token := mustIssueAccessTokenWithUserID(t, authService, userID, auth.RolePlayer)
	challengeID := uuid.New().String()
	body, _ := json.Marshal(map[string]any{"challengeId": challengeID})

	statuses := make(chan int, 2)
	codes := make(chan string, 2)

	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/instances/start", bytes.NewReader(body))
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			if err != nil {
				t.Errorf("request failed: %v", err)
				return
			}
			defer resp.Body.Close()

			statuses <- resp.StatusCode
			if resp.StatusCode != http.StatusCreated {
				codes <- decodeErrorCode(t, resp)
				return
			}
			codes <- ""
		}()
	}
	wg.Wait()
	close(statuses)
	close(codes)

	created := 0
	conflict := 0
	for s := range statuses {
		if s == http.StatusCreated {
			created++
		}
		if s == http.StatusConflict {
			conflict++
		}
	}
	if created != 1 || conflict != 1 {
		t.Fatalf("expected one 201 and one 409, got created=%d conflict=%d", created, conflict)
	}

	foundActiveCode := false
	for c := range codes {
		if c == "INSTANCE_ACTIVE_EXISTS" {
			foundActiveCode = true
		}
	}
	if !foundActiveCode {
		t.Fatal("expected INSTANCE_ACTIVE_EXISTS conflict code")
	}
}

func TestInstancesStartBlockedByCooldownReturnsRetryAt(t *testing.T) {
	repo := newInMemoryRepo()
	service := NewService(repo)
	authService := auth.NewService(nil, "test-secret", time.Hour)
	app := fiber.New(fiber.Config{ErrorHandler: middleware.ErrorHandler})
	v1 := app.Group("/api/v1")
	NewHandler(service).RegisterRoutes(v1, authService)

	userID := uuid.New()
	challengeID := uuid.New()
	retryAt := time.Now().UTC().Add(45 * time.Second).Truncate(time.Second)
	repo.seedCooldown(userID, challengeID, retryAt)

	token := mustIssueAccessTokenWithUserID(t, authService, userID, auth.RolePlayer)
	body, _ := json.Marshal(map[string]any{"challengeId": uuid.New().String()})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/instances/start", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}

	var payload struct {
		Error struct {
			Code    string         `json:"code"`
			Details map[string]any `json:"details"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if payload.Error.Code != "INSTANCE_COOLDOWN_ACTIVE" {
		t.Fatalf("expected INSTANCE_COOLDOWN_ACTIVE, got %s", payload.Error.Code)
	}
	rawRetryAt, ok := payload.Error.Details["retryAt"].(string)
	if !ok || rawRetryAt == "" {
		t.Fatal("expected retryAt in error details")
	}
	parsed, err := time.Parse(time.RFC3339, rawRetryAt)
	if err != nil {
		t.Fatalf("invalid retryAt format: %v", err)
	}
	if !parsed.Equal(retryAt) {
		t.Fatalf("expected retryAt %s, got %s", retryAt.Format(time.RFC3339), rawRetryAt)
	}
}

func TestInstancesStartWithExistingActiveInstanceReturnsConflict(t *testing.T) {
	app, authService := setupInstanceTestApp()
	userID := uuid.New()
	token := mustIssueAccessTokenWithUserID(t, authService, userID, auth.RolePlayer)

	_ = doJSONRequest[InstanceResponse](
		t,
		app,
		http.MethodPost,
		"/api/v1/instances/start",
		token,
		map[string]any{"challengeId": uuid.New().String()},
		http.StatusCreated,
	)

	resp := doRequestRaw(
		t,
		app,
		http.MethodPost,
		"/api/v1/instances/start",
		token,
		map[string]any{"challengeId": uuid.New().String()},
	)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}
	if decodeErrorCode(t, resp) != "INSTANCE_ACTIVE_EXISTS" {
		t.Fatalf("expected INSTANCE_ACTIVE_EXISTS")
	}
}

func TestInstancesStartWithExistingActiveInstanceReturnsConflictDetails(t *testing.T) {
	repo := newInMemoryRepo()
	now := time.Date(2026, 2, 17, 11, 0, 0, 0, time.UTC)
	service := NewServiceWithOptions(repo, time.Hour, time.Minute, fixedClock{now: now})
	authService := auth.NewService(nil, "test-secret", time.Hour)
	app := fiber.New(fiber.Config{ErrorHandler: middleware.ErrorHandler})
	v1 := app.Group("/api/v1")
	NewHandler(service).RegisterRoutes(v1, authService)

	userID := uuid.New()
	challengeID := uuid.New()
	seed := repo.seedRunningWithTimestamps(userID, challengeID, "container-active-1", now, now.Add(time.Hour))
	token := mustIssueAccessTokenWithUserID(t, authService, userID, auth.RolePlayer)

	resp := doRequestRaw(
		t,
		app,
		http.MethodPost,
		"/api/v1/instances/start",
		token,
		map[string]any{"challengeId": uuid.New().String()},
	)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}

	var payload struct {
		Error struct {
			Code    string         `json:"code"`
			Message string         `json:"message"`
			Details map[string]any `json:"details"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if payload.Error.Code != "INSTANCE_ACTIVE_EXISTS" {
		t.Fatalf("expected INSTANCE_ACTIVE_EXISTS, got %s", payload.Error.Code)
	}
	if payload.Error.Message != "An active instance already exists" {
		t.Fatalf("expected unchanged message, got %q", payload.Error.Message)
	}
	if payload.Error.Details == nil {
		t.Fatal("expected details payload for active-instance conflict")
	}

	required := map[string]string{
		"activeInstanceId":  seed.ID.String(),
		"activeUserId":      userID.String(),
		"activeChallengeId": challengeID.String(),
		"activeStatus":      string(StatusRunning),
		"activeStartedAt":   now.UTC().Format(time.RFC3339),
		"activeExpiresAt":   now.Add(time.Hour).UTC().Format(time.RFC3339),
	}

	for key, expected := range required {
		raw, ok := payload.Error.Details[key]
		if !ok {
			t.Fatalf("expected details key %s", key)
		}
		value, ok := raw.(string)
		if !ok {
			t.Fatalf("expected details[%s] to be string, got %T", key, raw)
		}
		if value != expected {
			t.Fatalf("expected details[%s]=%s, got %s", key, expected, value)
		}
	}

	if _, err := time.Parse(time.RFC3339, payload.Error.Details["activeStartedAt"].(string)); err != nil {
		t.Fatalf("activeStartedAt must be RFC3339: %v", err)
	}
	if _, err := time.Parse(time.RFC3339, payload.Error.Details["activeExpiresAt"].(string)); err != nil {
		t.Fatalf("activeExpiresAt must be RFC3339: %v", err)
	}
}

func TestInstancesStartWithExistingActiveInstanceLegacyDecodeCompatibility(t *testing.T) {
	app, authService := setupInstanceTestApp()
	userID := uuid.New()
	token := mustIssueAccessTokenWithUserID(t, authService, userID, auth.RolePlayer)

	_ = doJSONRequest[InstanceResponse](
		t,
		app,
		http.MethodPost,
		"/api/v1/instances/start",
		token,
		map[string]any{"challengeId": uuid.New().String()},
		http.StatusCreated,
	)

	resp := doRequestRaw(
		t,
		app,
		http.MethodPost,
		"/api/v1/instances/start",
		token,
		map[string]any{"challengeId": uuid.New().String()},
	)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}

	type legacyErrorEnvelope struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	var legacy legacyErrorEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&legacy); err != nil {
		t.Fatalf("decode legacy envelope: %v", err)
	}

	if legacy.Error.Code != "INSTANCE_ACTIVE_EXISTS" {
		t.Fatalf("expected code INSTANCE_ACTIVE_EXISTS, got %s", legacy.Error.Code)
	}
	if legacy.Error.Message != "An active instance already exists" {
		t.Fatalf("expected unchanged message, got %q", legacy.Error.Message)
	}
}

func TestInstancesInvalidTransitionRejected(t *testing.T) {
	app, authService := setupInstanceTestApp()
	userID := uuid.New()
	token := mustIssueAccessTokenWithUserID(t, authService, userID, auth.RolePlayer)

	created := doJSONRequest[InstanceResponse](
		t,
		app,
		http.MethodPost,
		"/api/v1/instances/start",
		token,
		map[string]any{"challengeId": uuid.New().String()},
		http.StatusCreated,
	)

	resp := doRequestRaw(
		t,
		app,
		http.MethodPost,
		"/api/v1/instances/"+created.ID+"/transition",
		token,
		map[string]any{"status": StatusStopped},
	)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}
	if decodeErrorCode(t, resp) != "INSTANCE_INVALID_TRANSITION" {
		t.Fatalf("expected INSTANCE_INVALID_TRANSITION")
	}
}

func TestInstancesStartWithRuntimeIncludesAccessInfoAndExpiresAt(t *testing.T) {
	repo := newInMemoryRepo()
	challengeID := uuid.New()
	challengeReader := stubChallengeReader{challenge: &challenge.Challenge{
		ID:                 challengeID,
		IsPublished:        true,
		RuntimeImage:       ptrString("nginx:alpine"),
		RuntimeCommand:     ptrString("nginx -g daemon off;"),
		RuntimeExposedPort: ptrInt(80),
	}}
	runtime := &stubRuntimeController{startResult: &RuntimeStartResult{
		ContainerID: "container-123",
		AccessInfo: &RuntimeAccessInfo{
			Host:             "localhost",
			Port:             39001,
			ConnectionString: "localhost:39001",
		},
	}}

	service := NewServiceWithRuntime(repo, challengeReader, runtime)
	authService := auth.NewService(nil, "test-secret", time.Hour)
	app := fiber.New(fiber.Config{ErrorHandler: middleware.ErrorHandler})
	v1 := app.Group("/api/v1")
	NewHandler(service).RegisterRoutes(v1, authService)

	userID := uuid.New()
	token := mustIssueAccessTokenWithUserID(t, authService, userID, auth.RolePlayer)

	resp := doJSONRequest[InstanceResponse](
		t,
		app,
		http.MethodPost,
		"/api/v1/instances/start",
		token,
		map[string]any{"challengeId": challengeID.String()},
		http.StatusCreated,
	)

	if resp.Status != StatusRunning {
		t.Fatalf("expected status running, got %s", resp.Status)
	}
	if resp.ContainerID == nil || *resp.ContainerID != "container-123" {
		t.Fatalf("expected containerId container-123, got %#v", resp.ContainerID)
	}
	if resp.ExpiresAt == nil || *resp.ExpiresAt == "" {
		t.Fatal("expected expiresAt in response")
	}
	if resp.AccessInfo == nil || resp.AccessInfo.Port != 39001 {
		t.Fatalf("expected accessInfo port 39001, got %#v", resp.AccessInfo)
	}
}

func TestInstancesStopRemovesContainerAndTransitionsStopped(t *testing.T) {
	repo := newInMemoryRepo()
	service := NewServiceWithRuntime(repo, stubChallengeReader{}, &stubRuntimeController{})
	authService := auth.NewService(nil, "test-secret", time.Hour)
	app := fiber.New(fiber.Config{ErrorHandler: middleware.ErrorHandler})
	v1 := app.Group("/api/v1")
	NewHandler(service).RegisterRoutes(v1, authService)

	userID := uuid.New()
	challengeID := uuid.New()
	seed := repo.seedRunning(userID, challengeID, "container-stop-1")
	token := mustIssueAccessTokenWithUserID(t, authService, userID, auth.RolePlayer)

	resp := doJSONRequest[InstanceResponse](
		t,
		app,
		http.MethodPost,
		"/api/v1/instances/stop",
		token,
		map[string]any{"instanceId": seed.ID.String()},
		http.StatusOK,
	)

	if resp.Status != StatusStopped {
		t.Fatalf("expected stopped, got %s", resp.Status)
	}
	if resp.CooldownUntil == nil || *resp.CooldownUntil == "" {
		t.Fatal("expected cooldownUntil after stop")
	}
}

func TestInstancesStopNonOwnerForbidden(t *testing.T) {
	repo := newInMemoryRepo()
	service := NewServiceWithRuntime(repo, stubChallengeReader{}, &stubRuntimeController{})
	authService := auth.NewService(nil, "test-secret", time.Hour)
	app := fiber.New(fiber.Config{ErrorHandler: middleware.ErrorHandler})
	v1 := app.Group("/api/v1")
	NewHandler(service).RegisterRoutes(v1, authService)

	ownerID := uuid.New()
	nonOwnerID := uuid.New()
	challengeID := uuid.New()
	seed := repo.seedRunning(ownerID, challengeID, "container-owner-1")
	nonOwnerToken := mustIssueAccessTokenWithUserID(t, authService, nonOwnerID, auth.RolePlayer)

	resp := doRequestRaw(
		t,
		app,
		http.MethodPost,
		"/api/v1/instances/stop",
		nonOwnerToken,
		map[string]any{"instanceId": seed.ID.String()},
	)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", resp.StatusCode)
	}
	if decodeErrorCode(t, resp) != "INSTANCE_FORBIDDEN" {
		t.Fatal("expected INSTANCE_FORBIDDEN")
	}
}

func TestInstancesAPIEvidenceAStartsBStopDenied(t *testing.T) {
	app, authService := setupInstanceTestApp()
	userA := uuid.New()
	userB := uuid.New()
	tokenA := mustIssueAccessTokenWithUserID(t, authService, userA, auth.RolePlayer)
	tokenB := mustIssueAccessTokenWithUserID(t, authService, userB, auth.RolePlayer)

	started := doJSONRequest[InstanceResponse](
		t,
		app,
		http.MethodPost,
		"/api/v1/instances/start",
		tokenA,
		map[string]any{"challengeId": uuid.New().String()},
		http.StatusCreated,
	)

	resp := doRequestRaw(
		t,
		app,
		http.MethodPost,
		"/api/v1/instances/stop",
		tokenB,
		map[string]any{"instanceId": started.ID},
	)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", resp.StatusCode)
	}
	if decodeErrorCode(t, resp) != "INSTANCE_FORBIDDEN" {
		t.Fatal("expected INSTANCE_FORBIDDEN")
	}
}

func TestInstancesMeReturnsActiveInstanceForCurrentUserOnly(t *testing.T) {
	repo := newInMemoryRepo()
	service := NewServiceWithRuntime(repo, stubChallengeReader{}, &stubRuntimeController{})
	authService := auth.NewService(nil, "test-secret", time.Hour)
	app := fiber.New(fiber.Config{ErrorHandler: middleware.ErrorHandler})
	v1 := app.Group("/api/v1")
	NewHandler(service).RegisterRoutes(v1, authService)

	ownerID := uuid.New()
	otherID := uuid.New()
	challengeID := uuid.New()
	seed := repo.seedRunning(ownerID, challengeID, "container-me-1")
	repo.seedRunning(otherID, uuid.New(), "container-me-2")
	token := mustIssueAccessTokenWithUserID(t, authService, ownerID, auth.RolePlayer)

	resp := doJSONRequest[MyInstanceResponse](
		t,
		app,
		http.MethodGet,
		"/api/v1/instances/me",
		token,
		nil,
		http.StatusOK,
	)

	if resp.Instance == nil {
		t.Fatal("expected instance in response")
	}
	if resp.Instance.ID != seed.ID.String() {
		t.Fatalf("expected instance id %s, got %s", seed.ID.String(), resp.Instance.ID)
	}
	if resp.Instance.UserID != ownerID.String() {
		t.Fatalf("expected userId %s, got %s", ownerID.String(), resp.Instance.UserID)
	}
}

func TestInstancesMeReturnsEmptyWhenNoActiveInstance(t *testing.T) {
	app, authService := setupInstanceTestApp()
	userID := uuid.New()
	token := mustIssueAccessTokenWithUserID(t, authService, userID, auth.RolePlayer)

	resp := doJSONRequest[MyInstanceResponse](
		t,
		app,
		http.MethodGet,
		"/api/v1/instances/me",
		token,
		nil,
		http.StatusOK,
	)

	if resp.Instance != nil {
		t.Fatalf("expected nil instance, got %#v", resp.Instance)
	}
}

func TestInstancesMeReturnsCooldownWhenNoActiveInstance(t *testing.T) {
	repo := newInMemoryRepo()
	now := time.Date(2026, 2, 17, 8, 0, 0, 0, time.UTC)
	retryAt := now.Add(45 * time.Second)

	userID := uuid.New()
	repo.seedCooldown(userID, uuid.New(), retryAt)

	service := NewServiceWithOptions(repo, time.Hour, time.Minute, fixedClock{now: now})
	authService := auth.NewService(nil, "test-secret", time.Hour)
	app := fiber.New(fiber.Config{ErrorHandler: middleware.ErrorHandler})
	v1 := app.Group("/api/v1")
	NewHandler(service).RegisterRoutes(v1, authService)

	token := mustIssueAccessTokenWithUserID(t, authService, userID, auth.RolePlayer)

	resp := doJSONRequest[MyInstanceResponse](
		t,
		app,
		http.MethodGet,
		"/api/v1/instances/me",
		token,
		nil,
		http.StatusOK,
	)

	if resp.Instance != nil {
		t.Fatalf("expected nil active instance, got %#v", resp.Instance)
	}
	if resp.Cooldown == nil {
		t.Fatal("expected cooldown metadata when cooldown is active")
	}
	if resp.Cooldown.RetryAt != retryAt.UTC().Format(time.RFC3339) {
		t.Fatalf("expected retryAt %s, got %s", retryAt.UTC().Format(time.RFC3339), resp.Cooldown.RetryAt)
	}
}

func TestInstancesMeOmitsCooldownAfterCooldownElapsed(t *testing.T) {
	repo := newInMemoryRepo()
	base := time.Date(2026, 2, 17, 8, 0, 0, 0, time.UTC)
	retryAt := base.Add(30 * time.Second)

	userID := uuid.New()
	repo.seedCooldown(userID, uuid.New(), retryAt)

	service := NewServiceWithOptions(repo, time.Hour, time.Minute, fixedClock{now: retryAt.Add(time.Second)})
	authService := auth.NewService(nil, "test-secret", time.Hour)
	app := fiber.New(fiber.Config{ErrorHandler: middleware.ErrorHandler})
	v1 := app.Group("/api/v1")
	NewHandler(service).RegisterRoutes(v1, authService)

	token := mustIssueAccessTokenWithUserID(t, authService, userID, auth.RolePlayer)

	resp := doJSONRequest[MyInstanceResponse](
		t,
		app,
		http.MethodGet,
		"/api/v1/instances/me",
		token,
		nil,
		http.StatusOK,
	)

	if resp.Instance != nil {
		t.Fatalf("expected nil active instance, got %#v", resp.Instance)
	}
	if resp.Cooldown != nil {
		t.Fatalf("expected cooldown metadata to be omitted after expiry, got %#v", resp.Cooldown)
	}
}

func TestInstancesStartRuntimeFailureMapsErrorCode(t *testing.T) {
	repo := newInMemoryRepo()
	challengeID := uuid.New()
	challengeReader := stubChallengeReader{challenge: &challenge.Challenge{
		ID:                 challengeID,
		IsPublished:        true,
		RuntimeImage:       ptrString("nginx:alpine"),
		RuntimeExposedPort: ptrInt(80),
	}}
	runtime := &stubRuntimeController{startErr: context.DeadlineExceeded}

	service := NewServiceWithRuntime(repo, challengeReader, runtime)
	authService := auth.NewService(nil, "test-secret", time.Hour)
	app := fiber.New(fiber.Config{ErrorHandler: middleware.ErrorHandler})
	v1 := app.Group("/api/v1")
	NewHandler(service).RegisterRoutes(v1, authService)

	userID := uuid.New()
	token := mustIssueAccessTokenWithUserID(t, authService, userID, auth.RolePlayer)

	resp := doRequestRaw(
		t,
		app,
		http.MethodPost,
		"/api/v1/instances/start",
		token,
		map[string]any{"challengeId": challengeID.String()},
	)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.StatusCode)
	}
	if decodeErrorCode(t, resp) != "INSTANCE_RUNTIME_START_FAILED" {
		t.Fatal("expected INSTANCE_RUNTIME_START_FAILED")
	}
}

func TestInstancesStopRuntimeFailureMapsErrorCode(t *testing.T) {
	repo := newInMemoryRepo()
	runtime := &stubRuntimeController{stopErr: context.DeadlineExceeded}
	service := NewServiceWithRuntime(repo, stubChallengeReader{}, runtime)
	authService := auth.NewService(nil, "test-secret", time.Hour)
	app := fiber.New(fiber.Config{ErrorHandler: middleware.ErrorHandler})
	v1 := app.Group("/api/v1")
	NewHandler(service).RegisterRoutes(v1, authService)

	userID := uuid.New()
	challengeID := uuid.New()
	seed := repo.seedRunning(userID, challengeID, "container-stop-err")
	token := mustIssueAccessTokenWithUserID(t, authService, userID, auth.RolePlayer)

	resp := doRequestRaw(
		t,
		app,
		http.MethodPost,
		"/api/v1/instances/stop",
		token,
		map[string]any{"instanceId": seed.ID.String()},
	)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.StatusCode)
	}
	if decodeErrorCode(t, resp) != "INSTANCE_RUNTIME_STOP_FAILED" {
		t.Fatal("expected INSTANCE_RUNTIME_STOP_FAILED")
	}
}

func setupInstanceTestApp() (*fiber.App, *auth.Service) {
	repo := newInMemoryRepo()
	service := NewService(repo)
	authService := auth.NewService(nil, "test-secret", time.Hour)

	app := fiber.New(fiber.Config{ErrorHandler: middleware.ErrorHandler})
	v1 := app.Group("/api/v1")
	NewHandler(service).RegisterRoutes(v1, authService)

	return app, authService
}

func mustIssueAccessTokenWithUserID(t *testing.T, authService *auth.Service, userID uuid.UUID, role auth.Role) string {
	t.Helper()

	token, err := authService.GenerateAccessToken(&auth.User{ID: userID, Role: role})
	if err != nil {
		t.Fatalf("issue access token failed: %v", err)
	}

	return token
}

func doJSONRequest[T any](
	t *testing.T,
	app *fiber.App,
	method, path, token string,
	body map[string]any,
	expectedStatus int,
) T {
	t.Helper()

	resp := doRequestRaw(t, app, method, path, token, body)
	defer resp.Body.Close()

	if resp.StatusCode != expectedStatus {
		t.Fatalf("expected %d, got %d", expectedStatus, resp.StatusCode)
	}

	var payload T
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	return payload
}

func doRequestRaw(
	t *testing.T,
	app *fiber.App,
	method, path, token string,
	body map[string]any,
) *http.Response {
	t.Helper()

	var reqBody *bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reqBody = bytes.NewReader(raw)
	} else {
		reqBody = bytes.NewReader(nil)
	}

	req := httptest.NewRequest(method, path, reqBody)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	return resp
}

func decodeErrorCode(t *testing.T, resp *http.Response) string {
	t.Helper()

	var payload struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode error response: %v", err)
	}

	return payload.Error.Code
}

type stubChallengeReader struct {
	challenge *challenge.Challenge
	err       error
}

func (s stubChallengeReader) GetForSubmission(_ context.Context, _ string, _ bool) (*challenge.Challenge, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.challenge == nil {
		return nil, apperrors.NotFound("CHALLENGE_NOT_FOUND", "Challenge not found")
	}
	copy := *s.challenge
	return &copy, nil
}

type stubRuntimeController struct {
	startResult *RuntimeStartResult
	startErr    error
	stopErr     error
}

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time { return c.now }

func (s *stubRuntimeController) Start(_ context.Context, _ RuntimeStartSpec) (*RuntimeStartResult, error) {
	if s.startErr != nil {
		return nil, s.startErr
	}
	if s.startResult == nil {
		return &RuntimeStartResult{ContainerID: "stub-container"}, nil
	}
	return s.startResult, nil
}

func (s *stubRuntimeController) Stop(_ context.Context, _ string) error {
	return s.stopErr
}

func ptrString(v string) *string { return &v }
func ptrInt(v int) *int          { return &v }
func ptrTime(v time.Time) *time.Time {
	return &v
}
