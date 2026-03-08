package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"ctf/backend/internal/admin"
	"ctf/backend/internal/auth"
	"ctf/backend/internal/config"
	"ctf/backend/internal/game"
	"ctf/backend/internal/httpx"
	"ctf/backend/internal/runtime"
	"ctf/backend/internal/store"
)

type Server struct {
	cfg     config.Config
	admin   *admin.Service
	auth    *auth.Service
	game    *game.Service
	runtime *runtime.Service
	db      *sql.DB
}

func NewServer(cfg config.Config) (*Server, error) {
	db, err := store.Open(cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}

	userRepo := store.NewUserRepository(db)
	adminRepo := store.NewAdminRepository(db)
	gameRepo := store.NewGameRepository(db)
	runtimeRepo := store.NewRuntimeRepository(db)
	tokens := auth.NewTokenManager(cfg.JWTSecret, cfg.JWTTTL)
	manager := runtime.NewDockerManager(cfg.DockerSocketPath)

	return &Server{
		cfg:     cfg,
		admin:   admin.NewService(adminRepo),
		auth:    auth.NewService(userRepo, tokens),
		game:    game.NewService(gameRepo),
		runtime: runtime.NewService(cfg.PublicBaseURL, manager, runtimeRepo),
		db:      db,
	}, nil
}

