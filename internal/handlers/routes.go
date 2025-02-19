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

	// User routes (requires any authenticated user)
	user := api.Group("/", middleware.AuthMiddleware)
	user.Use(middleware.Protected())

	// Wallet routes
	user.Post("/wallet", middleware.HasPermission(models.PermissionWalletRead), GetWallet)
	user.Post("/wallet/topup", middleware.HasPermission(models.PermissionWalletWrite), TopUpWallet)
	user.Post("/wallet/withdraw", middleware.HasPermission(models.PermissionWalletWrite), WithdrawToCard)

	// Transaction routes
	user.Post("/transaction", middleware.HasPermission(models.PermissionTransactionWrite), ProcessTransaction)
	user.Get("/transactions", middleware.HasPermission(models.PermissionTransactionRead), GetUserTransactions)

	// Other user routes
	user.Post("/credit-card", middleware.HasPermission(models.PermissionCreditCardWrite), LinkCreditCard)
	user.Post("/change-password", middleware.HasPermission(models.PermissionChangePassword), ChangePassword)
	user.Post("/refresh", RefreshToken)
	user.Post("/logout", LogoutUser)

	// Merchant routes (requires merchant role)
	merchant := api.Group("/merchant", middleware.AuthMiddleware)
	merchant.Use(middleware.Protected())
	merchantHandler := NewMerchantHandler()

	// Merchant management
	merchant.Post("/", merchantHandler.CreateMerchant)
	merchant.Get("/profile", middleware.HasPermission(models.PermissionMerchantRead), merchantHandler.GetMerchantProfile)
	merchant.Put("/profile", middleware.HasPermission(models.PermissionMerchantWrite), merchantHandler.UpdateMerchantProfile)

	// Merchant transactions
	merchant.Post("/:merchantId/transaction", middleware.HasPermission(models.PermissionMerchantTransaction), merchantHandler.ProcessTransaction)
	merchant.Get("/:merchantId/transactions", middleware.HasPermission(models.PermissionMerchantRead), merchantHandler.GetMerchantTransactions)

	// Merchant settings
	merchant.Post("/:merchantId/apikey", middleware.HasPermission(models.PermissionMerchantWrite), merchantHandler.GenerateAPIKey)
	merchant.Post("/:merchantId/webhook", middleware.HasPermission(models.PermissionMerchantWrite), merchantHandler.SetWebhookURL)

	// Enterprise routes (requires enterprise role)
	enterprise := api.Group("/enterprise", middleware.AuthMiddleware)
	enterprise.Use(middleware.Protected())
	enterpriseHandler := NewEnterpriseHandler()
	enterprise.Post("/", enterpriseHandler.CreateEnterprise)
	enterprise.Post("/:enterpriseId/apikey", enterpriseHandler.GenerateAPIKey)

	// Admin routes (require AdminAuthMiddleware)
	admin := api.Group("/admin", middleware.AdminAuthMiddleware)
	admin.Get("/transactions", middleware.HasPermission(models.PermissionReadAdmin), GetAllTransactions)
	admin.Get("/users", middleware.HasPermission(models.PermissionReadAdmin), GetUsersPaginated)
	admin.Delete("/users/:id", middleware.HasPermission(models.PermissionWriteAdmin), DeleteUser)
	admin.Get("/wallets", middleware.HasPermission(models.PermissionWriteAdmin), GetAllWallets)          // Admin view all wallets
	admin.Get("/credit-cards", middleware.HasPermission(models.PermissionWriteAdmin), GetAllCreditCards) // Admin view all credit cards
}
