package admin

import (
	"context"
	"time"
)

type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo, now: time.Now}
}

func (s *Service) Challenges(ctx context.Context) ([]ChallengeSummary, error) {
	return s.repo.ListChallenges(ctx)
}

func (s *Service) Challenge(ctx context.Context, challengeID int64) (ChallengeDetail, error) {
	return s.repo.GetChallenge(ctx, challengeID)
}

func (s *Service) CreateChallenge(ctx context.Context, input UpsertChallengeInput) (ChallengeSummary, error) {
	return s.repo.CreateChallenge(ctx, input)
}

func (s *Service) UpdateChallenge(ctx context.Context, challengeID int64, input UpsertChallengeInput) (ChallengeSummary, error) {
	return s.repo.UpdateChallenge(ctx, challengeID, input)
}

func (s *Service) Announcements(ctx context.Context) ([]Announcement, error) {
	return s.repo.ListAnnouncements(ctx)
}

func (s *Service) CreateAnnouncement(ctx context.Context, actorUserID int64, input CreateAnnouncementInput) (Announcement, error) {
	return s.repo.CreateAnnouncement(ctx, actorUserID, input)
}

func (s *Service) Submissions(ctx context.Context) ([]SubmissionRecord, error) {
	return s.repo.ListSubmissions(ctx)
}

func (s *Service) Instances(ctx context.Context) ([]InstanceRecord, error) {
	return s.repo.ListInstances(ctx)
}

func (s *Service) TerminateInstance(ctx context.Context, instanceID int64) (InstanceRecord, error) {
	return s.repo.TerminateInstance(ctx, instanceID, s.now().UTC())
}
