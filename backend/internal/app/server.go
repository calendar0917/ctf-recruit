package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"ctf/backend/internal/config"
	"ctf/backend/internal/httpx"
	"ctf/backend/internal/runtime"
	"ctf/backend/internal/store"
)

type Server struct {
	cfg     config.Config
	runtime *runtime.Service
	db      *sql.DB
}

func NewServer(cfg config.Config) (*Server, error) {
	db, err := store.Open(cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}

	repo := store.NewRuntimeRepository(db)
	manager := runtime.NewDockerManager(cfg.DockerSocketPath)
	return &Server{
		cfg:     cfg,
		runtime: runtime.NewService(cfg.PublicBaseURL, manager, repo),
		db:      db,
	}, nil
}

func NewServerWithRuntime(cfg config.Config, runtimeService *runtime.Service) *Server {
	return &Server{
		cfg:     cfg,
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
	mux.HandleFunc("GET /api/v1/challenges", s.handleChallenges)
	mux.HandleFunc("GET /api/v1/scoreboard", s.handleScoreboard)
	mux.HandleFunc("POST /api/v1/challenges/{challengeID}/instances/me", s.handleCreateInstance)
	mux.HandleFunc("GET /api/v1/challenges/{challengeID}/instances/me", s.handleGetInstance)
	mux.HandleFunc("DELETE /api/v1/challenges/{challengeID}/instances/me", s.handleDeleteInstance)
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

func (s *Server) handleScoreboard(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"items": []map[string]any{},
	})
}

func (s *Server) handleCreateInstance(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromRequest(r)
	if !ok {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_user", "missing or invalid X-User-ID header")
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
	userID, ok := userIDFromRequest(r)
	if !ok {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_user", "missing or invalid X-User-ID header")
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
	userID, ok := userIDFromRequest(r)
	if !ok {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_user", "missing or invalid X-User-ID header")
		return
	}

	instance, err := s.runtime.DeleteInstance(r.Context(), userID, r.PathValue("challengeID"))
	if err != nil {
		s.writeRuntimeError(w, err)
		return
	}
	writeInstanceResponse(w, http.StatusOK, instance)
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

func userIDFromRequest(r *http.Request) (int64, bool) {
	value := r.Header.Get("X-User-ID")
	if value == "" {
		return 0, false
	}

	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed <= 0 {
		return 0, false
	}
	return parsed, true
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}
