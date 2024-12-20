// @title Landmark API
// @version 1.0
// @description This is a landmark API server.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:5050
// @BasePath /api/v1

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-Key
package main

import (
	"landmark-api/internal/api/controllers"
	"landmark-api/internal/api/handlers"
	"landmark-api/internal/config"
	"landmark-api/internal/database"
	"landmark-api/internal/logger"
	"landmark-api/internal/middleware"
	"landmark-api/internal/repository"
	"landmark-api/internal/services"
	"log"
	"net/http"
	"os"
	"time"

	_ "landmark-api/cmd/api/docs"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
	"github.com/sirupsen/logrus"
	"github.com/stripe/stripe-go/v72"
	httpSwagger "github.com/swaggo/http-swagger"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
	}

	// Initialize database connection
	db, err := database.InitDB()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Get underlying *sql.DB instance for connection pool settings
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal("Failed to get underlying *sql.DB instance:", err)
	}
	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")
	rateLimitConfig := config.NewRateLimitConfig()
	cacheConfig := config.NewCacheConfig()
	cacheService, err := services.NewRedisCacheService(cacheConfig)
	if err != nil {
		log.Fatal("Failed to initialize cache service")
	}

	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(25)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	userRepo := repository.NewUserRepository(db)
	subscriptionRepo := repository.NewSubscriptionRepository(db)
	landmarkRepo := repository.NewLandmarkRepository(db)
	apiKeyRepo := repository.NewAPIKeyRepository(db)
	apiUsageRepo := repository.NewAPIUsageRepository(db)

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}

	apiKeyService := services.NewAPIKeyService(apiKeyRepo, userRepo, subscriptionRepo)

	authService := services.NewAuthService(
		userRepo,
		subscriptionRepo,
		apiKeyService,
		jwtSecret,
	)

	auditLogRepo := repository.NewAuditLogRepository(db)
	auditLogService := services.NewAuditLogService(auditLogRepo)
	auditLogHandler := handlers.NewAuditLogHandler(auditLogService)

	landmarkService := services.NewLandmarkService(landmarkRepo)

	authHandler := handlers.NewAuthHandler(authService)
	landmarkHandler := handlers.NewLandmarkHandler(landmarkService, auditLogService, cacheService, db)

	config := &handlers.SuggestionsConfig{
		MaxResults:         15,
		MinSimilarity:      50,
		EnabledSearchTypes: []string{"city", "country", "category", "name"},
		CacheDuration:      5 * time.Minute,
	}
	suggestionHandler, err := handlers.NewSuggestionsHandler(db, cacheService, config)
	if err != nil {
		log.Fatalf("Failed to initialize search capabilities: %v", err)
	}

	rateLimiter := middleware.NewRateLimiter(rateLimitConfig)
	apiUsageService := services.NewAPIUsageService(apiUsageRepo, subscriptionRepo, rateLimitConfig)
	apiUsageHandler := handlers.NewUsageHandler(apiUsageService, authService)

	requestLogRepo := repository.NewRequestLogRepository(db)
	requestLogService := services.NewRequestLogService(requestLogRepo)
	requestLogHandler := handlers.NewRequestLogHandler(requestLogService)
	requestLogger := middleware.NewRequestLogger(requestLogService)

	awsRegion := "eu-north-1"
	awsBucket := "properties-photos"
	if awsRegion == "" {
		log.Fatal("AWS Region is nedeed")
	}
	if awsBucket == "" {
		log.Fatal("AWS Bucket is nedeed")
	}

	fileUploadHandler, err := handlers.NewFileUploadHandler(awsRegion, awsBucket)
	if err != nil {
		log.Fatal("Error with file handler")
	}
	stripeHandler := handlers.NewStripeHandler(authService, subscriptionRepo, userRepo, apiKeyService)

	uptimeService := handlers.NewUptimeService()
	uptimeHandler := handlers.NewUptimeHandler(uptimeService)
	uptimeMiddleware := handlers.NewUptimeMiddleware(uptimeService)

	categoryRepo := repository.NewCategoryRepository(db)
	categoryService := services.NewCategoryService(categoryRepo)
	categoryHandler := handlers.NewCategoryHandler(categoryService)

	landmarkStatsRepo := repository.NewLandmarkStatsRepository(db)
	landmarkStatsService := services.NewLandmarkStatsService(landmarkStatsRepo)
	landmarkStatsHandler := handlers.NewLandmarkStatsHandler(landmarkStatsService)

	router := mux.NewRouter()
	router.Use(middleware.LoggingMiddleware)
	router.Use(uptimeMiddleware.Middleware)

	// Public routes
	router.HandleFunc("/auth/register", authHandler.Register).Methods("POST")
	router.HandleFunc("/auth/login", authHandler.Login).Methods("POST")
	router.HandleFunc("/auth/register-email", authHandler.RegisterWithEmail).Methods("POST")
	router.HandleFunc("/health", controllers.HealthCheckHandler(db)).Methods("GET")
	router.HandleFunc("/swagger", httpSwagger.WrapHandler).Methods("GET")
	router.HandleFunc("/uptime", uptimeHandler.ServeHTTP).Methods("GET")

	contributionRouter := router.PathPrefix("/api/v1/contribution").Subrouter()
	contributionRouter.HandleFunc("/submit-landmark", landmarkHandler.CreateSubmission).Methods("POST")
	contributionRouter.HandleFunc("/submit-photo", fileUploadHandler.SubmitPhotos).Methods("POST")

	// API routes (protected)
	apiRouter := router.PathPrefix("/api/v1").Subrouter()
	apiRouter.Use(middleware.APIKeyMiddleware(apiKeyService))
	apiRouter.Use(rateLimiter.RateLimit(authService, apiUsageService))
	apiRouter.Use(requestLogger.LogRequest)

	// Landmarks routes
	apiRouter.HandleFunc("/landmarks", landmarkHandler.ListLandmarks).Methods("GET")
	apiRouter.HandleFunc("/landmarks/{id}", landmarkHandler.GetLandmark).Methods("GET")
	apiRouter.HandleFunc("/landmarks/country/{country}", landmarkHandler.ListLandmarksByCountry).Methods("GET")
	apiRouter.HandleFunc("/landmarks/name/{name}", landmarkHandler.ListLandmarksByName).Methods("GET")
	apiRouter.HandleFunc("/landmarks/city/{city}", landmarkHandler.ListLandmarksByCity).Methods("GET")
	apiRouter.HandleFunc("/landmarks/category/{category}", landmarkHandler.ListLandmarkByCategory).Methods("GET")
	apiRouter.HandleFunc("/landmarks/search", landmarkHandler.SearchLandmarks).Methods("POST")

	suggestionRouter := router.PathPrefix("/api/v1/suggestions").Subrouter()
	suggestionRouter.Use(middleware.APIKeyMiddleware(apiKeyService))
	suggestionRouter.HandleFunc("/{type}", suggestionHandler.GetSuggestions).Methods("GET").Queries("search", "{search}")
	suggestionRouter.HandleFunc("/landmarks/{id}", landmarkHandler.GetLandmark).Methods("GET")
	suggestionRouter.HandleFunc("/landmarks/country/{country}", landmarkHandler.ListLandmarksByCountry).Methods("GET")
	suggestionRouter.HandleFunc("/landmarks/name/{name}", landmarkHandler.ListLandmarksByName).Methods("GET")
	suggestionRouter.HandleFunc("/landmarks/city/{city}", landmarkHandler.ListLandmarksByCity).Methods("GET")
	suggestionRouter.HandleFunc("/landmarks/category/{category}", landmarkHandler.ListLandmarkByCategory).Methods("GET")
	// User check routes
	userRouter := router.PathPrefix("/user/api/v1").Subrouter()
	userRouter.Use(middleware.AuthMiddleware(authService))
	userRouter.HandleFunc("/validate-token", authHandler.ValidateToken).Methods("GET")
	userRouter.HandleFunc("/me", authHandler.CheckUser).Methods("GET")
	userRouter.HandleFunc("/usage", apiUsageHandler.GetCurrentUsage).Methods("GET")
	userRouter.HandleFunc("/requests/logs", requestLogHandler.GetUserLogs).Methods("GET")
	userRouter.HandleFunc("/update", authHandler.UpdateUser).Methods("PUT")

	subscriptionRouter := router.PathPrefix("/subscription").Subrouter()
	subscriptionRouter.HandleFunc("/create-checkout", stripeHandler.HandleCreateCheckOut).Methods("POST")
	subscriptionRouter.HandleFunc("/create-user-account", authHandler.RegisterSub).Methods("POST")
	subscriptionRouter.HandleFunc("/stripe-webhook", stripeHandler.HandleStripeWebhook).Methods("POST")

	subscriptionRouterManage := router.PathPrefix("/subscription/manage").Subrouter()
	subscriptionRouterManage.Use(middleware.AuthMiddleware(authService))
	subscriptionRouterManage.HandleFunc("/get-billing", stripeHandler.HandleUserBillingInfo).Methods("GET")

	adminRouter := router.PathPrefix("/admin").Subrouter()
	adminRouter.Use(middleware.AdminMiddleware(authService))
	adminRouter.HandleFunc("/landmarks/upload-photo", fileUploadHandler.Upload).Methods("POST")
	adminRouter.HandleFunc("/landmarks/create", landmarkHandler.CreateLandmark).Methods("POST")
	adminRouter.HandleFunc("/landmarks", landmarkHandler.ListAdminLandmarks).Methods("GET")
	adminRouter.HandleFunc("/landmarks/{id}", landmarkHandler.AdminEditHandler).Methods("PUT")
	adminRouter.HandleFunc("/landmarks/{id}", landmarkHandler.AdminDeleteHandler).Methods("DELETE")
	adminRouter.HandleFunc("/landmarks/category", categoryHandler.ListAdminCategories).Methods("GET")
	adminRouter.HandleFunc("/landmarks/stats", landmarkStatsHandler.GetLandmarkStats).Methods("GET")
	adminRouter.HandleFunc("/audit-logs", auditLogHandler.ListAuditLogs).Methods("GET")
	adminRouter.HandleFunc("/submissions/landmarks", landmarkHandler.ListPendingSubmissions).Methods("GET")
	adminRouter.HandleFunc("/submissions/landmarks/approve/{id}", landmarkHandler.ApproveSubmission).Methods("PUT")
	adminRouter.HandleFunc("/submission/landmarks/reject/{id}", landmarkHandler.RejectSubmission).Methods("DELETE")

	go func() {
		for {
			time.Sleep(4 * time.Hour)
			if err := requestLogRepo.DeleteOldLogs(); err != nil {
				log.Printf("Error deleting old logs: %v", err)
			} else {
				log.Println("Old logs deleted successfully")
			}
		}
	}()

	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins: []string{"*"}, // Allow all origins
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowedHeaders: []string{
			"Accept",
			"Authorization",
			"Content-Type",
			"X-CSRF-Token",
			"X-API-Key",
			"*", // Allow all headers
		},
		ExposedHeaders: []string{
			"Link",
		},
		AllowCredentials: false, // Must be false when using AllowedOrigins: ["*"]
		MaxAge:           300,
	})

	// Create server with timeouts
	srv := &http.Server{
		Handler:      corsMiddleware.Handler(router),
		Addr:         ":" + getPort(),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	// Start server
	logger.LogEvent(logrus.InfoLevel, "API started", logrus.Fields{
		"port": "8080",
	})
	log.Fatal(srv.ListenAndServe())
}

func getPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = "5050"
	}
	return port
}
