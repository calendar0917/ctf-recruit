package recruitment

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

type ListFilter struct {
	Limit  int
	Offset int
}

type Repository interface {
	Create(ctx context.Context, submission *Submission) error
	GetByID(ctx context.Context, id string) (*Submission, error)
	List(ctx context.Context, filter ListFilter) ([]Submission, error)
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

func (r *GormRepository) GetByID(ctx context.Context, id string) (*Submission, error) {
	var submission Submission
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&submission).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &submission, nil
}

func (r *GormRepository) List(ctx context.Context, filter ListFilter) ([]Submission, error) {
	var submissions []Submission
	if err := r.db.WithContext(ctx).
		Model(&Submission{}).
		Order("created_at DESC").
		Limit(filter.Limit).
		Offset(filter.Offset).
		Find(&submissions).Error; err != nil {
		return nil, err
	}

	return submissions, nil
}
