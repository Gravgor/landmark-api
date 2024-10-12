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

	apiKeyService := services.NewAPIKeyService(apiKeyRepo)

	authService := services.NewAuthService(
		userRepo,
		subscriptionRepo,
		apiKeyService,
		jwtSecret,
	)
	landmarkService := services.NewLandmarkService(landmarkRepo)

	authHandler := handlers.NewAuthHandler(authService)
	landmarkHandler := handlers.NewLandmarkHandler(landmarkService, cacheService, db)

	rateLimiter := middleware.NewRateLimiter(rateLimitConfig)
	apiUsageService := services.NewAPIUsageService(apiUsageRepo, rateLimitConfig)
	apiUsageHandler := handlers.NewUsageHandler(apiUsageService, authService)

	requestLogRepo := repository.NewRequestLogRepository(db)
	requestLogService := services.NewRequestLogService(requestLogRepo)
	requestLogHandler := handlers.NewRequestLogHandler(requestLogService)
	requestLogger := middleware.NewRequestLogger(requestLogService)

	fileUploadHandler := handlers.NewFileUploadHandler()
	stripeHandler := handlers.NewStripeHandler(authService, subscriptionRepo, userRepo)

	router := mux.NewRouter()
	router.Use(middleware.LoggingMiddleware)

	// Public routes
	router.HandleFunc("/auth/register", authHandler.Register).Methods("POST")
	router.HandleFunc("/auth/login", authHandler.Login).Methods("POST")
	router.HandleFunc("/health", controllers.HealthCheckHandler(db)).Methods("GET")
	router.HandleFunc("/swagger", httpSwagger.WrapHandler).Methods("GET")

	// API routes (protected)
	apiRouter := router.PathPrefix("/api/v1").Subrouter()
	apiRouter.Use(middleware.AuthMiddleware(authService))
	apiRouter.Use(middleware.APIKeyMiddleware(apiKeyService))
	apiRouter.Use(rateLimiter.RateLimit(authService, apiUsageService))
	apiRouter.Use(requestLogger.LogRequest)

	// Landmarks routes
	apiRouter.HandleFunc("/landmarks", landmarkHandler.ListLandmarks).Methods("GET")
	apiRouter.HandleFunc("/landmarks/{id}", landmarkHandler.GetLandmark).Methods("GET")
	apiRouter.HandleFunc("/landmarks/country/{country}", landmarkHandler.ListLandmarksByCountry).Methods("GET")
	apiRouter.HandleFunc("/landmarks/name/{name}", landmarkHandler.ListLandmarksByName).Methods("GET")
	apiRouter.HandleFunc("/landmarks/category/{category}", landmarkHandler.ListLandmarkByCategory).Methods("GET")
	apiRouter.HandleFunc("/landmarks/search", landmarkHandler.SearchLandmarks).Methods("POST")

	// User check routes
	userRouter := router.PathPrefix("/user/api/v1").Subrouter()
	userRouter.Use(middleware.AuthMiddleware(authService))
	userRouter.HandleFunc("/validate-token", authHandler.ValidateToken).Methods("GET")
	userRouter.HandleFunc("/me", authHandler.CheckUser).Methods("GET")
	userRouter.HandleFunc("/usage", apiUsageHandler.GetCurrentUsage).Methods("GET")
	userRouter.HandleFunc("/requests/logs", requestLogHandler.GetUserLogs).Methods("GET")

	subscriptionRouter := router.PathPrefix("/subscription").Subrouter()
	subscriptionRouter.HandleFunc("/create-checkout", stripeHandler.HandleCreateCheckOut).Methods("POST")
	subscriptionRouter.HandleFunc("/create-user-account", authHandler.RegisterSub).Methods("POST")
	subscriptionRouter.HandleFunc("/stripe-webhook", stripeHandler.HandleStripeWebhook).Methods("POST")

	tokenRepo := repository.NewAdminTokenRepository(db)
	tokenService := services.NewAdminTokenService(tokenRepo)
	go func() {
		for {
			time.Sleep(24 * time.Hour)
			if _, err := tokenService.GetOrCreateAdminToken(); err != nil {
				log.Printf("Error updating admin token: %v", err)
			}
		}
	}()
	adminPath, err := tokenService.GetOrCreateAdminToken()
	if err != nil {
		log.Fatal(err)
	}

	adminRouter := router.PathPrefix("/" + adminPath + "/api/v1").Subrouter()
	adminRouter.Use(middleware.AdminMiddleware(authService))
	adminRouter.HandleFunc("/landmarks/upload-photo", fileUploadHandler.Upload).Methods("POST")
	adminRouter.HandleFunc("/landmarks/create", landmarkHandler.CreateLandmark).Methods("POST")

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
		AllowedOrigins: []string{"http://localhost:3000"},
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
		},
		ExposedHeaders: []string{
			"Link",
		},
		AllowCredentials: true,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
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
