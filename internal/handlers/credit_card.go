package handlers

import (
	"orus/internal/models"
	"orus/internal/repositories"
	creditcard "orus/internal/services/credit-card"
	"orus/internal/utils/response"

	"github.com/gofiber/fiber/v2"
)

type CreditCardHandler struct {
	cardService creditcard.Service
}

func NewCreditCardHandler(cardRepo repositories.CreditCardRepository) *CreditCardHandler {
	return &CreditCardHandler{
		cardService: creditcard.NewService(cardRepo),
	}
}

func (h *CreditCardHandler) LinkCard(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*models.UserClaims)

	var input creditcard.CreateCardInput
	if err := c.BodyParser(&input); err != nil {
		return response.BadRequest(c, "Invalid request format")
	}

	card, err := h.cardService.LinkCard(claims.UserID, input)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, err.Error())
	}

	return response.Success(c, "Credit card linked successfully", fiber.Map{
		"card_type": card.CardType,
		"last_four": card.CardNumber[len(card.CardNumber)-4:],
		"expiry":    card.ExpiryMonth + "/" + card.ExpiryYear,
	})
}

func (h *CreditCardHandler) GetCards(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*models.UserClaims)

	cards, err := h.cardService.GetUserCards(claims.UserID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch cards")
	}

	return response.Success(c, "Cards retrieved successfully", cards)
}

func (h *CreditCardHandler) DeleteCard(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*models.UserClaims)
	cardID, err := c.ParamsInt("id")
	if err != nil {
		return response.BadRequest(c, "Invalid card ID")
	}

	if err := h.cardService.DeleteCard(claims.UserID, uint(cardID)); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to delete card")
	}

	return response.Success(c, "Card deleted successfully", nil)
}
