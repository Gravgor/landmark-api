package models

import (
	"gorm.io/gorm"
)

type Landmark struct {
	gorm.Model
	Name            string  `json:"name" gorm:"not null"`
	Country         string  `json:"country" gorm:"not null;index"`
	City            string  `json:"city" gorm:"not null"`
	Description     string  `json:"description"`
	Height          float64 `json:"height"`
	YearBuilt       int     `json:"yearBuilt"`
	Architect       string  `json:"architect"`
	VisitorsPerYear int     `json:"visitorsPerYear"`
	ImageURL        string  `json:"imageUrl"`
	Latitude        float64 `json:"latitude"`
	Longitude       float64 `json:"longitude"`
}
