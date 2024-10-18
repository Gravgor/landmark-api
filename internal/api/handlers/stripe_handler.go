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
	"github.com/stripe/stripe-go/v72/webhook"
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
		PaymentMethodTypes: stripe.StringSlice([]string{
			"card",
		}),
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

	s, err := session.New(params)
	if err != nil {
		return "", err
	}

	return s.ID, nil
}

func (h *StripeHandler) HandleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	const MaxBodyBytes = int64(65536)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading request body: %v\n", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	// Pass the request body and Stripe-Signature header to ConstructEvent, along with the webhook signing key
	event, err := webhook.ConstructEvent(payload, r.Header.Get("Stripe-Signature"), os.Getenv("STRIPE_WEBHOOK_SECRET"))

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error verifying webhook signature: %v\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Handle the event
	switch event.Type {
	case "checkout.session.completed":
		var session stripe.CheckoutSession
		err := json.Unmarshal(event.Data.Raw, &session)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing webhook JSON: %v\n", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		h.handleCheckoutSessionCompleted(r.Context(), session)
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

func (h *StripeHandler) handleCheckoutSessionCompleted(ctx context.Context, session stripe.CheckoutSession) {
	user, err := h.authService.GetUserByStripeCustomerID(ctx, session.Customer.ID)
	if err != nil {
		log.Printf("Error retrieving user for customer %s: %v", session.Customer.ID, err)
		return
	}
	subscription := &models.Subscription{
		UserID:           user.ID,
		StripeCustomerID: session.Customer.ID,
		StripePlanID:     session.Subscription.ID,
		Status:           "active",
		PlanType:         "PRO",
		EndDate:          time.Now().Add(30 * 24 * time.Hour),
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
	fmt.Printf("Subscription created for customer: %s\n", session.Customer.ID)
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
