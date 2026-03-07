package auth

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, user *User) error
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id string) (*User, error)
	List(ctx context.Context, limit, offset int) ([]User, error)
	UpdateAdminFields(ctx context.Context, id string, role *Role, isDisabled *bool) (*User, error)
}

type GormRepository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

func (r *GormRepository) Create(ctx context.Context, user *User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *GormRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *GormRepository) GetByID(ctx context.Context, id string) (*User, error) {
	var user User
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *GormRepository) List(ctx context.Context, limit, offset int) ([]User, error) {
	var users []User
	err := r.db.WithContext(ctx).
		Order("email ASC").
		Limit(limit).
		Offset(offset).
		Find(&users).Error
	if err != nil {
		return nil, err
	}
	return users, nil
}

func (r *GormRepository) UpdateAdminFields(ctx context.Context, id string, role *Role, isDisabled *bool) (*User, error) {
	updates := map[string]interface{}{}
	if role != nil {
		updates["role"] = *role
	}
	if isDisabled != nil {
		updates["is_disabled"] = *isDisabled
	}

	if len(updates) > 0 {
		result := r.db.WithContext(ctx).Model(&User{}).Where("id = ?", id).Updates(updates)
		if result.Error != nil {
			return nil, result.Error
		}
		if result.RowsAffected == 0 {
			return nil, nil
		}
	}

	return r.GetByID(ctx, id)
}
