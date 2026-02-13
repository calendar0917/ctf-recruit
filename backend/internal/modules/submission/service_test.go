package submission

import (
	"context"
	apperrors "ctf-recruit/backend/internal/errors"
	"ctf-recruit/backend/internal/modules/auth"
	"ctf-recruit/backend/internal/modules/challenge"
	"ctf-recruit/backend/internal/modules/judge"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

type mockRepository struct {
	submissions []Submission
	awarded     map[string]struct{}
}

func newMockRepository() *mockRepository {
	return &mockRepository{submissions: []Submission{}, awarded: map[string]struct{}{}}
}

func (m *mockRepository) Create(_ context.Context, submission *Submission) error {
	if submission.AwardedPoints > 0 {
		key := submission.UserID.String() + ":" + submission.ChallengeID.String()
		if _, exists := m.awarded[key]; exists {
			return &pgconn.PgError{Code: "23505"}
		}
		m.awarded[key] = struct{}{}
	}

	copied := *submission
	m.submissions = append(m.submissions, copied)
	return nil
}

func (m *mockRepository) FinalizePending(_ context.Context, _ uuid.UUID, _ Status, _ int) (bool, error) {
	return false, nil
}

type mockChallengeReader struct {
	ch *challenge.Challenge
}

func (m mockChallengeReader) GetForSubmission(_ context.Context, id string, publishedOnly bool) (*challenge.Challenge, error) {
	if m.ch == nil {
		return nil, nil
	}
	if id != m.ch.ID.String() {
		return nil, nil
	}
	if publishedOnly && !m.ch.IsPublished {
		return nil, apperrors.NotFound("CHALLENGE_NOT_FOUND", "Challenge not found")
	}
	copied := *m.ch
	return &copied, nil
}

type mockQueue struct {
	enqueued []judge.EnqueueInput
	jobs     []judge.Job
}

func newMockQueue() *mockQueue {
	return &mockQueue{enqueued: []judge.EnqueueInput{}, jobs: []judge.Job{}}
}

func (m *mockQueue) Enqueue(_ context.Context, input judge.EnqueueInput) (*judge.Job, error) {
	m.enqueued = append(m.enqueued, input)
	job := judge.Job{ID: uuid.New(), SubmissionID: input.SubmissionID, Status: judge.JobStatusQueued}
	m.jobs = append(m.jobs, job)
	return &job, nil
}

func (m *mockQueue) FetchPending(_ context.Context, limit int) ([]judge.Job, error) {
	if limit <= 0 || limit > len(m.jobs) {
		limit = len(m.jobs)
	}
	result := make([]judge.Job, 0, limit)
	for i := 0; i < limit; i++ {
		result = append(result, m.jobs[i])
	}
	return result, nil
}

func (m *mockQueue) MarkRunning(_ context.Context, _ uuid.UUID) error { return nil }
func (m *mockQueue) MarkDone(_ context.Context, _ uuid.UUID) error    { return nil }
func (m *mockQueue) MarkFailed(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}

func TestServiceSubmitCorrectFlagAwardsPointsOnce(t *testing.T) {
	repo := newMockRepository()
	ch := &challenge.Challenge{
		ID:          uuid.New(),
		Points:      100,
		Mode:        challenge.ModeStatic,
		FlagHash:    hashFlag("flag{ok}"),
		IsPublished: true,
	}
	svc := NewService(repo, mockChallengeReader{ch: ch}, nil)

	resp, err := svc.Submit(context.Background(), uuid.New().String(), auth.RolePlayer, CreateSubmissionRequest{
		ChallengeID: ch.ID.String(),
		Flag:        "flag{ok}",
	})
	if err != nil {
		t.Fatalf("expected submit success, got error: %v", err)
	}
	if resp.Status != StatusCorrect {
		t.Fatalf("expected status correct, got %s", resp.Status)
	}
	if resp.AwardedPoints != 100 {
		t.Fatalf("expected awarded points 100, got %d", resp.AwardedPoints)
	}
	if resp.JudgeJobID != nil {
		t.Fatal("expected no judge job id for static challenge")
	}

	if len(repo.submissions) != 1 {
		t.Fatalf("expected 1 stored submission, got %d", len(repo.submissions))
	}
	if repo.submissions[0].AwardedPoints != 100 {
		t.Fatalf("expected stored awarded points 100, got %d", repo.submissions[0].AwardedPoints)
	}
}

func TestServiceSubmitWrongFlagStoresWrongWithZeroPoints(t *testing.T) {
	repo := newMockRepository()
	ch := &challenge.Challenge{
		ID:          uuid.New(),
		Points:      150,
		Mode:        challenge.ModeStatic,
		FlagHash:    hashFlag("flag{real}"),
		IsPublished: true,
	}
	svc := NewService(repo, mockChallengeReader{ch: ch}, nil)

	resp, err := svc.Submit(context.Background(), uuid.New().String(), auth.RolePlayer, CreateSubmissionRequest{
		ChallengeID: ch.ID.String(),
		Flag:        "flag{fake}",
	})
	if err != nil {
		t.Fatalf("expected submit success, got error: %v", err)
	}
	if resp.Status != StatusWrong {
		t.Fatalf("expected status wrong, got %s", resp.Status)
	}
	if resp.AwardedPoints != 0 {
		t.Fatalf("expected awarded points 0, got %d", resp.AwardedPoints)
	}
}

func TestServiceSubmitDuplicateCorrectDoesNotAwardTwice(t *testing.T) {
	repo := newMockRepository()
	ch := &challenge.Challenge{
		ID:          uuid.New(),
		Points:      100,
		Mode:        challenge.ModeStatic,
		FlagHash:    hashFlag("flag{ok}"),
		IsPublished: true,
	}
	svc := NewService(repo, mockChallengeReader{ch: ch}, nil)
	userID := uuid.New().String()

	resp1, err := svc.Submit(context.Background(), userID, auth.RolePlayer, CreateSubmissionRequest{
		ChallengeID: ch.ID.String(),
		Flag:        "flag{ok}",
	})
	if err != nil {
		t.Fatalf("expected first submit success, got error: %v", err)
	}
	if resp1.AwardedPoints != 100 {
		t.Fatalf("expected first awarded points 100, got %d", resp1.AwardedPoints)
	}

	resp2, err := svc.Submit(context.Background(), userID, auth.RolePlayer, CreateSubmissionRequest{
		ChallengeID: ch.ID.String(),
		Flag:        "flag{ok}",
	})
	if err != nil {
		t.Fatalf("expected second submit success, got error: %v", err)
	}
	if resp2.Status != StatusCorrect {
		t.Fatalf("expected second status correct, got %s", resp2.Status)
	}
	if resp2.AwardedPoints != 0 {
		t.Fatalf("expected second awarded points 0, got %d", resp2.AwardedPoints)
	}

	awardedCount := 0
	for _, s := range repo.submissions {
		if s.AwardedPoints > 0 {
			awardedCount++
		}
	}
	if awardedCount != 1 {
		t.Fatalf("expected exactly 1 awarded submission, got %d", awardedCount)
	}
	if len(repo.submissions) != 2 {
		t.Fatalf("expected 2 stored submissions, got %d", len(repo.submissions))
	}
}

func TestServiceSubmitDynamicChallengeEnqueuesJudgeJob(t *testing.T) {
	repo := newMockRepository()
	queue := newMockQueue()
	ch := &challenge.Challenge{
		ID:          uuid.New(),
		Points:      250,
		Mode:        challenge.ModeDynamic,
		FlagHash:    hashFlag("unused-for-dynamic"),
		IsPublished: true,
	}
	svc := NewService(repo, mockChallengeReader{ch: ch}, queue)

	resp, err := svc.Submit(context.Background(), uuid.New().String(), auth.RolePlayer, CreateSubmissionRequest{
		ChallengeID: ch.ID.String(),
		Flag:        "any-input",
	})
	if err != nil {
		t.Fatalf("expected submit success, got error: %v", err)
	}
	if resp.Status != StatusPending {
		t.Fatalf("expected status pending, got %s", resp.Status)
	}
	if resp.AwardedPoints != 0 {
		t.Fatalf("expected awarded points 0, got %d", resp.AwardedPoints)
	}
	if resp.JudgeJobID == nil || *resp.JudgeJobID == "" {
		t.Fatal("expected judgeJobId to be returned")
	}
	if len(queue.enqueued) != 1 {
		t.Fatalf("expected 1 enqueued judge job, got %d", len(queue.enqueued))
	}
}

func TestServiceSubmitPlayerCannotSubmitToUnpublishedChallenge(t *testing.T) {
	repo := newMockRepository()
	ch := &challenge.Challenge{
		ID:          uuid.New(),
		Points:      100,
		Mode:        challenge.ModeStatic,
		FlagHash:    hashFlag("flag{ok}"),
		IsPublished: false,
	}
	svc := NewService(repo, mockChallengeReader{ch: ch}, nil)

	_, err := svc.Submit(context.Background(), uuid.New().String(), auth.RolePlayer, CreateSubmissionRequest{
		ChallengeID: ch.ID.String(),
		Flag:        "flag{ok}",
	})
	if err == nil {
		t.Fatal("expected error")
	}

	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected app error, got %T", err)
	}
	if appErr.Code != "CHALLENGE_NOT_FOUND" {
		t.Fatalf("expected CHALLENGE_NOT_FOUND, got %s", appErr.Code)
	}
}
