package wallet

import (
	"context"
	"log"
	"orus/internal/repositories"
	"orus/internal/utils/cache"
)

// InvalidateWalletCache invalidates all cache entries for a wallet
func InvalidateWalletCache(walletID uint) {
	keys := cache.InvalidateWalletCache(walletID)
	for _, key := range keys {
		err := repositories.RedisClient.Del(context.Background(), key).Err()
		if err != nil {
			log.Printf("Error invalidating wallet cache for key %s: %v", key, err)
		} else {
			log.Printf("Invalidated wallet cache for key: %s", key)
		}
	}
}

// InvalidateUserWalletCache invalidates all cache entries for a user's wallet
func InvalidateUserWalletCache(userID uint) {
	keys := cache.InvalidateUserWalletCache(userID)
	for _, key := range keys {
		err := repositories.RedisClient.Del(context.Background(), key).Err()
		if err != nil {
			log.Printf("Error invalidating user wallet cache for key %s: %v", key, err)
		} else {
			log.Printf("Invalidated user wallet cache for key: %s", key)
		}
	}
}

// InvalidateTransactionParticipantsCache invalidates cache for all participants in a transaction
func InvalidateTransactionParticipantsCache(senderID, receiverID uint) {
	// Invalidate sender's wallet cache
	InvalidateUserWalletCache(senderID)

	// Invalidate receiver's wallet cache
	InvalidateUserWalletCache(receiverID)

	// Get wallet IDs and invalidate those too
	senderWallet, err := repositories.GetWalletByUserID(senderID)
	if err == nil && senderWallet != nil {
		InvalidateWalletCache(senderWallet.ID)
	}

	receiverWallet, err := repositories.GetWalletByUserID(receiverID)
	if err == nil && receiverWallet != nil {
		InvalidateWalletCache(receiverWallet.ID)
	}
}
