package handlers

import (
	"orus/internal/middleware"
	"orus/internal/models"
	"orus/internal/repositories"
	"orus/internal/services/auth"
	"orus/internal/services/credit_card"
	"orus/internal/services/merchant"
	"orus/internal/services/payment"
	qr "orus/internal/services/qr_code"
	"orus/internal/services/transaction"
	"orus/internal/services/wallet"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

var walletService wallet.Service

func SetupRoutes(app *fiber.App, db *gorm.DB) {
	// Initialize cache
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// Initialize repositories
	walletRepo := repositories.NewWalletRepository(repositories.DB)
	cacheRepo := repositories.NewRedisCacheRepository(redisClient)
	userRepo := repositories.NewUserRepository(repositories.DB)
	cardRepo := repositories.NewCreditCardRepository(repositories.DB)

	// Initialize auth service and handler
	authService := auth.NewService(userRepo)
	authHandler := NewAuthHandler(authService)

	// Initialize services in correct order
	cardService := credit_card.NewService(cardRepo)
	walletService = wallet.NewService(
		walletRepo,
		cacheRepo,
		cardService,
		wallet.WalletConfig{},
		&wallet.NoopMetricsCollector{},
	)

	transactionService := transaction.NewService(
		repositories.DB,
		walletService,
		walletService,
		cacheRepo,
	)

	qrService := qr.NewService(
		repositories.DB,
		cacheRepo,
		transactionService,
		walletService,
	)

	paymentService := payment.NewService(walletService, transactionService, qrService)

	// Initialize handlers
	paymentHandler := NewPaymentHandler(qrService, paymentService)
	merchantHandler := NewMerchantHandler(
		merchant.NewService(qrService, transactionService, walletService),
		qrService,
	)
	enterpriseHandler := NewEnterpriseHandler()
	userHandler := NewUserHandler(qrService, walletService)
	cardHandler := NewCreditCardHandler()

	// Public routes
	app.Get("/health", HealthCheck)
	app.Get("/", func(c *fiber.Ctx) error { return c.SendString("Welcome to OrusPay API!") })

	api := app.Group("/api")
	api.Post("/register", userHandler.RegisterUser)
	api.Post("/login", authHandler.LoginUser)
	api.Post("/refresh", authHandler.RefreshToken)

	// Authenticated routes
	authenticated := app.Group("/api", middleware.AuthMiddleware)

	// Setup different route groups
	setupUserRoutes(authenticated, paymentHandler, userHandler, cardHandler, authHandler)
	setupMerchantRoutes(authenticated, merchantHandler, paymentHandler)
	setupEnterpriseRoutes(authenticated, enterpriseHandler)
	setupAdminRoutes(app)
}

func setupUserRoutes(router fiber.Router, paymentHandler *PaymentHandler, userHandler *UserHandler, cardHandler *CreditCardHandler, authHandler *AuthHandler) {
	// Initialize wallet handler
	walletHandler := NewWalletHandler(walletService)

	// Wallet routes
	wallet := router.Group("/wallet")
	wallet.Get("/", middleware.HasPermission(models.PermissionWalletRead), walletHandler.GetWallet)
	wallet.Post("/topup", middleware.HasPermission(models.PermissionWalletWrite), walletHandler.TopUpWallet)
	wallet.Post("/withdraw", middleware.HasPermission(models.PermissionWalletWrite), walletHandler.WithdrawToCard)

	// Transaction routes
	router.Get("/transactions", userHandler.GetUserTransactions) //✅

	// User account routes
	router.Post("/credit-card", cardHandler.LinkCard)         // Add credit card route
	router.Get("/credit-card", cardHandler.GetCards)          // Get user's cards
	router.Delete("/credit-card/:id", cardHandler.DeleteCard) // Delete a card
	router.Post("/change-password", authHandler.ChangePassword)
	router.Post("/logout", authHandler.LogoutUser)

	// Payment routes
	payments := router.Group("/payment")
	payments.Post("/scan", paymentHandler.ProcessQRPayment) // For users scanning QRs
	payments.Post("/send", paymentHandler.SendMoney)        //✅
}

func setupMerchantRoutes(router fiber.Router, h *MerchantHandler, paymentHandler *PaymentHandler) {
	merchant := router.Group("/merchant", middleware.HasPermission(models.PermissionMerchantRead))

	// Profile Management
	merchant.Post("/", h.CreateMerchant)
	merchant.Get("/profile", h.GetMerchantProfile)
	merchant.Put("/profile", h.UpdateMerchantProfile)

	// Payment Processing
	payments := merchant.Group("/payments")
	payments.Post("/receive", paymentHandler.ProcessQRPayment) // For merchants receiving payments (scanning customer QRs)
	payments.Post("/charge", h.ProcessDirectCharge)            // For direct charges without QR

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
