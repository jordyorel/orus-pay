package transaction

import (
	"context"
	"errors"
	"fmt"
	"math"
	"orus/internal/models"
	"orus/internal/repositories"
	"time"

	"gorm.io/gorm"
)

var (
	ErrHighRiskTransaction = errors.New("transaction risk too high")
	highRiskThreshold      = 0.8
)

type Service interface {
	Process(ctx context.Context, tx *models.Transaction) error
	Rollback(ctx context.Context, tx *models.Transaction) error
	ProcessP2PTransfer(ctx context.Context, req TransferRequest) (*models.Transaction, error)
	ProcessTransaction(ctx context.Context, tx *models.Transaction) (*models.Transaction, error)
}

type service struct {
	db             *gorm.DB
	walletService  WalletService
	balanceService BalanceService
	cache          repositories.CacheRepository
	riskService    *RiskService
}

func NewService(
	db *gorm.DB,
	walletSvc WalletService,
	balanceSvc BalanceService,
	cache repositories.CacheRepository,
) Service {
	return &service{
		db:             db,
		walletService:  walletSvc,
		balanceService: balanceSvc,
		cache:          cache,
		riskService:    NewRiskService(),
	}
}

func (s *service) ProcessTransaction(ctx context.Context, tx *models.Transaction) (*models.Transaction, error) {
	// Round amount to 2 decimal places
	tx.Amount = math.Round(tx.Amount*100) / 100

	// Risk assessment
	riskScore := s.riskService.AssessTransaction(tx)
	if riskScore > highRiskThreshold {
		return nil, ErrHighRiskTransaction
	}

	// Process based on transaction type
	switch tx.Type {
	case models.TransactionTypeMerchantDirect, models.TransactionTypeQRPayment, "P2P_TRANSFER":
		if err := s.db.Transaction(func(txn *gorm.DB) error {
			if err := s.walletService.Debit(ctx, tx.SenderID, tx.Amount); err != nil {
				return err
			}
			if err := s.walletService.Credit(ctx, tx.ReceiverID, tx.Amount); err != nil {
				// Rollback debit if credit fails
				_ = s.walletService.Credit(ctx, tx.SenderID, tx.Amount)
				return err
			}
			return nil
		}); err != nil {
			return nil, err
		}
		tx.Status = "completed"

	default:
		return nil, fmt.Errorf("unsupported transaction type: %s", tx.Type)
	}

	if err := s.db.Create(tx).Error; err != nil {
		return nil, fmt.Errorf("failed to record transaction: %w", err)
	}

	return tx, nil
}

func (s *service) ProcessP2PTransfer(ctx context.Context, req TransferRequest) (*models.Transaction, error) {
	tx := &models.Transaction{
		TransactionID: fmt.Sprintf("TX-%d-%d", time.Now().Unix(), req.SenderID),
		Type:          "P2P_TRANSFER",
		SenderID:      req.SenderID,
		ReceiverID:    req.ReceiverID,
		Amount:        req.Amount,
		Description:   req.Description,
		Status:        "pending",
	}
	return s.ProcessTransaction(ctx, tx)
}

func (s *service) Process(ctx context.Context, tx *models.Transaction) error {
	if tx.Type == "debit" {
		return s.walletService.Process(ctx, tx)
	}
	return s.walletService.Process(ctx, tx)
}

func (s *service) Rollback(ctx context.Context, tx *models.Transaction) error {
	return s.walletService.Rollback(ctx, tx)
}

type RiskService struct{}

func NewRiskService() *RiskService {
	return &RiskService{}
}

func (s *RiskService) AssessTransaction(tx *models.Transaction) float64 {
	var riskScore float64 = 0.0
	if tx.Amount > 10000 {
		riskScore += 0.3
	}
	return riskScore
}
