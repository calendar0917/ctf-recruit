package admin

import (
	"context"
	"testing"
	"time"
)

type fakeRepo struct{}

func (r *fakeRepo) ListChallenges(context.Context) ([]ChallengeSummary, error) {
	return []ChallengeSummary{{ID: 1, Slug: "web-welcome"}}, nil
}
func (r *fakeRepo) GetChallenge(context.Context, int64) (ChallengeDetail, error) {
	return ChallengeDetail{ID: 1, Slug: "web-welcome", RuntimeConfig: RuntimeConfig{Enabled: true, ImageName: "ctf/web-welcome:dev"}}, nil
}
func (r *fakeRepo) CreateChallenge(context.Context, UpsertChallengeInput) (ChallengeSummary, error) {
	return ChallengeSummary{ID: 2, Slug: "new-challenge"}, nil
}
func (r *fakeRepo) UpdateChallenge(context.Context, int64, UpsertChallengeInput) (ChallengeSummary, error) {
	return ChallengeSummary{ID: 1, Slug: "web-welcome"}, nil
}
func (r *fakeRepo) ListAnnouncements(context.Context) ([]Announcement, error) {
	return []Announcement{{ID: 1, Title: "hello"}}, nil
}
func (r *fakeRepo) CreateAnnouncement(context.Context, int64, CreateAnnouncementInput) (Announcement, error) {
	return Announcement{ID: 2, Title: "new"}, nil
}
func (r *fakeRepo) ListSubmissions(context.Context) ([]SubmissionRecord, error) {
	return []SubmissionRecord{{ID: 1}}, nil
}
func (r *fakeRepo) ListInstances(context.Context) ([]InstanceRecord, error) {
	return []InstanceRecord{{ID: 1}}, nil
}
func (r *fakeRepo) TerminateInstance(context.Context, int64, time.Time) (InstanceRecord, error) {
	return InstanceRecord{ID: 1, Status: "terminated"}, nil
}

func TestChallenge(t *testing.T) {
	service := NewService(&fakeRepo{})
	challenge, err := service.Challenge(context.Background(), 1)
	if err != nil {
		t.Fatalf("challenge: %v", err)
	}
	if challenge.RuntimeConfig.ImageName != "ctf/web-welcome:dev" {
		t.Fatalf("unexpected challenge: %+v", challenge)
	}
}

func TestTerminateInstance(t *testing.T) {
	service := NewService(&fakeRepo{})
	instance, err := service.TerminateInstance(context.Background(), 1)
	if err != nil {
		t.Fatalf("terminate instance: %v", err)
	}
	if instance.Status != "terminated" {
		t.Fatalf("unexpected status: %s", instance.Status)
	}
}
