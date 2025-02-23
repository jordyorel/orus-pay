package models

type PaymentRequest struct {
	Amount      float64 `json:"amount" validate:"required,gt=0"`
	RecipientID uint    `json:"recipient_id" validate:"required"`
	Description string  `json:"description"`
	PaymentType string  `json:"payment_type" validate:"required,oneof=wallet card qr"`
}

const (
	PaymentTypeWallet = "wallet"
	PaymentTypeCard   = "card"
	PaymentTypeQR     = "qr"
)
