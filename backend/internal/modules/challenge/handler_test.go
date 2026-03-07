package challenge

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ctf-recruit/backend/internal/middleware"
	"ctf-recruit/backend/internal/modules/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func TestChallengeListPlayerSeesPublishedOnly(t *testing.T) {
	app, authService, _, _, _ := setupChallengeHandlerTestApp(t)

	playerToken := mustIssueAccessToken(t, authService, auth.RolePlayer)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/challenges", nil)
	req.Header.Set("Authorization", "Bearer "+playerToken)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var payload ChallengeListResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(payload.Items) != 1 {
		t.Fatalf("expected 1 published challenge for player, got %d", len(payload.Items))
	}
	if !payload.Items[0].IsPublished {
		t.Fatal("expected returned challenge to be published for player")
	}
}

func TestChallengeListAdminSeesPublishedAndUnpublished(t *testing.T) {
	app, authService, _, _, _ := setupChallengeHandlerTestApp(t)

	adminToken := mustIssueAccessToken(t, authService, auth.RoleAdmin)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/challenges", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var payload ChallengeListResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(payload.Items) != 2 {
		t.Fatalf("expected 2 challenges for admin, got %d", len(payload.Items))
	}

	hasDraft := false
	for _, item := range payload.Items {
		if !item.IsPublished {
			hasDraft = true
		}
	}
	if !hasDraft {
		t.Fatal("expected admin list to include unpublished challenge")
	}
}

func TestChallengeDetailPlayerCannotAccessUnpublished(t *testing.T) {
	app, authService, _, unpublished, _ := setupChallengeHandlerTestApp(t)

	playerToken := mustIssueAccessToken(t, authService, auth.RolePlayer)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/challenges/"+unpublished.ID, nil)
	req.Header.Set("Authorization", "Bearer "+playerToken)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}

	var payload struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Error.Code != "CHALLENGE_NOT_FOUND" {
		t.Fatalf("expected CHALLENGE_NOT_FOUND, got %s", payload.Error.Code)
	}
}

func TestChallengeDetailAdminCanAccessUnpublished(t *testing.T) {
	app, authService, _, unpublished, _ := setupChallengeHandlerTestApp(t)

	adminToken := mustIssueAccessToken(t, authService, auth.RoleAdmin)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/challenges/"+unpublished.ID, nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var payload ChallengeResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.ID != unpublished.ID {
		t.Fatalf("expected challenge id %s, got %s", unpublished.ID, payload.ID)
	}
	if payload.IsPublished {
		t.Fatal("expected admin detail response to include unpublished challenge")
	}
}

func TestChallengeAdminCRUDAndPublishToggle(t *testing.T) {
	app, authService, _, _, _ := setupChallengeHandlerTestApp(t)
	adminToken := mustIssueAccessToken(t, authService, auth.RoleAdmin)

	createBody := map[string]any{
		"title":       "Admin CRUD Challenge",
		"description": "created by admin",
		"category":    "web",
		"difficulty":  "easy",
		"mode":        "static",
		"points":      111,
		"flag":        "flag{admin-crud}",
		"isPublished": false,
	}
	created := doJSONRequest[ChallengeResponse](t, app, http.MethodPost, "/api/v1/challenges", adminToken, createBody, http.StatusCreated)

	if created.IsPublished {
		t.Fatal("expected created challenge to be unpublished")
	}

	updatedTitle := "Admin CRUD Challenge Updated"
	togglePublish := true
	updateBody := map[string]any{
		"title":       updatedTitle,
		"isPublished": togglePublish,
	}
	updated := doJSONRequest[ChallengeResponse](t, app, http.MethodPut, "/api/v1/challenges/"+created.ID, adminToken, updateBody, http.StatusOK)

	if updated.Title != updatedTitle {
		t.Fatalf("expected updated title %q, got %q", updatedTitle, updated.Title)
	}
	if !updated.IsPublished {
		t.Fatal("expected challenge to be published after toggle")
	}

	listAfterPublish := doJSONRequest[ChallengeListResponse](t, app, http.MethodGet, "/api/v1/challenges", adminToken, nil, http.StatusOK)
	var foundPublished bool
	for _, item := range listAfterPublish.Items {
		if item.ID == created.ID {
			foundPublished = true
			if !item.IsPublished {
				t.Fatal("expected admin list to reflect published state")
			}
		}
	}
	if !foundPublished {
		t.Fatal("expected created challenge in admin list")
	}

	deleteNoContentRequest(t, app, http.MethodDelete, "/api/v1/challenges/"+created.ID, adminToken, http.StatusNoContent)

	listAfterDelete := doJSONRequest[ChallengeListResponse](t, app, http.MethodGet, "/api/v1/challenges", adminToken, nil, http.StatusOK)
	for _, item := range listAfterDelete.Items {
		if item.ID == created.ID {
			t.Fatal("expected deleted challenge to be absent from admin list")
		}
	}
}

