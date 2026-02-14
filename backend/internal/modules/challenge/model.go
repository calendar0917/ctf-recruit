package challenge

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Difficulty string

type Mode string

const (
	DifficultyEasy   Difficulty = "easy"
	DifficultyMedium Difficulty = "medium"
	DifficultyHard   Difficulty = "hard"
)

const (
	ModeStatic  Mode = "static"
	ModeDynamic Mode = "dynamic"
)

type Challenge struct {
	ID          uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Title       string         `gorm:"size:255;not null"`
	Description string         `gorm:"type:text;not null"`
	Category    string         `gorm:"size:100;not null"`
	Difficulty  Difficulty     `gorm:"type:varchar(20);not null"`
	Mode        Mode           `gorm:"type:varchar(20);not null;default:static"`
	Points      int            `gorm:"not null"`
	FlagHash    string         `gorm:"size:255;not null"`
	IsPublished bool           `gorm:"not null;default:false"`
	CreatedAt   time.Time      `gorm:"not null;default:now()"`
	UpdatedAt   time.Time      `gorm:"not null;default:now()"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

func (Challenge) TableName() string {
	return "challenges"
}
