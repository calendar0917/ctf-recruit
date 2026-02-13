package judge

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, job *Job) error
	ListQueued(ctx context.Context, limit int) ([]Job, error)
	MarkRunning(ctx context.Context, id uuid.UUID) error
	MarkDone(ctx context.Context, id uuid.UUID) error
	MarkFailed(ctx context.Context, id uuid.UUID, message string) error
	FinalizeExecution(ctx context.Context, input FinalizeExecutionInput) error
}

type GormRepository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

func (r *GormRepository) Create(ctx context.Context, job *Job) error {
	return r.db.WithContext(ctx).Create(job).Error
}

func (r *GormRepository) ListQueued(ctx context.Context, limit int) ([]Job, error) {
	if limit <= 0 {
		limit = 1
	}

	var jobs []Job
	err := r.db.WithContext(ctx).
		Where("status = ?", JobStatusQueued).
		Order("queued_at ASC").
		Limit(limit).
		Find(&jobs).Error
	if err != nil {
		return nil, err
	}
	return jobs, nil
}

func (r *GormRepository) MarkRunning(ctx context.Context, id uuid.UUID) error {
	now := nowUTC()
	return r.db.WithContext(ctx).Model(&Job{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":     JobStatusRunning,
			"started_at": now,
			"attempts":   gorm.Expr("attempts + 1"),
			"updated_at": now,
		}).Error
}

func (r *GormRepository) MarkDone(ctx context.Context, id uuid.UUID) error {
	now := nowUTC()
	return r.db.WithContext(ctx).Model(&Job{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":        JobStatusDone,
			"finished_at":   now,
			"error_message": "",
			"updated_at":    now,
		}).Error
}

func (r *GormRepository) MarkFailed(ctx context.Context, id uuid.UUID, message string) error {
	now := nowUTC()
	return r.db.WithContext(ctx).Model(&Job{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":        JobStatusFailed,
			"finished_at":   now,
			"error_message": message,
			"updated_at":    now,
		}).Error
}

func (r *GormRepository) FinalizeExecution(ctx context.Context, input FinalizeExecutionInput) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		awardedPoints := 0
		if input.SubmissionStatus == "correct" {
			if input.AwardedPoints != nil {
				awardedPoints = *input.AwardedPoints
			} else {
				points, err := loadChallengePoints(ctx, tx, input.ChallengeID)
				if err != nil {
					return err
				}
				awardedPoints = points
			}
		}

		subUpdate := tx.Table("submissions").
			Where("id = ? AND status = ?", input.SubmissionID, "pending").
			Updates(map[string]any{
				"status":         input.SubmissionStatus,
				"awarded_points": awardedPoints,
			})
		if subUpdate.Error != nil {
			return subUpdate.Error
		}

		now := nowUTC()
		errorMessage := ""
		if input.JobStatus == JobStatusFailed {
			errorMessage = input.JobErrorMessage
		}
		jobUpdate := tx.Model(&Job{}).
			Where("id = ?", input.JobID).
			Updates(map[string]any{
				"status":        input.JobStatus,
				"finished_at":   now,
				"error_message": errorMessage,
				"updated_at":    now,
			})
		if jobUpdate.Error != nil {
			return jobUpdate.Error
		}
		return nil
	})
}

func loadChallengePoints(ctx context.Context, tx *gorm.DB, challengeID uuid.UUID) (int, error) {
	type challengePoints struct {
		Points int
	}
	var out challengePoints
	err := tx.WithContext(ctx).Table("challenges").Select("points").Where("id = ?", challengeID).Take(&out).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, err
	}
	return out.Points, nil
}

func nowUTC() time.Time {
	return time.Now().UTC()
}
