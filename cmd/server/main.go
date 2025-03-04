// Package main is the entry point for the application.
// It initializes all dependencies, sets up the HTTP server,
// and starts the application.
package main

import (
	"context"
	"log"
	"orus/internal/config"
	"strconv"

	// "orus/internal/handlers"
	"orus/internal/repositories"
	"orus/internal/routes"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

// main initializes and starts the HTTP server.
// It performs the following setup:
// - Loads configuration
// - Initializes database connection
// - Sets up dependency injection
// - Configures routes
// - Starts the HTTP server
func main() {
	// Load environment variables
	config.LoadEnv()

	// Initialize databases (PostgreSQL + Redis)
	repositories.InitDB()

	sqlDB, err := repositories.DB.DB()
	if err != nil {
		log.Fatalf("Failed to get database instance: %v", err)
	}

	maxIdleConns, _ := strconv.Atoi(config.GetEnv("DB_MAX_IDLE_CONNS", "10"))
	maxOpenConns, _ := strconv.Atoi(config.GetEnv("DB_MAX_OPEN_CONNS", "100"))
	connMaxLifetime, _ := time.ParseDuration(config.GetEnv("DB_CONN_MAX_LIFETIME", "1h"))
	connMaxIdleTime, _ := time.ParseDuration(config.GetEnv("DB_CONN_MAX_IDLE_TIME", "30m"))
	if err != nil {
		maxIdleConns = 10
		log.Printf("Invalid DB_MAX_IDLE_CONNS, using default: %d", maxIdleConns)
	}

	sqlDB.SetMaxIdleConns(maxIdleConns)       // Maximum number of idle connections
	sqlDB.SetMaxOpenConns(maxOpenConns)       // Maximum number of open connections
	sqlDB.SetConnMaxLifetime(connMaxLifetime) // Maximum lifetime of a connection
	sqlDB.SetConnMaxIdleTime(connMaxIdleTime) // Maximum idle time for a connection

	// Add a periodic check of connection pool stats
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			stats := sqlDB.Stats()
			log.Printf("DB Stats: Open=%d, Idle=%d, InUse=%d, WaitCount=%d, WaitDuration=%s",
				stats.OpenConnections, stats.Idle, stats.InUse, stats.WaitCount, stats.WaitDuration)
		}
	}()

	if err := sqlDB.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	} else {
		log.Println("✅ Successfully connected to database with connection pooling")
	}

	// Clear Redis cache on startup using CacheService
	if repositories.CacheService != nil {
		err := repositories.CacheService.FlushAll(context.Background())
		if err != nil {
			log.Printf("⚠️ Failed to flush Redis cache: %v", err)
		} else {
			log.Println("✅ Redis cache flushed on startup")
		}
	}

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

		// Close Redis connection via CacheService
		if repositories.CacheService != nil {
			if err := repositories.CacheService.Close(); err != nil {
				log.Printf("⚠️ Failed to close Redis connection: %v", err)
			}
		}
	}()

	// Create Fiber app
	app := fiber.New()

	// CORS middleware
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "http://localhost:5173",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowMethods:     "GET,POST,HEAD,PUT,DELETE,PATCH",
		AllowCredentials: true,
	}))

	// Middleware
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${latency} ${method} ${path}\n",
	}))

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
	routes.SetupRoutes(app, repositories.DB)

	// Start server
	log.Fatal(app.Listen(":" + config.GetEnv("PORT", "3000")))
}
