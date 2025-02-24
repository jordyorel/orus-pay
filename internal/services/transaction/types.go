package transaction

import (
	"orus/internal/models"
	"time"
)

// TransferRequest represents a P2P transfer request
type TransferRequest struct {
	SenderID       uint    `json:"sender_id,omitempty"`
	ReceiverID     uint    `json:"receiver_id" validate:"required"`
	Amount         float64 `json:"amount" validate:"required,gt=0"`
	Description    string  `json:"description"`
	Metadata       map[string]interface{}
	ProcessingMode string            // sync or async
	Callback       string            // webhook URL for async processing
	Options        map[string]string // Additional processing options
}

// TransactionConfig holds configuration for transaction processing
type TransactionConfig struct {
	MaxAmount       float64
	MinAmount       float64
	DailyLimit      float64
	MonthlyLimit    float64
	ProcessingDelay time.Duration
	RetryAttempts   int
	AllowedTypes    []string          // List of allowed transaction types
	BlockedUsers    []uint            // List of blocked user IDs
	RiskThreshold   float64           // Risk score threshold
	ValidationRules map[string]string // Custom validation rules
}

// TransactionResult represents the result of a transaction
type TransactionResult struct {
	Transaction    *models.Transaction
	Status         string
	Error          error
	Metadata       map[string]interface{}
	ProcessingTime time.Duration
	AttemptCount   int
	RiskScore      float64
}
