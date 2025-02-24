package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"orus/internal/models"

	"github.com/redis/go-redis/v9"
)

// NoContextCache implements wallet.CacheOperator
type NoContextCache struct {
	client *redis.Client
}

// ContextCache implements repositories.CacheRepository
type ContextCache struct {
	client *redis.Client
}

func NewCaches(client *redis.Client) (*NoContextCache, *ContextCache) {
	return &NoContextCache{client}, &ContextCache{client}
}

// NoContextCache implementation
func (c *NoContextCache) Get(key string) (interface{}, error) {
	return c.client.Get(context.Background(), key).Result()
}

func (c *NoContextCache) Set(key string, value interface{}, expiration time.Duration) error {
	return c.client.Set(context.Background(), key, value, expiration).Err()
}

func (c *NoContextCache) Delete(key string) error {
	return c.client.Del(context.Background(), key).Err()
}

// NoContextCache wallet methods
func (c *NoContextCache) GetWallet(ctx context.Context, userID uint) (*models.Wallet, error) {
	key := fmt.Sprintf("wallet:%d", userID)
	val, err := c.client.Get(context.Background(), key).Result()
	if err != nil {
		return nil, err
	}
	var wallet models.Wallet
	err = json.Unmarshal([]byte(val), &wallet)
	return &wallet, err
}

func (c *NoContextCache) InvalidateWallet(ctx context.Context, userID uint) error {
	key := fmt.Sprintf("wallet:%d", userID)
	return c.Delete(key)
}

func (c *NoContextCache) SetWallet(ctx context.Context, wallet *models.Wallet) error {
	key := fmt.Sprintf("wallet:%d", wallet.UserID)
	data, err := json.Marshal(wallet)
	if err != nil {
		return err
	}
	return c.client.Set(context.Background(), key, data, 24*time.Hour).Err()
}

// ContextCache implementation
func (c *ContextCache) Get(ctx context.Context, key string) (interface{}, error) {
	return c.client.Get(ctx, key).Result()
}

func (c *ContextCache) GetFloat64(ctx context.Context, key string) (float64, error) {
	val, err := c.client.Get(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	return strconv.ParseFloat(val, 64)
}

func (c *ContextCache) SetFloat64(ctx context.Context, key string, value float64, expiration time.Duration) error {
	return c.client.Set(ctx, key, value, expiration).Err()
}

func (c *ContextCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return c.client.Set(ctx, key, value, expiration).Err()
}

func (c *ContextCache) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

// ContextCache additional methods
func (c *ContextCache) DeleteMany(ctx context.Context, pattern string) error {
	keys, err := c.client.Keys(ctx, pattern).Result()
	if err != nil {
		return err
	}
	if len(keys) > 0 {
		return c.client.Del(ctx, keys...).Err()
	}
	return nil
}

func (c *ContextCache) GetWallet(ctx context.Context, userID uint) (*models.Wallet, error) {
	key := fmt.Sprintf("wallet:%d", userID)
	val, err := c.client.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	var wallet models.Wallet
	err = json.Unmarshal([]byte(val), &wallet)
	return &wallet, err
}
