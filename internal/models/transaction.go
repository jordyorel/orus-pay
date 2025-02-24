package models

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

const (
	TransactionTypeTopup          = "TOPUP"
	TransactionTypeWithdrawal     = "WITHDRAWAL"
	TransactionTypeQRPayment      = "qr_payment"
	TransactionTypeMerchantDirect = "merchant_direct"
	TransactionTypeMerchantScan   = "merchant_scan"
	TransactionTypeRefund         = "refund"
)

// Consolidated Transaction model
type Transaction struct {
	gorm.Model
	TransactionID    string  `json:"transaction_id"`
	Type             string  `json:"type"` // "qr", "direct", "card"
	SenderID         uint    `json:"sender_id"`
	ReceiverID       uint    `json:"receiver_id"`
	Amount           float64 `json:"amount"`
	Fee              float64
	Currency         string `json:"currency"`
	Status           string `json:"status"`
	Reference        string
	Description      string  `json:"description"`
	QRCodeID         *string `json:"qr_code_id,omitempty"`
	QRType           string  `json:"qr_type,omitempty"`
	QROwnerID        uint    `json:"qr_owner_id,omitempty"`
	QROwnerType      string  `json:"qr_owner_type,omitempty"`
	PaymentMethod    string  `json:"payment_method" gorm:"default:'qr'"`
	PaymentType      string  `json:"payment_type" gorm:"default:'direct'"`
	CardID           *uint
	MerchantID       *uint
	MerchantName     string
	MerchantCategory string
	Metadata         JSON `json:"metadata" gorm:"type:jsonb"`
	History          JSON `json:"history" gorm:"type:jsonb"`
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
