package submission

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusCorrect Status = "correct"
	StatusWrong   Status = "wrong"
	StatusPending Status = "pending"
	StatusFailed  Status = "failed"
)

type Submission struct {
	ID            uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID        uuid.UUID `gorm:"type:uuid;not null;index"`
	ChallengeID   uuid.UUID `gorm:"type:uuid;not null;index"`
	Status        Status    `gorm:"type:varchar(20);not null"`
	AwardedPoints int       `gorm:"not null;default:0"`
	CreatedAt     time.Time `gorm:"not null;default:now()"`
}

func (Submission) TableName() string {
	return "submissions"
}
