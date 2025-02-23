package repositories

import (
	"context"
	"orus/internal/models"
	"orus/internal/utils"
	"time"

	"gorm.io/gorm"
)

func CreateQRCode(qr *models.QRCode) (*models.QRCode, error) {
	if err := DB.Create(qr).Error; err != nil {
		return nil, err
	}
	return qr, nil
}

func GetQRCodeByCode(code string) (*models.QRCode, error) {
	var qr models.QRCode
	if err := DB.Where("code = ?", code).First(&qr).Error; err != nil {
		return nil, err
	}
	return &qr, nil
}

func GetQRCodeByCodeForUpdate(code string) (*models.QRCode, error) {
	var qr models.QRCode
	if err := DB.Set("gorm:for_update", true).Where("code = ?", code).First(&qr).Error; err != nil {
		return nil, err
	}
	return &qr, nil
}

func GetQRCodeDailyTotal(ctx context.Context, qrID uint) (float64, error) {
	var total float64
	today := time.Now().Truncate(24 * time.Hour)
	tomorrow := today.Add(24 * time.Hour)

	err := DB.WithContext(ctx).Model(&models.QRTransaction{}).
		Where("qr_code_id = ? AND created_at >= ? AND created_at < ?", qrID, today, tomorrow).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&total).Error

	return total, err
}

func GetQRCodeMonthlyTotal(ctx context.Context, qrID uint) (float64, error) {
	var total float64
	startOfMonth := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, -time.Now().UTC().Day()+1)
	startOfNextMonth := startOfMonth.AddDate(0, 1, 0)

	err := DB.WithContext(ctx).Model(&models.QRTransaction{}).
		Where("qr_code_id = ? AND created_at >= ? AND created_at < ?", qrID, startOfMonth, startOfNextMonth).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&total).Error

	return total, err
}

func GetUserStaticQR(userID uint) (*models.QRCode, error) {
	var qr models.QRCode
	err := DB.Where("user_id = ? AND user_type = ? AND type = ?",
		userID, "user", models.QRTypeStatic).First(&qr).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// If no static QR exists, create one
			qr = models.QRCode{
				Code:     utils.MustGenerateSecureCode(),
				UserID:   userID,
				UserType: "user",
				Type:     models.QRTypeStatic,
				Status:   "active",
				MaxUses:  -1, // Unlimited uses
			}

			// Set default limits
			dailyLimit := float64(1000)
			monthlyLimit := float64(5000)
			qr.DailyLimit = &dailyLimit
			qr.MonthlyLimit = &monthlyLimit

			if err := DB.Create(&qr).Error; err != nil {
				return nil, err
			}
			return &qr, nil
		}
		return nil, err
	}
	return &qr, nil
}

func GetUserPaymentQR(userID uint) (*models.QRCode, error) {
	var qr models.QRCode
	err := DB.Where("user_id = ? AND type = ?", userID, models.QRTypePayment).First(&qr).Error

	if err == gorm.ErrRecordNotFound {
		// Create new payment QR if not found
		qr = models.QRCode{
			Code:     utils.MustGenerateSecureCode(),
			UserID:   userID,
			UserType: "user",
			Type:     models.QRTypePayment,
			Status:   "active",
			MaxUses:  -1, // Never expires
		}

		if err := DB.Create(&qr).Error; err != nil {
			return nil, err
		}
		return &qr, nil
	} else if err != nil {
		return nil, err
	}

	return &qr, nil
}
