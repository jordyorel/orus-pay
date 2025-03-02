package repositories

import (
	"context"
	"fmt"
	"log"
	"orus/internal/config"
	"orus/internal/models"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	RedisCtx          = context.Background()
	RedisClient       *redis.Client
	cacheHits         int64
	cacheMisses       int64
	cacheWalletHits   int64
	cacheWalletMisses int64
	cacheUserHits     int64
	cacheUserMisses   int64
)

// RedisConfig holds all Redis connection configuration
type RedisConfig struct {
	Host         string
	Password     string
	DB           int
	PoolSize     int           // Maximum number of socket connections
	MinIdleConns int           // Minimum number of idle connections
	MaxConnAge   time.Duration // Maximum age of connections
	IdleTimeout  time.Duration // How long connections can be idle
	DialTimeout  time.Duration // Timeout for establishing new connections
	ReadTimeout  time.Duration // Timeout for socket reads
	WriteTimeout time.Duration // Timeout for socket writes
}

// NewRedisConfig creates a RedisConfig with values from environment or defaults
func NewRedisConfig() *RedisConfig {
	// Add validation for Redis URL
	redisHost := config.GetEnv("REDIS_HOST", "localhost")
	if redisHost == "" {
		log.Fatal("REDIS_HOST environment variable not set")
	}

	// Parse integer configs with fallbacks
	poolSize, err := strconv.Atoi(config.GetEnv("REDIS_POOL_SIZE", "10"))
	if err != nil {
		log.Printf("Warning: Invalid REDIS_POOL_SIZE, using default: %v", err)
		poolSize = 10
	}

	minIdleConns, err := strconv.Atoi(config.GetEnv("REDIS_MIN_IDLE_CONNS", "5"))
	if err != nil {
		log.Printf("Warning: Invalid REDIS_MIN_IDLE_CONNS, using default: %v", err)
		minIdleConns = 5
	}

	db, err := strconv.Atoi(config.GetEnv("REDIS_DB", "0"))
	if err != nil {
		log.Printf("Warning: Invalid REDIS_DB, using default: %v", err)
		db = 0
	}

	// Parse duration configs with fallbacks
	maxConnAge, err := time.ParseDuration(config.GetEnv("REDIS_MAX_CONN_AGE", "30m"))
	if err != nil {
		log.Printf("Warning: Invalid REDIS_MAX_CONN_AGE, using default: %v", err)
		maxConnAge = 30 * time.Minute
	}

	idleTimeout, err := time.ParseDuration(config.GetEnv("REDIS_IDLE_TIMEOUT", "5m"))
	if err != nil {
		log.Printf("Warning: Invalid REDIS_IDLE_TIMEOUT, using default: %v", err)
		idleTimeout = 5 * time.Minute
	}

	dialTimeout, err := time.ParseDuration(config.GetEnv("REDIS_DIAL_TIMEOUT", "5s"))
	if err != nil {
		log.Printf("Warning: Invalid REDIS_DIAL_TIMEOUT, using default: %v", err)
		dialTimeout = 5 * time.Second
	}

	readTimeout, err := time.ParseDuration(config.GetEnv("REDIS_READ_TIMEOUT", "3s"))
	if err != nil {
		log.Printf("Warning: Invalid REDIS_READ_TIMEOUT, using default: %v", err)
		readTimeout = 3 * time.Second
	}

	writeTimeout, err := time.ParseDuration(config.GetEnv("REDIS_WRITE_TIMEOUT", "3s"))
	if err != nil {
		log.Printf("Warning: Invalid REDIS_WRITE_TIMEOUT, using default: %v", err)
		writeTimeout = 3 * time.Second
	}

	return &RedisConfig{
		Host:         redisHost,
		Password:     config.GetEnv("REDIS_PASSWORD", ""),
		DB:           db,
		PoolSize:     poolSize,
		MinIdleConns: minIdleConns,
		MaxConnAge:   maxConnAge,
		IdleTimeout:  idleTimeout,
		DialTimeout:  dialTimeout,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	}
}

// InitRedis initializes the Redis client with connection pooling
func InitRedis() {
	cfg := NewRedisConfig()

	log.Printf("Connecting to Redis at %s (DB %d)", cfg.Host, cfg.DB)
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     cfg.Host + ":6379",
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// Add connection test
	_, err := RedisClient.Ping(RedisCtx).Result()
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	log.Println("✅ Redis connection verified")

	// Start periodic monitoring of Redis connection pool stats
	go monitorRedisPool()
}

// monitorRedisPool periodically logs Redis connection pool statistics
func monitorRedisPool() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		stats := RedisClient.PoolStats()
		cacheStats := GetCacheStats()

		log.Printf("Redis Pool Stats - Hits: %d, Misses: %d, Timeouts: %d, TotalConns: %d, IdleConns: %d, StaleConns: %d",
			stats.Hits, stats.Misses, stats.Timeouts, stats.TotalConns, stats.IdleConns, stats.StaleConns)

		log.Printf("Cache Hit Ratios - Overall: %.2f%%, Wallet: %.2f%%, User: %.2f%%",
			cacheStats["total"].(map[string]interface{})["ratio"],
			cacheStats["wallet"].(map[string]interface{})["ratio"],
			cacheStats["user"].(map[string]interface{})["ratio"])
	}
}

