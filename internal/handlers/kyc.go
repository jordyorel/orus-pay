package handlers

import (
	"orus/internal/models"
	"orus/internal/services"
	"orus/internal/utils/response"

	"github.com/gofiber/fiber/v2"
)

type KYCHandler struct {
	service services.KYCService
}

func NewKYCHandler(s services.KYCService) *KYCHandler { return &KYCHandler{service: s} }

func (h *KYCHandler) SubmitKYC(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*models.UserClaims)
	var input struct {
		DocumentID string `json:"document_id"`
		ScanURL    string `json:"scan_url"`
	}
	if err := c.BodyParser(&input); err != nil {
		return response.BadRequest(c, "invalid request")
	}
	kyc, err := h.service.SubmitKYC(c.Context(), claims.UserID, input.DocumentID, input.ScanURL)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, err.Error())
	}
	return response.Success(c, "KYC submitted", kyc)
}

func (h *KYCHandler) GetStatus(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*models.UserClaims)
	kyc, err := h.service.GetStatus(c.Context(), claims.UserID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, err.Error())
	}
	return response.Success(c, "KYC status", kyc)
}
