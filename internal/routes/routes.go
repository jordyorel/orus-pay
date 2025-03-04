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
	"orus/internal/services/dispute"
	"orus/internal/services/merchant"
	"orus/internal/services/payment"
	qr "orus/internal/services/qr_code"
	"orus/internal/services/transaction"
	"orus/internal/services/user"
	"orus/internal/services/wallet"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

var walletService wallet.Service

// SetupRoutes configures all application routes.
// It groups routes by functionality and applies appropriate middleware.
func SetupRoutes(app *fiber.App, db *gorm.DB) {
	// Initialize repositories
	walletRepo := repositories.NewWalletRepository(repositories.DB)
	userRepo := repositories.NewUserRepository(repositories.DB, repositories.CacheService)
	cardRepo := repositories.NewCreditCardRepository(repositories.DB)
	qrRepo := repositories.NewQRCodeRepository(repositories.DB)

	// Initialize auth service and handler
	jwtSecret := config.GetEnv("JWT_SECRET", "orus")
	refreshSecret := config.GetEnv("REFRESH_SECRET", "your-refresh-secret")
	authService := auth.NewService(userRepo, jwtSecret, refreshSecret)
	authHandler := handlers.NewAuthHandler(authService, refreshSecret)

	// Initialize services in correct order
	cardService := creditcard.NewService(cardRepo)
	userService := user.NewService(userRepo)
	walletService = wallet.NewService(
		walletRepo,
		repositories.CacheService,
		cardService,
		wallet.WalletConfig{},
		&wallet.NoopMetricsCollector{},
	)

	transactionService := transaction.NewService(
		repositories.DB,
		walletService,
		walletService,
		repositories.CacheService,
	)

	qrService := qr.NewService(
		repositories.DB,
		qrRepo,
		repositories.CacheService,
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

	// Initialize dispute service and handler
	disputeService := dispute.NewService(
		repositories.NewDisputeRepository(db),
		repositories.NewTransactionRepository(db),
		repositories.NewUserRepository(repositories.DB, repositories.CacheService),
		db,
	)
	disputeHandler := handlers.NewDisputeHandler(disputeService)

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

	// Public endpoints (no auth required)
	api.Post("/login", authHandler.LoginUser)       // This becomes /api/login
	api.Post("/register", userHandler.RegisterUser) // This becomes /api/register
	api.Post("/refresh", authHandler.RefreshToken)  // This becomes /api/refresh

	// Debug endpoints (public)
	api.Get("/debug/token-version/:id", authHandler.GetTokenVersion)
	api.Get("/debug/token", authHandler.DebugToken)

	// Also add a root welcome route
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Welcome to Orus API",
			"version": "1.0.0",
			"docs":    "/api",
		})
	})

	// Create middleware instance
	authMiddleware := middleware.NewAuthMiddleware(authService)

	// Protected routes with auth middleware
	protected := api.Use(authMiddleware.Handler) // Auth middleware starts here

	// Setup different route groups
	setupUserRoutes(protected, paymentHandler, userHandler, cardHandler, authHandler, qrService)
	setupMerchantRoutes(protected, merchantHandler, paymentHandler)
	// setupEnterpriseRoutes(protected, enterpriseHandler)
	setupAdminRoutes(app, authMiddleware)
	setupDisputeRoutes(protected, disputeHandler)

	// Add dashboard routes
	addDashboardRoutes(app, dashboardHandler, authMiddleware.Handler)

	// Add debug endpoint for protected routes
	protected.Get("/debug/claims", func(c *fiber.Ctx) error {
		claims, ok := c.Locals("claims").(*models.UserClaims)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "No claims found",
			})
		}

		return c.JSON(fiber.Map{
			"user_id":       claims.UserID,
			"email":         claims.Email,
			"role":          claims.Role,
			"permissions":   claims.Permissions,
			"token_version": claims.TokenVersion,
		})
	})

	// Add temporary cache stats route
	protected.Get("/test/cache-stats", handlers.CacheStats)
}

func setupUserRoutes(router fiber.Router, paymentHandler *handlers.PaymentHandler, userHandler *handlers.UserHandler, cardHandler *handlers.CreditCardHandler, authHandler *handlers.AuthHandler, qrService qr.Service) {
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

	// QR code routes
	qrHandler := handlers.NewQRHandler(qrService)
	router.Get("/qr-codes", middleware.HasPermission(models.PermissionWalletRead), qrHandler.GetUserQRCodes)
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

func setupAdminRoutes(app *fiber.App, authMiddleware *middleware.AuthMiddleware) {
	// Use the existing auth middleware instance
	admin := app.Group("/api/admin", authMiddleware.Handler, middleware.AdminAuthMiddleware)

	admin.Get("/transactions", middleware.HasPermission(models.PermissionReadAdmin), handlers.GetAllTransactions)
	admin.Get("/users", middleware.HasPermission(models.PermissionReadAdmin), handlers.GetUsersPaginated)
	admin.Delete("/users/:id", middleware.HasPermission(models.PermissionWriteAdmin), handlers.DeleteUser)
	admin.Get("/wallets", middleware.HasPermission(models.PermissionWriteAdmin), handlers.GetAllWallets)
	admin.Get("/credit-cards", middleware.HasPermission(models.PermissionWriteAdmin), handlers.GetAllCreditCards)

	// Add cache stats endpoint to admin routes
	admin.Get("/cache-stats", handlers.CacheStats)

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

func setupDisputeRoutes(router fiber.Router, disputeHandler *handlers.DisputeHandler) {
	dispute := router.Group("/disputes")

	dispute.Post("/", disputeHandler.FileDispute)                                                                       // Endpoint to file a dispute
	dispute.Get("/", disputeHandler.GetDisputes)                                                                        // Endpoint to get all disputes for a merchant
	dispute.Get("/merchant", disputeHandler.GetMerchantDisputes)                                                        // New endpoint to get merchant disputes
	dispute.Post("/:id/refund", middleware.HasPermission(models.PermissionMerchantWrite), disputeHandler.RefundDispute) // New endpoint for processing refunds
}
