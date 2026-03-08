package app

import (
	"context"
	"log"
	"net/http"
	"time"

	"ctf/backend/internal/config"
	"ctf/backend/internal/httpx"
)

type Server struct {
	cfg config.Config
}

func NewServer(cfg config.Config) *Server {
	return &Server{cfg: cfg}
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
			log.Printf("instance sweeper tick")
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
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"status":                  "ready",
		"database_url_configured": s.cfg.DatabaseURL != "",
		"docker_socket_path":      s.cfg.DockerSocketPath,
	})
}

func (s *Server) handleChallenges(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"items": []map[string]any{
			{
				"id":       1,
				"slug":     "web-welcome",
				"title":    "Welcome Panel",
				"category": "web",
				"points":   100,
				"dynamic":  true,
			},
		},
	})
}

func (s *Server) handleScoreboard(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"items": []map[string]any{},
	})
}

func (s *Server) handleCreateInstance(w http.ResponseWriter, r *http.Request) {
	challengeID := r.PathValue("challengeID")
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{
		"challenge_id": challengeID,
		"status":       "pending",
		"access_url":   s.cfg.PublicBaseURL + "/instance/demo",
		"expires_at":   time.Now().Add(30 * time.Minute).UTC().Format(time.RFC3339),
		"note":         "docker runtime integration not implemented yet",
	})
}

func (s *Server) handleGetInstance(w http.ResponseWriter, r *http.Request) {
	challengeID := r.PathValue("challengeID")
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"challenge_id": challengeID,
		"status":       "not_found",
	})
}

func (s *Server) handleDeleteInstance(w http.ResponseWriter, r *http.Request) {
	challengeID := r.PathValue("challengeID")
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"challenge_id": challengeID,
		"status":       "terminated",
	})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}
