package app

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ctf/backend/internal/admin"
	"ctf/backend/internal/auth"
	"ctf/backend/internal/config"
	"ctf/backend/internal/contest"
	"ctf/backend/internal/game"
	"ctf/backend/internal/runtime"
)

type testManager struct {
	stopped    []string
	containers map[string]runtime.ManagedContainer
}

func (m *testManager) Start(_ context.Context, req runtime.StartRequest) (runtime.StartedContainer, error) {
	if m.containers == nil {
		m.containers = make(map[string]runtime.ManagedContainer)
	}
	m.containers["test-container"] = runtime.ManagedContainer{ContainerID: "test-container", ChallengeID: req.ChallengeID, UserID: req.UserID}
	return runtime.StartedContainer{
		ContainerID:   "test-container",
		ContainerName: "test-name",
		HostIP:        "127.0.0.1",
		HostPort:      18081,
	}, nil
}

func (m *testManager) Stop(_ context.Context, containerID string) error {
	m.stopped = append(m.stopped, containerID)
	if m.containers != nil {
		delete(m.containers, containerID)
	}
	return nil
}

func (m *testManager) Exists(_ context.Context, containerID string) (bool, error) {
	_, ok := m.containers[containerID]
	return ok, nil
}

func (m *testManager) ListManagedContainers(_ context.Context) ([]runtime.ManagedContainer, error) {
	items := make([]runtime.ManagedContainer, 0, len(m.containers))
	for _, item := range m.containers {
		items = append(items, item)
	}
	return items, nil
}

type testRuntimeRepo struct {
	challenge runtime.RuntimeConfigRecord
	instance  *runtime.InstanceRecord
	history   *runtime.InstanceRecord
}

type testUserRepo struct {
	users      map[int64]auth.User
	identifier map[string]int64
	nextID     int64
}

type testContestRepo struct {
	current contest.Contest
}

type testGameRepo struct {
	announcements      []game.Announcement
	challenge          game.Challenge
	hiddenChallengeRef string
	flag               string
	submissions        []game.UserSubmission
	solves             []game.UserSolve
	scoreboard         []game.ScoreboardEntry
	solved             map[int64]bool
	nextSubmissionID   int64
	attachment         game.Attachment
	attachmentPath     string
}

type testAdminRepo struct {
	challenges    []admin.ChallengeSummary
	challenge     admin.ChallengeDetail
	attachments   map[int64]testAttachmentFile
	users         []admin.UserRecord
	auditLogs     []admin.AuditLogRecord
	announcements []admin.Announcement
	submissions   []admin.SubmissionRecord
	instances     []admin.InstanceRecord
}

type testAttachmentFile struct {
	attachment admin.Attachment
	path       string
}

