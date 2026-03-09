package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"ctf/backend/internal/admin"
	"ctf/backend/internal/auth"
	"ctf/backend/internal/config"
	"ctf/backend/internal/contest"
	"ctf/backend/internal/game"
	"ctf/backend/internal/httpx"
	"ctf/backend/internal/runtime"
	"ctf/backend/internal/store"
)

type Server struct {
	cfg      config.Config
	admin    *admin.Service
	auth     *auth.Service
	contest  *contest.Service
	game     *game.Service
	runtime  *runtime.Service
	limiters AppLimiters
	metrics  *metricsRegistry
	db       *sql.DB
}

func NewServer(cfg config.Config) (*Server, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	db, err := store.Open(cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}

	userRepo := store.NewUserRepository(db)
	adminRepo := store.NewAdminRepository(db)
	contestRepo := store.NewContestRepository(db)
	gameRepo := store.NewGameRepository(db)
	runtimeRepo := store.NewRuntimeRepository(db)
	tokens := auth.NewTokenManager(cfg.JWTSecret, cfg.JWTTTL)
	manager := runtime.NewDockerManager(cfg.DockerSocketPath)
	limiters := newAppLimiters(cfg)
	metrics := newMetricsRegistry()

	return &Server{
		cfg:      cfg,
		admin:    admin.NewServiceWithManager(adminRepo, cfg.AttachmentStorageDir, manager),
		auth:     auth.NewService(userRepo, tokens),
		contest:  contest.NewService(contestRepo),
		game:     game.NewService(gameRepo),
		runtime:  runtime.NewService(cfg.PublicBaseURL, manager, runtimeRepo),
		limiters: limiters,
		metrics:  metrics,
		db:       db,
	}, nil
}

func NewServerForTests(cfg config.Config, adminService *admin.Service, authService *auth.Service, contestService *contest.Service, gameService *game.Service, runtimeService *runtime.Service) *Server {
	return &Server{
		cfg:      cfg,
		admin:    adminService,
		auth:     authService,
		contest:  contestService,
		game:     gameService,
		runtime:  runtimeService,
		limiters: newAppLimiters(cfg),
		metrics:  newMetricsRegistry(),
	}
}

func (s *Server) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", s.handleHealth)
	mux.HandleFunc("GET /api/v1/ready", s.handleReady)
	mux.Handle("GET /api/v1/metrics", s.metrics)
	mux.HandleFunc("GET /api/v1/contest", s.handleContest)
	mux.HandleFunc("POST /api/v1/auth/register", s.handleRegister)
	mux.HandleFunc("POST /api/v1/auth/login", s.handleLogin)
	mux.Handle("GET /api/v1/me", s.authenticated(http.HandlerFunc(s.handleMe)))
	mux.Handle("GET /api/v1/me/submissions", s.authenticated(http.HandlerFunc(s.handleMeSubmissions)))
	mux.Handle("GET /api/v1/me/solves", s.authenticated(http.HandlerFunc(s.handleMeSolves)))
	mux.HandleFunc("GET /api/v1/announcements", s.handleAnnouncements)
	mux.HandleFunc("GET /api/v1/challenges", s.handleChallenges)
	mux.HandleFunc("GET /api/v1/challenges/{challengeID}", s.handleChallengeDetail)
	mux.HandleFunc("GET /api/v1/challenges/{challengeID}/attachments/{attachmentID}", s.handleChallengeAttachmentDownload)
	mux.HandleFunc("GET /api/v1/scoreboard", s.handleScoreboard)
	mux.Handle("POST /api/v1/challenges/{challengeID}/instances/me", s.authenticated(http.HandlerFunc(s.handleCreateInstance)))
	mux.Handle("GET /api/v1/challenges/{challengeID}/instances/me", s.authenticated(http.HandlerFunc(s.handleGetInstance)))
	mux.Handle("DELETE /api/v1/challenges/{challengeID}/instances/me", s.authenticated(http.HandlerFunc(s.handleDeleteInstance)))
	mux.Handle("POST /api/v1/challenges/{challengeID}/instances/me/renew", s.authenticated(http.HandlerFunc(s.handleRenewInstance)))
	mux.Handle("POST /api/v1/challenges/{challengeID}/submissions", s.authenticated(http.HandlerFunc(s.handleSubmitFlag)))
	mux.Handle("GET /api/v1/admin/contest", s.requirePermission("contest:read", http.HandlerFunc(s.handleAdminContest)))
	mux.Handle("PATCH /api/v1/admin/contest", s.requirePermission("contest:write", http.HandlerFunc(s.handleAdminUpdateContest)))
	mux.Handle("GET /api/v1/admin/challenges", s.requirePermission("challenge:read", http.HandlerFunc(s.handleAdminChallenges)))
	mux.Handle("POST /api/v1/admin/challenges", s.requirePermission("challenge:write", http.HandlerFunc(s.handleAdminCreateChallenge)))
	mux.Handle("GET /api/v1/admin/challenges/{challengeID}", s.requirePermission("challenge:read", http.HandlerFunc(s.handleAdminChallengeDetail)))
	mux.Handle("PATCH /api/v1/admin/challenges/{challengeID}", s.requirePermission("challenge:write", http.HandlerFunc(s.handleAdminUpdateChallenge)))
	mux.Handle("POST /api/v1/admin/challenges/{challengeID}/attachments", s.requirePermission("attachment:write", http.HandlerFunc(s.handleAdminCreateAttachment)))
	mux.Handle("GET /api/v1/admin/announcements", s.requirePermission("announcement:read", http.HandlerFunc(s.handleAdminAnnouncements)))
	mux.Handle("POST /api/v1/admin/announcements", s.requirePermission("announcement:write", http.HandlerFunc(s.handleAdminCreateAnnouncement)))
	mux.Handle("DELETE /api/v1/admin/announcements/{announcementID}", s.requirePermission("announcement:write", http.HandlerFunc(s.handleAdminDeleteAnnouncement)))
	mux.Handle("GET /api/v1/admin/submissions", s.requirePermission("submission:read", http.HandlerFunc(s.handleAdminSubmissions)))
	mux.Handle("GET /api/v1/admin/instances", s.requirePermission("instance:read", http.HandlerFunc(s.handleAdminInstances)))
	mux.Handle("POST /api/v1/admin/instances/{instanceID}/terminate", s.requirePermission("instance:write", http.HandlerFunc(s.handleAdminTerminateInstance)))
	mux.Handle("GET /api/v1/admin/users", s.requirePermission("user:read", http.HandlerFunc(s.handleAdminUsers)))
	mux.Handle("PATCH /api/v1/admin/users/{userID}", s.requirePermission("user:write", http.HandlerFunc(s.handleAdminUpdateUser)))
	mux.Handle("GET /api/v1/admin/audit-logs", s.requirePermission("audit:read", http.HandlerFunc(s.handleAdminAuditLogs)))
	return loggingMiddleware(s.metrics, mux)
}

