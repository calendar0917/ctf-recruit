package runtime

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"
)

type Service struct {
	manager Manager
	repo    Repository
	cfg     ServiceConfig
	now     func() time.Time
	mu      sync.Mutex
}

func NewService(cfg ServiceConfig, manager Manager, repo Repository) *Service {
	cfg.PublicBaseURL = strings.TrimSpace(cfg.PublicBaseURL)
	if cfg.PublicBaseURL == "" {
		cfg.PublicBaseURL = "http://localhost:8080"
	}
	cfg.RuntimeBaseURL = strings.TrimSpace(cfg.RuntimeBaseURL)
	if cfg.RuntimeBaseURL == "" {
		cfg.RuntimeBaseURL = cfg.PublicBaseURL
	}
	cfg.BindAddr = strings.TrimSpace(cfg.BindAddr)
	if cfg.BindAddr == "" {
		cfg.BindAddr = "127.0.0.1"
	}
	if cfg.PortMin < 0 {
		cfg.PortMin = 0
	}
	if cfg.PortMax < 0 {
		cfg.PortMax = 0
	}
	if cfg.PortMin > 0 && cfg.PortMax > 0 && cfg.PortMin > cfg.PortMax {
		cfg.PortMin = 0
		cfg.PortMax = 0
	}

	return &Service{
		manager: manager,
		repo:    repo,
		cfg:     cfg,
		now:     time.Now,
	}
}

func (s *Service) Challenges(ctx context.Context) ([]ChallengeSummary, error) {
	return s.repo.ListChallenges(ctx)
}

func (s *Service) StartInstance(ctx context.Context, userID int64, challengeRef string) (Instance, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, err := s.repo.GetChallengeConfig(ctx, challengeRef)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return Instance{}, false, ErrChallengeNotFound
		}
		return Instance{}, false, err
	}

	cfg := record.Challenge
	if !cfg.Dynamic {
		return Instance{}, false, ErrChallengeNotDynamic
	}
	if record.ID == 0 || cfg.ImageName == "" || cfg.ContainerPort == 0 {
		return Instance{}, false, ErrRuntimeConfigMissing
	}

	existing, err := s.repo.GetActiveInstance(ctx, userID, cfg.ID)
	if err == nil {
		existing.Instance.AccessURL = buildAccessURL(cfg.ExposedProtocol, s.cfg.RuntimeBaseURL, existing.Instance.HostPort)
		return existing.Instance, false, nil
	}
	if !errors.Is(err, ErrRepositoryNotFound) {
		return Instance{}, false, err
	}

	if cfg.MaxActiveInstances > 0 {
		activeCount, err := s.repo.CountActiveInstances(ctx, cfg.ID)
		if err != nil {
			return Instance{}, false, err
		}
		if activeCount >= cfg.MaxActiveInstances {
			return Instance{}, false, ErrInstanceCapacityReached
		}
	}

	if cfg.UserCooldown > 0 {
		latest, err := s.repo.GetLatestInstance(ctx, userID, cfg.ID)
		if err != nil && !errors.Is(err, ErrRepositoryNotFound) {
			return Instance{}, false, err
		}
		if err == nil {
			nextAllowedAt := latest.Instance.StartedAt.Add(cfg.UserCooldown)
			if nextAllowedAt.After(s.now().UTC()) {
				return Instance{}, false, ErrInstanceCooldownActive
			}
		}
	}

	started, hostPort, err := s.startContainer(ctx, userID, cfg)
	if err != nil {
		return Instance{}, false, err
	}

	now := s.now().UTC()
	instance := Instance{
		ChallengeID:   cfg.ID,
		UserID:        userID,
		Status:        "running",
		AccessURL:     buildAccessURL(cfg.ExposedProtocol, s.cfg.RuntimeBaseURL, hostPort),
		HostPort:      hostPort,
		RenewCount:    0,
		StartedAt:     now,
		ExpiresAt:     now.Add(cfg.TTL),
		ContainerID:   started.ContainerID,
		ContainerName: started.ContainerName,
		HostIP:        started.HostIP,
	}

	saved, err := s.repo.CreateInstance(ctx, record.ID, instance)
	if err != nil {
		_ = s.manager.Stop(context.Background(), started.ContainerID)

		if existing, lookupErr := s.repo.GetActiveInstance(ctx, userID, cfg.ID); lookupErr == nil {
			existing.Instance.AccessURL = buildAccessURL(cfg.ExposedProtocol, s.cfg.RuntimeBaseURL, existing.Instance.HostPort)
			return existing.Instance, false, nil
		}
		return Instance{}, false, err
	}

	saved.Instance.AccessURL = buildAccessURL(cfg.ExposedProtocol, s.cfg.RuntimeBaseURL, saved.Instance.HostPort)
	return saved.Instance, true, nil
}

