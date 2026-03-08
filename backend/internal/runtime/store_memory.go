package runtime

import (
	"fmt"
	"sync"
	"time"
)

type MemoryStore struct {
	mu     sync.RWMutex
	active map[string]Instance
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		active: make(map[string]Instance),
	}
}

func (s *MemoryStore) GetActive(userID int64, challengeID string) (Instance, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	instance, ok := s.active[instanceKey(userID, challengeID)]
	return instance, ok
}

func (s *MemoryStore) Save(instance Instance) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.active[instanceKey(instance.UserID, instance.ChallengeID)] = instance
}

func (s *MemoryStore) Delete(userID int64, challengeID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.active, instanceKey(userID, challengeID))
}

func (s *MemoryStore) ListExpired(now time.Time) []Instance {
	s.mu.RLock()
	defer s.mu.RUnlock()

	expired := make([]Instance, 0)
	for _, instance := range s.active {
		if !instance.ExpiresAt.After(now) {
			expired = append(expired, instance)
		}
	}
	return expired
}

func instanceKey(userID int64, challengeID string) string {
	return fmt.Sprintf("%d:%s", userID, challengeID)
}
