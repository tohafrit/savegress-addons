// Package billing provides Stripe payment integration
package billing

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/chainlens/chainlens/backend/internal/config"
	"github.com/chainlens/chainlens/backend/internal/database"
)

// Price IDs for subscription plans
const (
	PriceIDProMonthly        = "price_pro_monthly"
	PriceIDProYearly         = "price_pro_yearly"
	PriceIDEnterpriseMonthly = "price_enterprise_monthly"
	PriceIDEnterpriseYearly  = "price_enterprise_yearly"
)

// Stripe API base URL
const stripeAPIBase = "https://api.stripe.com/v1"

// StripeClient handles Stripe API interactions
type StripeClient struct {
	secretKey      string
	webhookSecret  string
	httpClient     *http.Client
}

// NewStripeClient creates a new Stripe client
func NewStripeClient(cfg *config.Config) *StripeClient {
	return &StripeClient{
		secretKey:     cfg.StripeSecretKey,
		webhookSecret: cfg.StripeWebhookSecret,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CheckoutSession represents a Stripe checkout session
type CheckoutSession struct {
	ID         string `json:"id"`
	URL        string `json:"url"`
	CustomerID string `json:"customer"`
}

// Customer represents a Stripe customer
type Customer struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

// Subscription represents a Stripe subscription
type Subscription struct {
	ID                 string `json:"id"`
	CustomerID         string `json:"customer"`
	Status             string `json:"status"`
	CurrentPeriodEnd   int64  `json:"current_period_end"`
	CancelAtPeriodEnd  bool   `json:"cancel_at_period_end"`
	Plan               string `json:"plan_id"`
}

// CreateCustomer creates a new Stripe customer
func (s *StripeClient) CreateCustomer(ctx context.Context, email, name string) (*Customer, error) {
	data := url.Values{}
	data.Set("email", email)
	data.Set("name", name)

	resp, err := s.post(ctx, "/customers", data)
	if err != nil {
		return nil, err
	}

	var customer Customer
	if err := json.Unmarshal(resp, &customer); err != nil {
		return nil, err
	}

	return &customer, nil
}

// CreateCheckoutSession creates a checkout session for a subscription
func (s *StripeClient) CreateCheckoutSession(ctx context.Context, customerID, priceID, successURL, cancelURL string) (*CheckoutSession, error) {
	data := url.Values{}
	data.Set("mode", "subscription")
	data.Set("success_url", successURL)
	data.Set("cancel_url", cancelURL)
	data.Set("line_items[0][price]", priceID)
	data.Set("line_items[0][quantity]", "1")

	if customerID != "" {
		data.Set("customer", customerID)
	}

	resp, err := s.post(ctx, "/checkout/sessions", data)
	if err != nil {
		return nil, err
	}

	var session CheckoutSession
	if err := json.Unmarshal(resp, &session); err != nil {
		return nil, err
	}

	return &session, nil
}

// CreatePortalSession creates a customer billing portal session
func (s *StripeClient) CreatePortalSession(ctx context.Context, customerID, returnURL string) (string, error) {
	data := url.Values{}
	data.Set("customer", customerID)
	data.Set("return_url", returnURL)

	resp, err := s.post(ctx, "/billing_portal/sessions", data)
	if err != nil {
		return "", err
	}

	var result struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return "", err
	}

	return result.URL, nil
}

// GetSubscription retrieves a subscription by ID
func (s *StripeClient) GetSubscription(ctx context.Context, subscriptionID string) (*Subscription, error) {
	resp, err := s.get(ctx, "/subscriptions/"+subscriptionID)
	if err != nil {
		return nil, err
	}

	var sub Subscription
	if err := json.Unmarshal(resp, &sub); err != nil {
		return nil, err
	}

	return &sub, nil
}

// CancelSubscription cancels a subscription at period end
func (s *StripeClient) CancelSubscription(ctx context.Context, subscriptionID string) error {
	data := url.Values{}
	data.Set("cancel_at_period_end", "true")

	_, err := s.post(ctx, "/subscriptions/"+subscriptionID, data)
	return err
}

// WebhookEvent represents a Stripe webhook event
type WebhookEvent struct {
	ID      string          `json:"id"`
	Type    string          `json:"type"`
	Data    json.RawMessage `json:"data"`
	Created int64           `json:"created"`
}

// WebhookEventData contains the event object
type WebhookEventData struct {
	Object json.RawMessage `json:"object"`
}

// VerifyWebhook verifies webhook signature and parses the event
func (s *StripeClient) VerifyWebhook(payload []byte, signature string) (*WebhookEvent, error) {
	// Parse signature header
	sigParts := parseSignatureHeader(signature)
	timestamp := sigParts["t"]
	v1Sig := sigParts["v1"]

	if timestamp == "" || v1Sig == "" {
		return nil, fmt.Errorf("invalid signature header")
	}

	// Compute expected signature
	signedPayload := timestamp + "." + string(payload)
	mac := hmac.New(sha256.New, []byte(s.webhookSecret))
	mac.Write([]byte(signedPayload))
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(v1Sig), []byte(expectedSig)) {
		return nil, fmt.Errorf("signature verification failed")
	}

	var event WebhookEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, err
	}

	return &event, nil
}