func newTestServer(t *testing.T) (*Server, *testRuntimeRepo) {
	t.Helper()

	attachmentDir := t.TempDir()
	manager := &testManager{}
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
				MaxRenewCount:   1,
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
	now := time.Date(2025, time.March, 8, 9, 0, 0, 0, time.UTC)
	attachmentPath := filepath.Join(attachmentDir, "statement.pdf")
	if err := os.WriteFile(attachmentPath, []byte("statement-data"), 0o644); err != nil {
		t.Fatalf("write attachment fixture: %v", err)
	}
	contestRepo := &testContestRepo{
		current: contest.Contest{ID: 1, Slug: "recruit-2025", Title: "Recruit 2025", Description: "demo contest", Status: contest.StatusRunning},
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
			Attachments: []game.Attachment{{ID: 1, Filename: "statement.pdf", ContentType: "application/pdf", SizeBytes: 14}},
		},
		hiddenChallengeRef: "2",
		flag:               "flag{welcome}",
		submissions: []game.UserSubmission{{
			ID:             1,
			ChallengeID:    1,
			ChallengeSlug:  "web-welcome",
			ChallengeTitle: "Welcome Panel",
			Category:       "web",
			Correct:        true,
			SubmittedAt:    now,
			SourceIP:       "127.0.0.1",
		}},
		solves: []game.UserSolve{{
			ID:             1,
			ChallengeID:    1,
			ChallengeSlug:  "web-welcome",
			ChallengeTitle: "Welcome Panel",
			Category:       "web",
			SubmissionID:   1,
			AwardedPoints:  100,
			SolvedAt:       now.Add(5 * time.Minute),
		}},
		scoreboard:       []game.ScoreboardEntry{{UserID: 1, Username: "alice", DisplayName: "Alice", Score: 100, Solves: []game.ScoreboardSolve{{ChallengeID: 1, ChallengeSlug: "web-welcome", ChallengeTitle: "Welcome Panel", Category: "web", Difficulty: "easy", AwardedPoints: 100, SolvedAt: now}}}},
		solved:           make(map[int64]bool),
		nextSubmissionID: 1,
		attachment:       game.Attachment{ID: 1, Filename: "statement.pdf", ContentType: "application/pdf", SizeBytes: 14},
		attachmentPath:   attachmentPath,
	}
	adminRepo := &testAdminRepo{
		challenges: []admin.ChallengeSummary{{ID: 1, Slug: "web-welcome", Title: "Welcome Panel", Category: "web", Points: 100, Visible: true, DynamicEnabled: true}},
		challenge: admin.ChallengeDetail{
			ID:             1,
			Slug:           "web-welcome",
			Title:          "Welcome Panel",
			Category:       "web",
			Description:    "demo",
			Points:         100,
			Difficulty:     "easy",
			FlagType:       "static",
			FlagValue:      "flag{welcome}",
			Visible:        true,
			DynamicEnabled: true,
			SortOrder:      10,
			Attachments:    []admin.Attachment{{ID: 1, Filename: "statement.pdf", ContentType: "application/pdf", SizeBytes: 14}},
			RuntimeConfig: admin.RuntimeConfig{
				Enabled:         true,
				ImageName:       "ctf/web-welcome:dev",
				ExposedProtocol: "http",
				ContainerPort:   80,
				DefaultTTL:      1800,
				MaxRenewCount:   1,
				MemoryLimitMB:   256,
				CPUMilli:        500,
			},
		},
		attachments: map[int64]testAttachmentFile{
			1: {attachment: admin.Attachment{ID: 1, Filename: "statement.pdf", ContentType: "application/pdf", SizeBytes: 14}, path: attachmentPath},
		},
		users: []admin.UserRecord{
			{ID: 1, Role: "admin", Username: "root", Email: "root@example.com", DisplayName: "Root", Status: "active", CreatedAt: now},
			{ID: 2, Role: "player", Username: "alice", Email: "alice@example.com", DisplayName: "Alice", Status: "active", CreatedAt: now},
			{ID: 3, Role: "ops", Username: "ops", Email: "ops@example.com", DisplayName: "Ops", Status: "active", CreatedAt: now},
		},
		auditLogs:     []admin.AuditLogRecord{{ID: 1, Action: "challenge.update", ResourceType: "challenge", ResourceID: "1", CreatedAt: now}},
		announcements: []admin.Announcement{{ID: 1, Title: "Welcome", Published: true}},
		submissions:   []admin.SubmissionRecord{{ID: 1, ChallengeSlug: "web-welcome", Username: "alice"}},
		instances:     []admin.InstanceRecord{{ID: 1, ChallengeID: 1, ChallengeSlug: "web-welcome", Username: "alice", Status: "running", ContainerID: "test-container"}},
	}

	cfg := config.Load()
	cfg.AttachmentStorageDir = attachmentDir
	cfg.RedisAddr = ""
	cfg.SubmissionRateLimitWindowSeconds = 60
	cfg.SubmissionRateLimitMax = 2
	cfg.LoginRateLimitWindowSeconds = 60
	cfg.LoginRateLimitMax = 2
	cfg.RegisterRateLimitWindowSeconds = 300
	cfg.RegisterRateLimitMax = 5
	cfg.AdminWriteRateLimitWindowSeconds = 60
	cfg.AdminWriteRateLimitMax = 1
	tokens := auth.NewTokenManager(cfg.JWTSecret, cfg.JWTTTL)
	authService := auth.NewService(userRepo, tokens)
	adminService := admin.NewServiceWithManager(adminRepo, cfg.AttachmentStorageDir, manager)
	contestService := contest.NewService(contestRepo)
	gameService := game.NewService(gameRepo)
	runtimeService := runtime.NewService("http://localhost:8080", manager, runtimeRepo)
	server := NewServerForTests(cfg, adminService, authService, contestService, gameService, runtimeService)
	if limiter, ok := server.limiters.Submission.(*memoryRateLimiter); ok {
		limiter.now = func() time.Time { return now }
	}
	if limiter, ok := server.limiters.Login.(*memoryRateLimiter); ok {
		limiter.now = func() time.Time { return now }
	}
	if limiter, ok := server.limiters.Register.(*memoryRateLimiter); ok {
		limiter.now = func() time.Time { return now }
	}
	if limiter, ok := server.limiters.AdminWrite.(*memoryRateLimiter); ok {
		limiter.now = func() time.Time { return now }
	}
	return server, runtimeRepo
}

func (r *testContestRepo) Current(context.Context) (contest.Contest, error) {
	return r.current, nil
}

func (r *testContestRepo) Update(_ context.Context, input contest.UpdateInput) (contest.Contest, error) {
	r.current.Status = contest.NormalizeStatus(input.Status)
	return r.current, nil
}

func (r *testRuntimeRepo) ListChallenges(context.Context) ([]runtime.ChallengeSummary, error) {
	cfg := r.challenge.Challenge
	return []runtime.ChallengeSummary{{ID: cfg.ID, Slug: cfg.Slug, Title: cfg.Title, Category: cfg.Category, Points: cfg.Points, Difficulty: "normal", Dynamic: cfg.Dynamic}}, nil
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
	r.history = &record
	return record, nil
}