func (s *Service) GetInstance(ctx context.Context, userID int64, challengeRef string) (Instance, error) {
	record, err := s.repo.GetChallengeConfig(ctx, challengeRef)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return Instance{}, ErrChallengeNotFound
		}
		return Instance{}, err
	}

	cfg := record.Challenge
	if !cfg.Dynamic {
		return Instance{}, ErrChallengeNotDynamic
	}

	instanceRecord, err := s.repo.GetActiveInstance(ctx, userID, cfg.ID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return Instance{}, ErrInstanceNotFound
		}
		return Instance{}, err
	}
	instanceRecord.Instance.AccessURL = buildAccessURL(cfg.ExposedProtocol, s.cfg.RuntimeBaseURL, instanceRecord.Instance.HostPort)
	return instanceRecord.Instance, nil
}

func (s *Service) RenewInstance(ctx context.Context, userID int64, challengeRef string) (Instance, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, err := s.repo.GetChallengeConfig(ctx, challengeRef)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return Instance{}, ErrChallengeNotFound
		}
		return Instance{}, err
	}

	cfg := record.Challenge
	if !cfg.Dynamic {
		return Instance{}, ErrChallengeNotDynamic
	}
	if record.ID == 0 || cfg.ImageName == "" || cfg.ContainerPort == 0 {
		return Instance{}, ErrRuntimeConfigMissing
	}

	instanceRecord, err := s.repo.GetActiveInstance(ctx, userID, cfg.ID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return Instance{}, ErrInstanceNotFound
		}
		return Instance{}, err
	}
	if instanceRecord.Instance.RenewCount >= cfg.MaxRenewCount {
		return Instance{}, ErrInstanceRenewLimitReached
	}

	now := s.now().UTC()
	base := instanceRecord.Instance.ExpiresAt.UTC()
	if base.Before(now) {
		base = now
	}
	updated, err := s.repo.RenewInstance(ctx, instanceRecord.ID, base.Add(cfg.TTL))
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return Instance{}, ErrInstanceNotFound
		}
		return Instance{}, err
	}
	updated.Instance.AccessURL = buildAccessURL(cfg.ExposedProtocol, s.cfg.RuntimeBaseURL, updated.Instance.HostPort)
	return updated.Instance, nil
}

func (s *Service) DeleteInstance(ctx context.Context, userID int64, challengeRef string) (Instance, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, err := s.repo.GetChallengeConfig(ctx, challengeRef)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return Instance{}, ErrChallengeNotFound
		}
		return Instance{}, err
	}

	cfg := record.Challenge
	if !cfg.Dynamic {
		return Instance{}, ErrChallengeNotDynamic
	}

	instanceRecord, err := s.repo.GetActiveInstance(ctx, userID, cfg.ID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return Instance{}, ErrInstanceNotFound
		}
		return Instance{}, err
	}

	if err := s.manager.Stop(ctx, instanceRecord.Instance.ContainerID); err != nil {
		return Instance{}, err
	}

	now := s.now().UTC()
	if err := s.repo.TerminateInstance(ctx, instanceRecord.ID, now); err != nil {
		return Instance{}, err
	}

	instanceRecord.Instance.Status = "terminated"
	instanceRecord.Instance.TerminatedAt = &now
	instanceRecord.Instance.AccessURL = buildAccessURL(cfg.ExposedProtocol, s.cfg.RuntimeBaseURL, instanceRecord.Instance.HostPort)
	return instanceRecord.Instance, nil
}

