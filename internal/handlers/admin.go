package handlers

import (
	"log"
	"orus/internal/models"
	"orus/internal/repositories"
	"strconv"

	"orus/internal/utils/pagination"

	"github.com/gofiber/fiber/v2"
)

func GetUsersPaginated(c *fiber.Ctx) error {
	// Verify admin permissions
	claims, ok := c.Locals("claims").(*models.UserClaims)
	if !ok || !claims.HasPermission(models.PermissionReadAdmin) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied. Admin privileges required",
		})
	}

	p := pagination.ParseFromRequest(c)

	users, total, err := repositories.GetUsersPaginated(p.Limit, p.Offset)
	if err != nil {
		log.Printf("Error fetching paginated users: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch users"})
	}

	p.Total = total
	return c.JSON(pagination.Response(p, users))
}

// GetAllWallets retrieves all wallets in a paginated manner (Admin only)
func GetAllWallets(c *fiber.Ctx) error {
	// Verify admin permissions
	claims, ok := c.Locals("claims").(*models.UserClaims)
	if !ok || !claims.HasPermission(models.PermissionReadAdmin) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied. Admin privileges required.",
		})
	}

	p := pagination.ParseFromRequest(c)

	wallets, total, err := repositories.GetWalletsPaginated(p.Limit, p.Offset)
	if err != nil {
		log.Printf("Error fetching wallets: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch wallets",
		})
	}

	p.Total = total
	return c.JSON(pagination.Response(p, wallets))
}

// GetAllCreditCards retrieves all credit cards in a paginated manner (Admin only)
func GetAllCreditCards(c *fiber.Ctx) error {
	// Verify admin permissions
	claims, ok := c.Locals("claims").(*models.UserClaims)
	if !ok || !claims.HasPermission(models.PermissionReadAdmin) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied. Admin privileges required.",
		})
	}

	p := pagination.ParseFromRequest(c)

	creditCards, total, err := repositories.GetCreditCardsPaginated(p.Limit, p.Offset)
	if err != nil {
		log.Printf("Error fetching credit cards: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch credit cards",
		})
	}

	p.Total = total
	return c.JSON(pagination.Response(p, creditCards))
}

func GetAllTransactions(c *fiber.Ctx) error {
	// Get claims from context
	claims, ok := c.Locals("claims").(*models.UserClaims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid claims",
		})
	}

	// Check if user has admin read permission
	if !claims.HasPermission(models.PermissionReadAdmin) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied. Admin privileges required.",
		})
	}

	p := pagination.ParseFromRequest(c)

	// Fetch all transactions
	transactions, total, err := repositories.GetTransactions(p.Limit, p.Offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch transactions",
		})
	}

	p.Total = total
	return c.JSON(pagination.Response(p, transactions))
}

// DeleteUser allows admins to delete a user by their ID
func DeleteUser(c *fiber.Ctx) error {
	// Verify admin permissions
	claims, ok := c.Locals("claims").(*models.UserClaims)
	if !ok || !claims.HasPermission(models.PermissionWriteAdmin) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied. Admin privileges required",
		})
	}

	userID := c.Params("id")

	// Add audit logging
	log.Printf("Admin %d attempting to delete user %s", claims.UserID, userID)

	if err := repositories.DeleteUserByID(userID); err != nil {
		log.Printf("Error deleting user %s: %v", userID, err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to delete user"})
	}

	// Add cache invalidation
	if userIDUint, err := strconv.ParseUint(userID, 10, 32); err == nil {
		repositories.InvalidateUserCache(uint(userIDUint))
	}

	return c.JSON(fiber.Map{
		"message": "User deleted successfully",
	})
}
