package game

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Announcements(ctx context.Context) ([]Announcement, error) {
	return s.repo.ListAnnouncements(ctx)
}

func (s *Service) Challenge(ctx context.Context, challengeRef string) (Challenge, error) {
	challenge, _, err := s.repo.GetChallenge(ctx, challengeRef)
	return challenge, err
}

func (s *Service) Attachment(ctx context.Context, challengeRef string, attachmentID int64) (Attachment, string, error) {
	return s.repo.GetChallengeAttachment(ctx, challengeRef, attachmentID)
}

func (s *Service) UserSubmissions(ctx context.Context, userID int64) ([]UserSubmission, error) {
	return s.repo.ListUserSubmissions(ctx, userID)
}

func (s *Service) UserSolves(ctx context.Context, userID int64) ([]UserSolve, error) {
	return s.repo.ListUserSolves(ctx, userID)
}

func (s *Service) SubmitFlag(ctx context.Context, userID int64, challengeRef, submittedFlag, sourceIP string) (SubmitResult, error) {
	challenge, flagValue, err := s.repo.GetChallenge(ctx, challengeRef)
	if err != nil {
		return SubmitResult{}, err
	}

	correct, err := evaluateFlag(challenge.FlagType, flagValue, submittedFlag)
	if err != nil {
		return SubmitResult{}, err
	}
	submissionID, _, err := s.repo.CreateSubmission(ctx, challenge.ID, userID, submittedFlag, correct, sourceIP)
	if err != nil {
		return SubmitResult{}, err
	}

	result := SubmitResult{
		SubmissionID: submissionID,
		Correct:      correct,
	}
	if !correct {
		result.Message = "incorrect flag"
		return result, nil
	}

	solved, err := s.repo.HasSolved(ctx, challenge.ID, userID)
	if err != nil {
		return SubmitResult{}, err
	}
	if solved {
		result.Solved = false
		result.Message = "flag accepted, challenge already solved"
		return result, nil
	}

	solvedAt, err := s.repo.CreateSolve(ctx, challenge.ID, userID, submissionID, challenge.Points)
	if err != nil {
		return SubmitResult{}, err
	}
	result.Solved = true
	result.Message = "flag accepted"
	result.AwardedPoints = challenge.Points
	result.SolvedAt = &solvedAt
	return result, nil
}

func (s *Service) Scoreboard(ctx context.Context) ([]ScoreboardEntry, error) {
	entries, err := s.repo.ListScoreboard(ctx)
	if err != nil {
		return nil, err
	}
	for i := range entries {
		entries[i].Rank = i + 1
	}
	return entries, nil
}

func ValidateFlagTypeConfig(flagType, expected string) (string, error) {
	normalized := normalizeFlagType(flagType)
	switch normalized {
	case FlagTypeStatic, FlagTypeCaseInsensitive:
		return normalized, nil
	case FlagTypeRegex:
		if _, err := compileFlagRegex(expected); err != nil {
			return "", err
		}
		return normalized, nil
	default:
		return "", fmt.Errorf("%w: unsupported flag_type %q", ErrInvalidFlagStrategy, flagType)
	}
}

func evaluateFlag(flagType, expected, submitted string) (bool, error) {
	normalized, err := ValidateFlagTypeConfig(flagType, expected)
	if err != nil {
		return false, err
	}

	switch normalized {
	case FlagTypeStatic:
		return submitted == expected, nil
	case FlagTypeCaseInsensitive:
		return strings.EqualFold(strings.TrimSpace(submitted), strings.TrimSpace(expected)), nil
	case FlagTypeRegex:
		re, err := compileFlagRegex(expected)
		if err != nil {
			return false, err
		}
		return re.MatchString(submitted), nil
	default:
		return false, fmt.Errorf("%w: unsupported flag_type %q", ErrInvalidFlagStrategy, flagType)
	}
}

func normalizeFlagType(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return FlagTypeStatic
	}
	return normalized
}

func compileFlagRegex(pattern string) (*regexp.Regexp, error) {
	re, err := regexp.Compile(strings.TrimSpace(pattern))
	if err != nil {
		return nil, fmt.Errorf("%w: invalid regex pattern", ErrInvalidFlagStrategy)
	}
	return re, nil
}
