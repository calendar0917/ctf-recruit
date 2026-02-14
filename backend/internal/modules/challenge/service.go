package challenge

import (
	"context"
	"crypto/sha256"
	apperrors "ctf-recruit/backend/internal/errors"
	"encoding/hex"
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

func (s *Service) GetForSubmission(ctx context.Context, id string, publishedOnly bool) (*Challenge, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, apperrors.BadRequest("CHALLENGE_VALIDATION_ERROR", "Challenge ID is required")
	}

	challenge, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.Internal("CHALLENGE_FETCH_FAILED", "Failed to fetch challenge", fmt.Errorf("get challenge: %w", err))
	}
	if challenge == nil || (publishedOnly && !challenge.IsPublished) {
		return nil, apperrors.NotFound("CHALLENGE_NOT_FOUND", "Challenge not found")
	}

	return challenge, nil
}

func (s *Service) Create(ctx context.Context, req CreateChallengeRequest) (*ChallengeResponse, error) {
	req.Title = strings.TrimSpace(req.Title)
	req.Description = strings.TrimSpace(req.Description)
	req.Category = strings.TrimSpace(req.Category)
	req.Flag = strings.TrimSpace(req.Flag)

	if req.Title == "" || req.Description == "" || req.Category == "" || req.Flag == "" {
		return nil, apperrors.BadRequest("CHALLENGE_VALIDATION_ERROR", "Title, description, category, and flag are required")
	}
	if !isValidDifficulty(req.Difficulty) {
		return nil, apperrors.BadRequest("CHALLENGE_VALIDATION_ERROR", "Difficulty must be easy, medium, or hard")
	}
	if req.Mode == "" {
		req.Mode = ModeStatic
	}
	if !isValidMode(req.Mode) {
		return nil, apperrors.BadRequest("CHALLENGE_VALIDATION_ERROR", "Mode must be static or dynamic")
	}
	if req.Points <= 0 {
		return nil, apperrors.BadRequest("CHALLENGE_VALIDATION_ERROR", "Points must be greater than zero")
	}

	challenge := &Challenge{
		Title:       req.Title,
		Description: req.Description,
		Category:    req.Category,
		Difficulty:  req.Difficulty,
		Mode:        req.Mode,
		Points:      req.Points,
		FlagHash:    hashFlag(req.Flag),
		IsPublished: req.IsPublished,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if err := s.repo.Create(ctx, challenge); err != nil {
		return nil, apperrors.Internal("CHALLENGE_CREATE_FAILED", "Failed to create challenge", fmt.Errorf("create challenge: %w", err))
	}

	resp := mapChallengeResponse(challenge)
	return &resp, nil
}

func (s *Service) Update(ctx context.Context, id string, req UpdateChallengeRequest) (*ChallengeResponse, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, apperrors.BadRequest("CHALLENGE_VALIDATION_ERROR", "Challenge ID is required")
	}

	challenge, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.Internal("CHALLENGE_UPDATE_FAILED", "Failed to update challenge", fmt.Errorf("get challenge: %w", err))
	}
	if challenge == nil {
		return nil, apperrors.NotFound("CHALLENGE_NOT_FOUND", "Challenge not found")
	}

	if req.Title != nil {
		value := strings.TrimSpace(*req.Title)
		if value == "" {
			return nil, apperrors.BadRequest("CHALLENGE_VALIDATION_ERROR", "Title cannot be empty")
		}
		challenge.Title = value
	}
	if req.Description != nil {
		value := strings.TrimSpace(*req.Description)
		if value == "" {
			return nil, apperrors.BadRequest("CHALLENGE_VALIDATION_ERROR", "Description cannot be empty")
		}
		challenge.Description = value
	}
	if req.Category != nil {
		value := strings.TrimSpace(*req.Category)
		if value == "" {
			return nil, apperrors.BadRequest("CHALLENGE_VALIDATION_ERROR", "Category cannot be empty")
		}
		challenge.Category = value
	}
	if req.Difficulty != nil {
		if !isValidDifficulty(*req.Difficulty) {
			return nil, apperrors.BadRequest("CHALLENGE_VALIDATION_ERROR", "Difficulty must be easy, medium, or hard")
		}
		challenge.Difficulty = *req.Difficulty
	}
	if req.Mode != nil {
		if !isValidMode(*req.Mode) {
			return nil, apperrors.BadRequest("CHALLENGE_VALIDATION_ERROR", "Mode must be static or dynamic")
		}
		challenge.Mode = *req.Mode
	}
	if req.Points != nil {
		if *req.Points <= 0 {
			return nil, apperrors.BadRequest("CHALLENGE_VALIDATION_ERROR", "Points must be greater than zero")
		}
		challenge.Points = *req.Points
	}
	if req.Flag != nil {
		value := strings.TrimSpace(*req.Flag)
		if value == "" {
			return nil, apperrors.BadRequest("CHALLENGE_VALIDATION_ERROR", "Flag cannot be empty")
		}
		challenge.FlagHash = hashFlag(value)
	}
	if req.IsPublished != nil {
		challenge.IsPublished = *req.IsPublished
	}

	challenge.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, challenge); err != nil {
		return nil, apperrors.Internal("CHALLENGE_UPDATE_FAILED", "Failed to update challenge", fmt.Errorf("update challenge: %w", err))
	}

	resp := mapChallengeResponse(challenge)
	return &resp, nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return apperrors.BadRequest("CHALLENGE_VALIDATION_ERROR", "Challenge ID is required")
	}

	challenge, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return apperrors.Internal("CHALLENGE_DELETE_FAILED", "Failed to delete challenge", fmt.Errorf("get challenge: %w", err))
	}
	if challenge == nil {
		return apperrors.NotFound("CHALLENGE_NOT_FOUND", "Challenge not found")
	}

	if err := s.repo.Delete(ctx, challenge); err != nil {
		return apperrors.Internal("CHALLENGE_DELETE_FAILED", "Failed to delete challenge", fmt.Errorf("delete challenge: %w", err))
	}

	return nil
}

