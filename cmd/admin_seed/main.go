package main

import (
	"context"
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

		if repositories.CacheService != nil {
			if err := repositories.CacheService.Close(); err != nil {
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

	log.Printf("Admin user created with ID: %d", adminUser.ID)

	if repositories.CacheService != nil {
		if err := repositories.InvalidateUserCache(adminUser.ID); err != nil {
			log.Printf("Warning: Failed to invalidate admin user cache: %v", err)
		}

		emailKey := repositories.CacheService.GenerateKey("user", "email", adminEmail)
		phoneKey := repositories.CacheService.GenerateKey("user", "phone", adminPhone)
		if err := repositories.CacheService.Delete(context.Background(), emailKey, phoneKey); err != nil {
			log.Printf("Warning: Failed to invalidate admin user email/phone cache: %v", err)
		}

		log.Println("Admin user cache invalidated")
	}

	log.Println("✅ Admin account created successfully!")
}
