package cache

import (
	"context"
	"strconv"
	"time"
)

// CacheAdapter implements both interfaces
type CacheAdapter struct {
	*RedisCache
}

func NewCacheAdapter(redis *RedisCache) *CacheAdapter {
	return &CacheAdapter{redis}
}

// wallet.CacheOperator methods
func (a *CacheAdapter) Get(key string) (interface{}, error) {
	return a.client.Get(context.Background(), key).Result()
}

func (a *CacheAdapter) Set(key string, value interface{}, expiration time.Duration) error {
	return a.client.Set(context.Background(), key, value, expiration).Err()
}

func (a *CacheAdapter) Delete(key string) error {
	return a.client.Del(context.Background(), key).Err()
}

// repositories.CacheRepository methods
func (a *CacheAdapter) GetWithContext(ctx context.Context, key string) (interface{}, error) {
	return a.client.Get(ctx, key).Result()
}

func (a *CacheAdapter) GetFloat64(ctx context.Context, key string) (float64, error) {
	val, err := a.client.Get(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	return strconv.ParseFloat(val, 64)
}

func (a *CacheAdapter) SetFloat64(ctx context.Context, key string, value float64, expiration time.Duration) error {
	return a.client.Set(ctx, key, value, expiration).Err()
}

func (a *CacheAdapter) SetWithContext(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return a.client.Set(ctx, key, value, expiration).Err()
}

func (a *CacheAdapter) DeleteWithContext(ctx context.Context, key string) error {
	return a.client.Del(ctx, key).Err()
}
