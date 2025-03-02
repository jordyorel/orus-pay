package repositories

import (
	"context"
	"encoding/json"
	"fmt"
	"orus/internal/models"
	"strconv"
	"time"

	"log"

	"github.com/redis/go-redis/v9"
)

type RedisCacheRepository struct {
	client *redis.Client
}

func NewRedisCacheRepository(client *redis.Client) CacheRepository {
	return &RedisCacheRepository{
		client: client,
	}
}

func (r *RedisCacheRepository) Get(ctx context.Context, key string) (interface{}, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var result interface{}
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (r *RedisCacheRepository) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return r.client.Set(ctx, key, data, expiration).Err()
}

func (r *RedisCacheRepository) GetFloat64(ctx context.Context, key string) (float64, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		return 0, err
	}

	return strconv.ParseFloat(val, 64)
}

func (r *RedisCacheRepository) SetFloat64(ctx context.Context, key string, value float64, expiration time.Duration) error {
	return r.client.Set(ctx, key, fmt.Sprintf("%f", value), expiration).Err()
}

func (r *RedisCacheRepository) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

func (r *RedisCacheRepository) DeleteMany(ctx context.Context, pattern string) error {
	iter := r.client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		if err := r.client.Del(ctx, iter.Val()).Err(); err != nil {
			return err
		}
	}
	return iter.Err()
}

func (r *RedisCacheRepository) GetWallet(ctx context.Context, userID uint) (*models.Wallet, error) {
	key := fmt.Sprintf("wallet:%d", userID)
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		// Cache miss
		LogCacheMiss(key)
		return nil, err
	}

	// Cache hit
	LogCacheHit(key)

	var wallet models.Wallet
	if err := json.Unmarshal(data, &wallet); err != nil {
		return nil, err
	}
	return &wallet, nil
}

func (r *RedisCacheRepository) SetWallet(ctx context.Context, userID uint, wallet *models.Wallet) error {
	key := fmt.Sprintf("wallet:%d", userID)
	data, err := json.Marshal(wallet)
	if err != nil {
		return err
	}

	// Log cache set operation with expiration time
	log.Printf("Cache SET: %s (expires in %s)", key, WalletExpiration)

	// Use the shorter expiration time for wallets
	return r.client.Set(ctx, key, data, WalletExpiration).Err()
}

func (r *RedisCacheRepository) DeleteWallet(ctx context.Context, userID uint) error {
	key := fmt.Sprintf("wallet:%d", userID)
	return r.client.Del(ctx, key).Err()
}
