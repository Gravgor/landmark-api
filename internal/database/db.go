package database

import (
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func InitDB() (*gorm.DB, error) {
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
	/*err := db.Migrator().DropTable(&models.SubmissionLandmark{}, &models.SubmissionLandmarkDetail{}, &models.SubmissionLandmarkImage{})
	if err != nil {
		log.Fatal("Failed to drop tables: ", err)
	}
	return db.AutoMigrate(&models.SubmissionLandmark{}, &models.SubmissionLandmarkDetail{}, &models.SubmissionLandmarkImage{})*/
	return db.AutoMigrate()
}
