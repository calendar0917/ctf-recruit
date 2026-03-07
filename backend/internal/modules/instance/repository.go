package instance

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository interface {
	StartInstance(ctx context.Context, userID, challengeID uuid.UUID, now time.Time, ttl, cooldown time.Duration) (*ChallengeInstance, *time.Time, error)
	GetByID(ctx context.Context, id uuid.UUID) (*ChallengeInstance, error)
	GetByIDForUser(ctx context.Context, id, userID uuid.UUID) (*ChallengeInstance, error)
	FindActiveForUser(ctx context.Context, userID uuid.UUID) (*ChallengeInstance, error)
	FindLatestForUser(ctx context.Context, userID uuid.UUID) (*ChallengeInstance, error)
	ListExpirableBefore(ctx context.Context, now time.Time, limit int) ([]ChallengeInstance, error)
	TransitionStatus(ctx context.Context, id, userID uuid.UUID, to Status, now time.Time, ttl, cooldown time.Duration, containerID *string) (*ChallengeInstance, error)
}

type GormRepository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

func (r *GormRepository) StartInstance(ctx context.Context, userID, challengeID uuid.UUID, now time.Time, ttl, cooldown time.Duration) (*ChallengeInstance, *time.Time, error) {
	var created *ChallengeInstance
	var retryAt *time.Time

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var lockID string
		if err := tx.Raw("SELECT id FROM users WHERE id = ? FOR UPDATE", userID).Scan(&lockID).Error; err != nil {
			return err
		}

		active, err := r.getLatestByStatusesTx(ctx, tx, userID, []Status{StatusStarting, StatusRunning})
		if err != nil {
			return err
		}
		if active != nil {
			return ErrActiveInstanceExists
		}

		latest, err := r.getLatestByStatusesTx(ctx, tx, userID, []Status{StatusCooldown, StatusStopped, StatusExpired, StatusFailed})
		if err != nil {
			return err
		}
		if latest != nil && latest.CooldownUntil != nil && latest.CooldownUntil.After(now) {
			retry := latest.CooldownUntil.UTC()
			retryAt = &retry
			return ErrCooldownActive
		}

		instance := &ChallengeInstance{
			ID:          uuid.New(),
			UserID:      userID,
			ChallengeID: challengeID,
			Status:      StatusStarting,
			CreatedAt:   now.UTC(),
			UpdatedAt:   now.UTC(),
		}
		_ = ttl
		_ = cooldown

		if err := tx.Create(instance).Error; err != nil {
			return err
		}

		created = instance
		return nil
	})
	if err != nil {
		return nil, retryAt, err
	}

	return created, nil, nil
}

func (r *GormRepository) GetByIDForUser(ctx context.Context, id, userID uuid.UUID) (*ChallengeInstance, error) {
	var instance ChallengeInstance
	err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).Take(&instance).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &instance, nil
}

func (r *GormRepository) GetByID(ctx context.Context, id uuid.UUID) (*ChallengeInstance, error) {
	var instance ChallengeInstance
	err := r.db.WithContext(ctx).Where("id = ?", id).Take(&instance).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &instance, nil
}

func (r *GormRepository) FindActiveForUser(ctx context.Context, userID uuid.UUID) (*ChallengeInstance, error) {
	var instance ChallengeInstance
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND status IN ?", userID, []Status{StatusStarting, StatusRunning, StatusStopping}).
		Order("created_at DESC").
		Take(&instance).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &instance, nil
}

func (r *GormRepository) FindLatestForUser(ctx context.Context, userID uuid.UUID) (*ChallengeInstance, error) {
	var instance ChallengeInstance
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Take(&instance).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &instance, nil
}

func (r *GormRepository) ListExpirableBefore(ctx context.Context, now time.Time, limit int) ([]ChallengeInstance, error) {
	if limit <= 0 {
		limit = 20
	}

	items := make([]ChallengeInstance, 0, limit)
	err := r.db.WithContext(ctx).
		Where("status = ? AND expires_at IS NOT NULL AND expires_at <= ?", StatusRunning, now.UTC()).
		Order("expires_at ASC").
		Limit(limit).
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *GormRepository) TransitionStatus(ctx context.Context, id, userID uuid.UUID, to Status, now time.Time, ttl, cooldown time.Duration, containerID *string) (*ChallengeInstance, error) {
	var updated *ChallengeInstance

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var lockID string
		if err := tx.Raw("SELECT id FROM users WHERE id = ? FOR UPDATE", userID).Scan(&lockID).Error; err != nil {
			return err
		}

		var current ChallengeInstance
		if err := tx.Where("id = ? AND user_id = ?", id, userID).Take(&current).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return ErrInstanceNotFound
			}
			return err
		}

		if !CanTransition(current.Status, to) {
			return ErrInvalidTransition
		}

		changes := map[string]any{
			"status":     to,
			"updated_at": now.UTC(),
		}

		if containerID != nil {
			changes["container_id"] = *containerID
		}

		switch to {
		case StatusRunning:
			s := now.UTC()
			changes["started_at"] = s
			changes["expires_at"] = s.Add(ttl)
			changes["cooldown_until"] = nil
		case StatusStopped, StatusExpired, StatusFailed, StatusCooldown:
			if cooldown > 0 {
				cd := now.UTC().Add(cooldown)
				changes["cooldown_until"] = cd
			}
		}

		if err := tx.Model(&ChallengeInstance{}).Where("id = ? AND user_id = ?", id, userID).Updates(changes).Error; err != nil {
			return err
		}

		if err := tx.Where("id = ?", id).Take(&current).Error; err != nil {
			return err
		}

		updated = &current
		return nil
	})
	if err != nil {
		return nil, err
	}

	return updated, nil
}

func (r *GormRepository) getLatestByStatusesTx(ctx context.Context, tx *gorm.DB, userID uuid.UUID, statuses []Status) (*ChallengeInstance, error) {
	if len(statuses) == 0 {
		return nil, nil
	}

	var instance ChallengeInstance
	err := tx.WithContext(ctx).
		Where("user_id = ? AND status IN ?", userID, statuses).
		Order("created_at DESC").
		Limit(1).
		Take(&instance).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &instance, nil
}
