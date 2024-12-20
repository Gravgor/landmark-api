package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"landmark-api/internal/models"
	"landmark-api/internal/repository"
	"landmark-api/internal/services"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/checkout/session"
	"github.com/stripe/stripe-go/v72/invoice"
	"github.com/stripe/stripe-go/v72/sub"
)

type StripeHandler struct {
	authService   services.AuthService
	subRepo       repository.SubscriptionRepository
	userRepo      repository.UserRepository
	apiKeyService services.APIKeyService
}

func NewStripeHandler(auth services.AuthService, subRepo repository.SubscriptionRepository, userRepo repository.UserRepository, apiKeyService services.APIKeyService) *StripeHandler {
	return &StripeHandler{
		authService:   auth,
		subRepo:       subRepo,
		userRepo:      userRepo,
		apiKeyService: apiKeyService,
	}
}

const (
	PlanTypeFree    = "free"
	PlanTypeMonthly = "monthly"
	PlanTypeAnnual  = "annual"

	ErrUserNotFound    = "user not found"
	ErrNoStripeID      = "user doesn't have a Stripe ID"
	ErrInvalidPlanType = "invalid plan type"
	ErrCreateCheckout  = "error creating checkout session"
	ErrNoPriceID       = "no price ID found for the selected plan"
)

