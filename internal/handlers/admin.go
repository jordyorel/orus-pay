package handlers

import (
	"log"
	"orus/internal/models"
	"orus/internal/repositories"
	"strconv"

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

	// Get pagination parameters
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	offset := (page - 1) * limit

	// Get paginated users
	users, total, err := repositories.GetUsersPaginated(limit, offset)
	if err != nil {
		log.Printf("Error fetching paginated users: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch users"})
	}

	// Calculate total pages
	totalPages := total / int64(limit)
	if total%int64(limit) > 0 {
		totalPages++
	}

	return c.JSON(fiber.Map{
		"data": users,
		"meta": fiber.Map{
			"current_page": page,
			"per_page":     limit,
			"total_items":  total,
			"total_pages":  totalPages,
		},
	})
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

	// Pagination
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	offset := (page - 1) * limit

	// Fetch paginated wallets
	wallets, total, err := repositories.GetWalletsPaginated(limit, offset)
	if err != nil {
		log.Printf("Error fetching wallets: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch wallets",
		})
	}

	// Calculate total pages
	totalPages := total / int64(limit)
	if total%int64(limit) > 0 {
		totalPages++
	}

	// Return paginated wallets
	return c.JSON(fiber.Map{
		"data": wallets,
		"meta": fiber.Map{
			"current_page": page,
			"per_page":     limit,
			"total_items":  total,
			"total_pages":  totalPages,
		},
	})
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

	// Pagination
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	offset := (page - 1) * limit

	// Fetch paginated credit cards
	creditCards, total, err := repositories.GetCreditCardsPaginated(limit, offset)
	if err != nil {
		log.Printf("Error fetching credit cards: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch credit cards",
		})
	}

	// Calculate total pages
	totalPages := total / int64(limit)
	if total%int64(limit) > 0 {
		totalPages++
	}

	// Return paginated credit cards
	return c.JSON(fiber.Map{
		"data": creditCards,
		"meta": fiber.Map{
			"current_page": page,
			"per_page":     limit,
			"total_items":  total,
			"total_pages":  totalPages,
		},
	})
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

	// Pagination and optional filtering
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	offset := (page - 1) * limit

	// Fetch all transactions
	transactions, err := repositories.GetTransactions(limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch transactions",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"transactions": transactions,
		"page":         page,
		"limit":        limit,
	})
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
