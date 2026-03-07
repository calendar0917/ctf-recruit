package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
)

func TestRateLimiterBlocksRapidCallsPerPathAndIP(t *testing.T) {
	limiter := NewRateLimiter(2, time.Minute)

	app := fiber.New(fiber.Config{ErrorHandler: ErrorHandler})
	app.Post("/api/v1/auth/login", limiter.Middleware(), func(c *fiber.Ctx) error {
		return c.Status(http.StatusOK).JSON(fiber.Map{"ok": true})
	})

	for i := 0; i < 2; i++ {
		resp := doRateLimitRequest(t, app, http.MethodPost, "/api/v1/auth/login")
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 on call %d, got %d", i+1, resp.StatusCode)
		}
		resp.Body.Close()
	}

	third := doRateLimitRequest(t, app, http.MethodPost, "/api/v1/auth/login")
	defer third.Body.Close()
	if third.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("expected 429 on third call, got %d", third.StatusCode)
	}
	t.Logf("rapid POST /api/v1/auth/login status=%d", third.StatusCode)

	var payload struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(third.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Error.Code != "RATE_LIMITED" {
		t.Fatalf("expected RATE_LIMITED, got %s", payload.Error.Code)
	}
	t.Logf("rapid POST /api/v1/auth/login error.code=%s", payload.Error.Code)
}

func TestRateLimiterUsesPathIsolation(t *testing.T) {
	limiter := NewRateLimiter(1, time.Minute)

	app := fiber.New(fiber.Config{ErrorHandler: ErrorHandler})
	app.Post("/api/v1/auth/login", limiter.Middleware(), func(c *fiber.Ctx) error {
		return c.SendStatus(http.StatusOK)
	})
	app.Post("/api/v1/submissions", limiter.Middleware(), func(c *fiber.Ctx) error {
		return c.SendStatus(http.StatusCreated)
	})

	loginFirst := doRateLimitRequest(t, app, http.MethodPost, "/api/v1/auth/login")
	if loginFirst.StatusCode != http.StatusOK {
		t.Fatalf("expected login first call 200, got %d", loginFirst.StatusCode)
	}
	loginFirst.Body.Close()

	submissionFirst := doRateLimitRequest(t, app, http.MethodPost, "/api/v1/submissions")
	if submissionFirst.StatusCode != http.StatusCreated {
		t.Fatalf("expected submissions first call 201, got %d", submissionFirst.StatusCode)
	}
	submissionFirst.Body.Close()

	loginSecond := doRateLimitRequest(t, app, http.MethodPost, "/api/v1/auth/login")
	defer loginSecond.Body.Close()
	if loginSecond.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("expected second login call 429, got %d", loginSecond.StatusCode)
	}
}

func doRateLimitRequest(t *testing.T, app *fiber.App, method, path string) *http.Response {
	t.Helper()

	req := httptest.NewRequest(method, path, nil)
	req.RemoteAddr = "127.0.0.1:12345"

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	return resp
}
