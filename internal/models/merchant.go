package models

import (
	"time"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

type Merchant struct {
	gorm.Model
	UserID             uint   `gorm:"uniqueIndex;not null" json:"user_id"`
	BusinessName       string `gorm:"not null" json:"business_name"`
	BusinessType       string `gorm:"not null" json:"business_type"`
	BusinessAddress    string `gorm:"not null" json:"business_address"`
	BusinessID         string `gorm:"unique" json:"business_id"`
	TaxID              string `json:"tax_id"`
	Website            string `json:"website"`
	MerchantCategory   string `json:"merchant_category"`
	LegalEntityType    string `json:"legal_entity_type"`
	RegistrationNumber string `json:"registration_number"`
	YearEstablished    int    `json:"year_established"`
	SupportEmail       string `json:"support_email"`
	SupportPhone       string `json:"support_phone"`
	IsActive           bool   `gorm:"default:true"`
	VerificationStatus string `gorm:"default:'pending'"`
	Status             string `json:"status"`

	// Financial Information
	BankAccounts      []MerchantBankAccount `gorm:"foreignKey:MerchantID"`
	MonthlyVolume     float64
	ProcessingFeeRate float64 `gorm:"default:2.5"`

	// Verification Status
	DocumentsSubmitted bool `gorm:"default:false"`
	ApprovedAt         *time.Time

	// Integration Settings
	APIKey      string `gorm:"unique"`
	WebhookURL  string
	IPWhitelist pq.StringArray `gorm:"type:text[]"`

	// Business Hours
	OperatingHours pq.StringArray `gorm:"type:text[]"`

	// Stats
	TotalTransactions int64
	TotalVolume       float64
	Rating            int `gorm:"default:0"`

	// Risk and Compliance
	RiskScore       int                `gorm:"default:50"`
	ComplianceLevel string             `gorm:"default:'medium_risk'"`
	DisputeRate     float64            `gorm:"default:0"`
	Chargeback      MerchantChargeback `gorm:"foreignKey:MerchantID"`

	// Limits fields integrated here
	DailyTransactionLimit   float64 `json:"daily_transaction_limit" gorm:"default:10000"`
	MonthlyTransactionLimit float64 `json:"monthly_transaction_limit" gorm:"default:100000"`
	MinTransactionAmount    float64 `json:"min_transaction_amount" gorm:"default:1"`
	MaxTransactionAmount    float64 `json:"max_transaction_amount" gorm:"default:5000"`
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
