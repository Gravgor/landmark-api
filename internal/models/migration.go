package models

import "gorm.io/gorm"

// MigrationRecord keeps track of which migrations have been run
type MigrationRecord struct {
	gorm.Model
	Name string `gorm:"uniqueIndex"`
}
