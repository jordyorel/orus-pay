package models

import (
	"time"
)

// Transaction types
const (
	TransactionTypeTopup          = "TOPUP"
	TransactionTypeWithdrawal     = "WITHDRAWAL"
	TransactionTypeQRPayment      = "qr_payment"
	TransactionTypeMerchantDirect = "merchant_direct"
	TransactionTypeMerchantScan   = "merchant_scan"
	TransactionTypeRefund         = "refund"
	TransactionTypeP2PTransfer    = "P2P_TRANSFER"
	TransactionTypeTransfer       = "transfer"
	TransactionTypeQRCode         = "QR_PAYMENT"
)

// Consolidated Transaction model
type Transaction struct {
	ID               uint    `gorm:"primarykey"`
	Type             string  `gorm:"not null"`
	SenderID         uint    `gorm:"not null"`
	ReceiverID       uint    `gorm:"not null"`
	Amount           float64 `gorm:"not null"`
	Description      string
	Status           string  `gorm:"not null;default:'pending'"`
	Fee              float64 `gorm:"default:0"`
	Metadata         JSON    `gorm:"type:jsonb"`
	Currency         string  `gorm:"default:'USD'"`
	TransactionID    string  // External reference ID
	Reference        string  // For linking related transactions
	PaymentType      string  // Payment method used
	PaymentMethod    string  // Additional payment details
	MerchantID       *uint   // Optional merchant reference
	MerchantName     string  // Merchant business name
	MerchantCategory string  // Merchant business type
	CardID           *uint   // Optional card reference
	QRCodeID         *string // Optional QR code reference
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type Location struct {
	Latitude  float64
	Longitude float64
	Address   string
}

type DeviceInfo struct {
	DeviceID   string
	DeviceType string
	IPAddress  string
}
