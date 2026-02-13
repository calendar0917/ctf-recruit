package submission

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, submission *Submission) error
	FinalizePending(ctx context.Context, submissionID uuid.UUID, status Status, awardedPoints int) (bool, error)
}

type GormRepository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

func (r *GormRepository) Create(ctx context.Context, submission *Submission) error {
	return r.db.WithContext(ctx).Create(submission).Error
}

func (r *GormRepository) FinalizePending(ctx context.Context, submissionID uuid.UUID, status Status, awardedPoints int) (bool, error) {
	result := r.db.WithContext(ctx).Model(&Submission{}).
		Where("id = ? AND status = ?", submissionID, StatusPending).
		Updates(map[string]any{
			"status":         status,
			"awarded_points": awardedPoints,
		})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}