func (s *Server) StartBackground(ctx context.Context) {
	interval, err := time.ParseDuration(s.cfg.InstanceSweeperPollInterval)
	if err != nil {
		logWarn("instance_sweeper.invalid_interval", map[string]any{"error": err.Error()})
		interval = 30 * time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	logInfo("instance_sweeper.started", map[string]any{"interval": interval.String()})
	if report, err := s.runtime.Reconcile(ctx); err != nil {
		logError("instance_reconcile.error", map[string]any{"error": err.Error()})
	} else if report.TerminatedRecords > 0 || report.RemovedContainers > 0 {
		logInfo("instance_reconcile.corrected", map[string]any{"terminated_records": report.TerminatedRecords, "removed_containers": report.RemovedContainers})
	}

	for {
		select {
		case <-ctx.Done():
			logInfo("instance_sweeper.stopped", nil)
			return
		case <-ticker.C:
			terminated, err := s.runtime.SweepExpired(ctx)
			if err != nil {
				logError("instance_sweeper.error", map[string]any{"error": err.Error()})
				continue
			}
			if terminated > 0 {
				logInfo("instance_sweeper.terminated", map[string]any{"count": terminated})
			}

			report, err := s.runtime.Reconcile(ctx)
			if err != nil {
				logError("instance_reconcile.error", map[string]any{"error": err.Error()})
				continue
			}
			if report.TerminatedRecords > 0 || report.RemovedContainers > 0 {
				logInfo("instance_reconcile.corrected", map[string]any{"terminated_records": report.TerminatedRecords, "removed_containers": report.RemovedContainers})
			}
		}
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	s.metrics.Inc("ctf_http_health_requests_total", nil)
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"status": "ok", "service": "api"})
}

func (s *Server) handleContest(w http.ResponseWriter, r *http.Request) {
	current, err := s.contest.Current(r.Context())
	if err != nil {
		logError("contest.current.failed", map[string]any{"error": err.Error()})
		httpx.WriteError(w, http.StatusBadGateway, "repository_error", "failed to load contest")
		return
	}
	phase := contest.BuildPhase(current)
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"contest": current, "phase": phase})
}

func (s *Server) handleReady(w http.ResponseWriter, _ *http.Request) {
	s.metrics.Inc("ctf_http_ready_requests_total", nil)
	ready := s.db != nil
	if s.db != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		ready = s.db.PingContext(ctx) == nil
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"status":                                "ready",
		"database_url_configured":               s.cfg.DatabaseURL != "",
		"database_connected":                    ready,
		"docker_socket_path":                    s.cfg.DockerSocketPath,
		"dynamic_runtime_enabled":               true,
		"attachment_storage_dir":                s.cfg.AttachmentStorageDir,
		"redis_addr":                            s.cfg.RedisAddr,
		"redis_rate_limit_enabled":              s.limiters.RedisAvailable,
		"login_rate_limit_window_seconds":       s.cfg.LoginRateLimitWindowSeconds,
		"login_rate_limit_max":                  s.cfg.LoginRateLimitMax,
		"register_rate_limit_window_seconds":    s.cfg.RegisterRateLimitWindowSeconds,
		"register_rate_limit_max":               s.cfg.RegisterRateLimitMax,
		"submission_rate_limit_window_seconds":  s.cfg.SubmissionRateLimitWindowSeconds,
		"submission_rate_limit_max":             s.cfg.SubmissionRateLimitMax,
		"admin_write_rate_limit_window_seconds": s.cfg.AdminWriteRateLimitWindowSeconds,
		"admin_write_rate_limit_max":            s.cfg.AdminWriteRateLimitMax,
	})
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	phase, ok := s.requireContestPhase(w, r, contestRequirement{registrationAllowed: true})
	if !ok {
		return
	}
	_ = phase
	var input auth.RegisterInput
	if err := decodeJSON(r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	allowed, err := enforceRateLimit(r.Context(), s.limiters.Register, registerRateLimitKey(r))
	if err != nil {
		s.metrics.Inc("ctf_rate_limit_errors_total", map[string]string{"scope": "register"})
		logError("rate_limit.register.error", map[string]any{"error": err.Error()})
		httpx.WriteError(w, http.StatusBadGateway, "rate_limit_error", "failed to enforce rate limit")
		return
	}
	if !allowed {
		s.metrics.Inc("ctf_rate_limit_hits_total", map[string]string{"scope": "register"})
		httpx.WriteError(w, http.StatusTooManyRequests, "register_rate_limited", "too many registration attempts, please try again later")
		return
	}

	result, err := s.auth.Register(r.Context(), input)
	if err != nil {
		logWarn("auth.register.failed", map[string]any{"error": err.Error()})
		httpx.WriteError(w, http.StatusBadRequest, "register_failed", "failed to register user")
		return
	}
	writeAuthResponse(w, http.StatusCreated, result)
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var input auth.LoginInput
	if err := decodeJSON(r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	allowed, err := enforceRateLimit(r.Context(), s.limiters.Login, authRateLimitKey("login", input.Identifier, r))
	if err != nil {
		s.metrics.Inc("ctf_rate_limit_errors_total", map[string]string{"scope": "login"})
		logError("rate_limit.login.error", map[string]any{"error": err.Error()})
		httpx.WriteError(w, http.StatusBadGateway, "rate_limit_error", "failed to enforce rate limit")
		return
	}
	if !allowed {
		s.metrics.Inc("ctf_rate_limit_hits_total", map[string]string{"scope": "login"})
		httpx.WriteError(w, http.StatusTooManyRequests, "login_rate_limited", "too many login attempts, please try again later")
		return
	}

	result, err := s.auth.Login(r.Context(), input)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			httpx.WriteError(w, http.StatusUnauthorized, "invalid_credentials", err.Error())
			return
		}
		logWarn("auth.login.failed", map[string]any{"error": err.Error()})
		httpx.WriteError(w, http.StatusBadGateway, "login_failed", "failed to login")
		return
	}
	writeAuthResponse(w, http.StatusOK, result)
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "missing authenticated user")
		return
	}

	user, err := s.auth.Me(r.Context(), userID)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "failed to load user")
		return
	}
	user.PasswordHash = ""
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"user": user})
}

