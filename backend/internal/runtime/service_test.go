package runtime

import (
	"context"
	"fmt"
	"testing"
	"time"
)

type fakeManager struct {
	startCalls int
	stopCalls  int
}

func (m *fakeManager) Start(_ context.Context, req StartRequest) (StartedContainer, error) {
	m.startCalls++
	return StartedContainer{
		ContainerID:   fmt.Sprintf("container-%d", m.startCalls),
		ContainerName: fmt.Sprintf("demo-%d", m.startCalls),
		HostIP:        "127.0.0.1",
		HostPort:      18080 + m.startCalls,
	}, nil
}

func (m *fakeManager) Stop(_ context.Context, _ string) error {
	m.stopCalls++
	return nil
}

func TestStartInstanceIsIdempotentPerUserAndChallenge(t *testing.T) {
	manager := &fakeManager{}
	service := NewService("http://localhost:8080", manager)

	first, created, err := service.StartInstance(context.Background(), 42, "1")
	if err != nil {
		t.Fatalf("start instance: %v", err)
	}
	if !created {
		t.Fatalf("expected first call to create an instance")
	}

	second, created, err := service.StartInstance(context.Background(), 42, "web-welcome")
	if err != nil {
		t.Fatalf("start instance again: %v", err)
	}
	if created {
		t.Fatalf("expected second call to reuse existing instance")
	}
	if manager.startCalls != 1 {
		t.Fatalf("expected one runtime start call, got %d", manager.startCalls)
	}
	if first.ContainerID != second.ContainerID {
		t.Fatalf("expected same instance to be returned")
	}
}

func TestSweepExpiredStopsContainers(t *testing.T) {
	manager := &fakeManager{}
	service := NewService("http://localhost:8080", manager)
	baseTime := time.Date(2025, time.March, 8, 9, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return baseTime }

	instance, created, err := service.StartInstance(context.Background(), 7, "1")
	if err != nil {
		t.Fatalf("start instance: %v", err)
	}
	if !created {
		t.Fatalf("expected instance to be created")
	}

	service.now = func() time.Time { return instance.ExpiresAt.Add(time.Second) }
	terminated, err := service.SweepExpired(context.Background())
	if err != nil {
		t.Fatalf("sweep expired: %v", err)
	}
	if terminated != 1 {
		t.Fatalf("expected one terminated instance, got %d", terminated)
	}
	if manager.stopCalls != 1 {
		t.Fatalf("expected one runtime stop call, got %d", manager.stopCalls)
	}
	if _, err := service.GetInstance(7, "1"); err != ErrInstanceNotFound {
		t.Fatalf("expected instance to be removed after sweep, got %v", err)
	}
}
