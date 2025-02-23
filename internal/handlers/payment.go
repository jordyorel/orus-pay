package handlers

import (
	"encoding/base64"
	"fmt"
	"log"
	"orus/internal/models"
	"orus/internal/repositories"
	"orus/internal/services"
	"orus/internal/utils/response"
	"orus/internal/utils/validation"

	"github.com/gofiber/fiber/v2"
	qrcode "github.com/skip2/go-qrcode"
)

type PaymentHandler struct {
	qrService          *services.QRService
	transactionService *services.TransactionService
}

func NewPaymentHandler() *PaymentHandler {
	return &PaymentHandler{
		qrService:          services.NewQRService(),
		transactionService: services.NewTransactionService(),
	}
}

// Generate a QR code for payment request
func GeneratePaymentQR(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uint)
	if !ok || userID == 0 {
		return c.Status(401).JSON(fiber.Map{"error": "Unauthorized"})
	}

	var request struct {
		Amount float64 `json:"amount"`
	}

	if err := c.BodyParser(&request); err != nil {
		log.Println("Error parsing request:", err)
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request format"})
	}

	if request.Amount <= 0 {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid amount"})
	}

	// Create a unique QR Code ID
	qrCodeID := fmt.Sprintf("QR-%d-%s", userID, services.GenerateUniqueID())

	// Save transaction with status "pending"
	transaction := models.Transaction{
		SenderID:   0, // Sender is unknown at this point
		ReceiverID: userID,
		Amount:     request.Amount,
		Status:     "pending",
		QRCodeID:   qrCodeID,
	}

	if err := repositories.CreateTransaction(&transaction); err != nil {
		log.Println("Error creating transaction:", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to create transaction"})
	}

	// Generate QR code containing the transaction ID
	qrData := fmt.Sprintf("orus://pay?qr_id=%s", qrCodeID)
	qrBytes, err := qrcode.Encode(qrData, qrcode.Medium, 256)
	if err != nil {
		log.Println("Error generating QR code:", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to generate QR code"})
	}

	// Convert QR code to Base64
	qrBase64 := base64.StdEncoding.EncodeToString(qrBytes)

	log.Printf("QR Code generated for User %d, Amount: %.2f", userID, request.Amount)

	return c.JSON(fiber.Map{
		"message":        "QR code generated",
		"transaction_id": transaction.ID,
		"amount":         request.Amount,
		"qr_code_base64": qrBase64,
		"qr_code_id":     qrCodeID,
	})
}

func ProcessPaymentQR(c *fiber.Ctx) error {
	// Extract payer ID from JWT
	payerID, ok := c.Locals("userID").(uint)
	if !ok || payerID == 0 {
		return c.Status(401).JSON(fiber.Map{"error": "Unauthorized"})
	}

	var request struct {
		QRCodeID string `json:"qr_code_id"`
	}

	if err := c.BodyParser(&request); err != nil {
		log.Println("Error parsing request body:", err)
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request format"})
	}

	// Find transaction by QR code ID
	transaction, err := repositories.GetTransactionByQRCode(request.QRCodeID)
	if err != nil {
		log.Println("Transaction not found for QR:", request.QRCodeID)
		return c.Status(404).JSON(fiber.Map{"error": "Transaction not found"})
	}

	// Ensure transaction is still valid
	if transaction.Status != "pending" {
		return c.Status(400).JSON(fiber.Map{"error": "Transaction already completed or expired"})
	}

	// Ensure payer is not the same as recipient
	if payerID == transaction.ReceiverID {
		return c.Status(400).JSON(fiber.Map{"error": "You cannot pay yourself"})
	}

	// Fetch payer's and recipient's wallets
	payerWallet, err := repositories.GetWalletByUserID(payerID)
	if err != nil || payerWallet.Balance < transaction.Amount {
		return c.Status(400).JSON(fiber.Map{"error": "Insufficient balance"})
	}

	receiverWallet, err := repositories.GetWalletByUserID(transaction.ReceiverID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Recipient wallet not found"})
	}

	// Perform transaction inside DB transaction
	tx := repositories.DB.Begin()

	// Deduct from payer
	payerWallet.Balance -= transaction.Amount
	if err := tx.Save(&payerWallet).Error; err != nil {
		tx.Rollback()
		return c.Status(500).JSON(fiber.Map{"error": "Failed to process transaction"})
	}

	// Credit recipient
	receiverWallet.Balance += transaction.Amount
	if err := tx.Save(&receiverWallet).Error; err != nil {
		tx.Rollback()
		return c.Status(500).JSON(fiber.Map{"error": "Failed to process transaction"})
	}

	// Update transaction details
	transaction.SenderID = payerID
	transaction.Status = "completed"
	if err := tx.Save(&transaction).Error; err != nil {
		tx.Rollback()
		return c.Status(500).JSON(fiber.Map{"error": "Failed to update transaction"})
	}

	tx.Commit()

	log.Printf("User %d paid %.2f to User %d using QR Code", payerID, transaction.Amount, transaction.ReceiverID)

	return c.JSON(fiber.Map{
		"message": "Payment successful",
		"transaction": fiber.Map{
			"id":          transaction.ID,
			"amount":      transaction.Amount,
			"sender_id":   transaction.SenderID,
			"receiver_id": transaction.ReceiverID,
			"status":      transaction.Status,
		},
		"updated_balance": payerWallet.Balance,
	})
}

// ProcessQRPayment handles both merchant and user QR code scanning
func (h *PaymentHandler) ProcessQRPayment(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*models.UserClaims)

	var input struct {
		QRCode      string  `json:"qr_code" validate:"required"`
		Amount      float64 `json:"amount" validate:"required,gt=0"`
		Description string  `json:"description"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request format"})
	}

	// Process payment through QR service
	tx, err := services.NewQRService().ProcessQRPayment(input.QRCode, claims.UserID, input.Amount, map[string]interface{}{
		"description": input.Description,
	})
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"message":     "Payment successful",
		"transaction": tx,
	})
}

// SendMoney handles direct user-to-user transfers
func (h *PaymentHandler) SendMoney(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*models.UserClaims)

	var input struct {
		ReceiverID  uint    `json:"receiver_id" validate:"required"`
		Amount      float64 `json:"amount" validate:"required,gt=0"`
		Description string  `json:"description"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request format"})
	}

	tx, err := services.NewTransactionService().ProcessP2PTransfer(claims.UserID, input.ReceiverID, input.Amount, input.Description)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"message":     "Transfer successful",
		"transaction": tx,
	})
}

