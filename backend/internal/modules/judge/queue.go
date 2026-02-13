package judge

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Queue interface {
	Enqueue(ctx context.Context, input EnqueueInput) (*Job, error)
	FetchPending(ctx context.Context, limit int) ([]Job, error)
	MarkRunning(ctx context.Context, id uuid.UUID) error
	MarkDone(ctx context.Context, id uuid.UUID) error
	MarkFailed(ctx context.Context, id uuid.UUID, message string) error
}

type WorkerQueue interface {
	FetchPending(ctx context.Context, limit int) ([]Job, error)
	MarkRunning(ctx context.Context, id uuid.UUID) error
	FinalizeExecution(ctx context.Context, input FinalizeExecutionInput) error
}

type FinalizeExecutionInput struct {
	JobID             uuid.UUID
	SubmissionID      uuid.UUID
	ChallengeID       uuid.UUID
	SubmissionStatus  string
	SubmissionMessage string
	AwardedPoints     *int
	JobStatus         JobStatus
	JobErrorMessage   string
}

type DBQueue struct {
	repo Repository
}

func NewQueue(repo Repository) *DBQueue {
	return &DBQueue{repo: repo}
}

func (q *DBQueue) Enqueue(ctx context.Context, input EnqueueInput) (*Job, error) {
	job := &Job{
		ID:           uuid.New(),
		SubmissionID: input.SubmissionID,
		UserID:       input.UserID,
		ChallengeID:  input.ChallengeID,
		Status:       JobStatusQueued,
		Attempts:     0,
		QueuedAt:     time.Now().UTC(),
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	if err := q.repo.Create(ctx, job); err != nil {
		return nil, err
	}
	return job, nil
}

func (q *DBQueue) FetchPending(ctx context.Context, limit int) ([]Job, error) {
	return q.repo.ListQueued(ctx, limit)
}

func (q *DBQueue) MarkRunning(ctx context.Context, id uuid.UUID) error {
	return q.repo.MarkRunning(ctx, id)
}

func (q *DBQueue) MarkDone(ctx context.Context, id uuid.UUID) error {
	return q.repo.MarkDone(ctx, id)
}

func (q *DBQueue) MarkFailed(ctx context.Context, id uuid.UUID, message string) error {
	if message == "" {
		message = "mock judge execution failed"
	}
	return q.repo.MarkFailed(ctx, id, message)
}

func (q *DBQueue) FinalizeExecution(ctx context.Context, input FinalizeExecutionInput) error {
	return q.repo.FinalizeExecution(ctx, input)
}

type Verdict string

const (
	VerdictCorrect Verdict = "correct"
	VerdictWrong   Verdict = "wrong"
)

type ExecutionResult struct {
	Verdict       Verdict
	AwardedPoints *int
	Message       string
}

type Executor interface {
	Execute(ctx context.Context, job Job) (ExecutionResult, error)
}

type MockExecutor struct{}

func NewMockExecutor() *MockExecutor {
	return &MockExecutor{}
}

func (e *MockExecutor) Execute(_ context.Context, job Job) (ExecutionResult, error) {
	if len(job.SubmissionID.String()) == 0 {
		return ExecutionResult{}, fmt.Errorf("invalid submission id")
	}

	last := job.SubmissionID.String()[len(job.SubmissionID.String())-1]
	if last == 'f' || last == 'F' || last == 'e' || last == 'E' {
		return ExecutionResult{}, fmt.Errorf("mock judge failure for submission %s", job.SubmissionID.String())
	}
	if last == 'd' || last == 'D' || last == 'b' || last == 'B' {
		return ExecutionResult{Verdict: VerdictWrong, Message: "mock wrong answer"}, nil
	}
	return ExecutionResult{Verdict: VerdictCorrect, Message: "mock accepted"}, nil
}

type Worker struct {
	queue WorkerQueue
	exec  Executor
}

func NewWorker(queue WorkerQueue, exec Executor) *Worker {
	return &Worker{queue: queue, exec: exec}
}

func (w *Worker) ProcessOnce(ctx context.Context, maxConcurrency int) (int, error) {
	if maxConcurrency <= 0 {
		maxConcurrency = 1
	}

	jobs, err := w.queue.FetchPending(ctx, maxConcurrency)
	if err != nil {
		return 0, err
	}

	processed := 0
	for _, job := range jobs {
		if err := w.queue.MarkRunning(ctx, job.ID); err != nil {
			return processed, err
		}
		result, execErr := w.exec.Execute(ctx, job)
		if execErr != nil {
			if finalizeErr := w.queue.FinalizeExecution(ctx, FinalizeExecutionInput{
				JobID:            job.ID,
				SubmissionID:     job.SubmissionID,
				ChallengeID:      job.ChallengeID,
				SubmissionStatus: "failed",
				AwardedPoints:    intPtr(0),
				JobStatus:        JobStatusFailed,
				JobErrorMessage:  execErr.Error(),
			}); finalizeErr != nil {
				return processed, finalizeErr
			}
			processed++
			continue
		}

		submissionStatus := "wrong"
		awardedPoints := intPtr(0)
		if result.Verdict == VerdictCorrect {
			submissionStatus = "correct"
			awardedPoints = result.AwardedPoints
		}

		if finalizeErr := w.queue.FinalizeExecution(ctx, FinalizeExecutionInput{
			JobID:             job.ID,
			SubmissionID:      job.SubmissionID,
			ChallengeID:       job.ChallengeID,
			SubmissionStatus:  submissionStatus,
			SubmissionMessage: result.Message,
			AwardedPoints:     awardedPoints,
			JobStatus:         JobStatusDone,
		}); finalizeErr != nil {
			return processed, finalizeErr
		}
		processed++
	}

	return processed, nil
}

func intPtr(v int) *int {
	return &v
}