func NewServerForTests(cfg config.Config, adminService *admin.Service, authService *auth.Service, gameService *game.Service, runtimeService *runtime.Service) *Server {
	return &Server{
		cfg:     cfg,
		admin:   adminService,
		auth:    authService,
		game:    gameService,
		runtime: runtimeService,
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
	mux.HandleFunc("POST /api/v1/auth/register", s.handleRegister)
	mux.HandleFunc("POST /api/v1/auth/login", s.handleLogin)
	mux.Handle("GET /api/v1/me", s.authenticated(http.HandlerFunc(s.handleMe)))
	mux.Handle("GET /api/v1/me/submissions", s.authenticated(http.HandlerFunc(s.handleMeSubmissions)))
	mux.Handle("GET /api/v1/me/solves", s.authenticated(http.HandlerFunc(s.handleMeSolves)))
	mux.HandleFunc("GET /api/v1/announcements", s.handleAnnouncements)
	mux.HandleFunc("GET /api/v1/challenges", s.handleChallenges)
	mux.HandleFunc("GET /api/v1/challenges/{challengeID}", s.handleChallengeDetail)
	mux.HandleFunc("GET /api/v1/scoreboard", s.handleScoreboard)
	mux.Handle("POST /api/v1/challenges/{challengeID}/instances/me", s.authenticated(http.HandlerFunc(s.handleCreateInstance)))
	mux.Handle("GET /api/v1/challenges/{challengeID}/instances/me", s.authenticated(http.HandlerFunc(s.handleGetInstance)))
	mux.Handle("DELETE /api/v1/challenges/{challengeID}/instances/me", s.authenticated(http.HandlerFunc(s.handleDeleteInstance)))
	mux.Handle("POST /api/v1/challenges/{challengeID}/instances/me/renew", s.authenticated(http.HandlerFunc(s.handleRenewInstance)))
	mux.Handle("POST /api/v1/challenges/{challengeID}/submissions", s.authenticated(http.HandlerFunc(s.handleSubmitFlag)))
	mux.Handle("GET /api/v1/admin/challenges", s.adminOnly(http.HandlerFunc(s.handleAdminChallenges)))
	mux.Handle("POST /api/v1/admin/challenges", s.adminOnly(http.HandlerFunc(s.handleAdminCreateChallenge)))
	mux.Handle("GET /api/v1/admin/challenges/{challengeID}", s.adminOnly(http.HandlerFunc(s.handleAdminChallengeDetail)))
	mux.Handle("PATCH /api/v1/admin/challenges/{challengeID}", s.adminOnly(http.HandlerFunc(s.handleAdminUpdateChallenge)))
	mux.Handle("GET /api/v1/admin/announcements", s.adminOnly(http.HandlerFunc(s.handleAdminAnnouncements)))
	mux.Handle("POST /api/v1/admin/announcements", s.adminOnly(http.HandlerFunc(s.handleAdminCreateAnnouncement)))
	mux.Handle("GET /api/v1/admin/submissions", s.adminOnly(http.HandlerFunc(s.handleAdminSubmissions)))
	mux.Handle("GET /api/v1/admin/instances", s.adminOnly(http.HandlerFunc(s.handleAdminInstances)))
	mux.Handle("POST /api/v1/admin/instances/{instanceID}/terminate", s.adminOnly(http.HandlerFunc(s.handleAdminTerminateInstance)))
	return loggingMiddleware(mux)
}

func (s *Server) StartBackground(ctx context.Context) {
	interval, err := time.ParseDuration(s.cfg.InstanceSweeperPollInterval)
	if err != nil {
		log.Printf("invalid INSTANCE_SWEEPER_POLL_INTERVAL: %v", err)
		interval = 30 * time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("instance sweeper started, interval=%s", interval)
	for {
		select {
		case <-ctx.Done():
			log.Printf("instance sweeper stopped")
			return
		case <-ticker.C:
			terminated, err := s.runtime.SweepExpired(ctx)
			if err != nil {
				log.Printf("instance sweeper error: %v", err)
				continue
			}
			if terminated > 0 {
				log.Printf("instance sweeper terminated %d expired instances", terminated)
			}
		}
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"status": "ok", "service": "api"})
}

func (s *Server) handleReady(w http.ResponseWriter, _ *http.Request) {
	ready := s.db != nil
	if s.db != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		ready = s.db.PingContext(ctx) == nil
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"status":                  "ready",
		"database_url_configured": s.cfg.DatabaseURL != "",
		"database_connected":      ready,
		"docker_socket_path":      s.cfg.DockerSocketPath,
		"dynamic_runtime_enabled": true,
	})
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var input auth.RegisterInput
	if err := decodeJSON(r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	result, err := s.auth.Register(r.Context(), input)
	if err != nil {
		log.Printf("register user: %v", err)
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

	result, err := s.auth.Login(r.Context(), input)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			httpx.WriteError(w, http.StatusUnauthorized, "invalid_credentials", err.Error())
			return
		}
		log.Printf("login user: %v", err)
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
		log.Printf("list my submissions: %v", err)
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
		log.Printf("list my solves: %v", err)
		httpx.WriteError(w, http.StatusBadGateway, "repository_error", "failed to load solves")
		return
	}
	for i := range items {
		items[i].SolvedAt = items[i].SolvedAt.UTC()
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleAnnouncements(w http.ResponseWriter, r *http.Request) {
	items, err := s.game.Announcements(r.Context())
	if err != nil {
		log.Printf("list announcements: %v", err)
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
	items, err := s.runtime.Challenges(r.Context())
	if err != nil {
		log.Printf("list challenges: %v", err)
		httpx.WriteError(w, http.StatusBadGateway, "repository_error", "failed to load challenges")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleChallengeDetail(w http.ResponseWriter, r *http.Request) {
	challenge, err := s.game.Challenge(r.Context(), r.PathValue("challengeID"))
	if err != nil {
		if errors.Is(err, game.ErrChallengeNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "challenge_not_found", err.Error())
			return
		}
		log.Printf("load challenge detail: %v", err)
		httpx.WriteError(w, http.StatusBadGateway, "repository_error", "failed to load challenge")
		return
	}
	writeChallengeResponse(w, http.StatusOK, challenge)
}

func (s *Server) handleScoreboard(w http.ResponseWriter, r *http.Request) {
	items, err := s.game.Scoreboard(r.Context())
	if err != nil {
		log.Printf("load scoreboard: %v", err)
		httpx.WriteError(w, http.StatusBadGateway, "repository_error", "failed to load scoreboard")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleCreateInstance(w http.ResponseWriter, r *http.Request) {
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
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "missing authenticated user")
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
		log.Printf("submit flag: %v", err)
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
		log.Printf("list admin challenges: %v", err)
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
		log.Printf("load admin challenge detail: %v", err)
		httpx.WriteError(w, http.StatusBadGateway, "repository_error", "failed to load admin challenge")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"challenge": challenge})
}

func (s *Server) handleAdminCreateChallenge(w http.ResponseWriter, r *http.Request) {
	var input admin.UpsertChallengeInput
	if err := decodeJSON(r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	challenge, err := s.admin.CreateChallenge(r.Context(), input)
	if err != nil {
		log.Printf("create admin challenge: %v", err)
		httpx.WriteError(w, http.StatusBadGateway, "create_failed", "failed to create challenge")
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{"challenge": challenge})
}

func (s *Server) handleAdminUpdateChallenge(w http.ResponseWriter, r *http.Request) {
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
		log.Printf("update admin challenge: %v", err)
		httpx.WriteError(w, http.StatusBadGateway, "update_failed", "failed to update challenge")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"challenge": challenge})
}

func (s *Server) handleAdminAnnouncements(w http.ResponseWriter, r *http.Request) {
	items, err := s.admin.Announcements(r.Context())
	if err != nil {
		log.Printf("list admin announcements: %v", err)
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
	var input admin.CreateAnnouncementInput
	if err := decodeJSON(r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	announcement, err := s.admin.CreateAnnouncement(r.Context(), actorUserID, input)
	if err != nil {
		log.Printf("create admin announcement: %v", err)
		httpx.WriteError(w, http.StatusBadGateway, "create_failed", "failed to create announcement")
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{"announcement": announcement})
}

func (s *Server) handleAdminSubmissions(w http.ResponseWriter, r *http.Request) {
	items, err := s.admin.Submissions(r.Context())
	if err != nil {
		log.Printf("list admin submissions: %v", err)
		httpx.WriteError(w, http.StatusBadGateway, "repository_error", "failed to load submissions")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleAdminInstances(w http.ResponseWriter, r *http.Request) {
	items, err := s.admin.Instances(r.Context())
	if err != nil {
		log.Printf("list admin instances: %v", err)
		httpx.WriteError(w, http.StatusBadGateway, "repository_error", "failed to load instances")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleAdminTerminateInstance(w http.ResponseWriter, r *http.Request) {
	instanceID, err := strconv.ParseInt(r.PathValue("instanceID"), 10, 64)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_instance_id", "instance id must be numeric")
		return
	}
	instance, err := s.admin.TerminateInstance(r.Context(), instanceID)
	if err != nil {
		if errors.Is(err, admin.ErrResourceNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "instance_not_found", err.Error())
			return
		}
		log.Printf("terminate admin instance: %v", err)
		httpx.WriteError(w, http.StatusBadGateway, "terminate_failed", "failed to terminate instance")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"instance": instance})
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
	default:
		log.Printf("runtime error: %v", err)
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

func (s *Server) adminOnly(next http.Handler) http.Handler {
	return s.authenticated(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role, ok := roleFromContext(r.Context())
		if !ok || role != "admin" {
			httpx.WriteError(w, http.StatusForbidden, "forbidden", "admin role required")
			return
		}
		next.ServeHTTP(w, r)
	}))
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

func requestSourceIP(r *http.Request) string {
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		return strings.TrimSpace(parts[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

type ctxKey string

const (
	userIDContextKey ctxKey = "user_id"
	roleContextKey   ctxKey = "role"
)

func withAuthContext(ctx context.Context, claims auth.TokenClaims) context.Context {
	ctx = context.WithValue(ctx, userIDContextKey, claims.UserID)
	ctx = context.WithValue(ctx, roleContextKey, claims.Role)
	return ctx
}

func userIDFromContext(ctx context.Context) (int64, bool) {
	value, ok := ctx.Value(userIDContextKey).(int64)
	return value, ok && value > 0
}

func roleFromContext(ctx context.Context) (string, bool) {
	value, ok := ctx.Value(roleContextKey).(string)
	return value, ok && value != ""
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}
