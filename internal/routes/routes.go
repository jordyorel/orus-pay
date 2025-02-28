// Package routes defines the API routing configuration.
// It sets up all HTTP routes and their corresponding handlers,
// including middleware and authentication requirements.
package routes

import (
	"orus/internal/config"
	"orus/internal/handlers"
	"orus/internal/middleware"
	"orus/internal/models"
	"orus/internal/repositories"
	"orus/internal/services/auth"
	creditcard "orus/internal/services/credit-card"
	"orus/internal/services/dashboard"
	"orus/internal/services/merchant"
	"orus/internal/services/payment"
	qr "orus/internal/services/qr_code"
	"orus/internal/services/transaction"
	"orus/internal/services/user"
	"orus/internal/services/wallet"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

var walletService wallet.Service

// SetupRoutes configures all application routes.
// It groups routes by functionality and applies appropriate middleware.
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
	jwtSecret := config.GetEnv("JWT_SECRET", "your-secret-key")
	refreshSecret := config.GetEnv("REFRESH_SECRET", "your-refresh-secret")
	authService := auth.NewService(userRepo, jwtSecret, refreshSecret)
	authHandler := handlers.NewAuthHandler(authService, refreshSecret)

	// Initialize services in correct order
	cardService := creditcard.NewService(cardRepo)
	userService := user.NewService(userRepo)
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

	// Initialize dashboard service and handler
	dashboardService := dashboard.NewService(
		repositories.NewTransactionRepository(db),
		repositories.NewWalletRepository(db),
		repositories.NewMerchantRepository(db),
		userRepo,
		db,
	)
	dashboardHandler := handlers.NewDashboardHandler(dashboardService)

	// Initialize handlers
	paymentHandler := handlers.NewPaymentHandler(qrService, paymentService)
	merchantHandler := handlers.NewMerchantHandler(
		merchant.NewService(qrService, transactionService, walletService),
		qrService,
		repositories.NewTransactionRepository(db),
	)
	// enterpriseHandler := handlers.NewEnterpriseHandler()
	userHandler := handlers.NewUserHandler(userService, walletService, qrService)
	cardHandler := handlers.NewCreditCardHandler(cardRepo)

	// Public routes
	api := app.Group("/api")
	api.Post("/register", userHandler.RegisterUser)
	api.Post("/login", authHandler.LoginUser)
	api.Post("/refresh", authHandler.RefreshToken)

	// Create middleware instance
	authMiddleware := middleware.NewAuthMiddleware(authService)

	// Protected routes with auth middleware
	protected := api.Use(authMiddleware.Handler)

	// Setup different route groups
	setupUserRoutes(protected, paymentHandler, userHandler, cardHandler, authHandler)
	setupMerchantRoutes(protected, merchantHandler, paymentHandler)
	// setupEnterpriseRoutes(protected, enterpriseHandler)
	setupAdminRoutes(app, jwtSecret)

	// Add dashboard routes
	addDashboardRoutes(app, dashboardHandler, authMiddleware.Handler)
}

func setupUserRoutes(router fiber.Router, paymentHandler *handlers.PaymentHandler, userHandler *handlers.UserHandler, cardHandler *handlers.CreditCardHandler, authHandler *handlers.AuthHandler) {
	// Initialize wallet handler
	walletHandler := handlers.NewWalletHandler(walletService)

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

func setupMerchantRoutes(router fiber.Router, h *handlers.MerchantHandler, paymentHandler *handlers.PaymentHandler) {
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

	// Transactions
	merchant.Get("/transactions", h.GetMerchantTransactions)
}

func setupAdminRoutes(app *fiber.App, jwtSecret string) {
	// First apply auth middleware, then admin check
	admin := app.Group("/api/admin", middleware.CreateAuthMiddleware(jwtSecret), middleware.AdminAuthMiddleware)

	admin.Get("/transactions", middleware.HasPermission(models.PermissionReadAdmin), handlers.GetAllTransactions)
	admin.Get("/users", middleware.HasPermission(models.PermissionReadAdmin), handlers.GetUsersPaginated)
	admin.Delete("/users/:id", middleware.HasPermission(models.PermissionWriteAdmin), handlers.DeleteUser)
	admin.Get("/wallets", middleware.HasPermission(models.PermissionWriteAdmin), handlers.GetAllWallets)
	admin.Get("/credit-cards", middleware.HasPermission(models.PermissionWriteAdmin), handlers.GetAllCreditCards)
}

func addDashboardRoutes(app *fiber.App, handler *handlers.DashboardHandler, authMiddleware fiber.Handler) {
	dashboard := app.Group("/api/dashboard", authMiddleware)

	// User dashboard routes
	dashboard.Get("/user", handler.GetUserDashboard)
	dashboard.Get("/user/analytics", handler.GetTransactionAnalytics)

	// Merchant dashboard routes
	dashboard.Get("/merchant", middleware.HasPermission(models.PermissionMerchantRead), handler.GetMerchantDashboard)
	dashboard.Get("/merchant/analytics", middleware.HasPermission(models.PermissionMerchantRead), handler.GetTransactionAnalytics)
}
