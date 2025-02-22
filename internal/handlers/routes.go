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

	// User routes with authentication
	authenticated := api.Group("/", middleware.AuthMiddleware)

	// Wallet routes
	wallet := authenticated.Group("/wallet")
	wallet.Get("/", middleware.HasPermission(models.PermissionWalletRead), GetWallet)
	wallet.Post("/topup", middleware.HasPermission(models.PermissionWalletWrite), TopUpWallet)
	wallet.Post("/withdraw", middleware.HasPermission(models.PermissionWalletWrite), WithdrawToCard)

	// Also add the direct wallet endpoint
	authenticated.Get("/wallet", middleware.HasPermission(models.PermissionWalletRead), GetWallet)

	// Transaction routes
	authenticated.Get("/transactions", GetUserTransactions)
	authenticated.Post("/transaction", ProcessTransaction)

	// Other user routes
	authenticated.Post("/credit-card", LinkCreditCard)
	authenticated.Post("/change-password", ChangePassword)
	authenticated.Post("/refresh", RefreshToken)
	authenticated.Post("/logout", LogoutUser)

	// Initialize handlers
	paymentHandler := NewPaymentHandler()
	merchantHandler := NewMerchantHandler()
	enterpriseHandler := NewEnterpriseHandler()

	// Merchant routes
	merchant := authenticated.Group("/merchant")
	merchant.Post("/", merchantHandler.CreateMerchant)

	// Use existing paymentHandler
	merchant.Post("/qr", middleware.HasPermission(models.PermissionMerchantTransaction), paymentHandler.GenerateQRCode)
	merchant.Post("/qr/dynamic", middleware.HasPermission(models.PermissionMerchantTransaction), paymentHandler.GenerateQRCode)
	merchant.Post("/qr/static", middleware.HasPermission(models.PermissionMerchantTransaction), paymentHandler.GenerateQRCode)

	// Other merchant routes that need merchant permissions
	merchant.Get("/profile", middleware.HasPermission(models.PermissionMerchantRead), merchantHandler.GetMerchantProfile)
	merchant.Put("/profile", middleware.HasPermission(models.PermissionMerchantWrite), merchantHandler.UpdateMerchantProfile)

	// Merchant transactions
	merchant.Post("/transaction", middleware.HasPermission(models.PermissionMerchantTransaction), merchantHandler.ProcessTransaction)
	merchant.Get("/:merchantId/transactions", middleware.HasPermission(models.PermissionMerchantRead), merchantHandler.GetMerchantTransactions)

	// Merchant settings
	merchant.Post("/:merchantId/apikey", middleware.HasPermission(models.PermissionMerchantWrite), merchantHandler.GenerateAPIKey)
	merchant.Post("/:merchantId/webhook", middleware.HasPermission(models.PermissionMerchantWrite), merchantHandler.SetWebhookURL)

	// Move these into the authenticated group
	authenticated.Post("/qr/dynamic", paymentHandler.GenerateQRCode)
	authenticated.Post("/payment/qr", paymentHandler.ProcessQRPayment)

	// Enterprise routes (requires enterprise role)
	enterprise := authenticated.Group("/enterprise")
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
