package scoreboard

import (
	"context"

	"gorm.io/gorm"
)

type Repository interface {
	ListAggregates(ctx context.Context) ([]ScoreboardAggregate, error)
}

type GormRepository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

func (r *GormRepository) ListAggregates(ctx context.Context) ([]ScoreboardAggregate, error) {
	items := make([]ScoreboardAggregate, 0)
	err := r.db.WithContext(ctx).
		Table("submissions AS s").
		Select(`
			s.user_id::text AS user_id,
			u.display_name AS display_name,
			SUM(s.awarded_points) AS total_points,
			COUNT(DISTINCT s.challenge_id) AS solved_count,
			MAX(s.created_at) AS last_accepted_at
		`).
		Joins("JOIN users AS u ON u.id = s.user_id").
		Where("s.awarded_points > 0").
		Group("s.user_id, u.display_name").
		Scan(&items).Error
	if err != nil {
		return nil, err
	}

	return items, nil
}
