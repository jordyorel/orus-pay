package repositories

import (
	"encoding/json"
	"log"
	"orus/internal/config"
	"orus/internal/models"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	DB *gorm.DB
)

func InitDB() error {
	initPostgres()
	InitRedis()

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
	)

	if err != nil {
		return err
	}

	log.Println("✅ Migrations applied successfully!")
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

	// Configure GORM logger to ignore "record not found" errors
	db.Logger = db.Logger.LogMode(logger.Silent) // Or use a custom logger configuration:

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

func ResetDatabase() error {
	// Drop tables
	err := DB.Migrator().DropTable(&models.User{}, &models.Wallet{}, &models.QRCode{} /* other tables */)
	if err != nil {
		return err
	}

	// Clear all caches
	if err := ClearAllCaches(); err != nil {
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

	// Clear all Redis cache
	err = RedisClient.FlushAll(RedisCtx).Err()
	if err != nil {
		return err
	}

	return nil
}
