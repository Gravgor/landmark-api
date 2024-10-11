package services

import (
	"context"
	"errors"
	"landmark-api/internal/models"
	"landmark-api/internal/repository"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type contextKey string

const (
	UserContextKey         contextKey = "user"
	SubscriptionContextKey contextKey = "subscription"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
)

type AuthService interface {
	Register(ctx context.Context, email, password string) (*models.User, error)
	Login(ctx context.Context, email, password string) (string, error)
	VerifyToken(token string) (*models.User, *models.Subscription, error)
	GetAPIKey(ctx context.Context, userID uuid.UUID) (*models.APIKey, error)
}

type authService struct {
	userRepo         repository.UserRepository
	subscriptionRepo repository.SubscriptionRepository
	apiKeyService    APIKeyService
	jwtSecret        string
}

func NewAuthService(
	userRepo repository.UserRepository,
	subscriptionRepo repository.SubscriptionRepository,
	apiKeyService APIKeyService,
	jwtSecret string,
) AuthService {
	return &authService{
		userRepo:         userRepo,
		subscriptionRepo: subscriptionRepo,
		apiKeyService:    apiKeyService,
		jwtSecret:        jwtSecret,
	}
}

func (s *authService) Register(ctx context.Context, email, password string) (*models.User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: string(hashedPassword),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	_, err = s.apiKeyService.AssignAPIKeyToUser(ctx, user.ID)
	if err != nil {
		return user, err
	}

	subscription := &models.Subscription{
		ID:        uuid.New(),
		UserID:    user.ID,
		PlanType:  models.FreePlan,
		StartDate: time.Now(),
		Status:    "active",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.subscriptionRepo.Create(ctx, subscription); err != nil {
		// Consider handling this error appropriately
		return user, err
	}

	return user, nil
}

func (s *authService) Login(ctx context.Context, email, password string) (string, error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return "", err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", ErrInvalidCredentials
	}

	subscription, err := s.subscriptionRepo.GetActiveByUserID(ctx, user.ID)
	if err != nil {
		return "", err
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":         user.ID.String(),
		"subscription_id": subscription.ID.String(),
		"plan_type":       string(subscription.PlanType),
		"exp":             time.Now().Add(time.Hour * 24).Unix(),
	})

	return token.SignedString([]byte(s.jwtSecret))
}

func (s *authService) GetAPIKey(ctx context.Context, userID uuid.UUID) (*models.APIKey, error) {
	userKey, err := s.apiKeyService.GetAPIKeyByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return userKey, nil

}

func (s *authService) VerifyToken(tokenString string) (*models.User, *models.Subscription, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(s.jwtSecret), nil
	})

	if err != nil || !token.Valid {
		return nil, nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, nil, ErrInvalidToken
	}

	userID, err := uuid.Parse(claims["user_id"].(string))
	if err != nil {
		return nil, nil, ErrInvalidToken
	}

	user, err := s.userRepo.GetByID(context.Background(), userID)
	if err != nil {
		return nil, nil, err
	}

	subscription, err := s.subscriptionRepo.GetActiveByUserID(context.Background(), userID)
	if err != nil {
		return nil, nil, err
	}

	return user, subscription, nil
}

// Helper function to add user and subscription to context
func WithUserAndSubscriptionContext(ctx context.Context, user *models.User, subscription *models.Subscription) context.Context {
	ctx = context.WithValue(ctx, UserContextKey, user)
	return context.WithValue(ctx, SubscriptionContextKey, subscription)
}

// Helper function to get user from context
func UserFromContext(ctx context.Context) (*models.User, bool) {
	user, ok := ctx.Value(UserContextKey).(*models.User)
	return user, ok
}

// Helper function to get subscription from context
func SubscriptionFromContext(ctx context.Context) (*models.Subscription, bool) {
	subscription, ok := ctx.Value(SubscriptionContextKey).(*models.Subscription)
	return subscription, ok
}
