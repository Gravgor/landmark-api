package services

import (
	"context"
	"errors"
	"fmt"
	"landmark-api/internal/models"
	"landmark-api/internal/repository"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/customer"
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
	Register(ctx context.Context, email, password, name string) (*models.User, error)
	RegisterSub(ctx context.Context, email, password, name string) (*models.User, error)
	RegisterWithEmail(ctx context.Context, email string) (*models.User, error)
	Login(ctx context.Context, email, password string) (string, error)
	UpdateUser(ctx context.Context, userID uuid.UUID, name, password string) error
	VerifyToken(token string) (*models.User, *models.Subscription, error)
	VerifyTokenAdmin(token string) (*models.User, *models.Subscription, error)
	GetAPIKey(ctx context.Context, userID uuid.UUID) (*models.APIKey, error)
	GetCurrentSubscription(ctx context.Context, userID uuid.UUID) (*models.Subscription, error)
	GetUserByID(ctx context.Context, userID uuid.UUID) (*models.User, error)
	GetUserByStripeCustomerID(ctx context.Context, customerID string) (*models.User, error)
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

func (s *authService) Register(ctx context.Context, email, password, name string) (*models.User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		ID:           uuid.New(),
		Name:         name,
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

func (s *authService) RegisterSub(ctx context.Context, email, password, name string) (*models.User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	params := &stripe.CustomerParams{
		Email: stripe.String(email),
	}
	c, err := customer.New(params)
	if err != nil {
		return nil, err
	}
	user := &models.User{
		ID:           uuid.New(),
		Name:         name,
		Email:        email,
		PasswordHash: string(hashedPassword),
		StripeID:     c.ID,
		HasAccess:    false,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *authService) RegisterWithEmail(ctx context.Context, email string) (*models.User, error) {
	// Generate a random password
	password := generateRandomPassword(12)

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		ID:           uuid.New(),
		Email:        email,
		Name:         "",
		OnBoarding:   true,
		PasswordHash: string(hashedPassword),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	// Assign API key
	_, err = s.apiKeyService.AssignAPIKeyToUser(ctx, user.ID)
	if err != nil {
		return user, err
	}

	// Create subscription
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
		return user, err
	}

	if err := s.sendPasswordEmail(user.Email, password); err != nil {
		return user, nil
	}

	return user, nil
}

func (s *authService) GetUserByID(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *authService) GetUserByStripeCustomerID(ctx context.Context, customerID string) (*models.User, error) {
	user, err := s.userRepo.GetByStripeCustomerID(ctx, customerID)
	if err != nil {
		return nil, err
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
		"role":            user.Role,
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

func (s *authService) GetCurrentSubscription(ctx context.Context, userID uuid.UUID) (*models.Subscription, error) {
	subscription, err := s.subscriptionRepo.GetActiveByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return subscription, nil
}

func (s *authService) UpdateUser(ctx context.Context, userID uuid.UUID, name, password string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	if name != "" {
		user.Name = name
	}

	if password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		user.PasswordHash = string(hashedPassword)
	}

	return s.userRepo.Update(ctx, user)
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

var (
	ErrUnauthorized = errors.New("user is not authorized as admin")
)

func (s *authService) VerifyTokenAdmin(tokenString string) (*models.User, *models.Subscription, error) {
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

	if user.Role != "admin" {
		return nil, nil, ErrUnauthorized
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

func generateRandomPassword(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()_+"
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	password := make([]byte, length)
	for i := range password {
		password[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(password)
}

func (s *authService) sendPasswordEmail(email, password string) error {
	from := mail.NewEmail("Landmark API", "noreply@landmark-api.com")
	subject := "Welcome to Landmark API Family!"
	to := mail.NewEmail("", email)
	log.Println(email)

	htmlContent := fmt.Sprintf(`
		<html lang="en" style="background-image: linear-gradient(to right, #4338ca, #312e81); color: #ffffff; font-family: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, 'Noto Sans', sans-serif, 'Apple Color Emoji', 'Segoe UI Emoji', 'Segoe UI Symbol', 'Noto Color Emoji';">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body style="background-image: linear-gradient(to right, #4338ca, #312e81); color: #ffffff; font-family: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, 'Noto Sans', sans-serif, 'Apple Color Emoji', 'Segoe UI Emoji', 'Segoe UI Symbol', 'Noto Color Emoji';">
    <div style="max-width: 42rem; margin-left: auto; margin-right: auto; padding: 2rem;">
        <div style="background-color: #3730a3; padding: 2rem; border-radius: 0.5rem; box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.1), 0 4px 6px -2px rgba(0, 0, 0, 0.05);">
            <h1 style="font-size: 1.875rem; line-height: 2.25rem; font-weight: 700; margin-bottom: 1.5rem;">Welcome to Landmark API!</h1>
            <p style="margin-bottom: 1rem;">Your account has been created successfully. Here are your login details:</p>
            <div style="background-color: #312e81; padding: 1rem; border-radius: 0.375rem; margin-bottom: 1.5rem;">
                <p style="margin-bottom: 0.5rem;"><strong>Email:</strong> %s</p>
                <p style="margin-bottom: 0;"><strong>Temporary Password:</strong> %s</p>
            </div>
            <p style="margin-bottom: 1.5rem;">Please log in and change your password as soon as possible.</p>
            <a href="https://landmark-api.com/auth?login=true" style="background-color: #2563eb; color: #ffffff; font-weight: 700; padding: 0.75rem 1.5rem; border-radius: 0.5rem; display: inline-block; text-decoration: none; transition-property: background-color, border-color, color, fill, stroke, opacity, box-shadow, transform; transition-timing-function: cubic-bezier(0.4, 0, 0.2, 1); transition-duration: 300ms;">Login Now</a>
        </div>
    </div>
</body>
</html>
	`, email, password)

	message := mail.NewSingleEmail(from, subject, to, "", htmlContent)
	client := sendgrid.NewSendClient(os.Getenv("SENDGRID_API_KEY"))
	response, err := client.Send(message)
	if err != nil {
		log.Println(err)
	} else {
		fmt.Println(response.StatusCode)
		fmt.Println(response.Body)
		fmt.Println(response.Headers)
	}

	if response.StatusCode >= 400 {
		return fmt.Errorf("error sending email: %v", response.Body)
	}

	return nil
}
