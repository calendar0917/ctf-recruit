package announcement

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
	Create(ctx context.Context, announcement *Announcement) error
	GetByID(ctx context.Context, id string) (*Announcement, error)
	Update(ctx context.Context, announcement *Announcement) error
	Delete(ctx context.Context, announcement *Announcement) error
	List(ctx context.Context, filter ListFilter) ([]Announcement, error)
}

type GormRepository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

func (r *GormRepository) Create(ctx context.Context, announcement *Announcement) error {
	return r.db.WithContext(ctx).Create(announcement).Error
}

func (r *GormRepository) GetByID(ctx context.Context, id string) (*Announcement, error) {
	var announcement Announcement
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&announcement).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &announcement, nil
}

func (r *GormRepository) Update(ctx context.Context, announcement *Announcement) error {
	return r.db.WithContext(ctx).Save(announcement).Error
}

func (r *GormRepository) Delete(ctx context.Context, announcement *Announcement) error {
	return r.db.WithContext(ctx).Delete(announcement).Error
}

func (r *GormRepository) List(ctx context.Context, filter ListFilter) ([]Announcement, error) {
	query := r.db.WithContext(ctx).Model(&Announcement{})
	if filter.PublishedOnly {
		query = query.Where("is_published = ?", true)
	}

	var announcements []Announcement
	if err := query.Order("COALESCE(published_at, created_at) DESC, created_at DESC").Limit(filter.Limit).Offset(filter.Offset).Find(&announcements).Error; err != nil {
		return nil, err
	}
	return announcements, nil
}
