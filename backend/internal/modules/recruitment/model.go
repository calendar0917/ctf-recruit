package recruitment

import (
	"time"

	"github.com/google/uuid"
)

type Submission struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index"`
	Name      string    `gorm:"size:100;not null"`
	School    string    `gorm:"size:255;not null"`
	Grade     string    `gorm:"size:100;not null"`
	Direction string    `gorm:"size:100;not null"`
	Contact   string    `gorm:"size:255;not null"`
	Bio       string    `gorm:"type:text;not null"`
	CreatedAt time.Time `gorm:"not null;default:now()"`
	UpdatedAt time.Time `gorm:"not null;default:now()"`
}

func (Submission) TableName() string {
	return "recruitment_submissions"
}