func TestChallengeAdminMutationsPlayerForbidden(t *testing.T) {
	app, authService, _, unpublished, _ := setupChallengeHandlerTestApp(t)
	playerToken := mustIssueAccessToken(t, authService, auth.RolePlayer)

	createBody := map[string]any{
		"title":       "Forbidden Create",
		"description": "player should not create",
		"category":    "web",
		"difficulty":  "easy",
		"mode":        "static",
		"points":      100,
		"flag":        "flag{forbidden}",
		"isPublished": false,
	}

	cases := []struct {
		name   string
		method string
		path   string
		body   map[string]any
	}{
		{name: "post create", method: http.MethodPost, path: "/api/v1/challenges", body: createBody},
		{name: "put update", method: http.MethodPut, path: "/api/v1/challenges/" + unpublished.ID, body: map[string]any{"title": "not allowed"}},
		{name: "delete", method: http.MethodDelete, path: "/api/v1/challenges/" + unpublished.ID},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := doRequestRaw(t, app, tc.method, tc.path, playerToken, tc.body)
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusForbidden {
				t.Fatalf("expected 403, got %d", resp.StatusCode)
			}

			code := decodeErrorCode(t, resp)
			if code != "AUTH_FORBIDDEN" {
				t.Fatalf("expected AUTH_FORBIDDEN, got %s", code)
			}
		})
	}
}

func setupChallengeHandlerTestApp(t *testing.T) (*fiber.App, *auth.Service, *Service, *ChallengeResponse, *ChallengeResponse) {
	t.Helper()

	repo := newMockRepository()
	challengeService := NewService(repo)

	published, err := challengeService.Create(context.Background(), CreateChallengeRequest{
		Title:       "Published Challenge",
		Description: "visible to players",
		Category:    "web",
		Difficulty:  DifficultyEasy,
		Mode:        ModeStatic,
		Points:      100,
		Flag:        "flag{pub}",
		IsPublished: true,
	})
	if err != nil {
		t.Fatalf("seed published challenge failed: %v", err)
	}

	unpublished, err := challengeService.Create(context.Background(), CreateChallengeRequest{
		Title:       "Draft Challenge",
		Description: "admin only",
		Category:    "pwn",
		Difficulty:  DifficultyMedium,
		Mode:        ModeStatic,
		Points:      200,
		Flag:        "flag{draft}",
		IsPublished: false,
	})
	if err != nil {
		t.Fatalf("seed unpublished challenge failed: %v", err)
	}

	authService := auth.NewService(nil, "test-secret", time.Hour)

	app := fiber.New(fiber.Config{ErrorHandler: middleware.ErrorHandler})
	v1 := app.Group("/api/v1")
	NewHandler(challengeService).RegisterRoutes(v1, authService)

	return app, authService, challengeService, unpublished, published
}

func mustIssueAccessToken(t *testing.T, authService *auth.Service, role auth.Role) string {
	t.Helper()

	token, err := authService.GenerateAccessToken(&auth.User{ID: uuid.New(), Role: role})
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

func deleteNoContentRequest(
	t *testing.T,
	app *fiber.App,
	method, path, token string,
	expectedStatus int,
) {
	t.Helper()

	resp := doRequestRaw(t, app, method, path, token, nil)
	defer resp.Body.Close()

	if resp.StatusCode != expectedStatus {
		t.Fatalf("expected %d, got %d", expectedStatus, resp.StatusCode)
	}
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