func (s *Service) SweepExpired(ctx context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	expired, err := s.repo.ListExpiredInstances(ctx, s.now().UTC())
	if err != nil {
		return 0, err
	}

	terminated := 0
	for _, item := range expired {
		if err := s.manager.Stop(ctx, item.Instance.ContainerID); err != nil {
			return terminated, err
		}
		if err := s.repo.TerminateInstance(ctx, item.ID, s.now().UTC()); err != nil {
			return terminated, err
		}
		terminated++
	}
	return terminated, nil
}

func (s *Service) Reconcile(ctx context.Context) (ReconcileReport, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	report := ReconcileReport{}
	active, err := s.repo.ListActiveInstances(ctx)
	if err != nil {
		return report, err
	}

	now := s.now().UTC()
	managedByKey := make(map[string]ManagedContainer)
	containers, err := s.manager.ListManagedContainers(ctx)
	if err != nil {
		return report, err
	}
	for _, container := range containers {
		managedByKey[managedContainerKey(container.ChallengeID, container.UserID)] = container
	}

	for _, item := range active {
		key := managedContainerKey(item.Instance.ChallengeID, item.Instance.UserID)
		exists, err := s.manager.Exists(ctx, item.Instance.ContainerID)
		if err != nil {
			return report, err
		}
		if !exists {
			delete(managedByKey, key)
			if err := s.repo.TerminateInstance(ctx, item.ID, now); err != nil {
				return report, err
			}
			report.TerminatedRecords++
			continue
		}
		delete(managedByKey, key)
	}

	for _, container := range managedByKey {
		if err := s.manager.Stop(ctx, container.ContainerID); err != nil {
			return report, err
		}
		report.RemovedContainers++
	}

	return report, nil
}

func (s *Service) startContainer(ctx context.Context, userID int64, cfg ChallengeConfig) (StartedContainer, int, error) {
	if s.cfg.PortMin <= 0 || s.cfg.PortMax <= 0 || s.cfg.PortMin > s.cfg.PortMax {
		started, err := s.manager.Start(ctx, StartRequest{
			ChallengeID: cfg.ID,
			UserID:      userID,
			BindAddr:    s.cfg.BindAddr,
			HostPort:    0,
			Config:      cfg,
		})
		return started, started.HostPort, err
	}

	ports, err := s.repo.ListActiveHostPorts(ctx)
	if err != nil {
		return StartedContainer{}, 0, err
	}
	used := make(map[int]struct{}, len(ports))
	for _, port := range ports {
		if port > 0 {
			used[port] = struct{}{}
		}
	}

	for port := s.cfg.PortMin; port <= s.cfg.PortMax; port++ {
		if _, ok := used[port]; ok {
			continue
		}
		started, err := s.manager.Start(ctx, StartRequest{
			ChallengeID: cfg.ID,
			UserID:      userID,
			BindAddr:    s.cfg.BindAddr,
			HostPort:    port,
			Config:      cfg,
		})
		if err != nil {
			if isPortBindError(err) {
				used[port] = struct{}{}
				continue
			}
			return StartedContainer{}, 0, err
		}
		return started, started.HostPort, nil
	}

	return StartedContainer{}, 0, ErrInstancePortExhausted
}

func isPortBindError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "port is already allocated") || strings.Contains(msg, "address already in use")
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

func managedContainerKey(challengeID string, userID int64) string {
	return fmt.Sprintf("%s:%d", challengeID, userID)
}
