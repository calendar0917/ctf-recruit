package app

import (
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"ctf/backend/internal/admin"
	"ctf/backend/internal/httpx"
	"ctf/backend/internal/store"
)

type adminBuildChallengeImageRequest struct {
	Template string `json:"template"`
	Tag      string `json:"tag"`
}

// handleAdminBuildChallengeImage builds a docker image for a whitelisted template.
// This is intended for development and controlled deployments.
func (s *Server) handleAdminBuildChallengeImage(w http.ResponseWriter, r *http.Request) {
	actorUserID, ok := userIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "missing authenticated user")
		return
	}
	allowed, err := enforceRateLimit(r.Context(), s.limiters.AdminWrite, adminRateLimitKey("image_build", r, actorUserID))
	if err != nil {
		s.metrics.Inc("ctf_rate_limit_errors_total", map[string]string{"scope": "admin_write"})
		logError("rate_limit.admin_write.error", map[string]any{"action": "image_build", "error": err.Error()})
		httpx.WriteError(w, http.StatusBadGateway, "rate_limit_error", "failed to enforce rate limit")
		return
	}
	if !allowed {
		s.metrics.Inc("ctf_rate_limit_hits_total", map[string]string{"scope": "admin_write"})
		httpx.WriteError(w, http.StatusTooManyRequests, "admin_rate_limited", "too many admin write requests, please try again later")
		return
	}

	var input adminBuildChallengeImageRequest
	if err := decodeJSON(r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	name := strings.TrimSpace(input.Template)
	if name == "" {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_template", "template is required")
		return
	}
	// Whitelist: templates are under ../challenges/templates/<name>
	// The API runs from backend/, so use ../challenges.
	base := filepath.Clean(filepath.Join("..", "challenges", "templates"))
	templateDir := filepath.Clean(filepath.Join(base, name))
	if !strings.HasPrefix(templateDir, base+string(filepath.Separator)) && templateDir != base {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_template", "template path is not allowed")
		return
	}

	tag := strings.TrimSpace(input.Tag)
	if tag == "" {
		tag = "ctf/" + name + ":dev"
	}

	// run: docker build -t <tag> <templateDir>
	result, runErr := admin.RunScript(r.Context(), []string{"docker", "build", "-t", tag, templateDir}, 10*time.Minute)
	_ = store.NewAdminRepository(s.db).CreateAuditLog(r.Context(), &actorUserID, "image.build", "challenge_template", name, map[string]any{
		"tag":        tag,
		"template":   name,
		"directory":  templateDir,
		"exit_code":  result.ExitCode,
		"duration_ms": result.DurationMS,
	})
	if runErr != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"result": result, "error": runErr.Error()})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"result": result})
}
