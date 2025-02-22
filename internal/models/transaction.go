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
)

type Transaction struct {
	gorm.Model
	TransactionID string `gorm:"unique;not null"`
	Type          string `gorm:"not null"` // "qr_payment", "transfer", "topup", etc.
	SenderID      uint   `gorm:"index"`
	ReceiverID    uint   `gorm:"index"`
	Amount        float64
	Fee           float64 `gorm:"default:0"`
	Currency      string  `gorm:"default:'USD'"`
	Status        string  `gorm:"default:'pending'"`
	PaymentMethod string
	Description   string
	Metadata      JSON `gorm:"type:jsonb"`

	// For merchant transactions
	MerchantID  *uint  `gorm:"index"`
	QRCodeID    string `gorm:"index"` // Added back for QR payments
	PaymentType string // Added back for different payment types (CARD, TRANSFER, etc.)
	CardID      *uint  // Added back for card payments
}

// BeforeCreate hook to generate TransactionID if not set
func (tx *Transaction) BeforeCreate(db *gorm.DB) error {
	if tx.TransactionID == "" {
		tx.TransactionID = fmt.Sprintf("TX-%d-%d", time.Now().Unix(), tx.ID)
	}
	return nil
}
