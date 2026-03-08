package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ctf/backend/internal/admin"
	"ctf/backend/internal/auth"
	"ctf/backend/internal/config"
	"ctf/backend/internal/game"
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

type testGameRepo struct {
	announcements    []game.Announcement
	challenge        game.Challenge
	flag             string
	scoreboard       []game.ScoreboardEntry
	solved           map[int64]bool
	nextSubmissionID int64
}

type testAdminRepo struct {
	challenges    []admin.ChallengeSummary
	announcements []admin.Announcement
	submissions   []admin.SubmissionRecord
	instances     []admin.InstanceRecord
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
	gameRepo := &testGameRepo{
		announcements: []game.Announcement{{ID: 1, Title: "Welcome", Content: "Hello", Pinned: true}},
		challenge: game.Challenge{
			ID:          1,
			Slug:        "web-welcome",
			Title:       "Welcome Panel",
			Category:    "web",
			Points:      100,
			Difficulty:  "easy",
			Description: "demo",
			Dynamic:     true,
		},
		flag:             "flag{welcome}",
		scoreboard:       []game.ScoreboardEntry{{UserID: 1, Username: "alice", DisplayName: "Alice", Score: 100}},
		solved:           make(map[int64]bool),
		nextSubmissionID: 1,
	}
	adminRepo := &testAdminRepo{
		challenges:    []admin.ChallengeSummary{{ID: 1, Slug: "web-welcome", Title: "Welcome Panel", Category: "web"}},
		announcements: []admin.Announcement{{ID: 1, Title: "Welcome", Published: true}},
		submissions:   []admin.SubmissionRecord{{ID: 1, ChallengeSlug: "web-welcome", Username: "alice"}},
		instances:     []admin.InstanceRecord{{ID: 1, ChallengeSlug: "web-welcome", Username: "alice", Status: "running"}},
	}

	cfg := config.Load()
	tokens := auth.NewTokenManager(cfg.JWTSecret, cfg.JWTTTL)
	authService := auth.NewService(userRepo, tokens)
	adminService := admin.NewService(adminRepo)
	gameService := game.NewService(gameRepo)
	runtimeService := runtime.NewService("http://localhost:8080", &testManager{}, runtimeRepo)
	return NewServerForTests(cfg, adminService, authService, gameService, runtimeService)
}

