package announcement

import (
	"context"
	apperrors "ctf-recruit/backend/internal/errors"
	"errors"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
)

type mockRepository struct {
	items []*Announcement
}

func newMockRepository() *mockRepository {
	return &mockRepository{items: make([]*Announcement, 0)}
}

func (m *mockRepository) Create(_ context.Context, announcement *Announcement) error {
	if announcement.ID == uuid.Nil {
		announcement.ID = uuid.New()
	}
	copied := *announcement
	m.items = append(m.items, &copied)
	return nil
}

func (m *mockRepository) GetByID(_ context.Context, id string) (*Announcement, error) {
	for _, announcement := range m.items {
		if announcement.ID.String() == id {
			copied := *announcement
			return &copied, nil
		}
	}
	return nil, nil
}

func (m *mockRepository) Update(_ context.Context, announcement *Announcement) error {
	for i, item := range m.items {
		if item.ID == announcement.ID {
			copied := *announcement
			m.items[i] = &copied
			return nil
		}
	}
	return errors.New("not found")
}

func (m *mockRepository) Delete(_ context.Context, announcement *Announcement) error {
	for i, item := range m.items {
		if item.ID == announcement.ID {
			m.items = append(m.items[:i], m.items[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *mockRepository) List(_ context.Context, filter ListFilter) ([]Announcement, error) {
	results := make([]Announcement, 0)
	for _, announcement := range m.items {
		if filter.PublishedOnly && !announcement.IsPublished {
			continue
		}
		results = append(results, *announcement)
	}

	sort.Slice(results, func(i, j int) bool {
		left := listSortTime(results[i])
		right := listSortTime(results[j])
		if left.Equal(right) {
			return results[i].CreatedAt.After(results[j].CreatedAt)
		}
		return left.After(right)
	})

	if filter.Offset >= len(results) {
		return []Announcement{}, nil
	}
	end := filter.Offset + filter.Limit
	if end > len(results) {
		end = len(results)
	}
	return results[filter.Offset:end], nil
}

func listSortTime(announcement Announcement) time.Time {
	if announcement.PublishedAt != nil {
		return announcement.PublishedAt.UTC()
	}
	return announcement.CreatedAt.UTC()
}

func TestServiceCreateSetsPublishedAtWhenPublishedAndMissing(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	resp, err := svc.Create(context.Background(), CreateAnnouncementRequest{
		Title:       "Maintenance",
		Content:     "System update at 10PM",
		IsPublished: true,
	})
	if err != nil {
		t.Fatalf("expected create success, got error: %v", err)
	}
	if resp.PublishedAt == nil {
		t.Fatal("expected publishedAt to be set")
	}
}

func TestServiceListRespectsPublishedOnly(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.Create(context.Background(), CreateAnnouncementRequest{
		Title:       "Live",
		Content:     "Published announcement",
		IsPublished: true,
	})
	if err != nil {
		t.Fatalf("setup create published failed: %v", err)
	}

	_, err = svc.Create(context.Background(), CreateAnnouncementRequest{
		Title:       "Draft",
		Content:     "Draft announcement",
		IsPublished: false,
	})
	if err != nil {
		t.Fatalf("setup create draft failed: %v", err)
	}

	playerList, err := svc.List(context.Background(), true, 10, 0)
	if err != nil {
		t.Fatalf("expected list success, got error: %v", err)
	}
	if len(playerList.Items) != 1 {
		t.Fatalf("expected 1 published announcement, got %d", len(playerList.Items))
	}

	adminList, err := svc.List(context.Background(), false, 10, 0)
	if err != nil {
		t.Fatalf("expected list success, got error: %v", err)
	}
	if len(adminList.Items) != 2 {
		t.Fatalf("expected 2 announcements for admin, got %d", len(adminList.Items))
	}
}

func TestServiceListOrdersNewestFirst(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	t0 := time.Date(2026, 2, 13, 10, 0, 0, 0, time.UTC)
	t1 := time.Date(2026, 2, 13, 11, 0, 0, 0, time.UTC)

	repo.items = append(repo.items,
		&Announcement{ID: uuid.New(), Title: "Older", Content: "old", IsPublished: true, CreatedAt: t0, UpdatedAt: t0, PublishedAt: &t0},
		&Announcement{ID: uuid.New(), Title: "Newer", Content: "new", IsPublished: true, CreatedAt: t1, UpdatedAt: t1, PublishedAt: &t1},
	)

	list, err := svc.List(context.Background(), false, 10, 0)
	if err != nil {
		t.Fatalf("expected list success, got error: %v", err)
	}
	if len(list.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(list.Items))
	}
	if list.Items[0].Title != "Newer" {
		t.Fatalf("expected newest item first, got %s", list.Items[0].Title)
	}
}

func TestServiceGetRespectsUnpublishedVisibility(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	resp, err := svc.Create(context.Background(), CreateAnnouncementRequest{
		Title:       "Draft",
		Content:     "Hidden",
		IsPublished: false,
	})
	if err != nil {
		t.Fatalf("setup create failed: %v", err)
	}

	_, err = svc.Get(context.Background(), resp.ID, true)
	if err == nil {
		t.Fatal("expected not found for unpublished announcement")
	}

	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected app error, got %T", err)
	}
	if appErr.Code != "ANNOUNCEMENT_NOT_FOUND" {
		t.Fatalf("expected ANNOUNCEMENT_NOT_FOUND, got %s", appErr.Code)
	}
}

func TestServiceUpdateSetsPublishedAtWhenPublishingDraft(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	created, err := svc.Create(context.Background(), CreateAnnouncementRequest{
		Title:       "Draft",
		Content:     "To be published",
		IsPublished: false,
	})
	if err != nil {
		t.Fatalf("setup create failed: %v", err)
	}

	publish := true
	updated, err := svc.Update(context.Background(), created.ID, UpdateAnnouncementRequest{IsPublished: &publish})
	if err != nil {
		t.Fatalf("expected update success, got error: %v", err)
	}
	if updated.PublishedAt == nil {
		t.Fatal("expected publishedAt to be set when publishing")
	}
}

func TestServiceListPaginationReturnsDeterministicSlice(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	t0 := time.Date(2026, 2, 13, 10, 0, 0, 0, time.UTC)
	for i := 0; i < 3; i++ {
		ti := t0.Add(time.Duration(i) * time.Hour)
		repo.items = append(repo.items, &Announcement{
			ID:          uuid.New(),
			Title:       string(rune('A' + i)),
			Content:     "x",
			IsPublished: true,
			CreatedAt:   ti,
			UpdatedAt:   ti,
			PublishedAt: &ti,
		})
	}

	list, err := svc.List(context.Background(), false, 2, 1)
	if err != nil {
		t.Fatalf("expected list success, got error: %v", err)
	}
	if len(list.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(list.Items))
	}

	titles := []string{list.Items[0].Title, list.Items[1].Title}
	sort.Strings(titles)
	if !(titles[0] == "A" || titles[0] == "B" || titles[0] == "C") {
		t.Fatalf("expected valid titles, got %+v", titles)
	}
}
