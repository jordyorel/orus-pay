package models

import (
	"time"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

type MerchantLimits struct {
	gorm.Model
	MerchantID              uint           `gorm:"not null"`
	DailyTransactionLimit   float64        `gorm:"not null;default:10000"`
	MonthlyTransactionLimit float64        `gorm:"not null;default:100000"`
	SingleTransactionLimit  float64        `gorm:"not null;default:5000"`
	MinTransactionAmount    float64        `gorm:"not null;default:1"`
	MaxTransactionAmount    float64        `gorm:"not null;default:5000"`
	ConcurrentTransactions  int            `gorm:"not null;default:10"`
	AllowedCurrencies       pq.StringArray `gorm:"type:text[]"`
}

type Merchant struct {
	gorm.Model
	UserID           uint   `gorm:"not null"`
	User             User   `gorm:"foreignKey:UserID"`
	BusinessName     string `gorm:"not null"`
	BusinessType     string
	BusinessAddress  string
	BusinessID       string `gorm:"unique_index"`
	TaxID            string
	Website          string
	MerchantCategory string

	// Business Details
	LegalEntityType    string
	RegistrationNumber string
	YearEstablished    int

	// Financial Information
	BankAccounts      []MerchantBankAccount `gorm:"foreignKey:MerchantID"`
	MonthlyVolume     float64
	ProcessingFeeRate float64 `gorm:"default:2.5"`

	// Verification Status
	VerificationStatus string `gorm:"default:'pending'"`
	DocumentsSubmitted bool   `gorm:"default:false"`
	ApprovedAt         *time.Time

	// Integration Settings
	APIKey      string `gorm:"unique"`
	WebhookURL  string
	IPWhitelist pq.StringArray `gorm:"type:text[]"`

	// Business Hours
	OperatingHours pq.StringArray `gorm:"type:text[]"`

	// Support Contact
	SupportEmail string
	SupportPhone string

	// Settlement Details
	SettlementCycle     string `gorm:"default:'daily'"`
	MinSettlementAmount float64

	// Stats
	TotalTransactions int64
	TotalVolume       float64
	Rating            int `gorm:"default:0"`

	// Add the Limits and Chargeback fields from earlier
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
