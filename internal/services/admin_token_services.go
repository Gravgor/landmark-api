package services

import (
	"crypto/rand"
	"encoding/hex"
	"landmark-api/internal/repository"
	"log"
	"time"

	"gorm.io/gorm"
)

type AdminTokenService struct {
	repo repository.AdminTokenRepository
}

func NewAdminTokenService(repo repository.AdminTokenRepository) *AdminTokenService {
	return &AdminTokenService{repo: repo}
}

func (s *AdminTokenService) GetOrCreateAdminToken() (string, error) {
	token, err := s.repo.GetLatestToken()
	if err == gorm.ErrRecordNotFound || time.Since(token.CreatedAt) > 24*time.Hour {
		newToken := generateSecureToken(32)
		if err := s.repo.CreateToken(newToken); err != nil {
			return "", err
		}
		if err := s.repo.DeleteOldTokens(); err != nil {
			log.Printf("Error deleting old tokens: %v", err)
		}
		return newToken, nil
	} else if err != nil {
		return "", err
	}
	return token.Token, nil
}

func generateSecureToken(length int) string {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}