func (h *StripeHandler) HandleCreateCheckOut(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID   uuid.UUID `json:"userId"`
		PlanType string    `json:"planType"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user, err := h.authService.GetUserByID(r.Context(), req.UserID)
	if err != nil {
		http.Error(w, ErrUserNotFound, http.StatusNotFound)
		return
	}

	if user.StripeID == "" {
		http.Error(w, ErrNoStripeID, http.StatusBadRequest)
		return
	}

	priceID, err := h.getPriceIDForPlan(req.PlanType)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	sessionID, err := h.createStripeCheckoutSession(user.StripeID, priceID)
	if err != nil {
		http.Error(w, ErrCreateCheckout, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"sessionId": sessionID})
}

func (h *StripeHandler) getPriceIDForPlan(planType string) (string, error) {
	switch planType {
	case PlanTypeFree:
		return os.Getenv("STRIPE_MONTHLY_FREE_PRICE_ID"), nil
	case PlanTypeMonthly:
		return os.Getenv("STRIPE_MONTHLY_PRICE_ID"), nil
	case PlanTypeAnnual:
		return os.Getenv("STRIPE_ANNUAL_PRICE_ID"), nil
	default:
		return "", errors.New(ErrInvalidPlanType)
	}
}

func (h *StripeHandler) createStripeCheckoutSession(customerID, priceID string) (string, error) {
	params := &stripe.CheckoutSessionParams{
		Customer: stripe.String(customerID),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(1),
			},
		},
		Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		SuccessURL: stripe.String("https://www.landmark-api.com/success"),
		CancelURL:  stripe.String("https://www.landmark-api.com/cancel"),
	}

	if priceID == os.Getenv("STRIPE_MONTHLY_FREE_PRICE_ID") {
		params.Discounts = []*stripe.CheckoutSessionDiscountParams{
			{
				Coupon: stripe.String("GMBDmApc"),
			},
		}
	} else {
		params.PaymentMethodTypes = stripe.StringSlice([]string{"card"})
	}

	s, err := session.New(params)
	if err != nil {
		return "", err
	}

	return s.ID, nil
}

// Other methods remain unchanged

func (h *StripeHandler) HandleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	const MaxBodyBytes = int64(65536)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading request body: %v\n", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	event := stripe.Event{}

	if err := json.Unmarshal(payload, &event); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse webhook body json: %v\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	switch event.Type {
	case "customer.subscription.created":
		var subscription stripe.Subscription
		err := json.Unmarshal(event.Data.Raw, &subscription)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing webhook JSON: %v\n", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		h.handleSubscriptionCreated(r.Context(), &subscription)
	case "customer.subscription.updated", "customer.subscription.deleted":
		var subscription stripe.Subscription
		err := json.Unmarshal(event.Data.Raw, &subscription)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing webhook JSON: %v\n", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		h.handleSubscriptionUpdated(r.Context(), subscription)
	default:
		fmt.Fprintf(os.Stderr, "Unhandled event type: %s\n", event.Type)
	}

	w.WriteHeader(http.StatusOK)
}

type BillingInfo struct {
	Invoices        []stripe.Invoice     `json:"invoices"`
	Subscription    *stripe.Subscription `json:"subscription,omitempty"`
	NextPaymentDate int64                `json:"next_payment_date,omitempty"`
}

func (h *StripeHandler) HandleUserBillingInfo(w http.ResponseWriter, r *http.Request) {
	tokenString := extractTokenFromHeader(r)
	user, _, err := h.authService.VerifyToken(tokenString)
	if err != nil {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	fullUser, err := h.authService.GetUserByID(r.Context(), user.ID)
	if err != nil {
		http.Error(w, "Failed to retrieve user", http.StatusBadRequest)
		return
	}

	if fullUser.StripeID == "" {
		http.Error(w, "User does not have a valid Stripe ID", http.StatusBadRequest)
		return
	}

	params := &stripe.InvoiceListParams{
		Customer: &fullUser.StripeID,
	}
	invoices := make([]stripe.Invoice, 0)
	i := invoice.List(params)
	for i.Next() {
		invoices = append(invoices, *i.Invoice())
	}
	if err := i.Err(); err != nil {
		http.Error(w, "Failed to fetch invoices", http.StatusInternalServerError)
		return
	}

	subParams := &stripe.SubscriptionListParams{
		Customer: fullUser.StripeID,
	}
	subs := sub.List(subParams)
	var subscription *stripe.Subscription
	if subs.Next() {
		subscription = subs.Subscription()
	}
	if err := subs.Err(); err != nil {
		http.Error(w, "Failed to fetch subscription", http.StatusInternalServerError)
		return
	}

	var nextPaymentDate int64
	if subscription != nil {
		nextPaymentDate = subscription.CurrentPeriodEnd
	}

	billingInfo := BillingInfo{
		Invoices:        invoices,
		Subscription:    subscription,
		NextPaymentDate: nextPaymentDate,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(billingInfo); err != nil {
		http.Error(w, "Failed to encode billing info", http.StatusInternalServerError)
		return
	}
}

func (h *StripeHandler) handleSubscriptionCreated(ctx context.Context, subscription *stripe.Subscription) error {
	if subscription == nil {
		return fmt.Errorf("subscription is nil")
	}

	if subscription.Customer == nil {
		return fmt.Errorf("customer is nil in the subscription")
	}

	user, err := h.authService.GetUserByStripeCustomerID(ctx, subscription.Customer.ID)
	if err != nil {
		return fmt.Errorf("error retrieving user for customer %s: %w", subscription.Customer.ID, err)
	}

	if len(subscription.Items.Data) == 0 {
		return fmt.Errorf("no subscription items found for customer %s", subscription.Customer.ID)
	}

	priceID := subscription.Items.Data[0].Price.ID
	if priceID == "" {
		return fmt.Errorf("price ID is empty for customer %s", subscription.Customer.ID)
	}

	planType, err := h.getPlanTypeFromPriceID(priceID)
	if err != nil {
		return fmt.Errorf("error determining plan type for price ID %s: %w", priceID, err)
	}

	subscriptionModel := &models.Subscription{
		UserID:           user.ID,
		StripeCustomerID: subscription.Customer.ID,
		StripePlanID:     subscription.ID,
		Status:           string(subscription.Status),
		PlanType:         planType,
		EndDate:          time.Unix(subscription.CurrentPeriodEnd, 0),
	}

	err = h.subRepo.Create(ctx, subscriptionModel)
	if err != nil {
		return fmt.Errorf("error creating/updating subscription for user %d: %w", user.ID, err)
	}

	_, err = h.apiKeyService.AssignAPIKeyToUser(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("error creating api key for user %d: %w", user.ID, err)
	}

	err = h.userRepo.GrantAccess(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("error granting service access to user %d: %w", user.ID, err)
	}

	log.Printf("Subscription created for customer: %s with plan type: %s", subscription.Customer.ID, planType)
	return nil
}

func (h *StripeHandler) handleCheckoutSessionCompleted(ctx context.Context, session stripe.CheckoutSession) {
	if session.Customer == nil {
		log.Printf("Error: Customer is nil in the checkout session")
		return
	}

	user, err := h.authService.GetUserByStripeCustomerID(ctx, session.Customer.ID)
	if err != nil {
		log.Printf("Error retrieving user for customer %s: %v", session.Customer.ID, err)
		return
	}

	if session.Subscription == nil {
		log.Printf("Error: Subscription is nil in the checkout session for customer %s", session.Customer.ID)
		return
	}

	if session.Subscription.Items == nil || len(session.Subscription.Items.Data) == 0 {
		log.Printf("No subscription items found for customer %s", session.Customer.ID)
		return
	}

	priceID := session.Subscription.Items.Data[0].Price.ID
	if priceID == "" {
		log.Printf("Error: Price ID is empty for customer %s", session.Customer.ID)
		return
	}

	planType, err := h.getPlanTypeFromPriceID(priceID)
	if err != nil {
		log.Printf("Error determining plan type for price ID %s: %v", priceID, err)
		return
	}

	subscription := &models.Subscription{
		UserID:           user.ID,
		StripeCustomerID: session.Customer.ID,
		StripePlanID:     session.Subscription.ID,
		Status:           string(session.Subscription.Status),
		PlanType:         planType,
		EndDate:          time.Unix(session.Subscription.CurrentPeriodEnd, 0),
	}

	err = h.subRepo.Create(ctx, subscription)
	if err != nil {
		log.Printf("Error creating/updating subscription for user %d: %v", user.ID, err)
		return
	}

	_, err = h.apiKeyService.AssignAPIKeyToUser(ctx, user.ID)
	if err != nil {
		log.Printf("Error creating api key for user %d: %v", user.ID, err)
		return
	}

	err = h.userRepo.GrantAccess(ctx, user.ID)
	if err != nil {
		log.Printf("Error granting service access to user %d: %v", user.ID, err)
		return
	}

	log.Printf("Subscription created for customer: %s with plan type: %s", session.Customer.ID, planType)
}
func (h *StripeHandler) getPlanTypeFromPriceID(priceID string) (models.SubscriptionPlan, error) {
	switch priceID {
	case os.Getenv("STRIPE_MONTHLY_FREE_PRICE_ID"):
		return models.FreePlan, nil
	case os.Getenv("STRIPE_MONTHLY_PRICE_ID"):
		return models.ProPlan, nil
	case os.Getenv("STRIPE_ENTERPRISE_PLAN_PRICE_ID"):
		return models.EnterprisePlan, nil
	default:
		return "", fmt.Errorf("unknown price ID: %s", priceID)
	}
}

func (h *StripeHandler) handleSubscriptionUpdated(ctx context.Context, subscription stripe.Subscription) {
	// 1. Retrieve the user based on subscription.Customer
	user, err := h.authService.GetUserByStripeCustomerID(ctx, subscription.Customer.ID)
	if err != nil {
		log.Printf("Error retrieving user for customer %s: %v", subscription.Customer.ID, err)
		return
	}

	updatedSubscription := &models.Subscription{
		UserID:           user.ID,
		StripeCustomerID: subscription.Customer.ID,
		StripePlanID:     subscription.ID,
		Status:           string(subscription.Status),
		PlanType:         "PRO",
		EndDate:          time.Unix(subscription.CurrentPeriodEnd, 0),
	}

	err = h.subRepo.Update(ctx, updatedSubscription)
	if err != nil {
		log.Printf("Error updating subscription for user %s: %v", user.ID, err)
		return
	}

	if subscription.Status == stripe.SubscriptionStatusActive {
		err = h.userRepo.GrantAccess(ctx, user.ID)
		if err != nil {
			log.Printf("Error granting service access to user %s: %v", user.ID, err)
			return
		}
	} else if subscription.Status == stripe.SubscriptionStatusCanceled ||
		subscription.Status == stripe.SubscriptionStatusUnpaid {
		err = h.userRepo.RevokeAccess(ctx, user.ID)
		if err != nil {
			log.Printf("Error revoking service access from user %s: %v", user.ID, err)
			return
		}
	}

	fmt.Printf("Subscription updated for customer: %s, status: %s\n", subscription.Customer.ID, subscription.Status)
}

func extractTokenFromHeader(r *http.Request) string {
	bearerToken := r.Header.Get("Authorization")
	if len(strings.Split(bearerToken, " ")) == 2 {
		return strings.Split(bearerToken, " ")[1]
	}
	return ""
}
