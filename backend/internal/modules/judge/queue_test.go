package judge

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
)

type mockRepository struct {
	created *Job
}

func (m *mockRepository) Create(_ context.Context, job *Job) error {
	copied := *job
	m.created = &copied
	return nil
}

func (m *mockRepository) ListQueued(_ context.Context, _ int) ([]Job, error) {
	return nil, nil
}

func (m *mockRepository) MarkRunning(_ context.Context, _ uuid.UUID) error { return nil }
func (m *mockRepository) MarkDone(_ context.Context, _ uuid.UUID) error    { return nil }
func (m *mockRepository) MarkFailed(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}
func (m *mockRepository) FinalizeExecution(_ context.Context, _ FinalizeExecutionInput) error {
	return nil
}

type mockWorkerQueue struct {
	jobs          []Job
	ignoreStatus  bool
	jobStatusMap  map[uuid.UUID]JobStatus
	jobErrors     map[uuid.UUID]string
	subStatusMap  map[uuid.UUID]string
	subPointsMap  map[uuid.UUID]int
	awardCountMap map[uuid.UUID]int
	chPoints      map[uuid.UUID]int
}

func newMockWorkerQueue(jobs []Job) *mockWorkerQueue {
	jobStatusMap := make(map[uuid.UUID]JobStatus, len(jobs))
	subStatusMap := make(map[uuid.UUID]string, len(jobs))
	subPointsMap := make(map[uuid.UUID]int, len(jobs))
	awardCountMap := make(map[uuid.UUID]int, len(jobs))
	for _, job := range jobs {
		jobStatusMap[job.ID] = job.Status
		subStatusMap[job.SubmissionID] = "pending"
		subPointsMap[job.SubmissionID] = 0
		awardCountMap[job.SubmissionID] = 0
	}
	return &mockWorkerQueue{
		jobs:          jobs,
		jobStatusMap:  jobStatusMap,
		jobErrors:     map[uuid.UUID]string{},
		subStatusMap:  subStatusMap,
		subPointsMap:  subPointsMap,
		awardCountMap: awardCountMap,
		chPoints:      map[uuid.UUID]int{},
	}
}

func (m *mockWorkerQueue) FetchPending(_ context.Context, limit int) ([]Job, error) {
	if limit <= 0 {
		limit = 1
	}
	result := make([]Job, 0, limit)
	for _, job := range m.jobs {
		if !m.ignoreStatus && m.jobStatusMap[job.ID] != JobStatusQueued {
			continue
		}
		result = append(result, job)
		if len(result) == limit {
			break
		}
	}
	return result, nil
}

func (m *mockWorkerQueue) MarkRunning(_ context.Context, id uuid.UUID) error {
	m.jobStatusMap[id] = JobStatusRunning
	return nil
}

func (m *mockWorkerQueue) FinalizeExecution(_ context.Context, input FinalizeExecutionInput) error {
	if m.subStatusMap[input.SubmissionID] == "pending" {
		points := 0
		if input.SubmissionStatus == "correct" {
			if input.AwardedPoints != nil {
				points = *input.AwardedPoints
			} else {
				points = m.chPoints[input.ChallengeID]
			}
		}
		m.subStatusMap[input.SubmissionID] = input.SubmissionStatus
		m.subPointsMap[input.SubmissionID] = points
		m.awardCountMap[input.SubmissionID]++
	}
	m.jobStatusMap[input.JobID] = input.JobStatus
	m.jobErrors[input.JobID] = input.JobErrorMessage
	return nil
}

func TestDBQueueEnqueueCreatesQueuedJob(t *testing.T) {
	repo := &mockRepository{}
	queue := NewQueue(repo)

	input := EnqueueInput{
		SubmissionID: uuid.New(),
		UserID:       uuid.New(),
		ChallengeID:  uuid.New(),
	}

	job, err := queue.Enqueue(context.Background(), input)
	if err != nil {
		t.Fatalf("expected enqueue success, got error: %v", err)
	}
	if repo.created == nil {
		t.Fatal("expected repository create to be called")
	}
	if job.Status != JobStatusQueued {
		t.Fatalf("expected queued status, got %s", job.Status)
	}
	if repo.created.Status != JobStatusQueued {
		t.Fatalf("expected stored queued status, got %s", repo.created.Status)
	}
	if repo.created.SubmissionID != input.SubmissionID {
		t.Fatal("expected submission id to be stored")
	}
}

