package instance

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusStarting Status = "starting"
	StatusRunning  Status = "running"
	StatusStopping Status = "stopping"
	StatusStopped  Status = "stopped"
	StatusExpired  Status = "expired"
	StatusFailed   Status = "failed"
	StatusCooldown Status = "cooldown"
)

type ChallengeInstance struct {
	ID            uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID        uuid.UUID  `gorm:"type:uuid;not null;index"`
	ChallengeID   uuid.UUID  `gorm:"type:uuid;not null;index"`
	Status        Status     `gorm:"type:varchar(20);not null;index"`
	ContainerID   *string    `gorm:"type:varchar(255)"`
	StartedAt     *time.Time `gorm:"type:timestamptz"`
	ExpiresAt     *time.Time `gorm:"type:timestamptz"`
	CooldownUntil *time.Time `gorm:"type:timestamptz"`
	CreatedAt     time.Time  `gorm:"not null;default:now()"`
	UpdatedAt     time.Time  `gorm:"not null;default:now()"`
}

func (ChallengeInstance) TableName() string {
	return "challenge_instances"
}
