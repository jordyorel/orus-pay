package models

import "gorm.io/gorm"

type KYCVerification struct {
	gorm.Model
	UserID     uint   `gorm:"not null"`
	Status     string `gorm:"default:'pending'"`
	DocumentID string
	ScanURL    string
}
