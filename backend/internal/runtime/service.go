package runtime

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type Service struct {
	manager       Manager
	store         *MemoryStore
	configs       map[string]ChallengeConfig
	aliases       map[string]string
	publicBaseURL string
	now           func() time.Time
	mu            sync.Mutex
}

func NewService(publicBaseURL string, manager Manager) *Service {
	service := &Service{
		manager:       manager,
		store:         NewMemoryStore(),
		configs:       make(map[string]ChallengeConfig),
		aliases:       make(map[string]string),
		publicBaseURL: publicBaseURL,
		now:           time.Now,
	}

	for _, cfg := range DefaultChallengeConfigs() {
		service.configs[cfg.ID] = cfg
		service.aliases[cfg.Slug] = cfg.ID
	}

	return service
}

func DefaultChallengeConfigs() []ChallengeConfig {
	return []ChallengeConfig{
		{
			ID:              "1",
			Slug:            "web-welcome",
			Title:           "Welcome Panel",
			Category:        "web",
			Points:          100,
			Dynamic:         true,
			ImageName:       "ctf/web-welcome:dev",
			ExposedProtocol: "http",
			ContainerPort:   80,
			TTL:             30 * time.Minute,
			MemoryLimitMB:   256,
			CPUMilli:        500,
		},
	}
}

func (s *Service) Challenges() []ChallengeSummary {
	items := make([]ChallengeSummary, 0, len(s.configs))
	for _, cfg := range s.configs {
		items = append(items, ChallengeSummary{
			ID:       cfg.ID,
			Slug:     cfg.Slug,
			Title:    cfg.Title,
			Category: cfg.Category,
			Points:   cfg.Points,
			Dynamic:  cfg.Dynamic,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})
	return items
}

func (s *Service) StartInstance(ctx context.Context, userID int64, challengeRef string) (Instance, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cfg, ok := s.lookupChallenge(challengeRef)
	if !ok {
		return Instance{}, false, ErrChallengeNotFound
	}
	if !cfg.Dynamic {
		return Instance{}, false, ErrChallengeNotDynamic
	}

	if existing, ok := s.store.GetActive(userID, cfg.ID); ok {
		return existing, false, nil
	}

	started, err := s.manager.Start(ctx, StartRequest{
		ChallengeID: cfg.ID,
		UserID:      userID,
		Config:      cfg,
	})
	if err != nil {
		return Instance{}, false, err
	}

	now := s.now().UTC()
	instance := Instance{
		ChallengeID:   cfg.ID,
		UserID:        userID,
		Status:        "running",
		AccessURL:     buildAccessURL(cfg.ExposedProtocol, s.publicBaseURL, started.HostPort),
		HostPort:      started.HostPort,
		StartedAt:     now,
		ExpiresAt:     now.Add(cfg.TTL),
		ContainerID:   started.ContainerID,
		ContainerName: started.ContainerName,
		HostIP:        started.HostIP,
	}
	s.store.Save(instance)

	return instance, true, nil
}

func (s *Service) GetInstance(userID int64, challengeRef string) (Instance, error) {
	cfg, ok := s.lookupChallenge(challengeRef)
	if !ok {
		return Instance{}, ErrChallengeNotFound
	}

	instance, ok := s.store.GetActive(userID, cfg.ID)
	if !ok {
		return Instance{}, ErrInstanceNotFound
	}
	return instance, nil
}

func (s *Service) DeleteInstance(ctx context.Context, userID int64, challengeRef string) (Instance, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cfg, ok := s.lookupChallenge(challengeRef)
	if !ok {
		return Instance{}, ErrChallengeNotFound
	}

	instance, ok := s.store.GetActive(userID, cfg.ID)
	if !ok {
		return Instance{}, ErrInstanceNotFound
	}

	if err := s.manager.Stop(ctx, instance.ContainerID); err != nil {
		return Instance{}, err
	}

	now := s.now().UTC()
	instance.Status = "terminated"
	instance.TerminatedAt = &now
	s.store.Delete(userID, cfg.ID)
	return instance, nil
}

func (s *Service) SweepExpired(ctx context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	expired := s.store.ListExpired(s.now().UTC())
	terminated := 0
	for _, instance := range expired {
		if err := s.manager.Stop(ctx, instance.ContainerID); err != nil {
			return terminated, err
		}
		s.store.Delete(instance.UserID, instance.ChallengeID)
		terminated++
	}
	return terminated, nil
}

func buildAccessURL(protocol, publicBaseURL string, hostPort int) string {
	scheme := protocol
	if scheme == "" {
		scheme = "http"
	}

	hostname := "localhost"
	if parsed, err := url.Parse(publicBaseURL); err == nil {
		if value := parsed.Hostname(); value != "" {
			hostname = value
		}
	}

	return fmt.Sprintf("%s://%s:%d", scheme, hostname, hostPort)
}

func (s *Service) lookupChallenge(challengeRef string) (ChallengeConfig, bool) {
	if cfg, ok := s.configs[challengeRef]; ok {
		return cfg, true
	}

	if canonicalID, ok := s.aliases[strings.ToLower(challengeRef)]; ok {
		cfg, exists := s.configs[canonicalID]
		return cfg, exists
	}

	return ChallengeConfig{}, false
}
