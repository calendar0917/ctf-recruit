package challenge

import (
	"context"
	apperrors "ctf-recruit/backend/internal/errors"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

type mockRepository struct {
	items map[string]*Challenge
}

func newMockRepository() *mockRepository {
	return &mockRepository{items: map[string]*Challenge{}}
}

func (m *mockRepository) Create(_ context.Context, challenge *Challenge) error {
	if challenge.ID == uuid.Nil {
		challenge.ID = uuid.New()
	}
	copied := *challenge
	m.items[challenge.ID.String()] = &copied
	return nil
}

func (m *mockRepository) GetByID(_ context.Context, id string) (*Challenge, error) {
	if challenge, ok := m.items[id]; ok {
		copied := *challenge
		return &copied, nil
	}
	return nil, nil
}

func (m *mockRepository) Update(_ context.Context, challenge *Challenge) error {
	if _, ok := m.items[challenge.ID.String()]; !ok {
		return errors.New("not found")
	}
	copied := *challenge
	m.items[challenge.ID.String()] = &copied
	return nil
}

func (m *mockRepository) Delete(_ context.Context, challenge *Challenge) error {
	delete(m.items, challenge.ID.String())
	return nil
}

func (m *mockRepository) List(_ context.Context, filter ListFilter) ([]Challenge, error) {
	results := make([]Challenge, 0)
	for _, challenge := range m.items {
		if filter.PublishedOnly && !challenge.IsPublished {
			continue
		}
		results = append(results, *challenge)
	}
	return results, nil
}

func TestServiceCreateHashesFlagAndHidesFlag(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	resp, err := svc.Create(context.Background(), CreateChallengeRequest{
		Title:       "Forensics 101",
		Description: "Find the flag",
		Category:    "forensics",
		Difficulty:  DifficultyEasy,
		Mode:        ModeStatic,
		Points:      100,
		Flag:        "flag{secret}",
		IsPublished: true,
	})
	if err != nil {
		t.Fatalf("expected create success, got error: %v", err)
	}

	stored, _ := repo.GetByID(context.Background(), resp.ID)
	if stored == nil {
		t.Fatal("expected stored challenge")
	}
	if stored.FlagHash == "" {
		t.Fatal("expected flag hash to be stored")
	}
	if stored.FlagHash == "flag{secret}" {
		t.Fatal("expected flag to be hashed")
	}
}

func TestServiceListRespectsPublishedOnly(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.Create(context.Background(), CreateChallengeRequest{
		Title:       "Web 1",
		Description: "Desc",
		Category:    "web",
		Difficulty:  DifficultyEasy,
		Points:      50,
		Flag:        "flag{one}",
		IsPublished: true,
	})
	if err != nil {
		t.Fatalf("setup create published failed: %v", err)
	}

	_, err = svc.Create(context.Background(), CreateChallengeRequest{
		Title:       "Web 2",
		Description: "Desc",
		Category:    "web",
		Difficulty:  DifficultyMedium,
		Points:      100,
		Flag:        "flag{two}",
		IsPublished: false,
	})
	if err != nil {
		t.Fatalf("setup create draft failed: %v", err)
	}

	list, err := svc.List(context.Background(), true, 10, 0)
	if err != nil {
		t.Fatalf("expected list success, got error: %v", err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("expected 1 published challenge, got %d", len(list.Items))
	}

	list, err = svc.List(context.Background(), false, 10, 0)
	if err != nil {
		t.Fatalf("expected list success, got error: %v", err)
	}
	if len(list.Items) != 2 {
		t.Fatalf("expected 2 challenges for admin, got %d", len(list.Items))
	}
}

func TestServiceCreateValidatesDifficulty(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.Create(context.Background(), CreateChallengeRequest{
		Title:       "Crypto",
		Description: "Desc",
		Category:    "crypto",
		Difficulty:  "legendary",
		Points:      100,
		Flag:        "flag{bad}",
		IsPublished: true,
	})
	if err == nil {
		t.Fatal("expected validation error")
	}

	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected app error, got %T", err)
	}
	if appErr.Code != "CHALLENGE_VALIDATION_ERROR" {
		t.Fatalf("expected CHALLENGE_VALIDATION_ERROR, got %s", appErr.Code)
	}
}

func TestServiceUpdateUpdatesTimestamp(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	resp, err := svc.Create(context.Background(), CreateChallengeRequest{
		Title:       "Reverse",
		Description: "Desc",
		Category:    "reversing",
		Difficulty:  DifficultyMedium,
		Points:      150,
		Flag:        "flag{rev}",
		IsPublished: true,
	})
	if err != nil {
		t.Fatalf("setup create failed: %v", err)
	}

	before := time.Now().UTC().Add(-time.Minute)
	updatedTitle := "Reverse 2"
	updated, err := svc.Update(context.Background(), resp.ID, UpdateChallengeRequest{Title: &updatedTitle})
	if err != nil {
		t.Fatalf("expected update success, got error: %v", err)
	}

	if updated.Title != updatedTitle {
		t.Fatalf("expected updated title, got %s", updated.Title)
	}

	parsed, err := time.Parse(time.RFC3339, updated.UpdatedAt)
	if err != nil {
		t.Fatalf("expected updated time to parse: %v", err)
	}
	if !parsed.After(before) {
		t.Fatalf("expected updated time to be after %v, got %v", before, parsed)
	}
}