func (r *testRuntimeRepo) RenewInstance(_ context.Context, instanceID int64, expiresAt time.Time) (runtime.InstanceRecord, error) {
	if r.instance == nil || r.instance.ID != instanceID {
		return runtime.InstanceRecord{}, runtime.ErrRepositoryNotFound
	}
	r.instance.Instance.RenewCount++
	r.instance.Instance.ExpiresAt = expiresAt
	record := *r.instance
	r.history = &record
	return *r.instance, nil
}

func (r *testRuntimeRepo) TerminateInstance(_ context.Context, instanceID int64, terminatedAt time.Time) error {
	if r.instance == nil || r.instance.ID != instanceID {
		return runtime.ErrRepositoryNotFound
	}
	record := *r.instance
	record.Instance.Status = "terminated"
	t := terminatedAt.UTC()
	record.Instance.TerminatedAt = &t
	r.history = &record
	r.instance = nil
	return nil
}

func (r *testRuntimeRepo) ListExpiredInstances(_ context.Context, now time.Time) ([]runtime.InstanceRecord, error) {
	if r.instance == nil || r.instance.Instance.ExpiresAt.After(now) {
		return nil, nil
	}
	return []runtime.InstanceRecord{*r.instance}, nil
}

func (r *testRuntimeRepo) CountActiveInstances(_ context.Context, challengeID string) (int, error) {
	if r.instance != nil && r.instance.Instance.ChallengeID == challengeID {
		return 1, nil
	}
	return 0, nil
}

func (r *testRuntimeRepo) GetLatestInstance(_ context.Context, userID int64, challengeID string) (runtime.InstanceRecord, error) {
	if r.history == nil || r.history.Instance.UserID != userID || r.history.Instance.ChallengeID != challengeID {
		return runtime.InstanceRecord{}, runtime.ErrRepositoryNotFound
	}
	return *r.history, nil
}