func parseSignatureHeader(header string) map[string]string {
	result := make(map[string]string)
	parts := strings.Split(header, ",")
	for _, part := range parts {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) == 2 {
			result[kv[0]] = kv[1]
		}
	}
	return result
}

// ProcessWebhookEvent processes a webhook event and updates the database
func (s *StripeClient) ProcessWebhookEvent(ctx context.Context, db *database.DB, event *WebhookEvent) error {
	var eventData WebhookEventData
	if err := json.Unmarshal(event.Data, &eventData); err != nil {
		return err
	}

	switch event.Type {
	case "checkout.session.completed":
		return s.handleCheckoutCompleted(ctx, db, eventData.Object)
	case "customer.subscription.created":
		return s.handleSubscriptionCreated(ctx, db, eventData.Object)
	case "customer.subscription.updated":
		return s.handleSubscriptionUpdated(ctx, db, eventData.Object)
	case "customer.subscription.deleted":
		return s.handleSubscriptionDeleted(ctx, db, eventData.Object)
	case "invoice.payment_succeeded":
		return s.handlePaymentSucceeded(ctx, db, eventData.Object)
	case "invoice.payment_failed":
		return s.handlePaymentFailed(ctx, db, eventData.Object)
	default:
		// Ignore unhandled events
		return nil
	}
}

func (s *StripeClient) handleCheckoutCompleted(ctx context.Context, db *database.DB, data json.RawMessage) error {
	var session struct {
		CustomerID     string `json:"customer"`
		SubscriptionID string `json:"subscription"`
		ClientRefID    string `json:"client_reference_id"`
	}
	if err := json.Unmarshal(data, &session); err != nil {
		return err
	}

	if session.ClientRefID == "" {
		return nil // No user ID to update
	}

	// Update user with Stripe IDs
	return db.UpdateUserStripe(ctx, session.ClientRefID, session.CustomerID, session.SubscriptionID)
}

func (s *StripeClient) handleSubscriptionCreated(ctx context.Context, db *database.DB, data json.RawMessage) error {
	var sub struct {
		ID         string `json:"id"`
		CustomerID string `json:"customer"`
		Status     string `json:"status"`
		Items      struct {
			Data []struct {
				Price struct {
					ID string `json:"id"`
				} `json:"price"`
			} `json:"data"`
		} `json:"items"`
	}
	if err := json.Unmarshal(data, &sub); err != nil {
		return err
	}

	if sub.Status != "active" && sub.Status != "trialing" {
		return nil
	}

	// Get user by Stripe customer ID and update plan
	user, err := db.GetUserByStripeCustomerID(ctx, sub.CustomerID)
	if err != nil {
		return nil // User not found, may be a new customer flow
	}

	plan := mapPriceIDToPlan(sub.Items.Data[0].Price.ID)
	return db.UpdateUserPlan(ctx, user.ID, plan)
}

func (s *StripeClient) handleSubscriptionUpdated(ctx context.Context, db *database.DB, data json.RawMessage) error {
	return s.handleSubscriptionCreated(ctx, db, data) // Same logic
}

func (s *StripeClient) handleSubscriptionDeleted(ctx context.Context, db *database.DB, data json.RawMessage) error {
	var sub struct {
		CustomerID string `json:"customer"`
	}
	if err := json.Unmarshal(data, &sub); err != nil {
		return err
	}

	user, err := db.GetUserByStripeCustomerID(ctx, sub.CustomerID)
	if err != nil {
		return nil
	}

	return db.UpdateUserPlan(ctx, user.ID, "free")
}

func (s *StripeClient) handlePaymentSucceeded(ctx context.Context, db *database.DB, data json.RawMessage) error {
	// Could send confirmation email or update payment status
	return nil
}

func (s *StripeClient) handlePaymentFailed(ctx context.Context, db *database.DB, data json.RawMessage) error {
	// Could send payment failed email or downgrade user
	return nil
}

func mapPriceIDToPlan(priceID string) string {
	switch priceID {
	case PriceIDProMonthly, PriceIDProYearly:
		return "pro"
	case PriceIDEnterpriseMonthly, PriceIDEnterpriseYearly:
		return "enterprise"
	default:
		return "free"
	}
}

// HTTP helpers

func (s *StripeClient) post(ctx context.Context, path string, data url.Values) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", stripeAPIBase+path, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	return s.doRequest(req)
}

func (s *StripeClient) get(ctx context.Context, path string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", stripeAPIBase+path, nil)
	if err != nil {
		return nil, err
	}
	return s.doRequest(req)
}

func (s *StripeClient) doRequest(req *http.Request) ([]byte, error) {
	req.SetBasicAuth(s.secretKey, "")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		var apiErr struct {
			Error struct {
				Message string `json:"message"`
				Type    string `json:"type"`
			} `json:"error"`
		}
		if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Error.Message != "" {
			return nil, fmt.Errorf("Stripe API error: %s", apiErr.Error.Message)
		}
		return nil, fmt.Errorf("Stripe API error: status %d", resp.StatusCode)
	}

	return body, nil
}

// ReadRequestBody reads and returns the request body for webhook verification
func ReadRequestBody(r *http.Request) ([]byte, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	// Restore body for potential re-reads
	r.Body = io.NopCloser(bytes.NewBuffer(body))
	return body, nil
}
