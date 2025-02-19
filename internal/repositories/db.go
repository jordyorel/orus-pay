package repositories

import (
	"context"
	"encoding/json"
	"log"
	"orus/internal/config"
	"orus/internal/models"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	DB          *gorm.DB
	RedisClient *redis.Client
	RedisCtx    = context.Background()
)

func InitDB() error {
	initPostgres()
	initRedis()

	// Auto-migrate the schema
	err := DB.AutoMigrate(
		&models.User{},
		&models.Wallet{},
		&models.Merchant{},
		&models.MerchantLimits{},
		&models.Transaction{},
		&models.CreateCreditCard{},
		&models.KYCVerification{},
		&models.MerchantBankAccount{},
		&models.Enterprise{},
		&models.EnterpriseLocation{},
		&models.EnterpriseAPIKey{},
	)

	return err
}

func initPostgres() {
	// First connect without database name to create it if needed
	initDsn := "host=" + config.GetEnv("DB_HOST", "localhost") +
		" user=" + config.GetEnv("DB_USER", "postgres") +
		" password=" + config.GetEnv("DB_PASSWORD", "postgres") +
		" port=5432 sslmode=disable"

	initDB, err := gorm.Open(postgres.Open(initDsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to postgres:", err)
	}

	// Create database if it doesn't exist
	dbName := config.GetEnv("DB_NAME", "orus")
	// initDB.Exec("CREATE DATABASE " + dbName + " WITH OWNER postgres;")

	// Close initial connection
	sqlDB, err := initDB.DB()
	if err != nil {
		log.Fatal("Failed to get database instance:", err)
	}
	sqlDB.Close()

	// Now connect to the actual database
	dsn := "host=" + config.GetEnv("DB_HOST", "localhost") +
		" user=" + config.GetEnv("DB_USER", "postgres") +
		" password=" + config.GetEnv("DB_PASSWORD", "postgres") +
		" dbname=" + dbName +
		" port=5432 sslmode=disable"

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	DB = db

	// Set up connection pooling
	sqlDB, err = db.DB()
	if err != nil {
		log.Fatal("Failed to get database instance:", err)
	}

	// Connection pooling configuration
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)

	// Schema setup
	db.Exec("CREATE SCHEMA IF NOT EXISTS public;")
	db.Exec("GRANT ALL ON SCHEMA public TO postgres;")
	db.Exec("GRANT ALL ON SCHEMA public TO public;")
	db.Exec("SET search_path TO public;")

	log.Println("✅ PostgreSQL connected & migrations applied successfully!")
}

func initRedis() {
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     config.GetEnv("REDIS_HOST", "redis") + ":6379",
		Password: config.GetEnv("REDIS_PASSWORD", ""),
		DB:       0,
	})

	_, err := RedisClient.Ping(RedisCtx).Result()
	if err != nil {
		log.Fatalf("🔥 Failed to connect to Redis: %v", err)
	}
	log.Println("✅ Redis connected successfully")
}

// Helper functions remain the same
func cacheGetUser(key string) (*models.User, error) {
	val, err := RedisClient.Get(RedisCtx, key).Result()
	if err != nil {
		return nil, err
	}

	var user models.User
	if err := json.Unmarshal([]byte(val), &user); err != nil {
		return nil, err
	}
	return &user, nil
}

func cacheSetUser(key string, user *models.User, expiration time.Duration) error {
	userBytes, err := json.Marshal(user)
	if err != nil {
		return err
	}
	return RedisClient.Set(RedisCtx, key, userBytes, expiration).Err()
}
