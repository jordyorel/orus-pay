package transaction

import (
	"context"
	"errors"
	"fmt"
	"orus/internal/models"
	"orus/internal/services/wallet"
	"time"

	"gorm.io/gorm"
)

var (
	ErrInvalidAmount     = errors.New("invalid transaction amount")
	ErrInvalidParties    = errors.New("invalid transaction parties")
	ErrTransactionFailed = errors.New("transaction failed")
	ErrInvalidType       = errors.New("invalid transaction type")
)

type TransactionType string

const (
	TypeTransfer   TransactionType = "TRANSFER"
	TypePayment    TransactionType = "PAYMENT"
	TypeRefund     TransactionType = "REFUND"
	TypeWithdrawal TransactionType = "WITHDRAWAL"
	TypeDeposit    TransactionType = "DEPOSIT"
)

type ProcessorConfig struct {
	DB            *gorm.DB
	WalletService wallet.Service
}

type Processor struct {
	db            *gorm.DB
	walletService wallet.Service
}

func NewProcessor(config ProcessorConfig) *Processor {
	if config.DB == nil {
		panic("db is required")
	}
	if config.WalletService == nil {
		panic("wallet service is required")
	}

	return &Processor{
		db:            config.DB,
		walletService: config.WalletService,
	}
}

type TransactionRequest struct {
	Type        TransactionType
	SenderID    uint
	ReceiverID  uint
	Amount      float64
	Description string
	Metadata    map[string]interface{}
	Reference   string
}

func (p *Processor) Process(ctx context.Context, req TransactionRequest) (*models.Transaction, error) {
	if err := p.validateRequest(req); err != nil {
		return nil, err
	}

	tx := &models.Transaction{
		TransactionID: fmt.Sprintf("TX-%d-%d", time.Now().Unix(), req.SenderID),
		Type:          string(req.Type),
		SenderID:      req.SenderID,
		ReceiverID:    req.ReceiverID,
		Amount:        req.Amount,
		Description:   req.Description,
		Status:        "pending",
		Metadata:      req.Metadata,
	}

	err := p.db.Transaction(func(dtx *gorm.DB) error {
		// Debit sender
		if req.SenderID != 0 {
			if err := p.walletService.Debit(ctx, req.SenderID, req.Amount); err != nil {
				return fmt.Errorf("failed to debit sender: %w", err)
			}
		}

		// Credit receiver
		if req.ReceiverID != 0 {
			if err := p.walletService.Credit(ctx, req.ReceiverID, req.Amount); err != nil {
				// Rollback sender debit if credit fails
				if req.SenderID != 0 {
					if rbErr := p.walletService.Credit(ctx, req.SenderID, req.Amount); rbErr != nil {
						return fmt.Errorf("critical error: credit failed and rollback failed: %v, %v", err, rbErr)
					}
				}
				return fmt.Errorf("failed to credit receiver: %w", err)
			}
		}

		tx.Status = "completed"
		return dtx.Save(tx).Error
	})

	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTransactionFailed, err)
	}

	return tx, nil
}

func (p *Processor) validateRequest(req TransactionRequest) error {
	if req.Amount <= 0 {
		return ErrInvalidAmount
	}

	if req.SenderID == 0 && req.ReceiverID == 0 {
		return ErrInvalidParties
	}

	switch req.Type {
	case TypeTransfer, TypePayment, TypeRefund, TypeWithdrawal, TypeDeposit:
		return nil
	default:
		return ErrInvalidType
	}
}