func (s *Service) Get(ctx context.Context, id string, publishedOnly bool) (*ChallengeResponse, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, apperrors.BadRequest("CHALLENGE_VALIDATION_ERROR", "Challenge ID is required")
	}

	challenge, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.Internal("CHALLENGE_FETCH_FAILED", "Failed to fetch challenge", fmt.Errorf("get challenge: %w", err))
	}
	if challenge == nil || (publishedOnly && !challenge.IsPublished) {
		return nil, apperrors.NotFound("CHALLENGE_NOT_FOUND", "Challenge not found")
	}

	resp := mapChallengeResponse(challenge)
	return &resp, nil
}

func (s *Service) List(ctx context.Context, publishedOnly bool, limit, offset int) (*ChallengeListResponse, error) {
	limit = normalizeLimit(limit)
	if offset < 0 {
		return nil, apperrors.BadRequest("CHALLENGE_VALIDATION_ERROR", "Offset must be zero or greater")
	}

	challenges, err := s.repo.List(ctx, ListFilter{PublishedOnly: publishedOnly, Limit: limit, Offset: offset})
	if err != nil {
		return nil, apperrors.Internal("CHALLENGE_LIST_FAILED", "Failed to list challenges", fmt.Errorf("list challenges: %w", err))
	}

	items := make([]ChallengeResponse, 0, len(challenges))
	for _, challenge := range challenges {
		challengeCopy := challenge
		items = append(items, mapChallengeResponse(&challengeCopy))
	}

	return &ChallengeListResponse{Items: items, Limit: limit, Offset: offset}, nil
}

func isValidDifficulty(value Difficulty) bool {
	switch value {
	case DifficultyEasy, DifficultyMedium, DifficultyHard:
		return true
	default:
		return false
	}
}

func isValidMode(value Mode) bool {
	switch value {
	case ModeStatic, ModeDynamic:
		return true
	default:
		return false
	}
}

func hashFlag(flag string) string {
	hash := sha256.Sum256([]byte(flag))
	return hex.EncodeToString(hash[:])
}

func mapChallengeResponse(challenge *Challenge) ChallengeResponse {
	return ChallengeResponse{
		ID:          challenge.ID.String(),
		Title:       challenge.Title,
		Description: challenge.Description,
		Category:    challenge.Category,
		Difficulty:  challenge.Difficulty,
		Mode:        challenge.Mode,
		Points:      challenge.Points,
		IsPublished: challenge.IsPublished,
		CreatedAt:   challenge.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:   challenge.UpdatedAt.UTC().Format(time.RFC3339),
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
