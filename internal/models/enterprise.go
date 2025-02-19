package models

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

type Enterprise struct {
	gorm.Model
	UserID                uint   `gorm:"unique"`
	User                  User   `gorm:"foreignKey:UserID"`
	CompanyName           string `gorm:"not null"`
	CompanyRegistrationNo string `gorm:"unique"`

	// Corporate Structure
	ParentCompany      string
	Subsidiaries       []Enterprise `gorm:"foreignKey:ParentEnterpriseID"`
	ParentEnterpriseID *uint

	// Enterprise Details
	IndustryType  string
	AnnualRevenue string
	EmployeeCount int

	// Multiple Locations
	Locations []EnterpriseLocation `gorm:"foreignKey:EnterpriseID"`

	// Billing
	BillingCycle      string `gorm:"default:'monthly'"`
	ContractStartDate time.Time
	ContractEndDate   time.Time
	CustomPricingPlan json.RawMessage `gorm:"type:jsonb"` // Store custom pricing as JSON

	// Integration
	APIKeys     []EnterpriseAPIKey `gorm:"foreignKey:EnterpriseID"`
	IPWhitelist []string           `gorm:"type:text[]"`

	// Compliance
	ComplianceOfficer string
	ComplianceEmail   string
	RiskLevel         string
	LastAuditDate     time.Time

	// Support
	DedicatedManager string
	SupportTier      string `gorm:"default:'premium'"`

	// Usage Statistics
	MonthlyVolume    float64
	UserCount        int
	TransactionLimit float64

	// Add similar fields as Merchant for consistency
	VerificationStatus string `gorm:"default:'pending'"`
	DocumentsSubmitted bool   `gorm:"default:false"`
	ApprovedAt         *time.Time
}

type EnterpriseLocation struct {
	gorm.Model
	EnterpriseID   uint
	Address        string
	City           string
	Country        string
	IsHeadquarters bool
	ContactPerson  string
	ContactEmail   string
	ContactPhone   string
}

type EnterpriseAPIKey struct {
	gorm.Model
	EnterpriseID uint
	KeyName      string
	APIKey       string `gorm:"unique"`
	Environment  string // 'production' or 'sandbox'
	LastUsed     time.Time
	Status       string
}
