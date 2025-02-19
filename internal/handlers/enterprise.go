package handlers

import (
	"orus/internal/models"
	"orus/internal/services"

	"github.com/gofiber/fiber/v2"
)

type EnterpriseHandler struct {
	enterpriseService *services.EnterpriseService
}

func NewEnterpriseHandler() *EnterpriseHandler {
	return &EnterpriseHandler{
		enterpriseService: services.NewEnterpriseService(),
	}
}

func (h *EnterpriseHandler) CreateEnterprise(c *fiber.Ctx) error {
	var enterprise models.Enterprise
	if err := c.BodyParser(&enterprise); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := h.enterpriseService.CreateEnterprise(&enterprise); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(enterprise)
}

func (h *EnterpriseHandler) GenerateAPIKey(c *fiber.Ctx) error {
	var req struct {
		KeyName     string `json:"key_name"`
		Environment string `json:"environment"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	enterpriseID, err := c.ParamsInt("enterpriseId")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid enterprise ID"})
	}

	apiKey, err := h.enterpriseService.GenerateAPIKey(uint(enterpriseID), req.KeyName, req.Environment)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(apiKey)
}
