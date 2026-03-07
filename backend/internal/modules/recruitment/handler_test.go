package recruitment

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

type mockRepository struct {
	items []*Submission
}

func newMockRepository() *mockRepository {
	return &mockRepository{items: make([]*Submission, 0)}
}

func (m *mockRepository) Create(_ context.Context, submission *Submission) error {
	if submission.ID == uuid.Nil {
		submission.ID = uuid.New()
	}
	copied := *submission
	m.items = append(m.items, &copied)
	return nil
}

func (m *mockRepository) GetByID(_ context.Context, id string) (*Submission, error) {
	for _, submission := range m.items {
		if submission.ID.String() == id {
			copied := *submission
			return &copied, nil
		}
	}
	return nil, nil
}

func (m *mockRepository) List(_ context.Context, filter ListFilter) ([]Submission, error) {
	items := make([]Submission, 0, len(m.items))
	for i := len(m.items) - 1; i >= 0; i-- {
		items = append(items, *m.items[i])
	}

	if filter.Offset >= len(items) {
		return []Submission{}, nil
	}

	end := filter.Offset + filter.Limit
	if end > len(items) {
		end = len(items)
	}

	return items[filter.Offset:end], nil
}

func TestRecruitmentPlayerCanSubmitAndAdminCanListAndGetDetail(t *testing.T) {
	app, authService := setupRecruitmentHandlerTestApp()

	playerToken := mustIssueAccessToken(t, authService, auth.RolePlayer)
	adminToken := mustIssueAccessToken(t, authService, auth.RoleAdmin)

	created := doJSONRequest[SubmissionResponse](
		t,
		app,
		http.MethodPost,
		"/api/v1/recruitments",
		playerToken,
		map[string]any{
			"name":      "Alice",
			"school":    "Test University",
			"grade":     "大二",
			"direction": "web",
			"contact":   "alice@example.com",
			"bio":       "Enjoys CTF and security learning.",
		},
		http.StatusCreated,
	)

	if created.ID == "" {
		t.Fatal("expected created submission ID")
	}
	if created.UserID == "" {
		t.Fatal("expected created submission user ID")
	}
	if created.Name != "Alice" {
		t.Fatalf("expected name Alice, got %s", created.Name)
	}

	list := doJSONRequest[SubmissionListResponse](
		t,
		app,
		http.MethodGet,
		"/api/v1/recruitments?limit=20&offset=0",
		adminToken,
		nil,
		http.StatusOK,
	)

	if len(list.Items) != 1 {
		t.Fatalf("expected 1 recruitment item, got %d", len(list.Items))
	}
	if list.Items[0].ID != created.ID {
		t.Fatalf("expected listed item id %s, got %s", created.ID, list.Items[0].ID)
	}

	detail := doJSONRequest[SubmissionResponse](
		t,
		app,
		http.MethodGet,
		"/api/v1/recruitments/"+created.ID,
		adminToken,
		nil,
		http.StatusOK,
	)

	if detail.ID != created.ID {
		t.Fatalf("expected detail ID %s, got %s", created.ID, detail.ID)
	}
	if detail.Contact != "alice@example.com" {
		t.Fatalf("expected detail contact alice@example.com, got %s", detail.Contact)
	}
}

func TestRecruitmentAdminOnlyListAndDetailForbiddenForPlayer(t *testing.T) {
	app, authService := setupRecruitmentHandlerTestApp()
	playerToken := mustIssueAccessToken(t, authService, auth.RolePlayer)
	adminToken := mustIssueAccessToken(t, authService, auth.RoleAdmin)

	created := doJSONRequest[SubmissionResponse](
		t,
		app,
		http.MethodPost,
		"/api/v1/recruitments",
		playerToken,
		map[string]any{
			"name":      "Bob",
			"school":    "Sample School",
			"grade":     "研一",
			"direction": "pwn",
			"contact":   "bob@example.com",
			"bio":       "CTF beginner",
		},
		http.StatusCreated,
	)

	_ = doJSONRequest[SubmissionListResponse](t, app, http.MethodGet, "/api/v1/recruitments", adminToken, nil, http.StatusOK)

	for _, tc := range []struct {
		name string
		path string
	}{
		{name: "list", path: "/api/v1/recruitments"},
		{name: "detail", path: "/api/v1/recruitments/" + created.ID},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp := doRequestRaw(t, app, http.MethodGet, tc.path, playerToken, nil)
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

func setupRecruitmentHandlerTestApp() (*fiber.App, *auth.Service) {
	repo := newMockRepository()
	recruitmentService := NewService(repo)
	authService := auth.NewService(nil, "test-secret", time.Hour)

	app := fiber.New(fiber.Config{ErrorHandler: middleware.ErrorHandler})
	v1 := app.Group("/api/v1")
	NewHandler(recruitmentService).RegisterRoutes(v1, authService)

	return app, authService
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
