package utils

import (
	"errors"

	"orus/internal/models"

	"github.com/gofiber/fiber/v2"
)

// GetUserClaims extracts the user claims from the Fiber context.
// It returns an error if the claims are missing or of an invalid type.
func GetUserClaims(c *fiber.Ctx) (*models.UserClaims, error) {
	v := c.Locals("claims")
	if v == nil {
		return nil, errors.New("claims not found in context")
	}

	claims, ok := v.(*models.UserClaims)
	if !ok {
		return nil, errors.New("invalid claims type")
	}
	return claims, nil
}
