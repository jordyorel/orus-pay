// Package repositories provides data access layer implementations.
// It handles all database operations and data persistence logic.
package repositories

import (
	"log"
	"orus/internal/config"
	"orus/internal/models"
	"os"
	"time"

	"orus/internal/repositories/cache"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB is the global database instance used across the application.
var DB *gorm.DB
var CacheService *cache.CacheService

// DBConfig holds database connection pool configuration
type DBConfig struct {
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

var dbConfig = DBConfig{
	MaxIdleConns:    10,
	MaxOpenConns:    100,
	ConnMaxLifetime: time.Hour,
	ConnMaxIdleTime: time.Minute * 30,
}

// InitDB initializes the database connection.
// It sets up the connection pool, performs migrations,
// and configures the database with proper settings.
func InitDB() error {
	initPostgres()

	// Initialize Redis with new config
	redisCfg := &cache.RedisConfig{
		Host:     config.GetEnv("REDIS_HOST", "localhost"),
		Port:     config.GetEnv("REDIS_PORT", "6379"),
		Password: config.GetEnv("REDIS_PASSWORD", ""),
		DB:       config.GetIntEnv("REDIS_DB", 0),
	}
	redisClient := cache.NewRedisClient(redisCfg)
	CacheService = cache.NewCacheService(redisClient, 24*time.Hour)

	// Auto-migrate the updated schema
	err := DB.AutoMigrate(
		&models.User{},
		&models.Wallet{},
		&models.Merchant{},    // Now includes limits
		&models.Transaction{}, // Consolidated transaction model
		&models.CreditCard{},
		&models.KYCVerification{},
		&models.Enterprise{}, // Consolidated enterprise model
		&models.QRCode{},
		&models.Dispute{},
	)

	if err != nil {
		return err
	}

	return nil
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

	// Apply the configuration from main.go
	sqlDB.SetMaxIdleConns(dbConfig.MaxIdleConns)
	sqlDB.SetMaxOpenConns(dbConfig.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(dbConfig.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(dbConfig.ConnMaxIdleTime)

	// Schema setup
	db.Exec("CREATE SCHEMA IF NOT EXISTS public;")
	db.Exec("GRANT ALL ON SCHEMA public TO postgres;")
	db.Exec("GRANT ALL ON SCHEMA public TO public;")
	db.Exec("SET search_path TO public;")

	// Configure GORM logger to ignore "record not found" errors
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Warn, // Only log warnings and errors
			IgnoreRecordNotFoundError: true,        // Ignore "record not found"
			Colorful:                  true,
		},
	)
	db.Logger = newLogger

	log.Println("✅ PostgreSQL connected & migrations applied successfully!")
}

func ResetDatabase() error {
	// Drop tables
	err := DB.Migrator().DropTable(&models.User{}, &models.Wallet{}, &models.QRCode{} /* other tables */)
	if err != nil {
		return err
	}

	// Run migrations
	return DB.AutoMigrate(&models.User{}, &models.Wallet{}, &models.QRCode{} /* other tables */)
}

func DropAllTables() error {
	// Drop tables
	err := DB.Migrator().DropTable(
		&models.User{},
		&models.Wallet{},
		&models.Transaction{},
		// ... other tables
	)
	if err != nil {
		return err
	}

	return nil
}
