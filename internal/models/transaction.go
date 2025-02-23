package models

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

const (
	TransactionTypeTopup          = "TOPUP"
	TransactionTypeWithdrawal     = "WITHDRAWAL"
	TransactionTypeQRPayment      = "QR_PAYMENT"
	TransactionTypeMerchantDirect = "MERCHANT_DIRECT"
	TransactionTypeMerchantScan   = "MERCHANT_SCAN"
)

type Transaction struct {
	gorm.Model
	TransactionID string `gorm:"unique;not null"`
	UserID        uint   `gorm:"index"`    // For direct wallet operations
	Type          string `gorm:"not null"` // QR_PAYMENT, MERCHANT_SCAN, etc.
	SenderID      uint   `gorm:"index"`
	ReceiverID    uint   `gorm:"index"`
	Amount        float64
	Fee           float64 `gorm:"default:0"`
	Currency      string  `gorm:"default:'USD'"`
	Status        string  `gorm:"default:'pending'"`
	Reference     string  // For tracking related transactions
	Description   string

	// QR specific fields
	QRCodeID    string `gorm:"index"`
	QRType      string // static, dynamic, payment
	QROwnerID   uint   `gorm:"index"`
	QROwnerType string // user, merchant

	// Payment details
	PaymentMethod string
	PaymentType   string // DIRECT, QR_SCAN, CARD
	CardID        *uint  `gorm:"index"` // For card-based transactions

	// Merchant specific
	MerchantID       *uint `gorm:"index"`
	MerchantName     string
	MerchantCategory string

	// Additional metadata
	Location   *Location              `gorm:"type:jsonb"`
	DeviceInfo *DeviceInfo            `gorm:"type:jsonb"`
	Metadata   map[string]interface{} `gorm:"type:jsonb"`
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

// BeforeCreate hook to generate TransactionID if not set
func (tx *Transaction) BeforeCreate(db *gorm.DB) error {
	if tx.TransactionID == "" {
		tx.TransactionID = fmt.Sprintf("TX-%d-%d", time.Now().Unix(), tx.ID)
	}
	return nil
}
