package merchant

// Input types for merchant operations
type UpdateMerchantInput struct {
	BusinessName    string  `json:"business_name"`
	BusinessType    string  `json:"business_type"`
	BusinessAddress string  `json:"business_address"`
	ProcessingFee   float64 `json:"processing_fee"`
	WebhookURL      string  `json:"webhook_url"`
}

type ChargeInput struct {
	Amount      float64 `json:"amount"`
	Description string  `json:"description"`
	PaymentType string  `json:"payment_type"`
	PaymentCode string  `json:"payment_code"`
}

type QRPaymentInput struct {
	QRCode      string                 `json:"qr_code"`
	Amount      float64                `json:"amount"`
	Description string                 `json:"description"`
	Metadata    map[string]interface{} `json:"metadata"`
}

type RefundInput struct {
	TransactionID string  `json:"transaction_id"`
	Amount        float64 `json:"amount"`
	Reason        string  `json:"reason"`
}
