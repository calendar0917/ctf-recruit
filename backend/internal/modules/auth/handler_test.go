package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
	"time"

	"ctf-recruit/backend/internal/middleware"
	"ctf-recruit/backend/internal/modules/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type testRepo struct {
	usersByEmail map[string]*auth.User
	usersByID    map[string]*auth.User
}

func newTestRepo() *testRepo {
	return &testRepo{
		usersByEmail: map[string]*auth.User{},
		usersByID:    map[string]*auth.User{},
	}
}

func (r *testRepo) Create(_ context.Context, user *auth.User) error {
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	copied := *user
	r.usersByEmail[user.Email] = &copied
	r.usersByID[user.ID.String()] = &copied
	return nil
}

func (r *testRepo) GetByEmail(_ context.Context, email string) (*auth.User, error) {
	if u, ok := r.usersByEmail[email]; ok {
		copied := *u
		return &copied, nil
	}
	return nil, nil
}

func (r *testRepo) GetByID(_ context.Context, id string) (*auth.User, error) {
	if u, ok := r.usersByID[id]; ok {
		copied := *u
		return &copied, nil
	}
	return nil, nil
}

func (r *testRepo) List(_ context.Context, limit, offset int) ([]auth.User, error) {
	all := make([]auth.User, 0, len(r.usersByID))
	for _, u := range r.usersByID {
		all = append(all, *u)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].Email < all[j].Email })

	if offset >= len(all) {
		return []auth.User{}, nil
	}
	if limit <= 0 {
		limit = len(all)
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}

	out := make([]auth.User, 0, end-offset)
	for _, item := range all[offset:end] {
		out = append(out, item)
	}
	return out, nil
}

func (r *testRepo) UpdateAdminFields(_ context.Context, id string, role *auth.Role, isDisabled *bool) (*auth.User, error) {
	u, ok := r.usersByID[id]
	if !ok {
		return nil, nil
	}
	updated := *u
	if role != nil {
		updated.Role = *role
	}
	if isDisabled != nil {
		updated.IsDisabled = *isDisabled
	}
	r.usersByID[id] = &updated
	r.usersByEmail[updated.Email] = &updated
	copyOut := updated
	return &copyOut, nil
}

func seedUser(t *testing.T, repo *testRepo, email, password string) {
	t.Helper()

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	err = repo.Create(context.Background(), &auth.User{
		Email:        email,
		PasswordHash: string(hash),
		DisplayName:  "Player One",
		Role:         auth.RolePlayer,
	})
	if err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}
}