func (r *testRuntimeRepo) ListChallenges(context.Context) ([]runtime.ChallengeSummary, error) {
	cfg := r.challenge.Challenge
	return []runtime.ChallengeSummary{{ID: cfg.ID, Slug: cfg.Slug, Title: cfg.Title, Category: cfg.Category, Points: cfg.Points, Dynamic: cfg.Dynamic}}, nil
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
	user := auth.User{ID: id, Role: params.RoleName, Username: params.Username, Email: params.Email, DisplayName: params.DisplayName, Status: "active", PasswordHash: params.PasswordHash}
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

func (r *testGameRepo) ListAnnouncements(context.Context) ([]game.Announcement, error) {
	return r.announcements, nil
}

func (r *testGameRepo) GetChallenge(_ context.Context, challengeRef string) (game.Challenge, string, error) {
	if challengeRef != r.challenge.Slug && challengeRef != "1" {
		return game.Challenge{}, "", game.ErrChallengeNotFound
	}
	return r.challenge, r.flag, nil
}

func (r *testGameRepo) CreateSubmission(_ context.Context, _ int64, _ int64, _ string, _ bool, _ string) (int64, time.Time, error) {
	id := r.nextSubmissionID
	r.nextSubmissionID++
	return id, time.Now().UTC(), nil
}

func (r *testGameRepo) HasSolved(_ context.Context, _ int64, userID int64) (bool, error) {
	return r.solved[userID], nil
}

func (r *testGameRepo) CreateSolve(_ context.Context, _ int64, userID int64, _ int64, _ int) (time.Time, error) {
	r.solved[userID] = true
	now := time.Now().UTC()
	return now, nil
}

func (r *testGameRepo) ListScoreboard(context.Context) ([]game.ScoreboardEntry, error) {
	return r.scoreboard, nil
}

func (r *testAdminRepo) ListChallenges(context.Context) ([]admin.ChallengeSummary, error) {
	return r.challenges, nil
}
func (r *testAdminRepo) CreateChallenge(_ context.Context, input admin.UpsertChallengeInput) (admin.ChallengeSummary, error) {
	challenge := admin.ChallengeSummary{ID: 2, Slug: input.Slug, Title: input.Title, Category: input.CategorySlug, Points: input.Points, Visible: input.Visible, DynamicEnabled: input.DynamicEnabled}
	r.challenges = append(r.challenges, challenge)
	return challenge, nil
}
func (r *testAdminRepo) UpdateChallenge(_ context.Context, challengeID int64, input admin.UpsertChallengeInput) (admin.ChallengeSummary, error) {
	return admin.ChallengeSummary{ID: challengeID, Slug: input.Slug, Title: input.Title, Category: input.CategorySlug, Points: input.Points, Visible: input.Visible, DynamicEnabled: input.DynamicEnabled}, nil
}
func (r *testAdminRepo) ListAnnouncements(context.Context) ([]admin.Announcement, error) {
	return r.announcements, nil
}
func (r *testAdminRepo) CreateAnnouncement(_ context.Context, _ int64, input admin.CreateAnnouncementInput) (admin.Announcement, error) {
	announcement := admin.Announcement{ID: 2, Title: input.Title, Content: input.Content, Pinned: input.Pinned, Published: input.Published}
	r.announcements = append(r.announcements, announcement)
	return announcement, nil
}
func (r *testAdminRepo) ListSubmissions(context.Context) ([]admin.SubmissionRecord, error) {
	return r.submissions, nil
}
func (r *testAdminRepo) ListInstances(context.Context) ([]admin.InstanceRecord, error) {
	return r.instances, nil
}
func (r *testAdminRepo) TerminateInstance(_ context.Context, instanceID int64, terminatedAt time.Time) (admin.InstanceRecord, error) {
	for i := range r.instances {
		if r.instances[i].ID == instanceID {
			r.instances[i].Status = "terminated"
			t := terminatedAt.UTC()
			r.instances[i].TerminatedAt = &t
			return r.instances[i], nil
		}
	}
	return admin.InstanceRecord{}, admin.ErrResourceNotFound
}

func TestHealthEndpoint(t *testing.T) {
	server := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	res := httptest.NewRecorder()
	server.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
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
	token := registerTestUser(t, server)
	meReq := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+token)
	meRes := httptest.NewRecorder()
	server.Handler().ServeHTTP(meRes, meReq)
	if meRes.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", meRes.Code)
	}
}

func TestChallengeDetailEndpoint(t *testing.T) {
	server := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/challenges/1", nil)
	res := httptest.NewRecorder()
	server.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
}

func TestSubmitFlagEndpoint(t *testing.T) {
	server := newTestServer(t)
	token := registerTestUser(t, server)
	body := []byte(`{"flag":"flag{welcome}"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/challenges/1/submissions", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.RemoteAddr = "127.0.0.1:54321"
	res := httptest.NewRecorder()
	server.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
}

func TestScoreboardEndpoint(t *testing.T) {
	server := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/scoreboard", nil)
	res := httptest.NewRecorder()
	server.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
}

func TestAdminEndpointRequiresAdminRole(t *testing.T) {
	server := newTestServer(t)
	playerToken := registerTestUser(t, server)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/challenges", nil)
	req.Header.Set("Authorization", "Bearer "+playerToken)
	res := httptest.NewRecorder()
	server.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.Code)
	}
}

func TestAdminChallengesEndpoint(t *testing.T) {
	server := newTestServer(t)
	adminToken := issueAdminToken(t, server)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/challenges", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	res := httptest.NewRecorder()
	server.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
}

func TestAdminTerminateInstanceEndpoint(t *testing.T) {
	server := newTestServer(t)
	adminToken := issueAdminToken(t, server)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/instances/1/terminate", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	res := httptest.NewRecorder()
	server.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
}

func registerTestUser(t *testing.T, server *Server) string {
	t.Helper()
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
	return token
}

func issueAdminToken(t *testing.T, server *Server) string {
	t.Helper()
	result, err := server.auth.Register(context.Background(), auth.RegisterInput{Username: "admin", Email: "admin@example.com", Password: "AdminPass123!", DisplayName: "Admin"})
	if err != nil {
		t.Fatalf("register admin: %v", err)
	}
	user, err := server.auth.Me(context.Background(), result.User.ID)
	if err != nil {
		t.Fatalf("load admin user: %v", err)
	}
	user.Role = "admin"
	token, _, err := auth.NewTokenManager(server.cfg.JWTSecret, server.cfg.JWTTTL).Sign(auth.TokenClaims{UserID: user.ID, Role: "admin"})
	if err != nil {
		t.Fatalf("issue admin token: %v", err)
	}
	return token
}