func (s *Server) handleMeSubmissions(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "missing authenticated user")
		return
	}

	items, err := s.game.UserSubmissions(r.Context(), userID)
	if err != nil {
		logError("me.submissions.failed", map[string]any{"error": err.Error()})
		httpx.WriteError(w, http.StatusBadGateway, "repository_error", "failed to load submissions")
		return
	}
	for i := range items {
		items[i].SubmittedAt = items[i].SubmittedAt.UTC()
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleMeSolves(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "missing authenticated user")
		return
	}

	items, err := s.game.UserSolves(r.Context(), userID)
	if err != nil {
		logError("me.solves.failed", map[string]any{"error": err.Error()})
		httpx.WriteError(w, http.StatusBadGateway, "repository_error", "failed to load solves")
		return
	}
	for i := range items {
		items[i].SolvedAt = items[i].SolvedAt.UTC()
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleAnnouncements(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireContestPhase(w, r, contestRequirement{announcementsVisible: true}); !ok {
		return
	}
	items, err := s.game.Announcements(r.Context())
	if err != nil {
		logError("announcements.list.failed", map[string]any{"error": err.Error()})
		httpx.WriteError(w, http.StatusBadGateway, "repository_error", "failed to load announcements")
		return
	}
	for i := range items {
		if items[i].PublishedAt != nil {
			t := items[i].PublishedAt.UTC()
			items[i].PublishedAt = &t
		}
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleChallenges(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireContestPhase(w, r, contestRequirement{challengeListVisible: true}); !ok {
		return
	}
	items, err := s.runtime.Challenges(r.Context())
	if err != nil {
		logError("challenges.list.failed", map[string]any{"error": err.Error()})
		httpx.WriteError(w, http.StatusBadGateway, "repository_error", "failed to load challenges")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleChallengeDetail(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireContestPhase(w, r, contestRequirement{challengeDetailVisible: true}); !ok {
		return
	}
	challenge, err := s.game.Challenge(r.Context(), r.PathValue("challengeID"))
	if err != nil {
		if errors.Is(err, game.ErrChallengeNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "challenge_not_found", err.Error())
			return
		}
		logError("challenge.detail.failed", map[string]any{"error": err.Error()})
		httpx.WriteError(w, http.StatusBadGateway, "repository_error", "failed to load challenge")
		return
	}
	writeChallengeResponse(w, http.StatusOK, challenge)
}

func (s *Server) handleChallengeAttachmentDownload(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireContestPhase(w, r, contestRequirement{attachmentVisible: true}); !ok {
		return
	}
	attachmentID, err := strconv.ParseInt(r.PathValue("attachmentID"), 10, 64)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_attachment_id", "attachment id must be numeric")
		return
	}

	attachment, storagePath, err := s.game.Attachment(r.Context(), r.PathValue("challengeID"), attachmentID)
	if err != nil {
		switch {
		case errors.Is(err, game.ErrChallengeNotFound):
			httpx.WriteError(w, http.StatusNotFound, "challenge_not_found", err.Error())
		case errors.Is(err, game.ErrAttachmentNotFound):
			httpx.WriteError(w, http.StatusNotFound, "attachment_not_found", err.Error())
		default:
			logError("challenge.attachment.lookup_failed", map[string]any{"error": err.Error()})
			httpx.WriteError(w, http.StatusBadGateway, "repository_error", "failed to load attachment")
		}
		return
	}
	file, err := os.Open(storagePath)
	if err != nil {
		logError("challenge.attachment.open_failed", map[string]any{"error": err.Error()})
		httpx.WriteError(w, http.StatusBadGateway, "attachment_unavailable", "failed to open attachment")
		return
	}
	defer file.Close()
	contentType := attachment.ContentType
	if contentType == "" {
		contentType = mime.TypeByExtension(filepath.Ext(attachment.Filename))
		if contentType == "" {
			contentType = "application/octet-stream"
		}
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", attachment.Filename))
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, file)
}

func (s *Server) handleScoreboard(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireContestPhase(w, r, contestRequirement{scoreboardVisible: true}); !ok {
		return
	}
	items, err := s.game.Scoreboard(r.Context())
	if err != nil {
		logError("scoreboard.load.failed", map[string]any{"error": err.Error()})
		httpx.WriteError(w, http.StatusBadGateway, "repository_error", "failed to load scoreboard")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleCreateInstance(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireContestPhase(w, r, contestRequirement{runtimeAllowed: true}); !ok {
		return
	}
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "missing authenticated user")
		return
	}
	instance, created, err := s.runtime.StartInstance(r.Context(), userID, r.PathValue("challengeID"))
	if err != nil {
		s.writeRuntimeError(w, err)
		return
	}
	status := http.StatusCreated
	if !created {
		status = http.StatusOK
	}
	writeInstanceResponse(w, status, instance)
}

func (s *Server) handleGetInstance(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireContestPhase(w, r, contestRequirement{runtimeAllowed: true}); !ok {
		return
	}
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "missing authenticated user")
		return
	}
	instance, err := s.runtime.GetInstance(r.Context(), userID, r.PathValue("challengeID"))
	if err != nil {
		s.writeRuntimeError(w, err)
		return
	}
	writeInstanceResponse(w, http.StatusOK, instance)
}

func (s *Server) handleDeleteInstance(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireContestPhase(w, r, contestRequirement{runtimeAllowed: true}); !ok {
		return
	}
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "missing authenticated user")
		return
	}
	instance, err := s.runtime.DeleteInstance(r.Context(), userID, r.PathValue("challengeID"))
	if err != nil {
		s.writeRuntimeError(w, err)
		return
	}
	writeInstanceResponse(w, http.StatusOK, instance)
}

