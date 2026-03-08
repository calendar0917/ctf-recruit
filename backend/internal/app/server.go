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
	"strings"
	"time"

	"ctf/backend/internal/auth"
	"ctf/backend/internal/config"
	"ctf/backend/internal/game"
	"ctf/backend/internal/httpx"
	"ctf/backend/internal/runtime"
	"ctf/backend/internal/store"
)

type Server struct {
	cfg     config.Config
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
	gameRepo := store.NewGameRepository(db)
	runtimeRepo := store.NewRuntimeRepository(db)
	tokens := auth.NewTokenManager(cfg.JWTSecret, cfg.JWTTTL)
	manager := runtime.NewDockerManager(cfg.DockerSocketPath)

	return &Server{
		cfg:     cfg,
		auth:    auth.NewService(userRepo, tokens),
		game:    game.NewService(gameRepo),
		runtime: runtime.NewService(cfg.PublicBaseURL, manager, runtimeRepo),
		db:      db,
	}, nil
}

func NewServerForTests(cfg config.Config, authService *auth.Service, gameService *game.Service, runtimeService *runtime.Service) *Server {
	return &Server{
		cfg:     cfg,
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
	mux.HandleFunc("GET /api/v1/announcements", s.handleAnnouncements)
	mux.HandleFunc("GET /api/v1/challenges", s.handleChallenges)
	mux.HandleFunc("GET /api/v1/challenges/{challengeID}", s.handleChallengeDetail)
	mux.HandleFunc("GET /api/v1/scoreboard", s.handleScoreboard)
	mux.Handle("POST /api/v1/challenges/{challengeID}/instances/me", s.authenticated(http.HandlerFunc(s.handleCreateInstance)))
	mux.Handle("GET /api/v1/challenges/{challengeID}/instances/me", s.authenticated(http.HandlerFunc(s.handleGetInstance)))
	mux.Handle("DELETE /api/v1/challenges/{challengeID}/instances/me", s.authenticated(http.HandlerFunc(s.handleDeleteInstance)))
	mux.Handle("POST /api/v1/challenges/{challengeID}/submissions", s.authenticated(http.HandlerFunc(s.handleSubmitFlag)))
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
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"service": "api",
	})
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

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"items": items,
	})
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

		next.ServeHTTP(w, r.WithContext(withUserID(r.Context(), claims.UserID)))
	})
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

const userIDContextKey ctxKey = "user_id"

func withUserID(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, userIDContextKey, userID)
}

func userIDFromContext(ctx context.Context) (int64, bool) {
	value, ok := ctx.Value(userIDContextKey).(int64)
	return value, ok && value > 0
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}
