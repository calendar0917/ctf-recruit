package game

import (
	"context"
	"testing"
	"time"
)

type fakeRepo struct {
	challenge     Challenge
	flag          string
	solved        bool
	announcements []Announcement
	scoreboard    []ScoreboardEntry
}

func (r *fakeRepo) ListAnnouncements(context.Context) ([]Announcement, error) {
	return r.announcements, nil
}

func (r *fakeRepo) GetChallenge(_ context.Context, challengeRef string) (Challenge, string, error) {
	if challengeRef != r.challenge.Slug && challengeRef != "1" {
		return Challenge{}, "", ErrChallengeNotFound
	}
	return r.challenge, r.flag, nil
}

func (r *fakeRepo) CreateSubmission(_ context.Context, _ int64, _ int64, _ string, _ bool, _ string) (int64, time.Time, error) {
	return 1, time.Now().UTC(), nil
}

func (r *fakeRepo) HasSolved(_ context.Context, _ int64, _ int64) (bool, error) {
	return r.solved, nil
}

func (r *fakeRepo) CreateSolve(_ context.Context, _ int64, _ int64, _ int64, _ int) (time.Time, error) {
	now := time.Now().UTC()
	return now, nil
}

func (r *fakeRepo) ListScoreboard(context.Context) ([]ScoreboardEntry, error) {
	return r.scoreboard, nil
}

func TestSubmitFlagCreatesSolveOnFirstCorrectSubmission(t *testing.T) {
	service := NewService(&fakeRepo{
		challenge: Challenge{ID: 1, Slug: "web-welcome", Points: 100},
		flag:      "flag{welcome}",
	})

	result, err := service.SubmitFlag(context.Background(), 7, "web-welcome", "flag{welcome}", "127.0.0.1")
	if err != nil {
		t.Fatalf("submit flag: %v", err)
	}
	if !result.Correct || !result.Solved || result.AwardedPoints != 100 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestSubmitFlagReturnsIncorrectForWrongFlag(t *testing.T) {
	service := NewService(&fakeRepo{
		challenge: Challenge{ID: 1, Slug: "web-welcome", Points: 100},
		flag:      "flag{welcome}",
	})

	result, err := service.SubmitFlag(context.Background(), 7, "web-welcome", "wrong", "127.0.0.1")
	if err != nil {
		t.Fatalf("submit flag: %v", err)
	}
	if result.Correct || result.Solved {
		t.Fatalf("unexpected result: %+v", result)
	}
}
