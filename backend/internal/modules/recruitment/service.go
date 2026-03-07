package recruitment

import (
	"context"
	apperrors "ctf-recruit/backend/internal/errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
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

func (s *Service) Create(ctx context.Context, userID string, req CreateSubmissionRequest) (*SubmissionResponse, error) {
	parsedUserID, err := uuid.Parse(strings.TrimSpace(userID))
	if err != nil {
		return nil, apperrors.Unauthorized("AUTH_INVALID_TOKEN", "Invalid token")
	}

	name := strings.TrimSpace(req.Name)
	school := strings.TrimSpace(req.School)
	grade := strings.TrimSpace(req.Grade)
	direction := strings.TrimSpace(req.Direction)
	contact := strings.TrimSpace(req.Contact)
	bio := strings.TrimSpace(req.Bio)

	if name == "" || school == "" || grade == "" || direction == "" || contact == "" || bio == "" {
		return nil, apperrors.BadRequest("RECRUITMENT_VALIDATION_ERROR", "All recruitment fields are required")
	}

	now := time.Now().UTC()
	submission := &Submission{
		UserID:    parsedUserID,
		Name:      name,
		School:    school,
		Grade:     grade,
		Direction: direction,
		Contact:   contact,
		Bio:       bio,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.repo.Create(ctx, submission); err != nil {
		return nil, apperrors.Internal("RECRUITMENT_CREATE_FAILED", "Failed to create recruitment submission", fmt.Errorf("create recruitment submission: %w", err))
	}

	resp := mapSubmissionResponse(submission)
	return &resp, nil
}

func (s *Service) List(ctx context.Context, limit, offset int) (*SubmissionListResponse, error) {
	limit = normalizeLimit(limit)
	if offset < 0 {
		return nil, apperrors.BadRequest("RECRUITMENT_VALIDATION_ERROR", "Offset must be zero or greater")
	}

	submissions, err := s.repo.List(ctx, ListFilter{Limit: limit, Offset: offset})
	if err != nil {
		return nil, apperrors.Internal("RECRUITMENT_LIST_FAILED", "Failed to list recruitment submissions", fmt.Errorf("list recruitment submissions: %w", err))
	}

	items := make([]SubmissionResponse, 0, len(submissions))
	for _, submission := range submissions {
		submissionCopy := submission
		items = append(items, mapSubmissionResponse(&submissionCopy))
	}

	return &SubmissionListResponse{Items: items, Limit: limit, Offset: offset}, nil
}

func (s *Service) Get(ctx context.Context, id string) (*SubmissionResponse, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, apperrors.BadRequest("RECRUITMENT_VALIDATION_ERROR", "Submission ID is required")
	}

	submission, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.Internal("RECRUITMENT_FETCH_FAILED", "Failed to fetch recruitment submission", fmt.Errorf("get recruitment submission: %w", err))
	}
	if submission == nil {
		return nil, apperrors.NotFound("RECRUITMENT_NOT_FOUND", "Recruitment submission not found")
	}

	resp := mapSubmissionResponse(submission)
	return &resp, nil
}

func mapSubmissionResponse(submission *Submission) SubmissionResponse {
	return SubmissionResponse{
		ID:        submission.ID.String(),
		UserID:    submission.UserID.String(),
		Name:      submission.Name,
		School:    submission.School,
		Grade:     submission.Grade,
		Direction: submission.Direction,
		Contact:   submission.Contact,
		Bio:       submission.Bio,
		CreatedAt: submission.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: submission.UpdatedAt.UTC().Format(time.RFC3339),
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