func (h *PaymentHandler) GeneratePaymentQR(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*models.UserClaims)

	var input struct {
		Amount float64 `json:"amount" validate:"required,gt=0"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request format"})
	}

	qr, err := services.NewQRService().GeneratePaymentQR(claims.UserID, input.Amount)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to generate QR code"})
	}

	return c.JSON(fiber.Map{
		"message": "QR code generated",
		"qr_code": qr,
		"amount":  input.Amount,
	})
}

func (h *PaymentHandler) ProcessPayment(c *fiber.Ctx) error {
	var req models.PaymentRequest
	if err := c.BodyParser(&req); err != nil {
		return response.ValidationError(c, "Invalid request format")
	}

	v := validation.New()
	v.Payment(&req) // Use the Payment validation method

	if !v.Valid() {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errors": v.Errors,
		})
	}

	// Get user ID from context
	claims := c.Locals("claims").(*models.UserClaims)
	if claims == nil {
		return response.Unauthorized(c)
	}

	// Create transaction request
	tx := &models.Transaction{
		Type:        req.PaymentType,
		SenderID:    claims.UserID,
		ReceiverID:  req.RecipientID,
		Amount:      req.Amount,
		Description: req.Description,
		Status:      "pending",
	}

	// Process transaction
	result, err := h.transactionService.ProcessTransaction(tx)
	if err != nil {
		return response.ServerError(c, err.Error())
	}

	return response.Success(c, "Payment processed successfully", result)
}
