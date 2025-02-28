package handlers

import (
	"orus/internal/models"
	"orus/internal/services/dashboard"
	"orus/internal/utils/response"
	"time"

	"github.com/gofiber/fiber/v2"
)

type DashboardHandler struct {
	dashboardService dashboard.Service
}

func NewDashboardHandler(dashboardService dashboard.Service) *DashboardHandler {
	return &DashboardHandler{
		dashboardService: dashboardService,
	}
}

// GetUserDashboard returns dashboard data for regular users
func (h *DashboardHandler) GetUserDashboard(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*models.UserClaims)

	stats, err := h.dashboardService.GetUserDashboard(c.Context(), claims.UserID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to get dashboard data")
	}

	return response.Success(c, "Dashboard data retrieved successfully", stats)
}

// GetMerchantDashboard returns dashboard data for merchants
func (h *DashboardHandler) GetMerchantDashboard(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*models.UserClaims)

	if claims.Role != "merchant" {
		return response.Error(c, fiber.StatusForbidden, "Access denied")
	}

	stats, err := h.dashboardService.GetMerchantDashboard(c.Context(), claims.UserID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to get merchant dashboard data")
	}

	return response.Success(c, "Merchant dashboard data retrieved successfully", stats)
}

// GetTransactionAnalytics returns detailed transaction analytics
func (h *DashboardHandler) GetTransactionAnalytics(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*models.UserClaims)

	// Parse date range from query parameters
	startDate := c.Query("start_date", time.Now().AddDate(0, -1, 0).Format("2006-01-02"))
	endDate := c.Query("end_date", time.Now().Format("2006-01-02"))

	start, _ := time.Parse("2006-01-02", startDate)
	end, _ := time.Parse("2006-01-02", endDate)

	analytics, err := h.dashboardService.GetTransactionAnalytics(c.Context(), claims.UserID, start, end)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to get transaction analytics")
	}

	return response.Success(c, "Transaction analytics retrieved successfully", analytics)
}
