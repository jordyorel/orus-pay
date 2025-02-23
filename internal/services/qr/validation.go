package qr

import (
	"context"
	"fmt"
	"orus/internal/models"
	"orus/internal/repositories"
	"time"
)

func (s *Service) validateAndLockQR(ctx context.Context, code string) (*models.QRCode, error) {
	var qr models.QRCode
	if err := s.db.WithContext(ctx).Set("gorm:for_update", true).
		Where("code = ?", code).
		First(&qr).Error; err != nil {
		return nil, fmt.Errorf("QR code not found: %w", err)
	}

	if qr.Status != "active" {
		return nil, ErrQRInactive
	}

	if qr.ExpiresAt != nil && time.Now().After(*qr.ExpiresAt) {
		return nil, ErrQRExpired
	}

	if qr.MaxUses > 0 && qr.UsageCount >= qr.MaxUses {
		return nil, ErrQRLimitExceeded
	}

	return &qr, nil
}

func (s *Service) checkLimits(ctx context.Context, qr *models.QRCode, amount float64) error {
	// Check daily limit
	if qr.DailyLimit != nil {
		daily, err := repositories.GetQRCodeDailyTotal(ctx, qr.ID)
		if err != nil {
			return err
		}
		if daily+amount > *qr.DailyLimit {
			return fmt.Errorf("daily limit exceeded: %v", *qr.DailyLimit)
		}
	}

	// Check monthly limit
	if qr.MonthlyLimit != nil {
		monthly, err := repositories.GetQRCodeMonthlyTotal(ctx, qr.ID)
		if err != nil {
			return err
		}
		if monthly+amount > *qr.MonthlyLimit {
			return fmt.Errorf("monthly limit exceeded: %v", *qr.MonthlyLimit)
		}
	}

	return nil
}

func (s *Service) updateQRUsage(ctx context.Context, qr *models.QRCode) error {
	qr.UsageCount++
	if qr.MaxUses > 0 && qr.UsageCount >= qr.MaxUses {
		qr.Status = "expired"
	}
	return s.db.WithContext(ctx).Save(qr).Error
}
