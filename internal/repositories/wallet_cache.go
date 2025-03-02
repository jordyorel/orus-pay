package repositories

import (
	"context"
	"encoding/json"
	"orus/internal/models"
	"time"
)

// GetCachedWallet retrieves a wallet from cache
func GetCachedWallet(key string) (*models.Wallet, error) {
	ctx := context.Background()
	data, err := RedisClient.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}

	var wallet models.Wallet
	err = json.Unmarshal(data, &wallet)
	if err != nil {
		return nil, err
	}

	return &wallet, nil
}

// CacheWallet stores a wallet in cache with the given TTL
func CacheWallet(key string, wallet *models.Wallet, ttl time.Duration) error {
	ctx := context.Background()
	data, err := json.Marshal(wallet)
	if err != nil {
		return err
	}

	return RedisClient.Set(ctx, key, data, ttl).Err()
}
