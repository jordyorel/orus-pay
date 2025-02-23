package qr

import (
	"time"
)

type QRType string

const (
	TypeStatic  QRType = "static"  // For receiving payments
	TypeDynamic QRType = "dynamic" // For receiving specific amount
	TypePayment QRType = "payment" // For making payments
)

type UserType string

const (
	UserTypeRegular  UserType = "user"
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

type QRLimits struct {
	DailyLimit   float64
	MonthlyLimit float64
	MaxUses      int
	ExpiresAt    *time.Time
}

// Default limits by user type
var defaultLimits = map[UserType]QRLimits{
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