func TestWorkerProcessOnceWritesBackSubmissionAndJobLifecycle(t *testing.T) {
	correctJob := Job{
		ID:           uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		SubmissionID: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		ChallengeID:  uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1"),
		Status:       JobStatusQueued,
	}
	wrongJob := Job{
		ID:           uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		SubmissionID: uuid.MustParse("22222222-2222-2222-2222-22222222222d"),
		ChallengeID:  uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbb2"),
		Status:       JobStatusQueued,
	}
	failJob := Job{
		ID:           uuid.MustParse("00000000-0000-0000-0000-000000000003"),
		SubmissionID: uuid.MustParse("33333333-3333-3333-3333-33333333333e"),
		ChallengeID:  uuid.MustParse("cccccccc-cccc-cccc-cccc-ccccccccccc3"),
		Status:       JobStatusQueued,
	}

	queue := newMockWorkerQueue([]Job{correctJob, wrongJob, failJob})
	queue.chPoints[correctJob.ChallengeID] = 500
	worker := NewWorker(queue, NewMockExecutor())

	processed, err := worker.ProcessOnce(context.Background(), 10)
	if err != nil {
		t.Fatalf("expected process success, got error: %v", err)
	}
	if processed != 3 {
		t.Fatalf("expected processed count 3, got %d", processed)
	}

	if queue.jobStatusMap[correctJob.ID] != JobStatusDone {
		t.Fatalf("expected correct job done, got %s", queue.jobStatusMap[correctJob.ID])
	}
	if queue.subStatusMap[correctJob.SubmissionID] != "correct" {
		t.Fatalf("expected correct submission status, got %s", queue.subStatusMap[correctJob.SubmissionID])
	}
	if queue.subPointsMap[correctJob.SubmissionID] != 500 {
		t.Fatalf("expected correct awarded points 500, got %d", queue.subPointsMap[correctJob.SubmissionID])
	}

	if queue.jobStatusMap[wrongJob.ID] != JobStatusDone {
		t.Fatalf("expected wrong job done, got %s", queue.jobStatusMap[wrongJob.ID])
	}
	if queue.subStatusMap[wrongJob.SubmissionID] != "wrong" {
		t.Fatalf("expected wrong submission status, got %s", queue.subStatusMap[wrongJob.SubmissionID])
	}
	if queue.subPointsMap[wrongJob.SubmissionID] != 0 {
		t.Fatalf("expected wrong awarded points 0, got %d", queue.subPointsMap[wrongJob.SubmissionID])
	}

	if queue.jobStatusMap[failJob.ID] != JobStatusFailed {
		t.Fatalf("expected failed job status, got %s", queue.jobStatusMap[failJob.ID])
	}
	if queue.subStatusMap[failJob.SubmissionID] != "failed" {
		t.Fatalf("expected failed submission status, got %s", queue.subStatusMap[failJob.SubmissionID])
	}
	if queue.subPointsMap[failJob.SubmissionID] != 0 {
		t.Fatalf("expected failed awarded points 0, got %d", queue.subPointsMap[failJob.SubmissionID])
	}
	if queue.jobErrors[failJob.ID] == "" {
		t.Fatal("expected failed job error message")
	}
}

func TestWorkerProcessOnceIdempotentSubmissionWriteback(t *testing.T) {
	job := Job{
		ID:           uuid.MustParse("00000000-0000-0000-0000-000000000004"),
		SubmissionID: uuid.MustParse("44444444-4444-4444-4444-444444444444"),
		ChallengeID:  uuid.MustParse("dddddddd-dddd-dddd-dddd-ddddddddddd4"),
		Status:       JobStatusQueued,
	}

	queue := newMockWorkerQueue([]Job{job})
	queue.ignoreStatus = true
	queue.chPoints[job.ChallengeID] = 300
	worker := NewWorker(queue, NewMockExecutor())

	if _, err := worker.ProcessOnce(context.Background(), 1); err != nil {
		t.Fatalf("first process failed: %v", err)
	}
	if _, err := worker.ProcessOnce(context.Background(), 1); err != nil {
		t.Fatalf("second process failed: %v", err)
	}

	if queue.subStatusMap[job.SubmissionID] != "correct" {
		t.Fatalf("expected submission status correct, got %s", queue.subStatusMap[job.SubmissionID])
	}
	if queue.subPointsMap[job.SubmissionID] != 300 {
		t.Fatalf("expected awarded points 300, got %d", queue.subPointsMap[job.SubmissionID])
	}
	if queue.awardCountMap[job.SubmissionID] != 1 {
		t.Fatalf("expected award applied once, got %d", queue.awardCountMap[job.SubmissionID])
	}
}

func TestMockExecutorDeterministicVerdicts(t *testing.T) {
	exec := NewMockExecutor()
	cases := []struct {
		name         string
		submissionID string
		wantVerdict  Verdict
		wantErr      bool
	}{
		{name: "correct", submissionID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1", wantVerdict: VerdictCorrect},
		{name: "wrong", submissionID: "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbd", wantVerdict: VerdictWrong},
		{name: "failure", submissionID: "cccccccc-cccc-cccc-cccc-ccccccccccce", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			job := Job{SubmissionID: uuid.MustParse(tc.submissionID)}
			result, err := exec.Execute(context.Background(), job)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Verdict != tc.wantVerdict {
				t.Fatalf("expected verdict %s, got %s", tc.wantVerdict, result.Verdict)
			}
		})
	}
}

var _ Repository = (*mockRepository)(nil)
var _ WorkerQueue = (*mockWorkerQueue)(nil)

func TestWorkerReturnsMarkRunningError(t *testing.T) {
	job := Job{ID: uuid.New(), SubmissionID: uuid.New(), Status: JobStatusQueued}
	queue := &markRunningErrQueue{job: job}
	worker := NewWorker(queue, NewMockExecutor())

	processed, err := worker.ProcessOnce(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error")
	}
	if processed != 0 {
		t.Fatalf("expected processed 0, got %d", processed)
	}
}

type markRunningErrQueue struct{ job Job }

func (m *markRunningErrQueue) FetchPending(_ context.Context, _ int) ([]Job, error) {
	return []Job{m.job}, nil
}
func (m *markRunningErrQueue) MarkRunning(_ context.Context, _ uuid.UUID) error {
	return fmt.Errorf("mark running failed")
}
func (m *markRunningErrQueue) FinalizeExecution(_ context.Context, _ FinalizeExecutionInput) error {
	return nil
}

var _ WorkerQueue = (*markRunningErrQueue)(nil)
