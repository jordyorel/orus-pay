package repositories

import (
	"context"
	"orus/internal/models"
	"time"
)

// CacheRepository defines the interface for cache operations
type CacheRepository interface {
	// Generic operations
	Get(ctx context.Context, key string) (interface{}, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Delete(ctx context.Context, key string) error

	// Type-specific operations
	GetFloat64(ctx context.Context, key string) (float64, error)
	SetFloat64(ctx context.Context, key string, value float64, expiration time.Duration) error

	// Wallet-specific operations
	GetWallet(ctx context.Context, userID uint) (*models.Wallet, error)
	SetWallet(ctx context.Context, userID uint, wallet *models.Wallet) error
	DeleteWallet(ctx context.Context, userID uint) error
}

// Default cache expiration time
const DefaultExpiration = 24 * time.Hour
