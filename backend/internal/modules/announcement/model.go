package announcement

import (
	"time"

	"github.com/google/uuid"
)

type Announcement struct {
	ID          uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Title       string     `gorm:"size:255;not null"`
	Content     string     `gorm:"type:text;not null"`
	IsPublished bool       `gorm:"not null;default:false"`
	PublishedAt *time.Time `gorm:"type:timestamptz"`
	CreatedAt   time.Time  `gorm:"not null;default:now()"`
	UpdatedAt   time.Time  `gorm:"not null;default:now()"`
}

func (Announcement) TableName() string {
	return "announcements"
}
