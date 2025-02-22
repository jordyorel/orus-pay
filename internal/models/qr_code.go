package models

import (
	"time"

	"gorm.io/gorm"
)

type QRCode struct {
	gorm.Model
	Code           string `gorm:"uniqueIndex;not null"`
	UserID         uint   `gorm:"not null;index"`
	UserType       string `gorm:"not null"`
	Type           string `gorm:"not null"`
	Amount         *float64
	ExpiresAt      *time.Time
	MaxUses        int    `gorm:"not null;default:1"`
	UsageCount     int    `gorm:"not null;default:0"`
	Status         string `gorm:"not null;default:'active'"`
	PaymentPurpose string

	DailyLimit       *float64
	MonthlyLimit     *float64
	AllowedCustomers []uint                 `gorm:"type:integer[]"`
	Metadata         map[string]interface{} `gorm:"type:jsonb"`
}

type QRTransaction struct {
	gorm.Model
	QRCodeID      uint `gorm:"index"`
	TransactionID uint `gorm:"unique"`
	CustomerID    uint `gorm:"index"`
	Amount        float64
	Status        string `gorm:"default:'pending'"`
	CompletedAt   *time.Time
	FailureReason string
}
