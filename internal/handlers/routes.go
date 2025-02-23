// package handlers

// import (
// 	"orus/internal/middleware"
// 	"orus/internal/models"

// 	"github.com/gofiber/fiber/v2"
// )

// func SetupRoutes(app *fiber.App) {
// 	// health check at the root
// 	app.Get("/health", HealthCheck)

// 	// Public routes
// 	app.Get("/", func(c *fiber.Ctx) error { return c.SendString("Welcome to OrusPay API!") })
// 	api := app.Group("/api")
// 	api.Post("/register", RegisterUser)
// 	api.Post("/login", LoginUser)
// 	api.Post("/refresh", RefreshToken)

// 	// User routes with authentication
// 	authenticated := api.Group("/", middleware.AuthMiddleware)

// 	// Wallet routes
// 	wallet := authenticated.Group("/wallet")
// 	wallet.Get("/", middleware.HasPermission(models.PermissionWalletRead), GetWallet)
// 	wallet.Post("/topup", middleware.HasPermission(models.PermissionWalletWrite), TopUpWallet)
// 	wallet.Post("/withdraw", middleware.HasPermission(models.PermissionWalletWrite), WithdrawToCard)

// 	// Also add the direct wallet endpoint
// 	authenticated.Get("/wallet", middleware.HasPermission(models.PermissionWalletRead), GetWallet)

// 	// Transaction routes
// 	authenticated.Get("/transactions", GetUserTransactions)
// 	authenticated.Post("/transaction", ProcessTransaction)

// 	// Other user routes
// 	authenticated.Post("/credit-card", LinkCreditCard)
// 	authenticated.Post("/change-password", ChangePassword)
// 	authenticated.Post("/refresh", RefreshToken)
// 	authenticated.Post("/logout", LogoutUser)

// 	// Initialize handlers
// 	paymentHandler := NewPaymentHandler()
// 	merchantHandler := NewMerchantHandler()
// 	enterpriseHandler := NewEnterpriseHandler()
// 	userHandler := NewUserHandler()

// 	// Merchant routes
// 	merchant := authenticated.Group("/merchant")
// 	merchant.Post("/", merchantHandler.CreateMerchant)

// 	// Use existing paymentHandler
// 	merchant.Post("/qr", middleware.HasPermission(models.PermissionMerchantTransaction), paymentHandler.GenerateQRCode)
// 	merchant.Post("/qr/dynamic", middleware.HasPermission(models.PermissionMerchantTransaction), paymentHandler.GenerateQRCode)
// 	merchant.Post("/qr/static", middleware.HasPermission(models.PermissionMerchantTransaction), paymentHandler.GenerateQRCode)

// 	// Other merchant routes that need merchant permissions
// 	merchant.Get("/profile", middleware.HasPermission(models.PermissionMerchantRead), merchantHandler.GetMerchantProfile)
// 	merchant.Put("/profile", middleware.HasPermission(models.PermissionMerchantWrite), merchantHandler.UpdateMerchantProfile)

// 	// Merchant transactions
// 	merchant.Post("/transaction", middleware.HasPermission(models.PermissionMerchantTransaction), merchantHandler.ProcessTransaction)
// 	merchant.Get("/:merchantId/transactions", middleware.HasPermission(models.PermissionMerchantRead), merchantHandler.GetMerchantTransactions)

// 	// Merchant settings
// 	merchant.Post("/:merchantId/apikey", middleware.HasPermission(models.PermissionMerchantWrite), merchantHandler.GenerateAPIKey)
// 	merchant.Post("/:merchantId/webhook", middleware.HasPermission(models.PermissionMerchantWrite), merchantHandler.SetWebhookURL)

// 	// Move these into the authenticated group
// 	authenticated.Post("/qr/dynamic", paymentHandler.GenerateQRCode)
// 	authenticated.Post("/payment/qr", paymentHandler.ProcessQRPayment)

// 	// User routes
// 	authenticated.Get("/payment-code", userHandler.GeneratePaymentCode) // User gets their payment QR
// 	authenticated.Get("/receive-code", userHandler.GetReceiveCode)      // For receiving payments

// 	// Enterprise routes (requires enterprise role)
// 	enterprise := authenticated.Group("/enterprise")
// 	enterprise.Post("/", enterpriseHandler.CreateEnterprise)
// 	enterprise.Post("/:enterpriseId/apikey", enterpriseHandler.GenerateAPIKey)

// 	// Admin routes (require AdminAuthMiddleware)
// 	admin := api.Group("/admin", middleware.AdminAuthMiddleware)
// 	admin.Get("/transactions", middleware.HasPermission(models.PermissionReadAdmin), GetAllTransactions)
// 	admin.Get("/users", middleware.HasPermission(models.PermissionReadAdmin), GetUsersPaginated)
// 	admin.Delete("/users/:id", middleware.HasPermission(models.PermissionWriteAdmin), DeleteUser)
// 	admin.Get("/wallets", middleware.HasPermission(models.PermissionWriteAdmin), GetAllWallets)          // Admin view all wallets
// 	admin.Get("/credit-cards", middleware.HasPermission(models.PermissionWriteAdmin), GetAllCreditCards) // Admin view all credit cards

