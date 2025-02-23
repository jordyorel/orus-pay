package models

type WalletOperation struct {
	UserID    uint
	Type      string
	Amount    float64
	Reference string
	Metadata  map[string]interface{}
}

const (
	WalletOperationCredit = "CREDIT"
	WalletOperationDebit  = "DEBIT"
)
