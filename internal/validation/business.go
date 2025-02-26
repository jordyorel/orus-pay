package validation

import (
	"orus/internal/models"
	"orus/internal/services/transaction"
	"strconv"
	"time"
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
	if req.Amount <= 0 {
		v.AddError("amount", "must be greater than 0")
	}
	if req.RecipientID == 0 {
		v.AddError("recipient_id", "is required")
	}
	if req.PaymentType == "" {
		v.AddError("payment_type", "is required")
	}
}

// UserRegistration validates user registration data
func (v *Validator) UserRegistration(input *models.CreateUserInput) {
	if !emailRegex.MatchString(input.Email) {
		v.AddError("email", "invalid format")
	}
	if !phoneRegex.MatchString(input.Phone) {
		v.AddError("phone", "invalid format")
	}
	if len(input.Password) < 8 || !HasSpecialChar(input.Password) {
		v.AddError("password", "must be at least 8 characters and contain special characters")
	}
	if !isValidRole(input.Role) {
		v.AddError("role", "must be one of: user, merchant, enterprise")
	}
}

// CardValidation validates credit card data
func (v *Validator) CardValidation(card *models.CreditCard) {
	if !isValidCardNumber(card.CardNumber) {
		v.AddError("card_number", "invalid number")
	}
	if !isValidExpiryDate(card.ExpiryMonth, card.ExpiryYear) {
		v.AddError("expiry_date", "invalid date")
	}
}

// QRPayment validates QR payment requests
func (v *Validator) QRPayment(input *models.QRPaymentRequest) {
	v.Required("qr_code", input.QRCode)
	v.Range("amount", input.Amount, 0.01, 1000000)

	if input.Amount <= 0 {
		v.AddError("amount", "must be greater than 0")
	}
}

// Transfer validates money transfer requests
func (v *Validator) Transfer(req *transaction.TransferRequest) {
	if req.ReceiverID == 0 {
		v.AddError("receiver_id", "must not be zero")
		return
	}

	v.Range("amount", req.Amount, 0.01, 1000000)

	if req.ReceiverID == req.SenderID {
		v.AddError("receiver_id", "cannot transfer to self")
	}
}

// Helper functions
func isValidRole(role string) bool {
	validRoles := map[string]bool{
		"user":       true,
		"merchant":   true,
		"enterprise": true,
	}
	return validRoles[role]
}

func isValidCardNumber(number string) bool {
	sum := 0
	isSecond := false

	for i := len(number) - 1; i >= 0; i-- {
		d := int(number[i] - '0')

		if isSecond {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}

		sum += d
		isSecond = !isSecond
	}

	return sum%10 == 0
}

func isValidExpiryDate(month, year string) bool {
	m, err := strconv.Atoi(month)
	if err != nil || m < 1 || m > 12 {
		return false
	}

	y, err := strconv.Atoi(year)
	if err != nil {
		return false
	}

	now := time.Now()
	expiry := time.Date(y, time.Month(m), 1, 0, 0, 0, 0, time.UTC)
	return expiry.After(now)
}
