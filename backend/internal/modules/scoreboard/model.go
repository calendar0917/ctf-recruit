package scoreboard

import "time"

type ScoreboardAggregate struct {
	UserID         string    `gorm:"column:user_id"`
	DisplayName    string    `gorm:"column:display_name"`
	TotalPoints    int       `gorm:"column:total_points"`
	SolvedCount    int       `gorm:"column:solved_count"`
	LastAcceptedAt time.Time `gorm:"column:last_accepted_at"`
}
