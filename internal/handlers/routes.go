package handlers

import (
	"orus/internal/middleware"
	"orus/internal/models"
	"orus/internal/repositories"
	"orus/internal/repositories/cache"
	"orus/internal/services"
	"orus/internal/services/qr"
	"orus/internal/services/transaction"
	"orus/internal/services/wallet"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
)

func SetupRoutes(app *fiber.App) {
	// Initialize cache
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	noCtxCache, ctxCache := cache.NewCaches(redisClient)

	// Initialize services in correct order
	walletService := wallet.NewService(
		repositories.DB,
		noCtxCache, // Uses no-context interface
		wallet.WalletConfig{},
		&wallet.NoopMetricsCollector{},
	)

	transactionService := transaction.NewService(
		repositories.DB,
		walletService,
		walletService,
		noCtxCache, // Uses no-context interface
	)

	qrService := qr.NewService(
		repositories.DB,
		ctxCache, // Uses context interface
		transactionService,
		walletService,
	)

	// Initialize handlers with services
	paymentHandler := NewPaymentHandler(qrService, transactionService)
	merchantHandler := NewMerchantHandler(services.NewMerchantService(), qrService)
	enterpriseHandler := NewEnterpriseHandler()
	userHandler := NewUserHandler(
		qrService,
		walletService,
	)

	// Public routes
	app.Get("/health", HealthCheck)
	app.Get("/", func(c *fiber.Ctx) error { return c.SendString("Welcome to OrusPay API!") })

	api := app.Group("/api")
	api.Post("/register", userHandler.RegisterUser)
	api.Post("/login", LoginUser)
	api.Post("/refresh", RefreshToken)

	// Authenticated routes
	authenticated := app.Group("/api", middleware.AuthMiddleware)

	// Setup different route groups
	setupUserRoutes(authenticated, userHandler, paymentHandler)
	setupMerchantRoutes(authenticated, merchantHandler)
	setupEnterpriseRoutes(authenticated, enterpriseHandler)
	setupAdminRoutes(app)
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

	// QR code routes
	qr := router.Group("/qr")
	qr.Get("/receive", userHandler.GetReceiveQR)     // For receiving money
	qr.Get("/payment", userHandler.GetPaymentCodeQR) // For payments at merchants
}

func setupMerchantRoutes(router fiber.Router, h *MerchantHandler) {
	merchant := router.Group("/merchant", middleware.HasPermission(models.PermissionMerchantRead))

	// Profile Management
	merchant.Post("/", h.CreateMerchant)                                                                        //✅
	merchant.Get("/profile", h.GetMerchantProfile)                                                              //✅
	merchant.Put("/profile", middleware.HasPermission(models.PermissionMerchantWrite), h.UpdateMerchantProfile) //✅

	// QR Management
	merchant.Get("/qr/receive", h.GetReceiveQR)     // For receiving money
	merchant.Get("/qr/payment", h.GetPaymentCodeQR) // For payments
	merchant.Post("/scan", h.ScanUserPaymentCode)   // For scanning user QRs

	// Transaction Management
	merchant.Post("/transaction", middleware.HasPermission(models.PermissionMerchantTransaction), h.ProcessTransaction)
	merchant.Get("/:merchantId/transactions", h.GetMerchantTransactions)

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
