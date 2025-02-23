package validation

import (
	"orus/internal/models"
)

// Transaction validates a transaction request
func (v *Validator) Transaction(tx *models.Transaction) {
	v.Required("type", tx.Type)
	v.Required("amount", tx.Amount)
	v.Range("amount", tx.Amount, 0.01, 1000000) // Example limits

	if tx.SenderID == 0 && tx.ReceiverID == 0 {
		v.AddError("parties", "transaction must have at least one party")
	}

	if tx.SenderID == tx.ReceiverID && tx.SenderID != 0 {
		v.AddError("parties", "sender and receiver cannot be the same")
	}
}

// QRCode validates QR code generation request
func (v *Validator) QRCode(qr *models.QRCode) {
	v.Required("user_id", qr.UserID)
	v.Required("type", qr.Type)

	if qr.Amount != nil {
		v.Range("amount", *qr.Amount, 0.01, 1000000)
	}

	if qr.ExpiresAt != nil {
		v.Future("expires_at", *qr.ExpiresAt)
	}

	if qr.MaxUses != -1 { // -1 indicates unlimited uses
		v.Range("max_uses", float64(qr.MaxUses), 1, 1000)
	}
}

// Wallet validates wallet operations
func (v *Validator) Wallet(op *models.WalletOperation) {
	v.Required("user_id", op.UserID)
	v.Required("type", op.Type)
	v.Check(op.Type == models.WalletOperationCredit || op.Type == models.WalletOperationDebit,
		"type", "must be either CREDIT or DEBIT")
	v.Range("amount", op.Amount, 0.01, 1000000)
}

// Payment validates payment requests
func (v *Validator) Payment(req *models.PaymentRequest) {
	v.Required("amount", req.Amount)
	v.Range("amount", req.Amount, 0.01, 10000)
	v.Required("recipient_id", req.RecipientID)
	v.Required("payment_type", req.PaymentType)
	v.Check(
		req.PaymentType == models.PaymentTypeWallet ||
			req.PaymentType == models.PaymentTypeCard ||
			req.PaymentType == models.PaymentTypeQR,
		"payment_type",
		"must be wallet, card, or qr",
	)
}
