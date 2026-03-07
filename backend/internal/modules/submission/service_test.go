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

func (m *mockRepository) ListByUser(_ context.Context, userID uuid.UUID, limit, offset int) ([]Submission, error) {
	items := make([]Submission, 0)
	for i := len(m.submissions) - 1; i >= 0; i-- {
		if m.submissions[i].UserID == userID {
			items = append(items, m.submissions[i])
		}
	}
	if offset >= len(items) {
		return []Submission{}, nil
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	return items[offset:end], nil
}

func (m *mockRepository) ListByUserAndChallenge(_ context.Context, userID, challengeID uuid.UUID, limit, offset int) ([]Submission, error) {
	items := make([]Submission, 0)
	for i := len(m.submissions) - 1; i >= 0; i-- {
		s := m.submissions[i]
		if s.UserID == userID && s.ChallengeID == challengeID {
			items = append(items, s)
		}
	}
	if offset >= len(items) {
		return []Submission{}, nil
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	return items[offset:end], nil
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
	enqErr   error
}

func newMockQueue() *mockQueue {
	return &mockQueue{enqueued: []judge.EnqueueInput{}, jobs: []judge.Job{}}
}

func (m *mockQueue) Enqueue(_ context.Context, input judge.EnqueueInput) (*judge.Job, error) {
	if m.enqErr != nil {
		return nil, m.enqErr
	}
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

func TestServiceSubmitCorrectSubmissionUpdatesScoreboardOnce(t *testing.T) {
	repo := newMockRepository()
	ch := &challenge.Challenge{
		ID:          uuid.New(),
		Points:      150,
		Mode:        challenge.ModeStatic,
		FlagHash:    hashFlag("flag{score}"),
		IsPublished: true,
	}
	svc := NewService(repo, mockChallengeReader{ch: ch}, nil)
	userID := uuid.New().String()

	if _, err := svc.Submit(context.Background(), userID, auth.RolePlayer, CreateSubmissionRequest{
		ChallengeID: ch.ID.String(),
		Flag:        "flag{score}",
	}); err != nil {
		t.Fatalf("expected submit success, got error: %v", err)
	}

	if score := calculateUserScore(repo.submissions, userID); score != 150 {
		t.Fatalf("expected scoreboard total 150, got %d", score)
	}
}

func TestServiceSubmitDuplicateCorrectSubmissionKeepsScoreboardStable(t *testing.T) {
	repo := newMockRepository()
	ch := &challenge.Challenge{
		ID:          uuid.New(),
		Points:      220,
		Mode:        challenge.ModeStatic,
		FlagHash:    hashFlag("flag{stable}"),
		IsPublished: true,
	}
	svc := NewService(repo, mockChallengeReader{ch: ch}, nil)
	userID := uuid.New().String()

	if _, err := svc.Submit(context.Background(), userID, auth.RolePlayer, CreateSubmissionRequest{
		ChallengeID: ch.ID.String(),
		Flag:        "flag{stable}",
	}); err != nil {
		t.Fatalf("expected first submit success, got error: %v", err)
	}
	if _, err := svc.Submit(context.Background(), userID, auth.RolePlayer, CreateSubmissionRequest{
		ChallengeID: ch.ID.String(),
		Flag:        "flag{stable}",
	}); err != nil {
		t.Fatalf("expected duplicate submit success, got error: %v", err)
	}

	if score := calculateUserScore(repo.submissions, userID); score != 220 {
		t.Fatalf("expected scoreboard total to stay 220, got %d", score)
	}
}

func TestServiceSubmitFlagRulesTrimWhitespaceAndKeepCaseSensitive(t *testing.T) {
	repo := newMockRepository()
	ch := &challenge.Challenge{
		ID:          uuid.New(),
		Points:      80,
		Mode:        challenge.ModeStatic,
		FlagHash:    hashFlag("Flag{CaseSensitive}"),
		IsPublished: true,
	}
	svc := NewService(repo, mockChallengeReader{ch: ch}, nil)
	userID := uuid.New().String()

	respTrimmed, err := svc.Submit(context.Background(), userID, auth.RolePlayer, CreateSubmissionRequest{
		ChallengeID: ch.ID.String(),
		Flag:        "  Flag{CaseSensitive}  ",
	})
	if err != nil {
		t.Fatalf("expected trimmed submit success, got error: %v", err)
	}
	if respTrimmed.Status != StatusCorrect {
		t.Fatalf("expected trimmed status correct, got %s", respTrimmed.Status)
	}

	respCase, err := svc.Submit(context.Background(), uuid.New().String(), auth.RolePlayer, CreateSubmissionRequest{
		ChallengeID: ch.ID.String(),
		Flag:        "flag{casesensitive}",
	})
	if err != nil {
		t.Fatalf("expected case submit success, got error: %v", err)
	}
	if respCase.Status != StatusWrong {
		t.Fatalf("expected case-mismatch status wrong, got %s", respCase.Status)
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

func calculateUserScore(subs []Submission, userID string) int {
	total := 0
	for _, s := range subs {
		if s.UserID.String() == userID {
			total += s.AwardedPoints
		}
	}
	return total
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

func TestServiceSubmitDynamicChallengeQueueUnavailableReturnsInternalError(t *testing.T) {
	repo := newMockRepository()
	ch := &challenge.Challenge{
		ID:          uuid.New(),
		Points:      250,
		Mode:        challenge.ModeDynamic,
		FlagHash:    hashFlag("unused-for-dynamic"),
		IsPublished: true,
	}
	svc := NewService(repo, mockChallengeReader{ch: ch}, nil)

	_, err := svc.Submit(context.Background(), uuid.New().String(), auth.RolePlayer, CreateSubmissionRequest{
		ChallengeID: ch.ID.String(),
		Flag:        "any-input",
	})
	if err == nil {
		t.Fatal("expected error")
	}

	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected app error, got %T", err)
	}
	if appErr.Code != "JUDGE_QUEUE_UNAVAILABLE" {
		t.Fatalf("expected JUDGE_QUEUE_UNAVAILABLE, got %s", appErr.Code)
	}
	if len(repo.submissions) != 1 {
		t.Fatalf("expected pending submission persisted, got %d", len(repo.submissions))
	}
	if repo.submissions[0].Status != StatusPending {
		t.Fatalf("expected persisted submission status pending, got %s", repo.submissions[0].Status)
	}
}

func TestServiceSubmitDynamicChallengeEnqueueFailureReturnsInternalError(t *testing.T) {
	repo := newMockRepository()
	queue := newMockQueue()
	queue.enqErr = errors.New("queue down")
	ch := &challenge.Challenge{
		ID:          uuid.New(),
		Points:      250,
		Mode:        challenge.ModeDynamic,
		FlagHash:    hashFlag("unused-for-dynamic"),
		IsPublished: true,
	}
	svc := NewService(repo, mockChallengeReader{ch: ch}, queue)

	_, err := svc.Submit(context.Background(), uuid.New().String(), auth.RolePlayer, CreateSubmissionRequest{
		ChallengeID: ch.ID.String(),
		Flag:        "any-input",
	})
	if err == nil {
		t.Fatal("expected error")
	}

	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected app error, got %T", err)
	}
	if appErr.Code != "JUDGE_JOB_ENQUEUE_FAILED" {
		t.Fatalf("expected JUDGE_JOB_ENQUEUE_FAILED, got %s", appErr.Code)
	}
	if len(repo.submissions) != 1 {
		t.Fatalf("expected pending submission persisted, got %d", len(repo.submissions))
	}
	if repo.submissions[0].Status != StatusPending {
		t.Fatalf("expected persisted submission status pending, got %s", repo.submissions[0].Status)
	}
}

func TestServiceListMineReturnsLatestFirstWithPagination(t *testing.T) {
	repo := newMockRepository()
	chA := &challenge.Challenge{ID: uuid.New(), Points: 100, Mode: challenge.ModeStatic, FlagHash: hashFlag("flag{a}"), IsPublished: true}
	chB := &challenge.Challenge{ID: uuid.New(), Points: 200, Mode: challenge.ModeStatic, FlagHash: hashFlag("flag{b}"), IsPublished: true}
	userID := uuid.New().String()

	svcA := NewService(repo, mockChallengeReader{ch: chA}, nil)
	if _, err := svcA.Submit(context.Background(), userID, auth.RolePlayer, CreateSubmissionRequest{ChallengeID: chA.ID.String(), Flag: "flag{a}"}); err != nil {
		t.Fatalf("seed submit A failed: %v", err)
	}

	svcB := NewService(repo, mockChallengeReader{ch: chB}, nil)
	if _, err := svcB.Submit(context.Background(), userID, auth.RolePlayer, CreateSubmissionRequest{ChallengeID: chB.ID.String(), Flag: "flag{wrong}"}); err != nil {
		t.Fatalf("seed submit B failed: %v", err)
	}

	resp, err := svcA.ListMine(context.Background(), userID, 1, 0)
	if err != nil {
		t.Fatalf("expected list success, got error: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 item due to limit, got %d", len(resp.Items))
	}
	if resp.Items[0].ChallengeID != chB.ID.String() {
		t.Fatalf("expected latest challenge %s, got %s", chB.ID.String(), resp.Items[0].ChallengeID)
	}
	if resp.Limit != 1 || resp.Offset != 0 {
		t.Fatalf("expected pagination limit=1 offset=0, got limit=%d offset=%d", resp.Limit, resp.Offset)
	}
}

func TestServiceListMineByChallengeFiltersChallengeAndValidatesInputs(t *testing.T) {
	repo := newMockRepository()
	ch := &challenge.Challenge{ID: uuid.New(), Points: 100, Mode: challenge.ModeStatic, FlagHash: hashFlag("flag{ok}"), IsPublished: true}
	other := &challenge.Challenge{ID: uuid.New(), Points: 100, Mode: challenge.ModeStatic, FlagHash: hashFlag("flag{other}"), IsPublished: true}
	userID := uuid.New().String()

	svc := NewService(repo, mockChallengeReader{ch: ch}, nil)
	if _, err := svc.Submit(context.Background(), userID, auth.RolePlayer, CreateSubmissionRequest{ChallengeID: ch.ID.String(), Flag: "flag{ok}"}); err != nil {
		t.Fatalf("seed submit target challenge failed: %v", err)
	}

	svcOther := NewService(repo, mockChallengeReader{ch: other}, nil)
	if _, err := svcOther.Submit(context.Background(), userID, auth.RolePlayer, CreateSubmissionRequest{ChallengeID: other.ID.String(), Flag: "flag{other}"}); err != nil {
		t.Fatalf("seed submit other challenge failed: %v", err)
	}

	resp, err := svc.ListMineByChallenge(context.Background(), userID, ch.ID.String(), 20, 0)
	if err != nil {
		t.Fatalf("expected challenge list success, got error: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 filtered item, got %d", len(resp.Items))
	}
	if resp.Items[0].ChallengeID != ch.ID.String() {
		t.Fatalf("expected challenge %s, got %s", ch.ID.String(), resp.Items[0].ChallengeID)
	}

	_, err = svc.ListMineByChallenge(context.Background(), userID, "invalid", 20, 0)
	if err == nil {
		t.Fatal("expected challenge id validation error")
	}
	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected app error, got %T", err)
	}
	if appErr.Code != "SUBMISSION_VALIDATION_ERROR" {
		t.Fatalf("expected SUBMISSION_VALIDATION_ERROR, got %s", appErr.Code)
	}

	_, err = svc.ListMine(context.Background(), userID, 20, -1)
	if err == nil {
		t.Fatal("expected offset validation error")
	}
	if !errors.As(err, &appErr) {
		t.Fatalf("expected app error, got %T", err)
	}
	if appErr.Code != "SUBMISSION_VALIDATION_ERROR" {
		t.Fatalf("expected SUBMISSION_VALIDATION_ERROR, got %s", appErr.Code)
	}
}