func (s *Server) handleRenewInstance(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireContestPhase(w, r, contestRequirement{runtimeAllowed: true}); !ok {
		return
	}
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "missing authenticated user")
		return
	}
	instance, err := s.runtime.RenewInstance(r.Context(), userID, r.PathValue("challengeID"))
	if err != nil {
		s.writeRuntimeError(w, err)
		return
	}
	writeInstanceResponse(w, http.StatusOK, instance)
}

func (s *Server) handleSubmitFlag(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireContestPhase(w, r, contestRequirement{submissionAllowed: true}); !ok {
		return
	}
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "missing authenticated user")
		return
	}
	limiterKey := limitKey("submission", fmt.Sprintf("%d", userID), r.PathValue("challengeID"))
	allowed, err := enforceRateLimit(r.Context(), s.limiters.Submission, limiterKey)
	if err != nil {
		s.metrics.Inc("ctf_rate_limit_errors_total", map[string]string{"scope": "submission"})
		logError("rate_limit.submission.error", map[string]any{"error": err.Error()})
		httpx.WriteError(w, http.StatusBadGateway, "rate_limit_error", "failed to enforce rate limit")
		return
	}
	if !allowed {
		s.metrics.Inc("ctf_rate_limit_hits_total", map[string]string{"scope": "submission"})
		httpx.WriteError(w, http.StatusTooManyRequests, "submission_rate_limited", "too many submissions, please try again later")
		return
	}

	var input struct {
		Flag string `json:"flag"`
	}
	if err := decodeJSON(r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	result, err := s.game.SubmitFlag(r.Context(), userID, r.PathValue("challengeID"), input.Flag, requestSourceIP(r))
	if err != nil {
		if errors.Is(err, game.ErrChallengeNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "challenge_not_found", err.Error())
			return
		}
		logError("flag.submit.failed", map[string]any{"error": err.Error()})
		httpx.WriteError(w, http.StatusBadGateway, "submit_failed", "failed to submit flag")
		return
	}
	if result.SolvedAt != nil {
		t := result.SolvedAt.UTC()
		result.SolvedAt = &t
	}
	httpx.WriteJSON(w, http.StatusOK, result)
}

func (s *Server) handleAdminChallenges(w http.ResponseWriter, r *http.Request) {
	items, err := s.admin.Challenges(r.Context())
	if err != nil {
		logError("admin.challenges.list.failed", map[string]any{"error": err.Error()})
		httpx.WriteError(w, http.StatusBadGateway, "repository_error", "failed to load admin challenges")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleAdminChallengeDetail(w http.ResponseWriter, r *http.Request) {
	challengeID, err := strconv.ParseInt(r.PathValue("challengeID"), 10, 64)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_challenge_id", "challenge id must be numeric")
		return
	}
	challenge, err := s.admin.Challenge(r.Context(), challengeID)
	if err != nil {
		if errors.Is(err, admin.ErrResourceNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "challenge_not_found", err.Error())
			return
		}
		logError("admin.challenge.detail.failed", map[string]any{"error": err.Error()})
		httpx.WriteError(w, http.StatusBadGateway, "repository_error", "failed to load admin challenge")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"challenge": challenge})
}

