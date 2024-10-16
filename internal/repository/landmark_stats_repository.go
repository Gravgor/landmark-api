package repository

import (
	"context"
	"landmark-api/internal/models"

	"gorm.io/gorm"
)

type LandmarkStatsRepository interface {
	GetTotalLandmarks(ctx context.Context) (int64, error)
	GetLandmarksByCategory(ctx context.Context) (map[string]int64, error)
	GetLandmarksByCountry(ctx context.Context) (map[string]int64, error)
	GetRecentlyAddedLandmarks(ctx context.Context, limit int) ([]models.Landmark, error)
}

type landmarkStatsRepository struct {
	db *gorm.DB
}

func NewLandmarkStatsRepository(db *gorm.DB) LandmarkStatsRepository {
	return &landmarkStatsRepository{
		db: db,
	}
}

func (r *landmarkStatsRepository) GetTotalLandmarks(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Landmark{}).Count(&count).Error
	return count, err
}

func (r *landmarkStatsRepository) GetLandmarksByCategory(ctx context.Context) (map[string]int64, error) {
	var results []struct {
		Category string
		Count    int64
	}
	err := r.db.WithContext(ctx).Model(&models.Landmark{}).
		Select("category, count(*) as count").
		Group("category").
		Find(&results).Error

	if err != nil {
		return nil, err
	}

	landmarksByCategory := make(map[string]int64)
	for _, result := range results {
		landmarksByCategory[result.Category] = result.Count
	}
	return landmarksByCategory, nil
}

func (r *landmarkStatsRepository) GetLandmarksByCountry(ctx context.Context) (map[string]int64, error) {
	var results []struct {
		Country string
		Count   int64
	}
	err := r.db.WithContext(ctx).Model(&models.Landmark{}).
		Select("country, count(*) as count").
		Group("country").
		Find(&results).Error

	if err != nil {
		return nil, err
	}

	landmarksByCountry := make(map[string]int64)
	for _, result := range results {
		landmarksByCountry[result.Country] = result.Count
	}
	return landmarksByCountry, nil
}

func (r *landmarkStatsRepository) GetRecentlyAddedLandmarks(ctx context.Context, limit int) ([]models.Landmark, error) {
	var landmarks []models.Landmark
	err := r.db.WithContext(ctx).
		Order("created_at DESC").
		Limit(limit).
		Find(&landmarks).Error
	return landmarks, err
}
