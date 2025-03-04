package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"orus/internal/models"

	"github.com/redis/go-redis/v9"
)

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

func NewRedisClient(cfg *RedisConfig) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})
}

type RedisCache struct {
	client *redis.Client
}

func NewRedisCache(client *redis.Client) *RedisCache {
	return &RedisCache{client: client}
}

// Methods for wallet.CacheOperator interface (no context)
func (c *RedisCache) Get(key string) (interface{}, error) {
	return c.client.Get(context.Background(), key).Result()
}

func (c *RedisCache) Set(key string, value interface{}, expiration time.Duration) error {
	return c.client.Set(context.Background(), key, value, expiration).Err()
}

func (c *RedisCache) Delete(key string) error {
	return c.client.Del(context.Background(), key).Err()
}

// Methods for repositories.CacheRepository interface (with context)
func (c *RedisCache) GetWithContext(ctx context.Context, key string) (interface{}, error) {
	return c.client.Get(ctx, key).Result()
}

func (c *RedisCache) SetWithContext(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return c.client.Set(ctx, key, value, expiration).Err()
}

func (c *RedisCache) DeleteWithContext(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

// Update InvalidateWallet to use non-context Delete
func (c *RedisCache) InvalidateWallet(ctx context.Context, userID uint) error {
	return c.Delete(walletKey(userID))
}

func (c *RedisCache) DeleteMany(ctx context.Context, pattern string) error {
	keys, err := c.client.Keys(ctx, pattern).Result()
	if err != nil {
		return err
	}
	if len(keys) > 0 {
		return c.client.Del(ctx, keys...).Err()
	}
	return nil
}

func (c *RedisCache) GetWallet(ctx context.Context, userID uint) (*models.Wallet, error) {
	key := fmt.Sprintf("wallet:%d", userID)
	val, err := c.client.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var wallet models.Wallet
	err = json.Unmarshal([]byte(val), &wallet)
	return &wallet, err
}

func (c *RedisCache) SetWallet(ctx context.Context, wallet *models.Wallet) error {
	key := walletKey(wallet.UserID)
	data, err := json.Marshal(wallet)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, data, 24*time.Hour).Err()
}

func walletKey(userID uint) string {
	return fmt.Sprintf("wallet:%d", userID)
}

// Add health check
func (s *CacheService) HealthCheck(ctx context.Context) error {
	if err := s.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis connection failed: %w", err)
	}
	return nil
}

// Add stats collection
func (s *CacheService) GetStats(ctx context.Context) *redis.PoolStats {
	return s.client.PoolStats()
}
