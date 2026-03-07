package submission

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, submission *Submission) error
	FinalizePending(ctx context.Context, submissionID uuid.UUID, status Status, awardedPoints int) (bool, error)
	ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]Submission, error)
	ListByUserAndChallenge(ctx context.Context, userID, challengeID uuid.UUID, limit, offset int) ([]Submission, error)
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

func (r *GormRepository) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]Submission, error) {
	items := make([]Submission, 0)
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *GormRepository) ListByUserAndChallenge(ctx context.Context, userID, challengeID uuid.UUID, limit, offset int) ([]Submission, error) {
	items := make([]Submission, 0)
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND challenge_id = ?", userID, challengeID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}