func (s *Server) handleAdminCreateChallenge(w http.ResponseWriter, r *http.Request) {
	actorUserID, ok := userIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "missing authenticated user")
		return
	}
	allowed, err := enforceRateLimit(r.Context(), s.limiters.AdminWrite, adminRateLimitKey("challenge_create", r, actorUserID))
	if err != nil {
		s.metrics.Inc("ctf_rate_limit_errors_total", map[string]string{"scope": "admin_write"})
		logError("rate_limit.admin_write.error", map[string]any{"action": "challenge_create", "error": err.Error()})
		httpx.WriteError(w, http.StatusBadGateway, "rate_limit_error", "failed to enforce rate limit")
		return
	}
	if !allowed {
		s.metrics.Inc("ctf_rate_limit_hits_total", map[string]string{"scope": "admin_write"})
		httpx.WriteError(w, http.StatusTooManyRequests, "admin_rate_limited", "too many admin write requests, please try again later")
		return
	}
	var input admin.UpsertChallengeInput
	if err := decodeJSON(r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	challenge, err := s.admin.CreateChallenge(r.Context(), input)
	if err != nil {
		logError("admin.challenge.create.failed", map[string]any{"error": err.Error()})
		httpx.WriteError(w, http.StatusBadGateway, "create_failed", "failed to create challenge")
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{"challenge": challenge})
}

func (s *Server) handleAdminUpdateChallenge(w http.ResponseWriter, r *http.Request) {
	actorUserID, ok := userIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "missing authenticated user")
		return
	}
	allowed, err := enforceRateLimit(r.Context(), s.limiters.AdminWrite, adminRateLimitKey("challenge_update", r, actorUserID))
	if err != nil {
		s.metrics.Inc("ctf_rate_limit_errors_total", map[string]string{"scope": "admin_write"})
		logError("rate_limit.admin_write.error", map[string]any{"action": "challenge_update", "error": err.Error()})
		httpx.WriteError(w, http.StatusBadGateway, "rate_limit_error", "failed to enforce rate limit")
		return
	}
	if !allowed {
		s.metrics.Inc("ctf_rate_limit_hits_total", map[string]string{"scope": "admin_write"})
		httpx.WriteError(w, http.StatusTooManyRequests, "admin_rate_limited", "too many admin write requests, please try again later")
		return
	}
	challengeID, err := strconv.ParseInt(r.PathValue("challengeID"), 10, 64)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_challenge_id", "challenge id must be numeric")
		return
	}
	var input admin.UpsertChallengeInput
	if err := decodeJSON(r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	challenge, err := s.admin.UpdateChallenge(r.Context(), challengeID, input)
	if err != nil {
		if errors.Is(err, admin.ErrResourceNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "challenge_not_found", err.Error())
			return
		}
		logError("admin.challenge.update.failed", map[string]any{"error": err.Error()})
		httpx.WriteError(w, http.StatusBadGateway, "update_failed", "failed to update challenge")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"challenge": challenge})
}

func (s *Server) handleAdminCreateAttachment(w http.ResponseWriter, r *http.Request) {
	actorUserID, ok := userIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "missing authenticated user")
		return
	}
	allowed, err := enforceRateLimit(r.Context(), s.limiters.AdminWrite, adminRateLimitKey("attachment_create", r, actorUserID))
	if err != nil {
		s.metrics.Inc("ctf_rate_limit_errors_total", map[string]string{"scope": "admin_write"})
		logError("rate_limit.admin_write.error", map[string]any{"action": "attachment_create", "error": err.Error()})
		httpx.WriteError(w, http.StatusBadGateway, "rate_limit_error", "failed to enforce rate limit")
		return
	}
	if !allowed {
		s.metrics.Inc("ctf_rate_limit_hits_total", map[string]string{"scope": "admin_write"})
		httpx.WriteError(w, http.StatusTooManyRequests, "admin_rate_limited", "too many admin write requests, please try again later")
		return
	}
	challengeID, err := strconv.ParseInt(r.PathValue("challengeID"), 10, 64)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_challenge_id", "challenge id must be numeric")
		return
	}
	filename, contentType, size, file, err := extractUpload(r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_upload", err.Error())
		return
	}
	defer file.Close()
	attachment, err := s.admin.CreateAttachment(r.Context(), actorUserID, challengeID, admin.CreateAttachmentInput{
		Filename:    filename,
		ContentType: contentType,
		Body:        file,
		SizeBytes:   size,
	})
	if err != nil {
		logError("admin.attachment.create.failed", map[string]any{"error": err.Error()})
		httpx.WriteError(w, http.StatusBadGateway, "create_failed", "failed to create attachment")
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{"attachment": attachment})
}

func (s *Server) handleAdminContest(w http.ResponseWriter, r *http.Request) {
	current, err := s.contest.Current(r.Context())
	if err != nil {
		logError("admin.contest.load.failed", map[string]any{"error": err.Error()})
		httpx.WriteError(w, http.StatusBadGateway, "repository_error", "failed to load contest")
		return
	}
	phase := contest.BuildPhase(current)
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"contest": current, "phase": phase})
}

func (s *Server) handleAdminUpdateContest(w http.ResponseWriter, r *http.Request) {
	actorUserID, ok := userIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "missing authenticated user")
		return
	}
	allowed, err := enforceRateLimit(r.Context(), s.limiters.AdminWrite, adminRateLimitKey("contest_update", r, actorUserID))
	if err != nil {
		s.metrics.Inc("ctf_rate_limit_errors_total", map[string]string{"scope": "admin_write"})
		logError("rate_limit.admin_write.error", map[string]any{"action": "contest_update", "error": err.Error()})
		httpx.WriteError(w, http.StatusBadGateway, "rate_limit_error", "failed to enforce rate limit")
		return
	}
	if !allowed {
		s.metrics.Inc("ctf_rate_limit_hits_total", map[string]string{"scope": "admin_write"})
		httpx.WriteError(w, http.StatusTooManyRequests, "admin_rate_limited", "too many admin write requests, please try again later")
		return
	}
	var input contest.UpdateInput
	if err := decodeJSON(r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	updated, err := s.contest.Update(r.Context(), input)
	if err != nil {
		logError("admin.contest.update.failed", map[string]any{"error": err.Error()})
		httpx.WriteError(w, http.StatusBadRequest, "update_failed", err.Error())
		return
	}
	phase := contest.BuildPhase(updated)
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"contest": updated, "phase": phase})
}

