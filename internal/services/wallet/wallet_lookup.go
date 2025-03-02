package wallet

import (
	"context"
	"fmt"
	"log"
	"orus/internal/models"
	"orus/internal/repositories"
)

// FindWalletByUserID is a robust function to find a wallet by user ID
// It tries multiple methods and provides detailed logging
func FindWalletByUserID(userID uint) (*models.Wallet, error) {
	log.Printf("Looking up wallet for user ID: %d", userID)

	// Method 1: Direct database query (most reliable)
	var wallet models.Wallet
	err := repositories.DB.Where("user_id = ?", userID).First(&wallet).Error
	if err == nil {
		log.Printf("Found wallet via direct DB query - User ID: %d, Wallet ID: %d, Balance: %.2f",
			userID, wallet.ID, wallet.Balance)
		return &wallet, nil
	}

	log.Printf("Direct DB query failed for user ID %d: %v", userID, err)

	// Method 2: Try repository method
	repo := repositories.NewWalletRepository(repositories.DB)
	wallet2, err2 := repo.GetByUserID(userID)
	if err2 == nil {
		log.Printf("Found wallet via repository - User ID: %d, Wallet ID: %d, Balance: %.2f",
			userID, wallet2.ID, wallet2.Balance)
		return wallet2, nil
	}

	log.Printf("Repository lookup failed for user ID %d: %v", userID, err2)

	// Method 3: Check if user exists but has no wallet
	var user models.User
	if err := repositories.DB.Where("id = ?", userID).First(&user).Error; err == nil {
		log.Printf("User exists but has no wallet - User ID: %d", userID)
		// User exists but has no wallet - could create one here
		return nil, fmt.Errorf("user exists but has no wallet: %w", err)
	}

	// All methods failed
	log.Printf("All wallet lookup methods failed for user ID: %d", userID)
	return nil, fmt.Errorf("wallet not found for user ID %d", userID)
}

// EnsureWalletExists makes sure a wallet exists for the user, creating one if needed
func EnsureWalletExists(ctx context.Context, userID uint) (*models.Wallet, error) {
	wallet, err := FindWalletByUserID(userID)
	if err == nil {
		return wallet, nil
	}

	// Check if user exists
	var user models.User
	if err := repositories.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Create new wallet
	log.Printf("Creating new wallet for user ID: %d", userID)
	newWallet := &models.Wallet{
		UserID:   userID,
		Balance:  0,
		Status:   "active",
		Currency: "USD",
	}

	if err := repositories.DB.Create(newWallet).Error; err != nil {
		return nil, fmt.Errorf("failed to create wallet: %w", err)
	}

	log.Printf("Created new wallet - User ID: %d, Wallet ID: %d", userID, newWallet.ID)
	return newWallet, nil
}
