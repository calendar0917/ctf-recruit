package middleware

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	apperrors "ctf-recruit/backend/internal/errors"

	"github.com/gofiber/fiber/v2"
)

func TestErrorHandlerInternalAppErrorIsStructured(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: ErrorHandler})
	app.Use(RequestID())
	app.Get("/boom", func(c *fiber.Ctx) error {
		return apperrors.Internal("JUDGE_QUEUE_UNAVAILABLE", "Judge queue is unavailable", errors.New("queue down"))
	})

	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.StatusCode)
	}

	if resp.Header.Get("X-Request-Id") == "" {
		t.Fatal("expected X-Request-Id header")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	var payload struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
		RequestID string `json:"requestId"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload.Error.Code != "JUDGE_QUEUE_UNAVAILABLE" {
		t.Fatalf("expected code JUDGE_QUEUE_UNAVAILABLE, got %s", payload.Error.Code)
	}
	if payload.Error.Message == "" {
		t.Fatal("expected non-empty message")
	}
	if payload.RequestID == "" {
		t.Fatal("expected requestId in response")
	}
}