func (s *Server) handleAdminAnnouncements(w http.ResponseWriter, r *http.Request) {
	items, err := s.admin.Announcements(r.Context())
	if err != nil {
		logError("admin.announcements.list.failed", map[string]any{"error": err.Error()})
		httpx.WriteError(w, http.StatusBadGateway, "repository_error", "failed to load admin announcements")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleAdminCreateAnnouncement(w http.ResponseWriter, r *http.Request) {
	actorUserID, ok := userIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "missing authenticated user")
		return
	}
	allowed, err := enforceRateLimit(r.Context(), s.limiters.AdminWrite, adminRateLimitKey("announcement_create", r, actorUserID))
	if err != nil {
		s.metrics.Inc("ctf_rate_limit_errors_total", map[string]string{"scope": "admin_write"})
		logError("rate_limit.admin_write.error", map[string]any{"action": "announcement_create", "error": err.Error()})
		httpx.WriteError(w, http.StatusBadGateway, "rate_limit_error", "failed to enforce rate limit")
		return
	}
	if !allowed {
		s.metrics.Inc("ctf_rate_limit_hits_total", map[string]string{"scope": "admin_write"})
		httpx.WriteError(w, http.StatusTooManyRequests, "admin_rate_limited", "too many admin write requests, please try again later")
		return
	}
	var input admin.CreateAnnouncementInput
	if err := decodeJSON(r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	announcement, err := s.admin.CreateAnnouncement(r.Context(), actorUserID, input)
	if err != nil {
		logError("admin.announcement.create.failed", map[string]any{"error": err.Error()})
		httpx.WriteError(w, http.StatusBadGateway, "create_failed", "failed to create announcement")
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{"announcement": announcement})
}

func (s *Server) handleAdminDeleteAnnouncement(w http.ResponseWriter, r *http.Request) {
	actorUserID, ok := userIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "missing authenticated user")
		return
	}
	allowed, err := enforceRateLimit(r.Context(), s.limiters.AdminWrite, adminRateLimitKey("announcement_delete", r, actorUserID))
	if err != nil {
		s.metrics.Inc("ctf_rate_limit_errors_total", map[string]string{"scope": "admin_write"})
		logError("rate_limit.admin_write.error", map[string]any{"action": "announcement_delete", "error": err.Error()})
		httpx.WriteError(w, http.StatusBadGateway, "rate_limit_error", "failed to enforce rate limit")
		return
	}
	if !allowed {
		s.metrics.Inc("ctf_rate_limit_hits_total", map[string]string{"scope": "admin_write"})
		httpx.WriteError(w, http.StatusTooManyRequests, "admin_rate_limited", "too many admin write requests, please try again later")
		return
	}
	announcementID, err := strconv.ParseInt(r.PathValue("announcementID"), 10, 64)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_announcement_id", "announcement id must be numeric")
		return
	}
	announcement, err := s.admin.DeleteAnnouncement(r.Context(), actorUserID, announcementID)
	if err != nil {
		if errors.Is(err, admin.ErrResourceNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "announcement_not_found", err.Error())
			return
		}
		logError("admin.announcement.delete.failed", map[string]any{"error": err.Error()})
		httpx.WriteError(w, http.StatusBadGateway, "delete_failed", "failed to delete announcement")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"announcement": announcement})
}

func (s *Server) handleAdminSubmissions(w http.ResponseWriter, r *http.Request) {
	items, err := s.admin.Submissions(r.Context())
	if err != nil {
		logError("admin.submissions.list.failed", map[string]any{"error": err.Error()})
		httpx.WriteError(w, http.StatusBadGateway, "repository_error", "failed to load submissions")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleAdminInstances(w http.ResponseWriter, r *http.Request) {
	items, err := s.admin.Instances(r.Context())
	if err != nil {
		logError("admin.instances.list.failed", map[string]any{"error": err.Error()})
		httpx.WriteError(w, http.StatusBadGateway, "repository_error", "failed to load instances")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleAdminTerminateInstance(w http.ResponseWriter, r *http.Request) {
	actorUserID, ok := userIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "missing authenticated user")
		return
	}
	allowed, err := enforceRateLimit(r.Context(), s.limiters.AdminWrite, adminRateLimitKey("instance_terminate", r, actorUserID))
	if err != nil {
		s.metrics.Inc("ctf_rate_limit_errors_total", map[string]string{"scope": "admin_write"})
		logError("rate_limit.admin_write.error", map[string]any{"action": "instance_terminate", "error": err.Error()})
		httpx.WriteError(w, http.StatusBadGateway, "rate_limit_error", "failed to enforce rate limit")
		return
	}
	if !allowed {
		s.metrics.Inc("ctf_rate_limit_hits_total", map[string]string{"scope": "admin_write"})
		httpx.WriteError(w, http.StatusTooManyRequests, "admin_rate_limited", "too many admin write requests, please try again later")
		return
	}
	instanceID, err := strconv.ParseInt(r.PathValue("instanceID"), 10, 64)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_instance_id", "instance id must be numeric")
		return
	}
	instance, err := s.admin.TerminateInstance(r.Context(), actorUserID, instanceID)
	if err != nil {
		if errors.Is(err, admin.ErrResourceNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "instance_not_found", err.Error())
			return
		}
		logError("admin.instance.terminate.failed", map[string]any{"error": err.Error()})
		httpx.WriteError(w, http.StatusBadGateway, "terminate_failed", "failed to terminate instance")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"instance": instance})
}

