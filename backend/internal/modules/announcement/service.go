package announcement

import (
	"context"
	apperrors "ctf-recruit/backend/internal/errors"
	"fmt"
	"strings"
	"time"
)

const (
	defaultLimit = 20
	maxLimit     = 100
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, req CreateAnnouncementRequest) (*AnnouncementResponse, error) {
	title := strings.TrimSpace(req.Title)
	content := strings.TrimSpace(req.Content)
	if title == "" || content == "" {
		return nil, apperrors.BadRequest("ANNOUNCEMENT_VALIDATION_ERROR", "Title and content are required")
	}

	now := time.Now().UTC()
	announcement := &Announcement{
		Title:       title,
		Content:     content,
		IsPublished: req.IsPublished,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if req.IsPublished {
		publishedAt, err := resolvePublishedAt(req.PublishedAt, now)
		if err != nil {
			return nil, err
		}
		announcement.PublishedAt = &publishedAt
	}

	if err := s.repo.Create(ctx, announcement); err != nil {
		return nil, apperrors.Internal("ANNOUNCEMENT_CREATE_FAILED", "Failed to create announcement", fmt.Errorf("create announcement: %w", err))
	}

	resp := mapAnnouncementResponse(announcement)
	return &resp, nil
}

func (s *Service) Update(ctx context.Context, id string, req UpdateAnnouncementRequest) (*AnnouncementResponse, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, apperrors.BadRequest("ANNOUNCEMENT_VALIDATION_ERROR", "Announcement ID is required")
	}

	announcement, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.Internal("ANNOUNCEMENT_UPDATE_FAILED", "Failed to update announcement", fmt.Errorf("get announcement: %w", err))
	}
	if announcement == nil {
		return nil, apperrors.NotFound("ANNOUNCEMENT_NOT_FOUND", "Announcement not found")
	}

	if req.Title != nil {
		value := strings.TrimSpace(*req.Title)
		if value == "" {
			return nil, apperrors.BadRequest("ANNOUNCEMENT_VALIDATION_ERROR", "Title cannot be empty")
		}
		announcement.Title = value
	}
	if req.Content != nil {
		value := strings.TrimSpace(*req.Content)
		if value == "" {
			return nil, apperrors.BadRequest("ANNOUNCEMENT_VALIDATION_ERROR", "Content cannot be empty")
		}
		announcement.Content = value
	}

	if req.IsPublished != nil {
		announcement.IsPublished = *req.IsPublished
		if !announcement.IsPublished {
			announcement.PublishedAt = nil
		}
	}

	if announcement.IsPublished {
		switch {
		case req.PublishedAt != nil:
			publishedAt, parseErr := parsePublishedAt(*req.PublishedAt)
			if parseErr != nil {
				return nil, parseErr
			}
			announcement.PublishedAt = &publishedAt
		case announcement.PublishedAt == nil:
			now := time.Now().UTC()
			announcement.PublishedAt = &now
		}
	}

	announcement.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, announcement); err != nil {
		return nil, apperrors.Internal("ANNOUNCEMENT_UPDATE_FAILED", "Failed to update announcement", fmt.Errorf("update announcement: %w", err))
	}

	resp := mapAnnouncementResponse(announcement)
	return &resp, nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return apperrors.BadRequest("ANNOUNCEMENT_VALIDATION_ERROR", "Announcement ID is required")
	}

	announcement, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return apperrors.Internal("ANNOUNCEMENT_DELETE_FAILED", "Failed to delete announcement", fmt.Errorf("get announcement: %w", err))
	}
	if announcement == nil {
		return apperrors.NotFound("ANNOUNCEMENT_NOT_FOUND", "Announcement not found")
	}

	if err := s.repo.Delete(ctx, announcement); err != nil {
		return apperrors.Internal("ANNOUNCEMENT_DELETE_FAILED", "Failed to delete announcement", fmt.Errorf("delete announcement: %w", err))
	}

	return nil
}

func (s *Service) Get(ctx context.Context, id string, publishedOnly bool) (*AnnouncementResponse, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, apperrors.BadRequest("ANNOUNCEMENT_VALIDATION_ERROR", "Announcement ID is required")
	}

	announcement, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.Internal("ANNOUNCEMENT_FETCH_FAILED", "Failed to fetch announcement", fmt.Errorf("get announcement: %w", err))
	}
	if announcement == nil || (publishedOnly && !announcement.IsPublished) {
		return nil, apperrors.NotFound("ANNOUNCEMENT_NOT_FOUND", "Announcement not found")
	}

	resp := mapAnnouncementResponse(announcement)
	return &resp, nil
}

func (s *Service) List(ctx context.Context, publishedOnly bool, limit, offset int) (*AnnouncementListResponse, error) {
	limit = normalizeLimit(limit)
	if offset < 0 {
		return nil, apperrors.BadRequest("ANNOUNCEMENT_VALIDATION_ERROR", "Offset must be zero or greater")
	}

	announcements, err := s.repo.List(ctx, ListFilter{PublishedOnly: publishedOnly, Limit: limit, Offset: offset})
	if err != nil {
		return nil, apperrors.Internal("ANNOUNCEMENT_LIST_FAILED", "Failed to list announcements", fmt.Errorf("list announcements: %w", err))
	}

	items := make([]AnnouncementResponse, 0, len(announcements))
	for _, announcement := range announcements {
		announcementCopy := announcement
		items = append(items, mapAnnouncementResponse(&announcementCopy))
	}

	return &AnnouncementListResponse{Items: items, Limit: limit, Offset: offset}, nil
}

func resolvePublishedAt(value *string, fallback time.Time) (time.Time, error) {
	if value == nil {
		return fallback.UTC(), nil
	}
	return parsePublishedAt(*value)
}

func parsePublishedAt(value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, apperrors.BadRequest("ANNOUNCEMENT_VALIDATION_ERROR", "publishedAt must be a valid RFC3339 timestamp")
	}
	return parsed.UTC(), nil
}

func mapAnnouncementResponse(announcement *Announcement) AnnouncementResponse {
	var publishedAt *string
	if announcement.PublishedAt != nil {
		value := announcement.PublishedAt.UTC().Format(time.RFC3339)
		publishedAt = &value
	}

	return AnnouncementResponse{
		ID:          announcement.ID.String(),
		Title:       announcement.Title,
		Content:     announcement.Content,
		IsPublished: announcement.IsPublished,
		PublishedAt: publishedAt,
		CreatedAt:   announcement.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:   announcement.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return defaultLimit
	}
	if limit > maxLimit {
		return maxLimit
	}
	return limit
}
