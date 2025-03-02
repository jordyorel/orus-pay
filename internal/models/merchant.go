package models

import (
	"time"

	"gorm.io/gorm"
)

type Merchant struct {
	ID                      uint   `gorm:"primarykey"`
	UserID                  uint   `gorm:"uniqueIndex;not null"`
	BusinessName            string `gorm:"not null"`
	BusinessType            string `gorm:"not null"`
	BusinessAddress         string
	RiskScore               int     `gorm:"default:0"`
	ComplianceLevel         string  `gorm:"default:'pending'"`
	Status                  string  `gorm:"default:'pending'"`
	ProcessingFeeRate       float64 `gorm:"default:0"`
	DailyTransactionLimit   float64
	MonthlyTransactionLimit float64
	MinTransactionAmount    float64
	MaxTransactionAmount    float64
	WebhookURL              string
	MonthlyVolume           float64
	Metadata                JSON `gorm:"type:jsonb"`
	CreatedAt               time.Time
	UpdatedAt               time.Time
	APIKey                  string `gorm:"column:api_key"`
}

type MerchantBankAccount struct {
	gorm.Model
	MerchantID         uint
	BankName           string
	AccountNumber      string
	AccountType        string
	IsDefault          bool
	VerificationStatus string
}

type MerchantChargeback struct {
	gorm.Model
	MerchantID       uint
	Count            int
	Volume           float64
	Ratio            float64
	LastChargebackAt time.Time
}
