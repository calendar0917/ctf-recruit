package game

import (
	"context"
	"testing"
	"time"
)

type fakeRepo struct {
	challenge          Challenge
	flag               string
	solved             bool
	announcements      []Announcement
	submissions        []UserSubmission
	solves             []UserSolve
	scoreboard         []ScoreboardEntry
	attachment         Attachment
	attachmentPath     string
	attachmentVisible  bool
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

func (r *fakeRepo) GetChallengeAttachment(_ context.Context, challengeRef string, attachmentID int64) (Attachment, string, error) {
	if challengeRef != r.challenge.Slug && challengeRef != "1" {
		return Attachment{}, "", ErrChallengeNotFound
	}
	if !r.attachmentVisible || attachmentID != r.attachment.ID {
		return Attachment{}, "", ErrAttachmentNotFound
	}
	return r.attachment, r.attachmentPath, nil
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

func (r *fakeRepo) ListUserSubmissions(context.Context, int64) ([]UserSubmission, error) {
	return r.submissions, nil
}

func (r *fakeRepo) ListUserSolves(context.Context, int64) ([]UserSolve, error) {
	return r.solves, nil
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

func TestAttachmentReturnsVisibleAttachment(t *testing.T) {
	service := NewService(&fakeRepo{
		challenge:         Challenge{ID: 1, Slug: "web-welcome", Points: 100},
		attachment:        Attachment{ID: 2, Filename: "statement.pdf"},
		attachmentPath:    "/tmp/statement.pdf",
		attachmentVisible: true,
	})

	attachment, path, err := service.Attachment(context.Background(), "web-welcome", 2)
	if err != nil {
		t.Fatalf("attachment: %v", err)
	}
	if attachment.Filename != "statement.pdf" || path != "/tmp/statement.pdf" {
		t.Fatalf("unexpected attachment result: %+v %s", attachment, path)
	}
}

func TestAttachmentRejectsHiddenAttachment(t *testing.T) {
	service := NewService(&fakeRepo{
		challenge:         Challenge{ID: 1, Slug: "web-welcome", Points: 100},
		attachment:        Attachment{ID: 2, Filename: "statement.pdf"},
		attachmentPath:    "/tmp/statement.pdf",
		attachmentVisible: false,
	})

	_, _, err := service.Attachment(context.Background(), "web-welcome", 2)
	if err != ErrAttachmentNotFound {
		t.Fatalf("expected attachment not found, got %v", err)
	}
}

func TestUserHistoryMethodsReturnRepositoryData(t *testing.T) {
	now := time.Date(2025, time.March, 8, 10, 0, 0, 0, time.UTC)
	repo := &fakeRepo{
		submissions: []UserSubmission{{ID: 1, ChallengeID: 2, ChallengeSlug: "misc-1", SubmittedAt: now}},
		solves:      []UserSolve{{ID: 3, ChallengeID: 2, ChallengeSlug: "misc-1", SolvedAt: now, AwardedPoints: 200}},
	}
	service := NewService(repo)

	submissions, err := service.UserSubmissions(context.Background(), 7)
	if err != nil {
		t.Fatalf("user submissions: %v", err)
	}
	if len(submissions) != 1 || submissions[0].ID != 1 {
		t.Fatalf("unexpected submissions: %+v", submissions)
	}

	solves, err := service.UserSolves(context.Background(), 7)
	if err != nil {
		t.Fatalf("user solves: %v", err)
	}
	if len(solves) != 1 || solves[0].ID != 3 || solves[0].AwardedPoints != 200 {
		t.Fatalf("unexpected solves: %+v", solves)
	}
}

func TestScoreboardReturnsSolveDetailsAndRanks(t *testing.T) {
	now := time.Date(2025, time.March, 8, 10, 0, 0, 0, time.UTC)
	service := NewService(&fakeRepo{
		scoreboard: []ScoreboardEntry{{
			UserID:      7,
			Username:    "alice",
			DisplayName: "Alice",
			Score:       200,
			Solves:      []ScoreboardSolve{{ChallengeID: 2, ChallengeSlug: "cipher-note", ChallengeTitle: "Cipher Note", Category: "crypto", Difficulty: "hard", AwardedPoints: 200, SolvedAt: now}},
		}},
	})

	items, err := service.Scoreboard(context.Background())
	if err != nil {
		t.Fatalf("scoreboard: %v", err)
	}
	if len(items) != 1 || items[0].Rank != 1 {
		t.Fatalf("unexpected scoreboard: %+v", items)
	}
	if len(items[0].Solves) != 1 || items[0].Solves[0].Difficulty != "hard" {
		t.Fatalf("expected solve details, got %+v", items[0].Solves)
	}
}
