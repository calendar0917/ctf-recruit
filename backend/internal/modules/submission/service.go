package submission

import (
	"context"
	"crypto/sha256"
	apperrors "ctf-recruit/backend/internal/errors"
	"ctf-recruit/backend/internal/modules/auth"
	"ctf-recruit/backend/internal/modules/challenge"
	"ctf-recruit/backend/internal/modules/judge"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	defaultLimit = 20
	maxLimit     = 100
)

type ChallengeReader interface {
	GetForSubmission(ctx context.Context, id string, publishedOnly bool) (*challenge.Challenge, error)
}

type Service struct {
	repo       Repository
	challenges ChallengeReader
	queue      judge.Queue
}

func NewService(repo Repository, challenges ChallengeReader, queue judge.Queue) *Service {
	return &Service{repo: repo, challenges: challenges, queue: queue}
}

func (s *Service) Submit(ctx context.Context, userID string, role auth.Role, req CreateSubmissionRequest) (*SubmissionResponse, error) {
	userUUID, err := parseUserID(userID)
	if err != nil {
		return nil, err
	}

	challengeID := strings.TrimSpace(req.ChallengeID)
	if challengeID == "" {
		challengeID = strings.TrimSpace(req.ChallengeIDSnake)
	}
	flag := strings.TrimSpace(req.Flag)

	if challengeID == "" || flag == "" {
		return nil, apperrors.BadRequest("SUBMISSION_VALIDATION_ERROR", "challengeId and flag are required")
	}

	challengeUUID, err := uuid.Parse(challengeID)
	if err != nil {
		return nil, apperrors.BadRequest("SUBMISSION_VALIDATION_ERROR", "challengeId must be a valid UUID")
	}

	publishedOnly := true
	if role == auth.RoleAdmin {
		publishedOnly = false
	}

	ch, err := s.challenges.GetForSubmission(ctx, challengeUUID.String(), publishedOnly)
	if err != nil {
		return nil, err
	}
	if ch == nil {
		return nil, apperrors.NotFound("CHALLENGE_NOT_FOUND", "Challenge not found")
	}

	status := StatusWrong
	awardedPoints := 0
	if ch.Mode == challenge.ModeDynamic {
		status = StatusPending
	} else if hashFlag(flag) == ch.FlagHash {
		status = StatusCorrect
		awardedPoints = ch.Points
	}

	sub := &Submission{
		ID:            uuid.New(),
		UserID:        userUUID,
		ChallengeID:   challengeUUID,
		Status:        status,
		AwardedPoints: awardedPoints,
		CreatedAt:     time.Now().UTC(),
	}

	if err := s.repo.Create(ctx, sub); err != nil {
		if status == StatusCorrect && awardedPoints > 0 && isUniqueViolation(err) {
			sub.ID = uuid.New()
			sub.AwardedPoints = 0
			if err2 := s.repo.Create(ctx, sub); err2 != nil {
				return nil, apperrors.Internal("SUBMISSION_CREATE_FAILED", "Failed to record submission", fmt.Errorf("create submission after duplicate award: %w", err2))
			}
		} else {
			return nil, apperrors.Internal("SUBMISSION_CREATE_FAILED", "Failed to record submission", fmt.Errorf("create submission: %w", err))
		}
	}

	var judgeJobID *string
	if ch.Mode == challenge.ModeDynamic {
		if s.queue == nil {
			return nil, apperrors.Internal("JUDGE_QUEUE_UNAVAILABLE", "Judge queue is unavailable", nil)
		}
		job, err := s.queue.Enqueue(ctx, judge.EnqueueInput{
			SubmissionID: sub.ID,
			UserID:       userUUID,
			ChallengeID:  challengeUUID,
		})
		if err != nil {
			return nil, apperrors.Internal("JUDGE_JOB_ENQUEUE_FAILED", "Failed to enqueue judge job", fmt.Errorf("enqueue judge job: %w", err))
		}
		jobID := job.ID.String()
		judgeJobID = &jobID
	}

	resp := mapSubmissionResponse(sub)
	resp.JudgeJobID = judgeJobID

	slog.Info("submission_scoring_event",
		"userId", userUUID.String(),
		"challengeId", challengeUUID.String(),
		"status", status,
		"awardedPoints", sub.AwardedPoints,
		"at", sub.CreatedAt.UTC().Format(time.RFC3339),
	)

	return &resp, nil
}

func (s *Service) ListMine(ctx context.Context, userID string, limit, offset int) (*SubmissionListResponse, error) {
	userUUID, err := parseUserID(userID)
	if err != nil {
		return nil, err
	}

	limit, offset, err = normalizePagination(limit, offset)
	if err != nil {
		return nil, err
	}

	items, err := s.repo.ListByUser(ctx, userUUID, limit, offset)
	if err != nil {
		return nil, apperrors.Internal("SUBMISSION_LIST_FAILED", "Failed to list submissions", fmt.Errorf("list submissions by user: %w", err))
	}

	return &SubmissionListResponse{Items: mapSubmissionResponses(items), Limit: limit, Offset: offset}, nil
}

func (s *Service) ListMineByChallenge(ctx context.Context, userID, challengeID string, limit, offset int) (*SubmissionListResponse, error) {
	userUUID, err := parseUserID(userID)
	if err != nil {
		return nil, err
	}

	challengeID = strings.TrimSpace(challengeID)
	challengeUUID, err := uuid.Parse(challengeID)
	if err != nil {
		return nil, apperrors.BadRequest("SUBMISSION_VALIDATION_ERROR", "challengeId must be a valid UUID")
	}

	limit, offset, err = normalizePagination(limit, offset)
	if err != nil {
		return nil, err
	}

	items, err := s.repo.ListByUserAndChallenge(ctx, userUUID, challengeUUID, limit, offset)
	if err != nil {
		return nil, apperrors.Internal("SUBMISSION_LIST_FAILED", "Failed to list submissions", fmt.Errorf("list submissions by user and challenge: %w", err))
	}

	return &SubmissionListResponse{Items: mapSubmissionResponses(items), Limit: limit, Offset: offset}, nil
}

func hashFlag(flag string) string {
	hash := sha256.Sum256([]byte(flag))
	return hex.EncodeToString(hash[:])
}

func mapSubmissionResponse(sub *Submission) SubmissionResponse {
	return SubmissionResponse{
		ID:            sub.ID.String(),
		ChallengeID:   sub.ChallengeID.String(),
		Status:        sub.Status,
		AwardedPoints: sub.AwardedPoints,
		CreatedAt:     sub.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func mapSubmissionResponses(items []Submission) []SubmissionResponse {
	out := make([]SubmissionResponse, 0, len(items))
	for i := range items {
		item := mapSubmissionResponse(&items[i])
		out = append(out, item)
	}
	return out
}

func parseUserID(userID string) (uuid.UUID, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return uuid.Nil, apperrors.Unauthorized("AUTH_INVALID_TOKEN", "Invalid token")
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return uuid.Nil, apperrors.Unauthorized("AUTH_INVALID_TOKEN", "Invalid token")
	}

	return userUUID, nil
}

func normalizePagination(limit, offset int) (int, int, error) {
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	if offset < 0 {
		return 0, 0, apperrors.BadRequest("SUBMISSION_VALIDATION_ERROR", "offset must be zero or greater")
	}
	return limit, offset, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
