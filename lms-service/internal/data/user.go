package data

import (
	"github.com/jinzhu/gorm"
)

type UserInfo struct {
	ID             uint   `gorm:"primaryKey"`
	CreatedAt      string `gorm:"type:timestamp with time zone;default:now()"`
	UpdatedAt      string `gorm:"type:timestamp with time zone;default:now()"`
	FName          string
	SName          string
	Email          string `gorm:"unique;not null"`
	PasswordHash   []byte `gorm:"not null"`
	UserRole       string
	Activated      bool `gorm:"not null"`
	Version        int  `gorm:"default:1"`
	ActivationLink string
}

type UserModel struct {
	DB *gorm.DB
}