func (s *Server) handleAdminUsers(w http.ResponseWriter, r *http.Request) {
	items, err := s.admin.Users(r.Context())
	if err != nil {
		logError("admin.users.list.failed", map[string]any{"error": err.Error()})
		httpx.WriteError(w, http.StatusBadGateway, "repository_error", "failed to load users")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleAdminUpdateUser(w http.ResponseWriter, r *http.Request) {
	actorUserID, ok := userIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "missing authenticated user")
		return
	}
	allowed, err := enforceRateLimit(r.Context(), s.limiters.AdminWrite, adminRateLimitKey("user_update", r, actorUserID))
	if err != nil {
		s.metrics.Inc("ctf_rate_limit_errors_total", map[string]string{"scope": "admin_write"})
		logError("rate_limit.admin_write.error", map[string]any{"action": "user_update", "error": err.Error()})
		httpx.WriteError(w, http.StatusBadGateway, "rate_limit_error", "failed to enforce rate limit")
		return
	}
	if !allowed {
		s.metrics.Inc("ctf_rate_limit_hits_total", map[string]string{"scope": "admin_write"})
		httpx.WriteError(w, http.StatusTooManyRequests, "admin_rate_limited", "too many admin write requests, please try again later")
		return
	}
	userID, err := strconv.ParseInt(r.PathValue("userID"), 10, 64)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_user_id", "user id must be numeric")
		return
	}
	var input admin.UpdateUserInput
	if err := decodeJSON(r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	user, err := s.admin.UpdateUser(r.Context(), actorUserID, userID, input)
	if err != nil {
		if errors.Is(err, admin.ErrResourceNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "user_not_found", err.Error())
			return
		}
		logError("admin.user.update.failed", map[string]any{"error": err.Error()})
		httpx.WriteError(w, http.StatusBadGateway, "update_failed", "failed to update user")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"user": user})
}

func (s *Server) handleAdminAuditLogs(w http.ResponseWriter, r *http.Request) {
	items, err := s.admin.AuditLogs(r.Context())
	if err != nil {
		logError("admin.audit_logs.list.failed", map[string]any{"error": err.Error()})
		httpx.WriteError(w, http.StatusBadGateway, "repository_error", "failed to load audit logs")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) writeRuntimeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, runtime.ErrChallengeNotFound):
		httpx.WriteError(w, http.StatusNotFound, "challenge_not_found", err.Error())
	case errors.Is(err, runtime.ErrChallengeNotDynamic):
		httpx.WriteError(w, http.StatusConflict, "challenge_not_dynamic", err.Error())
	case errors.Is(err, runtime.ErrRuntimeConfigMissing):
		httpx.WriteError(w, http.StatusConflict, "runtime_config_missing", err.Error())
	case errors.Is(err, runtime.ErrInstanceNotFound):
		httpx.WriteError(w, http.StatusNotFound, "instance_not_found", err.Error())
	case errors.Is(err, runtime.ErrInstanceRenewLimitReached):
		httpx.WriteError(w, http.StatusConflict, "instance_renew_limit_reached", err.Error())
	case errors.Is(err, runtime.ErrInstanceCapacityReached):
		httpx.WriteError(w, http.StatusConflict, "instance_capacity_reached", err.Error())
	case errors.Is(err, runtime.ErrInstanceCooldownActive):
		httpx.WriteError(w, http.StatusConflict, "instance_cooldown_active", err.Error())
	default:
		logError("runtime.error", map[string]any{"error": err.Error()})
		httpx.WriteError(w, http.StatusBadGateway, "runtime_error", fmt.Sprintf("%v", err))
	}
}

func (s *Server) authenticated(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r.Header.Get("Authorization"))
		if token == "" {
			httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "missing bearer token")
			return
		}

		claims, err := s.auth.Authenticate(token)
		if err != nil {
			code := "unauthorized"
			if auth.IsTokenError(err) {
				code = "invalid_token"
			}
			httpx.WriteError(w, http.StatusUnauthorized, code, err.Error())
			return
		}

		next.ServeHTTP(w, r.WithContext(withAuthContext(r.Context(), claims)))
	})
}

func (s *Server) requirePermission(permission string, next http.Handler) http.Handler {
	return s.authenticated(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role, ok := roleFromContext(r.Context())
		if !ok {
			httpx.WriteError(w, http.StatusForbidden, "forbidden", "missing role")
			return
		}
		if !hasPermission(role, permission) {
			httpx.WriteError(w, http.StatusForbidden, "forbidden", fmt.Sprintf("missing permission: %s", permission))
			return
		}
		next.ServeHTTP(w, r)
	}))
}

type contestRequirement struct {
	announcementsVisible  bool
	challengeListVisible  bool
	challengeDetailVisible bool
	attachmentVisible     bool
	scoreboardVisible     bool
	submissionAllowed     bool
	runtimeAllowed        bool
	registrationAllowed   bool
}

