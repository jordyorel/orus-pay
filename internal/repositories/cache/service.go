package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"orus/internal/models"
	"time"

	"github.com/redis/go-redis/v9"
)

type CacheService struct {
	client *redis.Client
	ttl    time.Duration
}

func NewCacheService(client *redis.Client, defaultTTL time.Duration) *CacheService {
	return &CacheService{
		client: client,
		ttl:    defaultTTL,
	}
}

// Base operations
func (s *CacheService) Set(ctx context.Context, key string, value interface{}) error {
	return s.SetWithTTL(ctx, key, value, s.ttl)
}

func (s *CacheService) SetWithTTL(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal cache value: %w", err)
	}
	return s.client.Set(ctx, key, data, ttl).Err()
}

func (s *CacheService) Get(ctx context.Context, key string, dest interface{}) (bool, error) {
	data, err := s.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, fmt.Errorf("failed to get cache value: %w", err)
	}

	if err := json.Unmarshal(data, dest); err != nil {
		return false, fmt.Errorf("failed to unmarshal cache value: %w", err)
	}
	return true, nil
}

func (s *CacheService) Delete(ctx context.Context, keys ...string) error {
	return s.client.Del(ctx, keys...).Err()
}

// Key generation
func (s *CacheService) GenerateKey(entityType, keyType string, value interface{}) string {
	return fmt.Sprintf("%s:%s:%v", entityType, keyType, value)
}

// User caching
func (s *CacheService) CacheUser(ctx context.Context, user *models.User) error {
	if user == nil {
		return errors.New("cannot cache nil user")
	}

	keys := []string{
		s.GenerateKey("user", "id", user.ID),
		s.GenerateKey("user", "email", user.Email),
	}
	if user.Phone != "" {
		keys = append(keys, s.GenerateKey("user", "phone", user.Phone))
	}

	for _, key := range keys {
		if err := s.Set(ctx, key, user); err != nil {
			return err
		}
	}
	return nil
}

func (s *CacheService) GetUser(ctx context.Context, key string) (*models.User, error) {
	var user models.User
	found, err := s.Get(ctx, key, &user)
	if err != nil || !found {
		if !found {
			return nil, errors.New("user not found in cache")
		}
		return nil, err
	}
	return &user, nil
}

// Wallet caching
func (s *CacheService) CacheWallet(ctx context.Context, wallet *models.Wallet) error {
	key := s.GenerateKey("wallet", "user", wallet.UserID)
	return s.Set(ctx, key, wallet)
}

func (s *CacheService) GetWallet(ctx context.Context, userID uint) (*models.Wallet, error) {
	key := s.GenerateKey("wallet", "user", userID)
	var wallet models.Wallet
	found, err := s.Get(ctx, key, &wallet)
	if err != nil || !found {
		return nil, err
	}
	return &wallet, nil
}

// Invalidation patterns
func (s *CacheService) InvalidateUser(ctx context.Context, userID uint) error {
	user, err := s.GetUser(ctx, s.GenerateKey("user", "id", userID))
	if err != nil {
		return err
	}

	keys := []string{
		s.GenerateKey("user", "id", userID),
		s.GenerateKey("user", "email", user.Email),
	}
	if user.Phone != "" {
		keys = append(keys, s.GenerateKey("user", "phone", user.Phone))
	}

	return s.Delete(ctx, keys...)
}

func (s *CacheService) InvalidateWallet(ctx context.Context, userID uint) error {
	return s.Delete(ctx, s.GenerateKey("wallet", "user", userID))
}

// FlushAll flushes all keys from the cache
func (s *CacheService) FlushAll(ctx context.Context) error {
	return s.client.FlushAll(ctx).Err()
}

// Close closes the Redis client connection
func (s *CacheService) Close() error {
	return s.client.Close()
}
