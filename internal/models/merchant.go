package models

import (
	"time"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

type MerchantLimits struct {
	gorm.Model
	MerchantID              uint           `gorm:"not null" json:"merchant_id"`
	DailyTransactionLimit   float64        `gorm:"not null;default:10000" json:"daily_limit"`
	MonthlyTransactionLimit float64        `gorm:"not null;default:100000" json:"monthly_limit"`
	SingleTransactionLimit  float64        `gorm:"not null;default:5000" json:"min_transaction"`
	MinTransactionAmount    float64        `gorm:"not null;default:1" json:"min_transaction_amount"`
	MaxTransactionAmount    float64        `gorm:"not null;default:5000" json:"max_transaction"`
	ConcurrentTransactions  int            `gorm:"not null;default:10"`
	AllowedCurrencies       pq.StringArray `gorm:"type:text[]" json:"allowed_currencies"`
}

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
	Limits          *MerchantLimits    `gorm:"foreignKey:MerchantID"`
	RiskScore       int                `gorm:"default:50"`
	ComplianceLevel string             `gorm:"default:'medium_risk'"`
	DisputeRate     float64            `gorm:"default:0"`
	Chargeback      MerchantChargeback `gorm:"foreignKey:MerchantID"`
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
