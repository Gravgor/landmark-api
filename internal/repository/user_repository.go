package repository

import (
	"context"
	"fmt"
	"landmark-api/internal/errors"
	"landmark-api/internal/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	GetByStripeCustomerID(ctx context.Context, id string) (*models.User, error)
	GrantAccess(ctx context.Context, id uuid.UUID) error
	RevokeAccess(ctx context.Context, id uuid.UUID) error
	Update(ctx context.Context, user *models.User) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *models.User) error {
	result := r.db.WithContext(ctx).Create(user)
	if result.Error != nil {
		return errors.Wrap(result.Error, "failed to create user")
	}
	return nil
}

func (r *userRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var user models.User
	result := r.db.WithContext(ctx).First(&user, "id = ?", id)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, errors.ErrNotFound
		}
		return nil, errors.Wrap(result.Error, "failed to get user by ID")
	}

	return &user, nil
}

func (r *userRepository) GetByStripeCustomerID(ctx context.Context, id string) (*models.User, error) {
	var user models.User
	result := r.db.WithContext(ctx).First(&user, "stripe_id = ?", id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, errors.ErrNotFound
		}
		return nil, errors.Wrap(result.Error, "failed to get user by customer id")
	}
	return &user, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	result := r.db.WithContext(ctx).First(&user, "email = ?", email)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, errors.ErrNotFound
		}
		return nil, errors.Wrap(result.Error, "failed to get user by email")
	}

	return &user, nil
}

func (r *userRepository) Update(ctx context.Context, user *models.User) error {
	result := r.db.WithContext(ctx).Model(user).Updates(map[string]interface{}{
		"email":         user.Email,
		"password_hash": user.PasswordHash,
		"on_boarding":   false,
		"updated_at":    user.UpdatedAt,
	})

	if result.Error != nil {
		return errors.Wrap(result.Error, "failed to update user")
	}

	if result.RowsAffected == 0 {
		return errors.ErrNotFound
	}

	return nil
}

func (r *userRepository) GrantAccess(ctx context.Context, userID uuid.UUID) error {
	var user models.User
	result := r.db.WithContext(ctx).Model(&user).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"has_access":        true,
			"access_granted_at": time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to grant access: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("no user found with ID: %s", userID)
	}

	return nil
}

func (r *userRepository) RevokeAccess(ctx context.Context, userID uuid.UUID) error {
	var user models.User
	result := r.db.WithContext(ctx).Model(&user).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"has_access":        false,
			"access_revoked_at": time.Now(),
			"access_granted_at": nil,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to revoke access: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("no user found with ID: %s", userID)
	}

	return nil
}

func (r *userRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.User{}, "id = ?", id)

	if result.Error != nil {
		return errors.Wrap(result.Error, "failed to delete user")
	}

	if result.RowsAffected == 0 {
		return errors.ErrNotFound
	}

	return nil
}
