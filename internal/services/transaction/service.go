package transaction

import (
	"context"
	"errors"
	"fmt"
	"math"
	"orus/internal/models"
	"orus/internal/repositories"
	"orus/internal/services/wallet"

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

	// Get user role from context
	roleVal := ctx.Value(wallet.UserRoleContextKey)
	role, ok := roleVal.(string)
	if !ok || role == "" {
		role = "user" // Default to user role
	}
	// Create new context with role for wallet operations
	ctxWithRole := context.WithValue(ctx, wallet.UserRoleContextKey, role)

	// Risk assessment
	riskScore := s.riskService.AssessTransaction(tx)
	if riskScore > highRiskThreshold {
		return nil, ErrHighRiskTransaction
	}

	// Process based on transaction type
	switch tx.Type {
	case models.TransactionTypeMerchantDirect, models.TransactionTypeQRPayment, "P2P_TRANSFER", "transfer":
		if err := s.db.Transaction(func(txn *gorm.DB) error {
			if err := s.walletService.Debit(ctxWithRole, tx.SenderID, tx.Amount); err != nil {
				return err
			}
			if err := s.walletService.Credit(ctxWithRole, tx.ReceiverID, tx.Amount); err != nil {
				// Rollback debit if credit fails
				_ = s.walletService.Credit(ctxWithRole, tx.SenderID, tx.Amount)
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
	// Create transaction
	tx := &models.Transaction{
		Type:        "transfer",
		SenderID:    req.SenderID,
		ReceiverID:  req.ReceiverID,
		Amount:      req.Amount,
		Description: req.Description,
		Status:      "pending",
		Metadata:    models.NewJSON(req.Metadata),
	}

	// Get user role from context and ensure it's passed through
	roleVal := ctx.Value(wallet.UserRoleContextKey)
	role, ok := roleVal.(string)
	if !ok || role == "" {
		role = "user" // Default to user role
	}
	// Create new context with role
	ctxWithRole := context.WithValue(ctx, wallet.UserRoleContextKey, role)

	// Process the transaction
	return s.ProcessTransaction(ctxWithRole, tx)
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