func ClearAllCaches() error {
	return RedisClient.FlushAll(RedisCtx).Err()
}

// Add these functions to ensure consistent cache key generation
func GetUserCacheKeyByID(userID uint) string {
	return fmt.Sprintf("user:id:%d", userID)
}

func GetUserCacheKeyByEmail(email string) string {
	return fmt.Sprintf("user:email:%s", email)
}

func GetUserCacheKeyByPhone(phone string) string {
	return fmt.Sprintf("user:phone:%s", phone)
}

// Improve the InvalidateUserCache function to be more thorough
func InvalidateUserCache(userID uint) error {
	log.Printf("Invalidating cache for user ID: %d", userID)

	// Get the user to find all identifiers
	var user models.User
	if err := DB.First(&user, userID).Error; err != nil {
		log.Printf("Error fetching user %d for cache invalidation: %v", userID, err)
		// Even if we can't get the user, try to invalidate by ID
		if err := RedisClient.Del(RedisCtx, GetUserCacheKeyByID(userID)).Err(); err != nil {
			log.Printf("Failed to delete cache key by ID for user %d: %v", userID, err)
		}
		return err
	}

	// Delete all cache keys for this user
	keys := []string{
		GetUserCacheKeyByID(userID),
	}

	if user.Email != "" {
		keys = append(keys, GetUserCacheKeyByEmail(user.Email))
	}

	if user.Phone != "" {
		keys = append(keys, GetUserCacheKeyByPhone(user.Phone))
	}

	// Add pattern-based keys too
	patternKeys, err := RedisClient.Keys(RedisCtx, fmt.Sprintf("user:%d:*", userID)).Result()
	if err == nil && len(patternKeys) > 0 {
		keys = append(keys, patternKeys...)
	}

	// Also check for any other user-related patterns
	otherPatterns := []string{
		fmt.Sprintf("user:*:%d", userID),
		fmt.Sprintf("user:*:%d:*", userID),
	}

	for _, pattern := range otherPatterns {
		patternKeys, err := RedisClient.Keys(RedisCtx, pattern).Result()
		if err == nil && len(patternKeys) > 0 {
			keys = append(keys, patternKeys...)
		}
	}

	// Delete all keys
	if len(keys) > 0 {
		if err := RedisClient.Del(RedisCtx, keys...).Err(); err != nil {
			log.Printf("Error deleting cache keys for user %d: %v", userID, err)
			return err
		}
		log.Printf("Successfully invalidated %d cache keys for user ID %d: %v", len(keys), userID, keys)
	} else {
		log.Printf("No cache keys found to invalidate for user ID %d", userID)
	}

	return nil
}

func LogCacheHit(key string) {
	atomic.AddInt64(&cacheHits, 1)
	log.Printf("Cache HIT: %s", key)

	// Track specific types
	if strings.HasPrefix(key, "wallet:") {
		atomic.AddInt64(&cacheWalletHits, 1)
	} else if strings.HasPrefix(key, "user:") {
		atomic.AddInt64(&cacheUserHits, 1)
	}
}

func LogCacheMiss(key string) {
	atomic.AddInt64(&cacheMisses, 1)
	log.Printf("Cache MISS: %s", key)

	// Track specific types
	if strings.HasPrefix(key, "wallet:") {
		atomic.AddInt64(&cacheWalletMisses, 1)
	} else if strings.HasPrefix(key, "user:") {
		atomic.AddInt64(&cacheUserMisses, 1)
	}
}

// Add a function to get cache statistics
func GetCacheStats() map[string]interface{} {
	hits := atomic.LoadInt64(&cacheHits)
	misses := atomic.LoadInt64(&cacheMisses)
	walletHits := atomic.LoadInt64(&cacheWalletHits)
	walletMisses := atomic.LoadInt64(&cacheWalletMisses)
	userHits := atomic.LoadInt64(&cacheUserHits)
	userMisses := atomic.LoadInt64(&cacheUserMisses)

	total := hits + misses
	walletTotal := walletHits + walletMisses
	userTotal := userHits + userMisses

	hitRatio := 0.0
	walletHitRatio := 0.0
	userHitRatio := 0.0

	if total > 0 {
		hitRatio = float64(hits) / float64(total) * 100
	}
	if walletTotal > 0 {
		walletHitRatio = float64(walletHits) / float64(walletTotal) * 100
	}
	if userTotal > 0 {
		userHitRatio = float64(userHits) / float64(userTotal) * 100
	}

	return map[string]interface{}{
		"total": map[string]interface{}{
			"hits":   hits,
			"misses": misses,
			"ratio":  hitRatio,
		},
		"wallet": map[string]interface{}{
			"hits":   walletHits,
			"misses": walletMisses,
			"ratio":  walletHitRatio,
		},
		"user": map[string]interface{}{
			"hits":   userHits,
			"misses": userMisses,
			"ratio":  userHitRatio,
		},
	}
}
