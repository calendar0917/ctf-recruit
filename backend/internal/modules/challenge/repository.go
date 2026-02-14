package challenge

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

type ListFilter struct {
	PublishedOnly bool
	Limit         int
	Offset        int
}

type Repository interface {
	Create(ctx context.Context, challenge *Challenge) error
	GetByID(ctx context.Context, id string) (*Challenge, error)
	Update(ctx context.Context, challenge *Challenge) error
	Delete(ctx context.Context, challenge *Challenge) error
	List(ctx context.Context, filter ListFilter) ([]Challenge, error)
}

type GormRepository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

func (r *GormRepository) Create(ctx context.Context, challenge *Challenge) error {
	return r.db.WithContext(ctx).Create(challenge).Error
}

func (r *GormRepository) GetByID(ctx context.Context, id string) (*Challenge, error) {
	var challenge Challenge
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&challenge).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &challenge, nil
}

func (r *GormRepository) Update(ctx context.Context, challenge *Challenge) error {
	return r.db.WithContext(ctx).Save(challenge).Error
}

func (r *GormRepository) Delete(ctx context.Context, challenge *Challenge) error {
	return r.db.WithContext(ctx).Delete(challenge).Error
}

func (r *GormRepository) List(ctx context.Context, filter ListFilter) ([]Challenge, error) {
	query := r.db.WithContext(ctx).Model(&Challenge{})
	if filter.PublishedOnly {
		query = query.Where("is_published = ?", true)
	}

	var challenges []Challenge
	if err := query.Order("created_at DESC").Limit(filter.Limit).Offset(filter.Offset).Find(&challenges).Error; err != nil {
		return nil, err
	}
	return challenges, nil
}
