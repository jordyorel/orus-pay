package repositories

import (
	"context"
	"log"
	"orus/internal/config"

	"github.com/redis/go-redis/v9"
)

var (
	RedisCtx    = context.Background()
	RedisClient *redis.Client
)

func InitRedis() {
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     config.GetEnv("REDIS_HOST", "localhost") + ":6379",
		Password: config.GetEnv("REDIS_PASSWORD", ""),
		DB:       0,
	})

	// Test connection
	if err := RedisClient.Ping(RedisCtx).Err(); err != nil {
		log.Fatalf("🔥 Failed to connect to Redis: %v", err)
	}
	log.Println("✅ Redis connected successfully")
}

func ClearAllCaches() error {
	return RedisClient.FlushAll(RedisCtx).Err()
}
