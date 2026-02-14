package judge

import (
	"time"

	"github.com/google/uuid"
)

type JobStatus string

const (
	JobStatusQueued  JobStatus = "queued"
	JobStatusRunning JobStatus = "running"
	JobStatusDone    JobStatus = "done"
	JobStatusFailed  JobStatus = "failed"
)

type Job struct {
	ID          uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	SubmissionID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:uq_judge_jobs_submission_id"`
	UserID      uuid.UUID `gorm:"type:uuid;not null;index"`
	ChallengeID uuid.UUID `gorm:"type:uuid;not null;index"`
	Status      JobStatus `gorm:"type:varchar(20);not null;index"`
	Attempts    int       `gorm:"not null;default:0"`
	ErrorMessage string   `gorm:"type:text"`
	QueuedAt    time.Time `gorm:"not null;default:now()"`
	StartedAt   *time.Time
	FinishedAt  *time.Time
	CreatedAt   time.Time `gorm:"not null;default:now()"`
	UpdatedAt   time.Time `gorm:"not null;default:now()"`
}

func (Job) TableName() string {
	return "judge_jobs"
}

type EnqueueInput struct {
	SubmissionID uuid.UUID
	UserID       uuid.UUID
	ChallengeID  uuid.UUID
}
