package main

import (
	"log"
	"os"

	"orus/internal/config"
	"orus/internal/models"
	"orus/internal/repositories"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	config.LoadEnv()

	adminEmail := os.Getenv("ADMIN_EMAIL")
	adminPassword := os.Getenv("ADMIN_PASSWORD")
	adminPhone := os.Getenv("ADMIN_PHONE")

	if adminEmail == "" || adminPassword == "" || adminPhone == "" {
		log.Fatal("ADMIN_EMAIL, ADMIN_PASSWORD, and ADMIN_PHONE must be set in environment")
	}

	repositories.InitDB()
	defer func() {
		if repositories.DB != nil {
			sqlDB, err := repositories.DB.DB()
			if err != nil {
				log.Printf("⚠️ Failed to get SQL DB instance: %v", err)
			} else {
				if err := sqlDB.Close(); err != nil {
					log.Printf("⚠️ Failed to close PostgreSQL connection: %v", err)
				}
			}
		}

		if repositories.RedisClient != nil {
			if err := repositories.RedisClient.Close(); err != nil {
				log.Printf("⚠️ Failed to close Redis connection: %v", err)
			}
		}
	}()

	var existingAdmin models.User
	result := repositories.DB.Where("email = ?", adminEmail).First(&existingAdmin)
	if result.Error == nil {
		log.Println("Admin user already exists")
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal("Failed to hash password:", err)
	}

	adminUser := models.User{
		Email:        adminEmail,
		Password:     string(hashedPassword),
		Phone:        adminPhone,
		Role:         "admin",
		TokenVersion: 1,
	}

	if err := repositories.DB.Create(&adminUser).Error; err != nil {
		log.Fatal("Failed to create admin user:", err)
	}

	if repositories.RedisClient != nil {
		repositories.RedisClient.Del(repositories.RedisCtx,
			repositories.GetUserCacheKeyByEmail(adminEmail),
			repositories.GetUserCacheKeyByPhone(adminPhone),
		)
	}

	log.Println("✅ Admin account created successfully!")
}
