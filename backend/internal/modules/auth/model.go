package auth

import "github.com/google/uuid"

type Role string

const (
	RoleAdmin  Role = "admin"
	RolePlayer Role = "player"
)

type User struct {
	ID           uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Email        string    `gorm:"size:255;not null;uniqueIndex"`
	PasswordHash string    `gorm:"size:255;not null"`
	DisplayName  string    `gorm:"size:100;not null"`
	Role         Role      `gorm:"type:varchar(20);not null;default:'player'"`
}

func (User) TableName() string {
	return "users"
}
