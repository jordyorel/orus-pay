package qr

import (
	"fmt"
	"time"
)

type QRType string
type UserType string

const (
	// QR Types
	TypeStatic       QRType = "static"         // Base type
	TypeReceive      QRType = "static_receive" // For receiving money
	TypePaymentCode  QRType = "payment_code"   // For payments
	TypeMerchantScan QRType = "static_scan"
	TypeDynamic      QRType = "dynamic"
	TypePayment      QRType = "payment"

	// User Types
	UserTypeRegular  UserType = "regular"
	UserTypeMerchant UserType = "merchant"
)

// Add this before DefaultLimits
type Limits struct {
	DailyLimit   float64
	MonthlyLimit float64
	MaxUses      int
}

// Update limits based on user type
var DefaultLimits = map[UserType]Limits{
	UserTypeRegular: {
		DailyLimit:   1000,
		MonthlyLimit: 5000,
		MaxUses:      -1, // Unlimited
	},
	UserTypeMerchant: {
		DailyLimit:   10000,
		MonthlyLimit: 100000,
		MaxUses:      -1, // Unlimited
	},
}

type GenerateQRRequest struct {
	UserID       uint
	UserType     UserType
	QRType       QRType
	Amount       *float64
	ExpiresAt    *time.Time
	MaxUses      int
	DailyLimit   *float64
	MonthlyLimit *float64
	Metadata     map[string]interface{}
}

// Add String methods
func (t QRType) String() string {
	return string(t)
}

func (t UserType) String() string {
	return string(t)
}

// Add validation methods
func (r *GenerateQRRequest) Validate() error {
	if r.UserID == 0 {
		return fmt.Errorf("user ID is required")
	}

	switch r.QRType {
	case TypeStatic, TypeReceive, TypePaymentCode, TypeMerchantScan, TypeDynamic, TypePayment:
		// Valid types
	default:
		return fmt.Errorf("invalid QR type: %s", r.QRType)
	}

	switch r.UserType {
	case UserTypeRegular, UserTypeMerchant:
		// Valid user types
	default:
		return fmt.Errorf("invalid user type")
	}

	if r.QRType == TypeDynamic && (r.Amount == nil || *r.Amount <= 0) {
		return fmt.Errorf("amount is required for dynamic QR")
	}

	return nil
}
