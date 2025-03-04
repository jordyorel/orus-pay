package handlers

import (
	"orus/internal/repositories"

	"github.com/gofiber/fiber/v2"
)

func HealthCheck(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":  "ok",
		"version": "1.0.0",
		"services": fiber.Map{
			"database": "connected",
			"redis":    "connected",
		},
	})
}

func CacheStats(c *fiber.Ctx) error {
	poolStats := repositories.CacheService.GetStats(c.Context())

	return c.JSON(fiber.Map{
		"pool_stats": fiber.Map{
			"hits":        poolStats.Hits,
			"misses":      poolStats.Misses,
			"timeouts":    poolStats.Timeouts,
			"total_conns": poolStats.TotalConns,
			"idle_conns":  poolStats.IdleConns,
			"stale_conns": poolStats.StaleConns,
		},
	})
}
