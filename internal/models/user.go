package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Email                 string  `gorm:"uniqueIndex;not null"` // Unique index on Email
	Password              string  `gorm:"not null"`
	Name                  string  `gorm:"not null"`
	Phone                 string  `gorm:"uniqueIndex;not null"` // Unique index on Phone
	UserType              string  `gorm:"default:'regular'"`
	Role                  string  `gorm:"default:'user'"`
	WalletID              *uint   `gorm:"unique;default:null"` // Make it a pointer to allow NULL
	Wallet                *Wallet `gorm:"foreignKey:WalletID"`
	Status                string  `gorm:"default:'active'"`
	KYCStatus             string  `gorm:"default:'pending'"`
	LastLoginAt           time.Time
	LastLoginIP           string
	TwoFactorEnabled      bool `gorm:"default:false"`
	FailedLoginAttempts   int  `gorm:"default:0"`
	AccountLockoutUntil   *time.Time
	TokenVersion          int       `gorm:"default:1"`
	MerchantProfileStatus string    `gorm:"default:'not_applicable'"`
	Balance               float64   `gorm:"default:0"`
	LastActiveAt          time.Time `gorm:"index"`
}

// CreateUserInput represents the data needed to create a new user
type CreateUserInput struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

// UpdateUserInput represents the data needed to update a user
type UpdateUserInput struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
	Email string `json:"email"`
}
