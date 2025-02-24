package response

import (
	"github.com/gofiber/fiber/v2"
)

func Success(c *fiber.Ctx, message string, data interface{}) error {
	return c.JSON(fiber.Map{
		"message": message,
		"data":    data,
	})
}

func Error(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(fiber.Map{
		"error": message,
	})
}

func BadRequest(c *fiber.Ctx, message string) error {
	return Error(c, fiber.StatusBadRequest, message)
}

func ServerError(c *fiber.Ctx, message string) error {
	return Error(c, fiber.StatusInternalServerError, message)
}

func Unauthorized(c *fiber.Ctx) error {
	return Error(c, fiber.StatusUnauthorized, "Unauthorized")
}

func ValidationError(c *fiber.Ctx, message string) error {
	return Error(c, fiber.StatusBadRequest, message)
}
