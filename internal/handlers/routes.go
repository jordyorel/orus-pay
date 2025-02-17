package handlers

import (
	"orus/internal/middleware"
	"orus/internal/models"

	"github.com/gofiber/fiber/v2"
)

func SetupRoutes(app *fiber.App) {
	// health check at the root
	app.Get("/health", HealthCheck)

	// Public routes
	app.Get("/", func(c *fiber.Ctx) error { return c.SendString("Welcome to OrusPay API!") })
	api := app.Group("/api")
	api.Post("/register", RegisterUser)
	api.Post("/login", LoginUser)
	api.Post("/refresh", RefreshToken)

	// Secured routes (require AuthMiddleware)
	secured := api.Use(middleware.AuthMiddleware)

	// User management routes
	// secured.Post("/logout", LogoutUser)
	secured.Post("/logout", middleware.AuthMiddleware, LogoutUser)
	secured.Post("/change-password", middleware.HasPermission(models.PermissionChangePassword), ChangePassword)

	// Wallet routes
	secured.Post("/wallet", middleware.HasPermission(models.PermissionWalletRead), GetWallet)
	secured.Post("/wallet/topup", middleware.HasPermission(models.PermissionWalletWrite), TopUpWallet)

	// Transaction routes
	secured.Post("/transaction", middleware.HasPermission(models.PermissionTransactionWrite), ProcessTransaction)
	secured.Get("/transactions", middleware.HasPermission(models.PermissionTransactionRead), GetUserTransactions)

	// Payment and credit card routes
	secured.Post("/credit-card", middleware.HasPermission(models.PermissionCreditCardWrite), LinkCreditCard)
	secured.Post("/payment/qr", middleware.HasPermission(models.PermissionPaymentWrite), GeneratePaymentQR) // To be emplemented

	// Admin routes (require AdminAuthMiddleware)
	admin := secured.Group("/admin", middleware.AdminAuthMiddleware)
	admin.Get("/transactions", middleware.HasPermission(models.PermissionReadAdmin), GetAllTransactions)
	admin.Get("/users", middleware.HasPermission(models.PermissionReadAdmin), GetUsersPaginated)
	admin.Delete("/users/:id", middleware.HasPermission(models.PermissionWriteAdmin), DeleteUser)
	admin.Get("/wallets", middleware.HasPermission(models.PermissionWriteAdmin), GetAllWallets)          // Admin view all wallets
	admin.Get("/credit-cards", middleware.HasPermission(models.PermissionWriteAdmin), GetAllCreditCards) // Admin view all credit cards

}
