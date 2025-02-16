package main

import (
	"log"
	"orus/internal/config"
	"orus/internal/handlers"
	"orus/internal/repositories"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func main() {
	// Load environment variables
	config.LoadEnv()

	// Initialize databases (PostgreSQL + Redis)
	repositories.InitDB()
	defer func() {
		// Close PostgreSQL connection
		if repositories.DB != nil {
			sqlDB, err := repositories.DB.DB()
			if err != nil {
				log.Printf("⚠️ Failed to get database instance: %v", err)
				return
			}
			if err := sqlDB.Close(); err != nil {
				log.Printf("⚠️ Failed to close database connection: %v", err)
			}
		}

		// Close Redis connection
		if repositories.RedisClient != nil {
			if err := repositories.RedisClient.Close(); err != nil {
				log.Printf("⚠️ Failed to close Redis connection: %v", err)
			}
		}
	}()

	// Create Fiber app
	app := fiber.New()

	// CORS middleware
	// app.Use(cors.New(cors.Config{
	// 	AllowOrigins:     "https://example.com, https://anotherdomain.com", // Add your allowed origins
	// 	AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
	// 	AllowMethods:     "GET, POST, PUT, DELETE, OPTIONS",
	// 	AllowCredentials: true, // Set to true if you're using cookies or Authorization headers
	// }))

	// Middleware
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${latency} ${method} ${path}\n",
	}))

	// Rate limiting setup remains the same
	app.Use("/api/register", limiter.New(limiter.Config{
		Max:        5,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(429).JSON(fiber.Map{
				"error": "Too many requests. Please try again later.",
			})
		},
	}))

	app.Use("/api/login", limiter.New(limiter.Config{
		Max:        5,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(429).JSON(fiber.Map{
				"error": "Too many requests. Please try again later.",
			})
		},
	}))

	// Routes
	handlers.SetupRoutes(app)

	// Start server
	log.Fatal(app.Listen(":" + config.GetEnv("PORT", "3000")))
}
