package utils

import "github.com/gofiber/fiber/v2"

// Respond sends a JSON response with the specified status code.
func Respond(c *fiber.Ctx, status int, data interface{}) error {
	return c.Status(status).JSON(data)
}

// Success sends a successful JSON response.
func Success(c *fiber.Ctx, data interface{}) error {
	return Respond(c, fiber.StatusOK, data)
}

// BadRequest sends a JSON error response with status 400.
func BadRequest(c *fiber.Ctx, message string) error {
	return Respond(c, fiber.StatusBadRequest, fiber.Map{"error": message})
}

// Unauthorized sends a JSON error response with status 401.
func Unauthorized(c *fiber.Ctx, message string) error {
	return Respond(c, fiber.StatusUnauthorized, fiber.Map{"error": message})
}

// Forbidden sends a JSON error response with status 403.
func Forbidden(c *fiber.Ctx, message string) error {
	return Respond(c, fiber.StatusForbidden, fiber.Map{"error": message})
}

// NotFound sends a JSON error response with status 404.
func NotFound(c *fiber.Ctx, message string) error {
	return Respond(c, fiber.StatusNotFound, fiber.Map{"error": message})
}

// InternalError sends a JSON error response with status 500.
func InternalError(c *fiber.Ctx, message string) error {
	return Respond(c, fiber.StatusInternalServerError, fiber.Map{"error": message})
}
