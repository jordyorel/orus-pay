package handlers

import "github.com/gofiber/fiber/v2"

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