func (r *testRuntimeRepo) ListActiveInstances(context.Context) ([]runtime.InstanceRecord, error) {
	if r.instance == nil {
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

func (r *testUserRepo) UpdateLastLogin(_ context.Context, userID int64, loggedInAt time.Time) error {
	user, ok := r.users[userID]
	if !ok {
		return runtime.ErrRepositoryNotFound
	}
	user.LastLoginAt = &loggedInAt
	r.users[userID] = user
	return nil
}

func (r *testGameRepo) ListAnnouncements(context.Context) ([]game.Announcement, error) {
	return r.announcements, nil
}

func (r *testGameRepo) GetChallenge(_ context.Context, challengeRef string) (game.Challenge, string, error) {
	if challengeRef == r.hiddenChallengeRef {
		return game.Challenge{}, "", game.ErrChallengeNotFound
	}
	if challengeRef != r.challenge.Slug && challengeRef != "1" {
		return game.Challenge{}, "", game.ErrChallengeNotFound
	}
	return r.challenge, r.flag, nil
}

func (r *testGameRepo) GetChallengeAttachment(_ context.Context, challengeRef string, attachmentID int64) (game.Attachment, string, error) {
	if challengeRef == r.hiddenChallengeRef {
		return game.Attachment{}, "", game.ErrChallengeNotFound
	}
	if challengeRef != r.challenge.Slug && challengeRef != "1" {
		return game.Attachment{}, "", game.ErrChallengeNotFound
	}
	if attachmentID != r.attachment.ID {
		return game.Attachment{}, "", game.ErrAttachmentNotFound
	}
	return r.attachment, r.attachmentPath, nil
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

func (r *testGameRepo) ListUserSubmissions(_ context.Context, _ int64) ([]game.UserSubmission, error) {
	return r.submissions, nil
}

func (r *testGameRepo) ListUserSolves(_ context.Context, _ int64) ([]game.UserSolve, error) {
	return r.solves, nil
}

func (r *testGameRepo) ListScoreboard(context.Context) ([]game.ScoreboardEntry, error) {
	return r.scoreboard, nil
}

func (r *testAdminRepo) ListChallenges(context.Context) ([]admin.ChallengeSummary, error) {
	return r.challenges, nil
}
func (r *testAdminRepo) GetChallenge(_ context.Context, challengeID int64) (admin.ChallengeDetail, error) {
	if r.challenge.ID != challengeID {
		return admin.ChallengeDetail{}, admin.ErrResourceNotFound
	}
	return r.challenge, nil
}
func (r *testAdminRepo) CreateChallenge(_ context.Context, input admin.UpsertChallengeInput) (admin.ChallengeSummary, error) {
	challenge := admin.ChallengeSummary{ID: 2, Slug: input.Slug, Title: input.Title, Category: input.CategorySlug, Points: input.Points, Visible: input.Visible, DynamicEnabled: input.DynamicEnabled}
	r.challenges = append(r.challenges, challenge)
	return challenge, nil
}
func (r *testAdminRepo) UpdateChallenge(_ context.Context, challengeID int64, input admin.UpsertChallengeInput) (admin.ChallengeSummary, error) {
	r.challenge = admin.ChallengeDetail{ID: challengeID, Slug: input.Slug, Title: input.Title, Category: input.CategorySlug, Description: input.Description, Points: input.Points, Difficulty: input.Difficulty, FlagType: input.FlagType, FlagValue: input.FlagValue, Visible: input.Visible, DynamicEnabled: input.DynamicEnabled, SortOrder: input.SortOrder, Attachments: r.challenge.Attachments}
	if input.RuntimeConfig != nil {
		r.challenge.RuntimeConfig = *input.RuntimeConfig
	}
	return admin.ChallengeSummary{ID: challengeID, Slug: input.Slug, Title: input.Title, Category: input.CategorySlug, Points: input.Points, Visible: input.Visible, DynamicEnabled: input.DynamicEnabled}, nil
}
func (r *testAdminRepo) CreateAttachment(_ context.Context, _ int64, filename, storagePath, contentType string, sizeBytes int64) (admin.Attachment, error) {
	id := int64(len(r.attachments) + 1)
	item := admin.Attachment{ID: id, Filename: filename, ContentType: contentType, SizeBytes: sizeBytes}
	r.attachments[id] = testAttachmentFile{attachment: item, path: storagePath}
	r.challenge.Attachments = append(r.challenge.Attachments, item)
	return item, nil
}
func (r *testAdminRepo) GetAttachment(_ context.Context, challengeID int64, attachmentID int64) (admin.Attachment, string, error) {
	if r.challenge.ID != challengeID {
		return admin.Attachment{}, "", admin.ErrResourceNotFound
	}
	item, ok := r.attachments[attachmentID]
	if !ok {
		return admin.Attachment{}, "", admin.ErrResourceNotFound
	}
	return item.attachment, item.path, nil
}
func (r *testAdminRepo) ListUsers(context.Context) ([]admin.UserRecord, error) {
	return r.users, nil
}
func (r *testAdminRepo) UpdateUser(_ context.Context, userID int64, input admin.UpdateUserInput) (admin.UserRecord, error) {
	for i := range r.users {
		if r.users[i].ID == userID {
			r.users[i].Role = input.Role
			r.users[i].DisplayName = input.DisplayName
			r.users[i].Status = input.Status
			return r.users[i], nil
		}
	}
	return admin.UserRecord{}, admin.ErrResourceNotFound
}
func (r *testAdminRepo) ListAuditLogs(context.Context) ([]admin.AuditLogRecord, error) {
	return r.auditLogs, nil
}
func (r *testAdminRepo) CreateAuditLog(_ context.Context, actorUserID *int64, action, resourceType, resourceID string, details map[string]any) error {
	id := int64(len(r.auditLogs) + 1)
	r.auditLogs = append(r.auditLogs, admin.AuditLogRecord{ID: id, ActorUserID: actorUserID, Action: action, ResourceType: resourceType, ResourceID: resourceID, Details: details, CreatedAt: time.Now().UTC()})
	return nil
}
func (r *testAdminRepo) ListAnnouncements(context.Context) ([]admin.Announcement, error) {
	return r.announcements, nil
}
func (r *testAdminRepo) CreateAnnouncement(_ context.Context, _ int64, input admin.CreateAnnouncementInput) (admin.Announcement, error) {
	announcement := admin.Announcement{ID: 2, Title: input.Title, Content: input.Content, Pinned: input.Pinned, Published: input.Published}
	r.announcements = append(r.announcements, announcement)
	return announcement, nil
}
func (r *testAdminRepo) DeleteAnnouncement(_ context.Context, announcementID int64) (admin.Announcement, error) {
	for i := range r.announcements {
		if r.announcements[i].ID == announcementID {
			item := r.announcements[i]
			r.announcements = append(r.announcements[:i], r.announcements[i+1:]...)
			return item, nil
		}
	}
	return admin.Announcement{}, admin.ErrResourceNotFound
}
func (r *testAdminRepo) ListSubmissions(context.Context) ([]admin.SubmissionRecord, error) {
	return r.submissions, nil
}
func (r *testAdminRepo) ListInstances(context.Context) ([]admin.InstanceRecord, error) {
	return r.instances, nil
}
func (r *testAdminRepo) GetInstance(_ context.Context, instanceID int64) (admin.InstanceRecord, error) {
	for _, item := range r.instances {
		if item.ID == instanceID {
			return item, nil
		}
	}
	return admin.InstanceRecord{}, admin.ErrResourceNotFound
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
	server, _ := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	res := httptest.NewRecorder()
	server.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
}

func TestMetricsEndpoint(t *testing.T) {
	server, _ := newTestServer(t)
	healthReq := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	healthRes := httptest.NewRecorder()
	server.Handler().ServeHTTP(healthRes, healthReq)

	metricsReq := httptest.NewRequest(http.MethodGet, "/api/v1/metrics", nil)
	metricsRes := httptest.NewRecorder()
	server.Handler().ServeHTTP(metricsRes, metricsReq)
	if metricsRes.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", metricsRes.Code)
	}
	body := metricsRes.Body.String()
	if !strings.Contains(body, "ctf_http_requests_total") {
		t.Fatalf("expected request counter in metrics, got %q", body)
	}
	if !strings.Contains(body, "ctf_http_health_requests_total") {
		t.Fatalf("expected health counter in metrics, got %q", body)
	}
}

func loginAsPlayer(t *testing.T, server *Server) string {
	t.Helper()
	registerTestUser(t, server)
	body := []byte(`{"identifier":"alice@example.com","password":"Password123!"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.RemoteAddr = "127.0.0.1:54321"
	res := httptest.NewRecorder()
	server.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected login 200, got %d: %s", res.Code, res.Body.String())
	}
	var payload struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if payload.Token == "" {
		t.Fatalf("expected token in login response")
	}
	return payload.Token
}

func TestProtectedInstanceEndpointRequiresBearerToken(t *testing.T) {
	server, _ := newTestServer(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/challenges/1/instances/me", nil)
	res := httptest.NewRecorder()
	server.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", res.Code)
	}
}

func TestRegisterThenAccessMe(t *testing.T) {
	server, _ := newTestServer(t)
	token := registerTestUser(t, server)
	meReq := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+token)
	meRes := httptest.NewRecorder()
	server.Handler().ServeHTTP(meRes, meReq)
	if meRes.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", meRes.Code)
	}
}

func TestMySubmissionsEndpoint(t *testing.T) {
	server, _ := newTestServer(t)
	token := registerTestUser(t, server)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me/submissions", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()
	server.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
}

func TestMySolvesEndpoint(t *testing.T) {
	server, _ := newTestServer(t)
	token := registerTestUser(t, server)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me/solves", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()
	server.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
}

func TestChallengeDetailEndpoint(t *testing.T) {
	server, _ := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/challenges/1", nil)
	res := httptest.NewRecorder()
	server.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
}

func TestChallengeAttachmentDownloadEndpoint(t *testing.T) {
	server, _ := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/challenges/1/attachments/1", nil)
	res := httptest.NewRecorder()
	server.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
	if got := res.Header().Get("Content-Disposition"); got == "" {
		t.Fatalf("expected content disposition header")
	}
	if body := res.Body.String(); body != "statement-data" {
		t.Fatalf("unexpected attachment body: %q", body)
	}
}

func TestChallengeAttachmentDownloadRejectsHiddenChallenge(t *testing.T) {
	server, _ := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/challenges/2/attachments/1", nil)
	res := httptest.NewRecorder()
	server.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", res.Code)
	}
}

func TestSubmitFlagEndpoint(t *testing.T) {
	server, _ := newTestServer(t)
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

func TestRegisterRateLimitEndpoint(t *testing.T) {
	server, _ := newTestServer(t)
	if limiter, ok := server.limiters.Register.(*memoryRateLimiter); ok {
		limiter.max = 1
	}
	body := []byte(`{"username":"alice","email":"alice@example.com","password":"Password123!","display_name":"Alice"}`)
	firstReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	firstReq.RemoteAddr = "127.0.0.1:54321"
	firstRes := httptest.NewRecorder()
	server.Handler().ServeHTTP(firstRes, firstReq)
	if firstRes.Code != http.StatusCreated {
		t.Fatalf("expected first register 201, got %d", firstRes.Code)
	}

	secondReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	secondReq.RemoteAddr = "127.0.0.1:54321"
	secondRes := httptest.NewRecorder()
	server.Handler().ServeHTTP(secondRes, secondReq)
	if secondRes.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", secondRes.Code)
	}
	assertAPIErrorCode(t, secondRes.Body.Bytes(), "register_rate_limited")
}

func TestLoginRateLimitEndpoint(t *testing.T) {
	server, _ := newTestServer(t)
	registerTestUser(t, server)
	body := []byte(`{"identifier":"alice@example.com","password":"wrong-password"}`)
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
		req.RemoteAddr = "127.0.0.1:54321"
		res := httptest.NewRecorder()
		server.Handler().ServeHTTP(res, req)
		if res.Code != http.StatusUnauthorized {
			t.Fatalf("expected warmup 401, got %d", res.Code)
		}
	}
	thirdReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	thirdReq.RemoteAddr = "127.0.0.1:54321"
	thirdRes := httptest.NewRecorder()
	server.Handler().ServeHTTP(thirdRes, thirdReq)
	if thirdRes.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", thirdRes.Code)
	}
	assertAPIErrorCode(t, thirdRes.Body.Bytes(), "login_rate_limited")
}

func TestSubmitFlagRateLimitEndpoint(t *testing.T) {
	server, _ := newTestServer(t)
	token := registerTestUser(t, server)
	body := []byte(`{"flag":"wrong"}`)
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/challenges/1/submissions", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		res := httptest.NewRecorder()
		server.Handler().ServeHTTP(res, req)
		if res.Code != http.StatusOK {
			t.Fatalf("expected warmup 200, got %d", res.Code)
		}
	}
	thirdReq := httptest.NewRequest(http.MethodPost, "/api/v1/challenges/1/submissions", bytes.NewReader(body))
	thirdReq.Header.Set("Authorization", "Bearer "+token)
	thirdRes := httptest.NewRecorder()
	server.Handler().ServeHTTP(thirdRes, thirdReq)
	if thirdRes.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", thirdRes.Code)
	}
}

func TestRenewInstanceEndpoint(t *testing.T) {
	server, _ := newTestServer(t)
	token := registerTestUser(t, server)
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/challenges/1/instances/me", nil)
	createReq.Header.Set("Authorization", "Bearer "+token)
	createRes := httptest.NewRecorder()
	server.Handler().ServeHTTP(createRes, createReq)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", createRes.Code)
	}
	renewReq := httptest.NewRequest(http.MethodPost, "/api/v1/challenges/1/instances/me/renew", nil)
	renewReq.Header.Set("Authorization", "Bearer "+token)
	renewRes := httptest.NewRecorder()
	server.Handler().ServeHTTP(renewRes, renewReq)
	if renewRes.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", renewRes.Code)
	}
}

func TestRenewInstanceEndpointRejectsLimitReached(t *testing.T) {
	server, _ := newTestServer(t)
	token := registerTestUser(t, server)
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/challenges/1/instances/me", nil)
	createReq.Header.Set("Authorization", "Bearer "+token)
	createRes := httptest.NewRecorder()
	server.Handler().ServeHTTP(createRes, createReq)
	firstRenewReq := httptest.NewRequest(http.MethodPost, "/api/v1/challenges/1/instances/me/renew", nil)
	firstRenewReq.Header.Set("Authorization", "Bearer "+token)
	firstRenewRes := httptest.NewRecorder()
	server.Handler().ServeHTTP(firstRenewRes, firstRenewReq)
	secondRenewReq := httptest.NewRequest(http.MethodPost, "/api/v1/challenges/1/instances/me/renew", nil)
	secondRenewReq.Header.Set("Authorization", "Bearer "+token)
	secondRenewRes := httptest.NewRecorder()
	server.Handler().ServeHTTP(secondRenewRes, secondRenewReq)
	if secondRenewRes.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", secondRenewRes.Code)
	}
}

func TestCreateInstanceEndpointRejectsChallengeCapacity(t *testing.T) {
	server, runtimeRepo := newTestServer(t)
	runtimeRepo.challenge.Challenge.MaxActiveInstances = 1

	firstToken := registerTestUser(t, server)
	firstReq := httptest.NewRequest(http.MethodPost, "/api/v1/challenges/1/instances/me", nil)
	firstReq.Header.Set("Authorization", "Bearer "+firstToken)
	firstRes := httptest.NewRecorder()
	server.Handler().ServeHTTP(firstRes, firstReq)
	if firstRes.Code != http.StatusCreated {
		t.Fatalf("expected first create 201, got %d", firstRes.Code)
	}

	secondToken := registerAnotherTestUser(t, server)
	secondReq := httptest.NewRequest(http.MethodPost, "/api/v1/challenges/1/instances/me", nil)
	secondReq.Header.Set("Authorization", "Bearer "+secondToken)
	secondRes := httptest.NewRecorder()
	server.Handler().ServeHTTP(secondRes, secondReq)
	if secondRes.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", secondRes.Code)
	}
	assertAPIErrorCode(t, secondRes.Body.Bytes(), "instance_capacity_reached")
}

func TestCreateInstanceEndpointRejectsCooldownActive(t *testing.T) {
	server, runtimeRepo := newTestServer(t)
	runtimeRepo.challenge.Challenge.UserCooldown = time.Hour

	token := registerTestUser(t, server)
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/challenges/1/instances/me", nil)
	createReq.Header.Set("Authorization", "Bearer "+token)
	createRes := httptest.NewRecorder()
	server.Handler().ServeHTTP(createRes, createReq)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("expected create 201, got %d", createRes.Code)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/v1/challenges/1/instances/me", nil)
	deleteReq.Header.Set("Authorization", "Bearer "+token)
	deleteRes := httptest.NewRecorder()
	server.Handler().ServeHTTP(deleteRes, deleteReq)
	if deleteRes.Code != http.StatusOK {
		t.Fatalf("expected delete 200, got %d", deleteRes.Code)
	}

	retryReq := httptest.NewRequest(http.MethodPost, "/api/v1/challenges/1/instances/me", nil)
	retryReq.Header.Set("Authorization", "Bearer "+token)
	retryRes := httptest.NewRecorder()
	server.Handler().ServeHTTP(retryRes, retryReq)
	if retryRes.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", retryRes.Code)
	}
	assertAPIErrorCode(t, retryRes.Body.Bytes(), "instance_cooldown_active")
}

func assertAPIErrorCode(t *testing.T, body []byte, want string) {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if got, _ := payload["error"].(string); got != want {
		t.Fatalf("expected error code %q, got %q", want, got)
	}
}

func TestScoreboardEndpoint(t *testing.T) {
	server, _ := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/scoreboard", nil)
	res := httptest.NewRecorder()
	server.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
}

func TestAdminEndpointRequiresPermission(t *testing.T) {
	server, _ := newTestServer(t)
	playerToken := registerTestUser(t, server)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/challenges", nil)
	req.Header.Set("Authorization", "Bearer "+playerToken)
	res := httptest.NewRecorder()
	server.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.Code)
	}
}

func TestOpsRoleCanAccessInstanceActionsButNotUserManagement(t *testing.T) {
	server, _ := newTestServer(t)
	opsToken := issueRoleToken(t, server, "ops")
	instanceReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/instances", nil)
	instanceReq.Header.Set("Authorization", "Bearer "+opsToken)
	instanceRes := httptest.NewRecorder()
	server.Handler().ServeHTTP(instanceRes, instanceReq)
	if instanceRes.Code != http.StatusOK {
		t.Fatalf("expected ops instance read 200, got %d", instanceRes.Code)
	}
	userReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
	userReq.Header.Set("Authorization", "Bearer "+opsToken)
	userRes := httptest.NewRecorder()
	server.Handler().ServeHTTP(userRes, userReq)
	if userRes.Code != http.StatusForbidden {
		t.Fatalf("expected ops user read 403, got %d", userRes.Code)
	}
}

func TestAdminChallengesEndpoint(t *testing.T) {
	server, _ := newTestServer(t)
	adminToken := issueAdminToken(t, server)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/challenges", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	res := httptest.NewRecorder()
	server.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
}

func TestAdminChallengeDetailEndpoint(t *testing.T) {
	server, _ := newTestServer(t)
	adminToken := issueAdminToken(t, server)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/challenges/1", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	res := httptest.NewRecorder()
	server.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
}

func TestAdminUpdateChallengePersistsRuntimeConfigPayload(t *testing.T) {
	server, _ := newTestServer(t)
	adminToken := issueAdminToken(t, server)
	body := []byte(`{"slug":"web-welcome","title":"Welcome Panel","category_slug":"web","description":"updated","points":100,"difficulty":"easy","flag_type":"static","flag_value":"flag{welcome}","dynamic_enabled":true,"visible":true,"sort_order":10,"runtime_config":{"enabled":true,"image_name":"ctf/web-welcome:v2","exposed_protocol":"http","container_port":8080,"default_ttl_seconds":2400,"max_renew_count":2,"memory_limit_mb":512,"cpu_limit_millicores":1000,"max_active_instances":5,"user_cooldown_seconds":120,"env":{"MODE":"prod"},"command":["/app/start"]}}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/challenges/1", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	res := httptest.NewRecorder()
	server.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
}

func TestAdminCreateChallengeRejectsInvalidFlagType(t *testing.T) {
	server, _ := newTestServer(t)
	adminToken := issueAdminToken(t, server)
	body := []byte(`{"slug":"regex-demo","title":"Regex Demo","category_slug":"web","description":"demo","points":100,"difficulty":"easy","flag_type":"script","flag_value":"flag{demo}","dynamic_enabled":false,"visible":true,"sort_order":10}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/challenges", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	res := httptest.NewRecorder()
	server.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.Code)
	}
	assertAPIErrorCode(t, res.Body.Bytes(), "invalid_challenge_input")
}

func TestAdminUpdateChallengeRejectsInvalidRegexFlagType(t *testing.T) {
	server, _ := newTestServer(t)
	adminToken := issueAdminToken(t, server)
	body := []byte(`{"slug":"web-welcome","title":"Welcome Panel","category_slug":"web","description":"updated","points":100,"difficulty":"easy","flag_type":"regex","flag_value":"^(broken$","dynamic_enabled":true,"visible":true,"sort_order":10}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/challenges/1", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	res := httptest.NewRecorder()
	server.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.Code)
	}
	assertAPIErrorCode(t, res.Body.Bytes(), "invalid_challenge_input")
}

func TestAdminWriteRateLimitEndpoint(t *testing.T) {
	server, _ := newTestServer(t)
	adminToken := issueAdminToken(t, server)
	body := []byte(`{"title":"One","content":"hello","pinned":false,"published":true}`)
	firstReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/announcements", bytes.NewReader(body))
	firstReq.Header.Set("Authorization", "Bearer "+adminToken)
	firstReq.RemoteAddr = "127.0.0.1:54321"
	firstRes := httptest.NewRecorder()
	server.Handler().ServeHTTP(firstRes, firstReq)
	if firstRes.Code != http.StatusCreated {
		t.Fatalf("expected first create 201, got %d", firstRes.Code)
	}

	secondReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/announcements", bytes.NewReader(body))
	secondReq.Header.Set("Authorization", "Bearer "+adminToken)
	secondReq.RemoteAddr = "127.0.0.1:54321"
	secondRes := httptest.NewRecorder()
	server.Handler().ServeHTTP(secondRes, secondReq)
	if secondRes.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", secondRes.Code)
	}
	assertAPIErrorCode(t, secondRes.Body.Bytes(), "admin_rate_limited")
}

func TestAdminCreateAttachmentEndpoint(t *testing.T) {
	server, _ := newTestServer(t)
	adminToken := issueAdminToken(t, server)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "readme.txt")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write([]byte("hello")); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/challenges/1/attachments", &body)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	res := httptest.NewRecorder()
	server.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", res.Code)
	}
}

