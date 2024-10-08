package db

import (
	"fmt"
	"landmark-api/internal/db/migrations"
	"landmark-api/internal/models"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Connect() *gorm.DB {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Create migrations table
	err = db.AutoMigrate(&models.MigrationRecord{})
	if err != nil {
		log.Fatalf("Failed to create migrations table: %v", err)
	}

	// Run migrations
	if err := runMigrations(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	return db
}

func runMigrations(db *gorm.DB) error {
	migrationsList := migrations.GetMigrations()

	for _, migration := range migrationsList {
		var record models.MigrationRecord
		result := db.Where("name = ?", migration.Name).First(&record)

		if result.Error == gorm.ErrRecordNotFound {
			log.Printf("Running migration: %s", migration.Name)

			err := db.Transaction(func(tx *gorm.DB) error {
				if err := migration.Run(tx); err != nil {
					return err
				}

				return tx.Create(&models.MigrationRecord{Name: migration.Name}).Error
			})

			if err != nil {
				return fmt.Errorf("migration '%s' failed: %v", migration.Name, err)
			}
		} else if result.Error != nil {
			return fmt.Errorf("failed to check migration status: %v", result.Error)
		}
	}

	return nil
}
