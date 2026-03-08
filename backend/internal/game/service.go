package game

import "context"

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

	correct := submittedFlag == flagValue
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
