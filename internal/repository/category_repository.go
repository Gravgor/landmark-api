package repository

import (
	"context"
	"landmark-api/internal/models"

	"gorm.io/gorm"
)

type CategoryRepository interface {
	ListAllCategories(ctx context.Context) ([]string, error)
}

type categoryRepository struct {
	db *gorm.DB
}

func NewCategoryRepository(db *gorm.DB) CategoryRepository {
	return &categoryRepository{
		db: db,
	}
}

func (r *categoryRepository) ListAllCategories(ctx context.Context) ([]string, error) {
	var categories []string
	err := r.db.WithContext(ctx).
		Model(&models.Landmark{}).
		Distinct("category").
		Pluck("category", &categories).
		Error
	return categories, err
}