func TestAdminUsersEndpoint(t *testing.T) {
	server, _ := newTestServer(t)
	adminToken := issueAdminToken(t, server)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	res := httptest.NewRecorder()
	server.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
}

func TestAdminUpdateUserEndpoint(t *testing.T) {
	server, _ := newTestServer(t)
	adminToken := issueAdminToken(t, server)
	body := []byte(`{"role":"ops","display_name":"Alice Ops","status":"suspended"}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/users/2", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	res := httptest.NewRecorder()
	server.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
}

func TestAdminAuditLogsEndpoint(t *testing.T) {
	server, _ := newTestServer(t)
	adminToken := issueAdminToken(t, server)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/audit-logs", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	res := httptest.NewRecorder()
	server.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
}

func TestAdminDeleteAnnouncementEndpoint(t *testing.T) {
	server, _ := newTestServer(t)
	adminToken := issueAdminToken(t, server)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/announcements/1", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	res := httptest.NewRecorder()
	server.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
}

func TestAdminTerminateInstanceEndpoint(t *testing.T) {
	server, _ := newTestServer(t)
	adminToken := issueAdminToken(t, server)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/instances/1/terminate", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	res := httptest.NewRecorder()
	server.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
}

func registerAnotherTestUser(t *testing.T, server *Server) string {
	t.Helper()
	registerBody := []byte(`{"username":"bob","email":"bob@example.com","password":"Password123!","display_name":"Bob"}`)
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

func issueRoleToken(t *testing.T, server *Server, role string) string {
	t.Helper()
	result, err := server.auth.Register(context.Background(), auth.RegisterInput{Username: role, Email: role + "@example.com", Password: "AdminPass123!", DisplayName: role})
	if err != nil {
		t.Fatalf("register role user: %v", err)
	}
	token, _, err := auth.NewTokenManager(server.cfg.JWTSecret, server.cfg.JWTTTL).Sign(auth.TokenClaims{UserID: result.User.ID, Role: role})
	if err != nil {
		t.Fatalf("issue role token: %v", err)
	}
	return token
}

func issueAdminToken(t *testing.T, server *Server) string {
	return issueRoleToken(t, server, "admin")
}

func TestContestEndpoint(t *testing.T) {
	server, _ := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/contest", nil)
	res := httptest.NewRecorder()
	server.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
	if !strings.Contains(res.Body.String(), `"status":"running"`) {
		t.Fatalf("expected running contest in response, got %s", res.Body.String())
	}
}

func TestChallengesBlockedWhenContestDraft(t *testing.T) {
	server, _ := newTestServer(t)
	server.contest = contest.NewService(&testContestRepo{current: contest.Contest{ID: 1, Slug: "recruit-2025", Status: contest.StatusDraft}})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/challenges", nil)
	res := httptest.NewRecorder()
	server.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.Code)
	}
	if !strings.Contains(res.Body.String(), `"error":"contest_not_public"`) {
		t.Fatalf("expected contest_not_public, got %s", res.Body.String())
	}
}

func TestSubmissionBlockedWhenContestFrozen(t *testing.T) {
	server, _ := newTestServer(t)
	server.contest = contest.NewService(&testContestRepo{current: contest.Contest{ID: 1, Slug: "recruit-2025", Status: contest.StatusFrozen}})
	token := loginAsPlayer(t, server)
	body := bytes.NewBufferString(`{"flag":"flag{welcome}"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/challenges/1/submissions", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	server.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.Code)
	}
	if !strings.Contains(res.Body.String(), `"error":"submission_closed"`) {
		t.Fatalf("expected submission_closed, got %s", res.Body.String())
	}
}

func TestRuntimeBlockedWhenContestFrozen(t *testing.T) {
	server, _ := newTestServer(t)
	server.contest = contest.NewService(&testContestRepo{current: contest.Contest{ID: 1, Slug: "recruit-2025", Status: contest.StatusFrozen}})
	token := loginAsPlayer(t, server)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/challenges/1/instances/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()
	server.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.Code)
	}
	if !strings.Contains(res.Body.String(), `"error":"runtime_closed"`) {
		t.Fatalf("expected runtime_closed, got %s", res.Body.String())
	}
}