// 	// Merchant routes
// 	merchant.Post("/scan", merchantHandler.ScanUserPaymentCode) // Merchant scans user's payment QR

// 	// Merchant QR routes
// 	merchant.Get("/qr", merchantHandler.GetMerchantQR)              // Get static QR
// 	merchant.Post("/qr/dynamic", merchantHandler.GenerateDynamicQR) // Only for dynamic QRs
// }

package handlers

import (
	"orus/internal/middleware"
	"orus/internal/models"

	"github.com/gofiber/fiber/v2"
)

func SetupRoutes(app *fiber.App) {
	// Initialize handlers first
	paymentHandler := NewPaymentHandler()
	merchantHandler := NewMerchantHandler()
	enterpriseHandler := NewEnterpriseHandler()
	userHandler := NewUserHandler()

	// Public routes
	setupPublicRoutes(app)

	// Authenticated routes
	authenticated := app.Group("/api", middleware.AuthMiddleware)

	// Setup different route groups
	setupUserRoutes(authenticated, userHandler, paymentHandler)
	setupMerchantRoutes(authenticated, merchantHandler)
	setupEnterpriseRoutes(authenticated, enterpriseHandler)
	setupAdminRoutes(app)
}

func setupPublicRoutes(app *fiber.App) {
	app.Get("/health", HealthCheck)
	app.Get("/", func(c *fiber.Ctx) error { return c.SendString("Welcome to OrusPay API!") })

	api := app.Group("/api")
	api.Post("/register", RegisterUser)
	api.Post("/login", LoginUser)
	api.Post("/refresh", RefreshToken)
}

func setupUserRoutes(router fiber.Router, userHandler *UserHandler, paymentHandler *PaymentHandler) {
	// Wallet routes
	wallet := router.Group("/wallet")
	wallet.Get("/", middleware.HasPermission(models.PermissionWalletRead), GetWallet)                //✅
	wallet.Post("/topup", middleware.HasPermission(models.PermissionWalletWrite), TopUpWallet)       //✅
	wallet.Post("/withdraw", middleware.HasPermission(models.PermissionWalletWrite), WithdrawToCard) //✅

	// Transaction routes
	router.Get("/transactions", GetUserTransactions) //✅

	// User account routes
	router.Post("/credit-card", LinkCreditCard)     //✅
	router.Post("/change-password", ChangePassword) //✅
	router.Post("/logout", LogoutUser)              //✅

	// Payment routes
	payments := router.Group("/payment")
	payments.Post("/scan", paymentHandler.ProcessQRPayment) //✅
	payments.Post("/send", paymentHandler.SendMoney)        //✅
	payments.Post("/qr", paymentHandler.GeneratePaymentQR)  //✅

	// QR code routes
	qr := router.Group("/qr")
	qr.Get("/receive", userHandler.GetReceiveQR) //✅
}

func setupMerchantRoutes(router fiber.Router, h *MerchantHandler) {
	merchant := router.Group("/merchant", middleware.HasPermission(models.PermissionMerchantRead))

	// Profile Management
	merchant.Post("/", h.CreateMerchant)                                                                        //✅
	merchant.Get("/profile", h.GetMerchantProfile)                                                              //✅
	merchant.Put("/profile", middleware.HasPermission(models.PermissionMerchantWrite), h.UpdateMerchantProfile) //✅

	// QR Management
	merchant.Get("/qr/static", h.GetMerchantQR)
	merchant.Post("/qr/dynamic", h.GenerateDynamicQR)

	// Transaction Management
	merchant.Post("/transaction", middleware.HasPermission(models.PermissionMerchantTransaction), h.ProcessTransaction)
	merchant.Get("/:merchantId/transactions", h.GetMerchantTransactions)
	merchant.Post("/scan", middleware.HasPermission(models.PermissionMerchantTransaction), h.ScanUserPaymentCode)

	// Integration Settings
	merchant.Post("/:merchantId/apikey", middleware.HasPermission(models.PermissionMerchantWrite), h.GenerateAPIKey)
	merchant.Post("/:merchantId/webhook", middleware.HasPermission(models.PermissionMerchantWrite), h.SetWebhookURL)
}

func setupEnterpriseRoutes(router fiber.Router, h *EnterpriseHandler) {
	enterprise := router.Group("/enterprise", middleware.HasPermission("enterprise:read"))
	enterprise.Post("/", h.CreateEnterprise)
	enterprise.Post("/:enterpriseId/apikey", h.GenerateAPIKey)
}

func setupAdminRoutes(app *fiber.App) {
	admin := app.Group("/api/admin", middleware.AdminAuthMiddleware)
	admin.Get("/transactions", middleware.HasPermission(models.PermissionReadAdmin), GetAllTransactions)
	admin.Get("/users", middleware.HasPermission(models.PermissionReadAdmin), GetUsersPaginated)
	admin.Delete("/users/:id", middleware.HasPermission(models.PermissionWriteAdmin), DeleteUser)
	admin.Get("/wallets", middleware.HasPermission(models.PermissionWriteAdmin), GetAllWallets)
	admin.Get("/credit-cards", middleware.HasPermission(models.PermissionWriteAdmin), GetAllCreditCards)
}
