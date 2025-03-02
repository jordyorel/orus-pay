package models

import (
	"time"

	"gorm.io/gorm"
)

type Dispute struct {
	gorm.Model
	TransactionID uint   `gorm:"not null"`
	MerchantID    uint   `gorm:"not null"`
	UserID        uint   `gorm:"not null"`
	Reason        string `gorm:"not null"`
	Status        string `gorm:"default:'pending'"`
	Refunded      bool   `gorm:"default:false"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
