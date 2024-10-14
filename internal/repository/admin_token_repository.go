package repository

import (
	"landmark-api/internal/models"
	"time"

	"gorm.io/gorm"
)

type AdminTokenRepository interface {
	GetLatestToken() (*models.AdminToken, error)
	CreateToken(token string) error
	DeleteOldTokens() error
}

type adminTokenRepository struct {
	db *gorm.DB
}

func NewAdminTokenRepository(db *gorm.DB) AdminTokenRepository {
	return &adminTokenRepository{db: db}
}

func (r *adminTokenRepository) GetLatestToken() (*models.AdminToken, error) {
	var token models.AdminToken
	if err := r.db.Order("created_at DESC").First(&token).Error; err != nil {
		return nil, err
	}
	return &token, nil
}

func (r *adminTokenRepository) CreateToken(token string) error {
	return r.db.Create(&models.AdminToken{Token: token}).Error
}

func (r *adminTokenRepository) DeleteOldTokens() error {
	return r.db.Where("created_at < ?", time.Now().Add(-24*time.Hour)).Delete(&models.AdminToken{}).Error
}