func (s *Server) requireContestPhase(w http.ResponseWriter, r *http.Request, requirement contestRequirement) (contest.Phase, bool) {
	phase, err := s.contest.Phase(r.Context())
	if err != nil {
		logError("contest.phase.failed", map[string]any{"error": err.Error()})
		httpx.WriteError(w, http.StatusBadGateway, "repository_error", "failed to load contest state")
		return contest.Phase{}, false
	}
	if requirement.announcementsVisible && !phase.AnnouncementVisible {
		httpx.WriteError(w, http.StatusForbidden, "contest_not_public", phase.Message)
		return contest.Phase{}, false
	}
	if requirement.challengeListVisible && !phase.ChallengeListVisible {
		httpx.WriteError(w, http.StatusForbidden, "contest_not_public", phase.Message)
		return contest.Phase{}, false
	}
	if requirement.challengeDetailVisible && !phase.ChallengeDetailVisible {
		httpx.WriteError(w, http.StatusForbidden, "contest_not_public", phase.Message)
		return contest.Phase{}, false
	}
	if requirement.attachmentVisible && !phase.AttachmentVisible {
		httpx.WriteError(w, http.StatusForbidden, "contest_not_public", phase.Message)
		return contest.Phase{}, false
	}
	if requirement.scoreboardVisible && !phase.ScoreboardVisible {
		httpx.WriteError(w, http.StatusForbidden, "scoreboard_not_public", phase.Message)
		return contest.Phase{}, false
	}
	if requirement.submissionAllowed && !phase.SubmissionAllowed {
		httpx.WriteError(w, http.StatusForbidden, "submission_closed", phase.Message)
		return contest.Phase{}, false
	}
	if requirement.runtimeAllowed && !phase.RuntimeAllowed {
		httpx.WriteError(w, http.StatusForbidden, "runtime_closed", phase.Message)
		return contest.Phase{}, false
	}
	if requirement.registrationAllowed && !phase.RegistrationAllowed {
		httpx.WriteError(w, http.StatusForbidden, "registration_closed", phase.Message)
		return contest.Phase{}, false
	}
	return phase, true
}

func hasPermission(role, permission string) bool {
	permissions := map[string]map[string]bool{
		"admin": {
			"challenge:read":     true,
			"challenge:write":    true,
			"attachment:write":   true,
			"contest:read":       true,
			"contest:write":      true,
			"announcement:read":  true,
			"announcement:write": true,
			"submission:read":    true,
			"instance:read":      true,
			"instance:write":     true,
			"user:read":          true,
			"user:write":         true,
			"audit:read":         true,
		},
		"ops": {
			"contest:read":      true,
			"challenge:read":    true,
			"attachment:write":  true,
			"announcement:read": true,
			"submission:read":   true,
			"instance:read":     true,
			"instance:write":    true,
			"audit:read":        true,
		},
	}
	if rolePerms, ok := permissions[role]; ok {
		return rolePerms[permission]
	}
	return false
}

func writeAuthResponse(w http.ResponseWriter, status int, result auth.AuthResult) {
	httpx.WriteJSON(w, status, map[string]any{
		"token":      result.Token,
		"expires_at": result.ExpiresAt.UTC().Format(time.RFC3339),
		"user":       result.User,
	})
}

func writeChallengeResponse(w http.ResponseWriter, status int, challenge game.Challenge) {
	httpx.WriteJSON(w, status, map[string]any{"challenge": challenge})
}

func writeInstanceResponse(w http.ResponseWriter, status int, instance runtime.Instance) {
	httpx.WriteJSON(w, status, map[string]any{
		"challenge_id":  instance.ChallengeID,
		"status":        instance.Status,
		"access_url":    instance.AccessURL,
		"host_port":     instance.HostPort,
		"renew_count":   instance.RenewCount,
		"started_at":    instance.StartedAt.UTC().Format(time.RFC3339),
		"expires_at":    instance.ExpiresAt.UTC().Format(time.RFC3339),
		"terminated_at": formatTime(instance.TerminatedAt),
	})
}

func formatTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC().Format(time.RFC3339)
}

func decodeJSON(r *http.Request, target any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(target)
}

func bearerToken(header string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}

func withAuthContext(ctx context.Context, claims auth.TokenClaims) context.Context {
	ctx = context.WithValue(ctx, authUserIDKey{}, claims.UserID)
	ctx = context.WithValue(ctx, authRoleKey{}, claims.Role)
	return ctx
}

type authUserIDKey struct{}
type authRoleKey struct{}

func userIDFromContext(ctx context.Context) (int64, bool) {
	value, ok := ctx.Value(authUserIDKey{}).(int64)
	return value, ok
}

func roleFromContext(ctx context.Context) (string, bool) {
	value, ok := ctx.Value(authRoleKey{}).(string)
	return value, ok
}

func requestSourceIP(r *http.Request) string {
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && host != "" {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}

func extractUpload(r *http.Request) (string, string, int64, multipart.File, error) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		return "", "", 0, nil, fmt.Errorf("parse multipart form: %w", err)
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		return "", "", 0, nil, fmt.Errorf("load file part: %w", err)
	}
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = mime.TypeByExtension(filepath.Ext(header.Filename))
	}
	return header.Filename, contentType, header.Size, file, nil
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func loggingMiddleware(metrics *metricsRegistry, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)
		durationMs := float64(time.Since(start).Milliseconds())
		if metrics != nil {
			metrics.Inc("ctf_http_requests_total", map[string]string{"method": r.Method, "path": r.URL.Path, "status": strconv.Itoa(recorder.status)})
			metrics.Add("ctf_http_request_duration_ms_total", durationMs, map[string]string{"method": r.Method, "path": r.URL.Path})
		}
		logInfo("http.request", map[string]any{
			"method":      r.Method,
			"path":        r.URL.Path,
			"status":      recorder.status,
			"duration_ms": durationMs,
			"remote_ip":   requestSourceIP(r),
		})
	})
}
