package models

// TransferRequest represents a transfer between wallets
type TransferRequest struct {
	SenderID    uint                   `json:"sender_id"`
	ReceiverID  uint                   `json:"receiver_id"`
	Amount      float64                `json:"amount"`
	Description string                 `json:"description"`
	Metadata    map[string]interface{} `json:"metadata"`
}

type PaymentRequest struct {
	Amount      float64 `json:"amount" validate:"required,gt=0"`
	RecipientID uint    `json:"recipient_id" validate:"required"`
	Description string  `json:"description"`
	PaymentType string  `json:"payment_type" validate:"required,oneof=wallet card qr"`
}

// QRPaymentRequest represents a QR code payment request
type QRPaymentRequest struct {
	QRCode      string                 `json:"qr_code"`
	Amount      float64                `json:"amount"`
	Description string                 `json:"description"`
	Metadata    map[string]interface{} `json:"metadata"`
}

const (
	PaymentTypeWallet = "wallet"
	PaymentTypeCard   = "card"
	PaymentTypeQR     = "qr"
)
