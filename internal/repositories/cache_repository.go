package repositories

import (
	"context"
	"time"
)

// CacheRepository defines the interface for cache operations
type CacheRepository interface {
	Get(ctx context.Context, key string) (interface{}, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	GetFloat64(ctx context.Context, key string) (float64, error)
	SetFloat64(ctx context.Context, key string, value float64, expiration time.Duration) error
	Delete(ctx context.Context, key string) error
	DeleteMany(ctx context.Context, pattern string) error
}

// Default cache expiration time
const DefaultExpiration = 24 * time.Hour
