package submission

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
	"ctf-recruit/backend/internal/modules/challenge"
	"ctf-recruit/backend/internal/modules/scoreboard"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func TestSubmissionListMineReturnsOwnHistory(t *testing.T) {
	app, authService, svc, ch := setupSubmissionHandlerTestApp(t)
	playerA := uuid.New()
	playerB := uuid.New()

	if _, err := svc.Submit(context.Background(), playerA.String(), auth.RolePlayer, CreateSubmissionRequest{ChallengeID: ch.ID.String(), Flag: "flag{ok}"}); err != nil {
		t.Fatalf("seed submit player A failed: %v", err)
	}
	if _, err := svc.Submit(context.Background(), playerB.String(), auth.RolePlayer, CreateSubmissionRequest{ChallengeID: ch.ID.String(), Flag: "flag{ok}"}); err != nil {
		t.Fatalf("seed submit player B failed: %v", err)
	}

	tokenA := mustIssueAccessTokenWithUserID(t, authService, playerA, auth.RolePlayer)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/submissions/me?limit=20&offset=0", nil)
	req.Header.Set("Authorization", "Bearer "+tokenA)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var payload SubmissionListResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(payload.Items) != 1 {
		t.Fatalf("expected 1 item for player A, got %d", len(payload.Items))
	}
	if payload.Items[0].ChallengeID != ch.ID.String() {
		t.Fatalf("expected challenge %s, got %s", ch.ID.String(), payload.Items[0].ChallengeID)
	}
}

func TestSubmissionListMineByChallengeFiltersHistory(t *testing.T) {
	app, authService, svc, ch := setupSubmissionHandlerTestApp(t)
	player := uuid.New()
	otherChallengeID := uuid.New()

	if _, err := svc.Submit(context.Background(), player.String(), auth.RolePlayer, CreateSubmissionRequest{ChallengeID: ch.ID.String(), Flag: "flag{wrong}"}); err != nil {
		t.Fatalf("seed submit challenge A failed: %v", err)
	}

	repo := svc.repo.(*mockRepository)
	if err := repo.Create(context.Background(), &Submission{
		ID:            uuid.New(),
		UserID:        player,
		ChallengeID:   otherChallengeID,
		Status:        StatusCorrect,
		AwardedPoints: 10,
		CreatedAt:     time.Now().UTC(),
	}); err != nil {
		t.Fatalf("seed submit challenge B failed: %v", err)
	}

	token := mustIssueAccessTokenWithUserID(t, authService, player, auth.RolePlayer)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/submissions/challenge/"+ch.ID.String()+"?limit=20&offset=0", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var payload SubmissionListResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(payload.Items) != 1 {
		t.Fatalf("expected 1 filtered item, got %d", len(payload.Items))
	}
	if payload.Items[0].ChallengeID != ch.ID.String() {
		t.Fatalf("expected challenge %s, got %s", ch.ID.String(), payload.Items[0].ChallengeID)
	}
}

func TestSubmissionAPIAndScoreboardDuplicateCorrectIsStable(t *testing.T) {
	repo := newMockRepository()
	ch := &challenge.Challenge{
		ID:          uuid.New(),
		Points:      120,
		Mode:        challenge.ModeStatic,
		FlagHash:    hashFlag("flag{idempotent}"),
		IsPublished: true,
	}
	svc := NewService(repo, mockChallengeReader{ch: ch}, nil)
	authService := auth.NewService(nil, "test-secret", time.Hour)

	app := fiber.New(fiber.Config{ErrorHandler: middleware.ErrorHandler})
	v1 := app.Group("/api/v1")
	NewHandler(svc).RegisterRoutes(v1, authService)
	scoreSvc := scoreboard.NewService(scoreboardRepoFromSubmissionRepo{repo: repo})
	scoreboard.NewHandler(scoreSvc).RegisterRoutes(v1, authService)

	userID := uuid.New()
	token := mustIssueAccessTokenWithUserID(t, authService, userID, auth.RolePlayer)
	body, _ := json.Marshal(map[string]any{"challengeId": ch.ID.String(), "flag": "flag{idempotent}"})

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/submissions", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("submit request failed: %v", err)
		}
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("expected 201, got %d", resp.StatusCode)
		}
		resp.Body.Close()
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/scoreboard?limit=20&offset=0", nil)
	getReq.Header.Set("Authorization", "Bearer "+token)
	getResp, err := app.Test(getReq)
	if err != nil {
		t.Fatalf("scoreboard request failed: %v", err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", getResp.StatusCode)
	}

	var payload scoreboard.ScoreboardResponse
	if err := json.NewDecoder(getResp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Items) != 1 {
		t.Fatalf("expected 1 scoreboard item, got %d", len(payload.Items))
	}
	if payload.Items[0].TotalPoints != 120 {
		t.Fatalf("expected stable total points 120, got %d", payload.Items[0].TotalPoints)
	}
}

type scoreboardRepoFromSubmissionRepo struct {
	repo *mockRepository
}

func (s scoreboardRepoFromSubmissionRepo) ListAggregates(_ context.Context) ([]scoreboard.ScoreboardAggregate, error) {
	type key struct {
		userID string
	}
	acc := map[key]scoreboard.ScoreboardAggregate{}
	seenSolve := map[key]map[string]struct{}{}

	for _, sub := range s.repo.submissions {
		if sub.AwardedPoints <= 0 {
			continue
		}
		k := key{userID: sub.UserID.String()}
		row := acc[k]
		row.UserID = sub.UserID.String()
		row.DisplayName = sub.UserID.String()
		row.TotalPoints += sub.AwardedPoints
		if row.LastAcceptedAt.IsZero() || row.LastAcceptedAt.Before(sub.CreatedAt) {
			row.LastAcceptedAt = sub.CreatedAt
		}
		if seenSolve[k] == nil {
			seenSolve[k] = map[string]struct{}{}
		}
		challengeKey := sub.ChallengeID.String()
		if _, ok := seenSolve[k][challengeKey]; !ok {
			seenSolve[k][challengeKey] = struct{}{}
			row.SolvedCount++
		}
		acc[k] = row
	}

	out := make([]scoreboard.ScoreboardAggregate, 0, len(acc))
	for _, row := range acc {
		out = append(out, row)
	}
	return out, nil
}

func setupSubmissionHandlerTestApp(t *testing.T) (*fiber.App, *auth.Service, *Service, *challenge.Challenge) {
	t.Helper()

	repo := newMockRepository()
	ch := &challenge.Challenge{
		ID:          uuid.New(),
		Points:      100,
		Mode:        challenge.ModeStatic,
		FlagHash:    hashFlag("flag{ok}"),
		IsPublished: true,
	}
	svc := NewService(repo, mockChallengeReader{ch: ch}, nil)
	authService := auth.NewService(nil, "test-secret", time.Hour)

	app := fiber.New(fiber.Config{ErrorHandler: middleware.ErrorHandler})
	v1 := app.Group("/api/v1")
	NewHandler(svc).RegisterRoutes(v1, authService)

	return app, authService, svc, ch
}

func mustIssueAccessTokenWithUserID(t *testing.T, authService *auth.Service, userID uuid.UUID, role auth.Role) string {
	t.Helper()

	token, err := authService.GenerateAccessToken(&auth.User{ID: userID, Role: role})
	if err != nil {
		t.Fatalf("issue access token failed: %v", err)
	}

	return token
}
