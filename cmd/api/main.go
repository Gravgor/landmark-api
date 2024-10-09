package main

import (
	"fmt"
	"landmark-api/internal/api/handlers"
	"landmark-api/internal/middleware"
	"landmark-api/internal/repository"
	"landmark-api/internal/services"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
	}

	// Initialize database connection
	db, err := initDB()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Get underlying *sql.DB instance for connection pool settings
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal("Failed to get underlying *sql.DB instance:", err)
	}

	// Configure connection pool
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(25)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	subscriptionRepo := repository.NewSubscriptionRepository(db)
	landmarkRepo := repository.NewLandmarkRepository(db)
	apiKeyRepo := repository.NewAPIKeyRepository(db)

	// Initialize services
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

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService)
	landmarkHandler := handlers.NewLandmarkHandler(landmarkService, db)

	// Initialize rate limiter
	rateLimiter := middleware.NewRateLimiter()

	// Initialize router
	router := mux.NewRouter()

	// Public routes
	router.HandleFunc("/auth/register", authHandler.Register).Methods("POST")
	router.HandleFunc("/auth/login", authHandler.Login).Methods("POST")
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	}).Methods("GET")

	// API routes (protected)
	apiRouter := router.PathPrefix("/api/v1").Subrouter()
	apiRouter.Use(middleware.AuthMiddleware(authService))
	apiRouter.Use(middleware.APIKeyMiddleware(apiKeyService))
	apiRouter.Use(rateLimiter.RateLimit)

	// Landmarks routes
	apiRouter.HandleFunc("/landmarks", landmarkHandler.ListLandmarks).Methods("GET")
	apiRouter.HandleFunc("/landmarks/{id}", landmarkHandler.GetLandmark).Methods("GET")
	apiRouter.HandleFunc("/landmarks/country/{country}", landmarkHandler.ListLandmarksByCountry).Methods("GET")
	apiRouter.HandleFunc("/landmarks/name/{name}", landmarkHandler.ListLandmarksByName).Methods("GET")

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
	log.Printf("Server starting on port %s...", getPort())
	log.Fatal(srv.ListenAndServe())
}

func initDB() (*gorm.DB, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is required")
	}

	// Configure GORM logger
	gormLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Info,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)

	// Open connection
	db, err := gorm.Open(postgres.Open(dbURL), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("error opening database: %v", err)
	}

	if err := autoMigrate(db); err != nil {
		return nil, fmt.Errorf("error migrating database: %v", err)
	}
	//migrations.MigrateLandmarks(db)

	return db, nil
}

func autoMigrate(db *gorm.DB) error {
	return db.AutoMigrate()
}

func getPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = "5050"
	}
	return port
}
