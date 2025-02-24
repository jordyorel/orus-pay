package models

import "time"

type TransactionHistory struct {
	ID          uint `gorm:"primarykey"`
	WalletID    uint `gorm:"index"`
	Amount      float64
	Type        string  // credit/debit
	Balance     float64 // Balance after transaction
	Description string
	CreatedAt   time.Time
}
