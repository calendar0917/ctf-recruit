package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ctf/backend/internal/config"
)

func TestHealthEndpoint(t *testing.T) {
	server := NewServer(config.Load())
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

func TestCreateInstanceRequiresUserHeader(t *testing.T) {
	server := NewServer(config.Load())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/challenges/1/instances/me", nil)
	res := httptest.NewRecorder()

	server.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.Code)
	}
}
