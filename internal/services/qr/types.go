package qr

import (
	domainQR "orus/internal/domain/qr"
	"time"
)

// QRType represents the type of QR code
type QRType string

// UserType represents the type of user
type UserType string

const (
	// QR Types
	TypeStatic      QRType = "static"         // Base type
	TypeReceive     QRType = "static_receive" // For receiving money from users
	TypePaymentCode QRType = "payment_code"   // For payments at merchants
	TypeDynamic     QRType = "dynamic"
	TypePayment     QRType = "payment"

	// User Types
	UserTypeRegular  UserType = "regular"
	UserTypeMerchant UserType = "merchant"
)

type QRConfig struct {
	UserID       uint
	UserType     UserType
	Type         QRType
	Amount       *float64
	ExpiresAt    *time.Time
	MaxUses      int
	DailyLimit   *float64
	MonthlyLimit *float64
	Metadata     map[string]interface{}
}

// QRLimits defines the usage limits for QR codes
type QRLimits struct {
	DailyLimit   float64
	MonthlyLimit float64
	MaxUses      int
	ExpiresAt    *time.Time
}

// DefaultLimits defines the default limits by user type
var DefaultLimits = map[domainQR.UserType]QRLimits{
	domainQR.UserTypeRegular: {
		DailyLimit:   1000,
		MonthlyLimit: 5000,
		MaxUses:      -1, // Unlimited
	},
	domainQR.UserTypeMerchant: {
		DailyLimit:   10000,
		MonthlyLimit: 100000,
		MaxUses:      -1, // Unlimited
	},
}

func (t UserType) String() string {
	return string(t)
}

func (t QRType) String() string {
	return string(t)
}