func TestLoginHandlerReturnsAccessToken(t *testing.T) {
	repo := newTestRepo()
	seedUser(t, repo, "player@example.com", "password123")

	svc := auth.NewService(repo, "test-secret", time.Hour)
	handler := auth.NewHandler(svc)

	app := fiber.New(fiber.Config{ErrorHandler: middleware.ErrorHandler})
	app.Post("/api/v1/auth/login", handler.Login)

	body, _ := json.Marshal(map[string]string{
		"email":    "player@example.com",
		"password": "password123",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var payload struct {
		AccessToken string `json:"accessToken"`
		TokenType   string `json:"tokenType"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.AccessToken == "" {
		t.Fatal("expected accessToken in login response")
	}
	if payload.TokenType != "Bearer" {
		t.Fatalf("expected tokenType Bearer, got %s", payload.TokenType)
	}
}

func TestLoginHandlerInvalidCredentialsUnauthorized(t *testing.T) {
	repo := newTestRepo()
	seedUser(t, repo, "player@example.com", "password123")

	svc := auth.NewService(repo, "test-secret", time.Hour)
	handler := auth.NewHandler(svc)

	app := fiber.New(fiber.Config{ErrorHandler: middleware.ErrorHandler})
	app.Post("/api/v1/auth/login", handler.Login)

	body, _ := json.Marshal(map[string]string{
		"email":    "player@example.com",
		"password": "wrong-password",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}

	var payload struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Error.Code != "AUTH_INVALID_CREDENTIALS" {
		t.Fatalf("expected AUTH_INVALID_CREDENTIALS, got %s", payload.Error.Code)
	}
}

func TestMeRouteWithoutTokenUnauthorized(t *testing.T) {
	repo := newTestRepo()
	svc := auth.NewService(repo, "test-secret", time.Hour)
	handler := auth.NewHandler(svc)

	app := fiber.New(fiber.Config{ErrorHandler: middleware.ErrorHandler})
	app.Get("/api/v1/auth/me", middleware.Auth(svc), handler.Me)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}

	var payload struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Error.Code != "AUTH_MISSING_TOKEN" {
		t.Fatalf("expected AUTH_MISSING_TOKEN, got %s", payload.Error.Code)
	}
}

func TestMeRouteWithMalformedTokenUnauthorized(t *testing.T) {
	repo := newTestRepo()
	svc := auth.NewService(repo, "test-secret", time.Hour)
	handler := auth.NewHandler(svc)

	app := fiber.New(fiber.Config{ErrorHandler: middleware.ErrorHandler})
	app.Get("/api/v1/auth/me", middleware.Auth(svc), handler.Me)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer not-a-jwt")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}

	var payload struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Error.Code != "AUTH_INVALID_TOKEN" {
		t.Fatalf("expected AUTH_INVALID_TOKEN, got %s", payload.Error.Code)
	}
}

func TestMeRouteWithExpiredTokenUnauthorized(t *testing.T) {
	repo := newTestRepo()
	svc := auth.NewService(repo, "test-secret", -time.Minute)
	handler := auth.NewHandler(svc)

	token, err := svc.GenerateAccessToken(&auth.User{ID: uuid.New(), Role: auth.RolePlayer})
	if err != nil {
		t.Fatalf("generate token failed: %v", err)
	}

	app := fiber.New(fiber.Config{ErrorHandler: middleware.ErrorHandler})
	app.Get("/api/v1/auth/me", middleware.Auth(svc), handler.Me)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}

	var payload struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Error.Code != "AUTH_INVALID_TOKEN" {
		t.Fatalf("expected AUTH_INVALID_TOKEN, got %s", payload.Error.Code)
	}
}

func TestAdminUsersEndpointsPlayerForbidden(t *testing.T) {
	repo := newTestRepo()
	seedUser(t, repo, "admin@example.com", "password123")
	seedUser(t, repo, "player@example.com", "password123")

	adminUser, _ := repo.GetByEmail(context.Background(), "admin@example.com")
	adminRole := auth.RoleAdmin
	if _, err := repo.UpdateAdminFields(context.Background(), adminUser.ID.String(), &adminRole, nil); err != nil {
		t.Fatalf("failed to set admin role: %v", err)
	}

	svc := auth.NewService(repo, "test-secret", time.Hour)
	handler := auth.NewHandler(svc)

	app := fiber.New(fiber.Config{ErrorHandler: middleware.ErrorHandler})
	v1 := app.Group("/api/v1")
	adminUsers := v1.Group("/admin/users", middleware.Auth(svc), middleware.RequireRoles(auth.RoleAdmin))
	adminUsers.Get("", handler.AdminListUsers)
	adminUsers.Patch("/:id", handler.AdminUpdateUser)

	playerToken, err := svc.GenerateAccessToken(&auth.User{ID: uuid.New(), Role: auth.RolePlayer})
	if err != nil {
		t.Fatalf("generate player token failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
	req.Header.Set("Authorization", "Bearer "+playerToken)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", resp.StatusCode)
	}
	t.Logf("player GET /api/v1/admin/users status=%d", resp.StatusCode)

	var payload struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Error.Code != "AUTH_FORBIDDEN" {
		t.Fatalf("expected AUTH_FORBIDDEN, got %s", payload.Error.Code)
	}
	t.Logf("player GET /api/v1/admin/users error.code=%s", payload.Error.Code)
}

func TestAdminRoleChangeReflectedInMe(t *testing.T) {
	repo := newTestRepo()
	seedUser(t, repo, "admin@example.com", "password123")
	seedUser(t, repo, "player@example.com", "password123")

	adminUser, _ := repo.GetByEmail(context.Background(), "admin@example.com")
	adminRole := auth.RoleAdmin
	if _, err := repo.UpdateAdminFields(context.Background(), adminUser.ID.String(), &adminRole, nil); err != nil {
		t.Fatalf("failed to set admin role: %v", err)
	}

	playerUser, _ := repo.GetByEmail(context.Background(), "player@example.com")

	svc := auth.NewService(repo, "test-secret", time.Hour)
	handler := auth.NewHandler(svc)

	app := fiber.New(fiber.Config{ErrorHandler: middleware.ErrorHandler})
	v1 := app.Group("/api/v1")
	v1.Get("/auth/me", middleware.Auth(svc), handler.Me)
	adminUsers := v1.Group("/admin/users", middleware.Auth(svc), middleware.RequireRoles(auth.RoleAdmin))
	adminUsers.Patch("/:id", handler.AdminUpdateUser)

	adminToken, err := svc.GenerateAccessToken(&auth.User{ID: adminUser.ID, Role: auth.RoleAdmin})
	if err != nil {
		t.Fatalf("generate admin token failed: %v", err)
	}
	playerToken, err := svc.GenerateAccessToken(&auth.User{ID: playerUser.ID, Role: auth.RolePlayer})
	if err != nil {
		t.Fatalf("generate player token failed: %v", err)
	}

	updateBody, _ := json.Marshal(map[string]any{"role": "admin"})
	patchReq := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/users/"+playerUser.ID.String(), bytes.NewReader(updateBody))
	patchReq.Header.Set("Authorization", "Bearer "+adminToken)
	patchReq.Header.Set("Content-Type", "application/json")

	patchResp, err := app.Test(patchReq)
	if err != nil {
		t.Fatalf("patch request failed: %v", err)
	}
	defer patchResp.Body.Close()
	if patchResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", patchResp.StatusCode)
	}
	t.Logf("admin PATCH /api/v1/admin/users/:id status=%d", patchResp.StatusCode)

	meReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+playerToken)
	meResp, err := app.Test(meReq)
	if err != nil {
		t.Fatalf("me request failed: %v", err)
	}
	defer meResp.Body.Close()
	if meResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", meResp.StatusCode)
	}

	var mePayload struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(meResp.Body).Decode(&mePayload); err != nil {
		t.Fatalf("decode me response: %v", err)
	}
	if mePayload.Role != "admin" {
		t.Fatalf("expected me role admin, got %s", mePayload.Role)
	}
	t.Logf("player GET /api/v1/auth/me role=%s", mePayload.Role)
}

var _ auth.Repository = (*testRepo)(nil)
