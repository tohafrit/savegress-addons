package billing

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"getchainlens.com/chainlens/backend/internal/config"
	"getchainlens.com/chainlens/backend/internal/database"
)

// mockDB implements BillingDB for testing
type mockDB struct {
	users             map[string]*database.User // keyed by stripe customer ID
	updateStripeCalls []updateStripeCall
	updatePlanCalls   []updatePlanCall
	shouldError       bool
}

type updateStripeCall struct {
	id, customerID, subscriptionID string
}

type updatePlanCall struct {
	id, plan string
}

func newMockDB() *mockDB {
	return &mockDB{
		users: make(map[string]*database.User),
	}
}

func (m *mockDB) UpdateUserStripe(ctx context.Context, id, customerID, subscriptionID string) error {
	if m.shouldError {
		return errors.New("mock db error")
	}
	m.updateStripeCalls = append(m.updateStripeCalls, updateStripeCall{id, customerID, subscriptionID})
	return nil
}

func (m *mockDB) GetUserByStripeCustomerID(ctx context.Context, customerID string) (*database.User, error) {
	if m.shouldError {
		return nil, errors.New("mock db error")
	}
	user, ok := m.users[customerID]
	if !ok {
		return nil, errors.New("user not found")
	}
	return user, nil
}

func (m *mockDB) UpdateUserPlan(ctx context.Context, id, plan string) error {
	if m.shouldError {
		return errors.New("mock db error")
	}
	m.updatePlanCalls = append(m.updatePlanCalls, updatePlanCall{id, plan})
	return nil
}

// Helper to create a test server
func newTestServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *StripeClient) {
	server := httptest.NewServer(handler)
	client := &StripeClient{
		secretKey:     "sk_test_123",
		webhookSecret: "whsec_test",
		httpClient:    server.Client(),
		baseURL:       server.URL,
	}
	return server, client
}

func TestNewStripeClient(t *testing.T) {
	cfg := &config.Config{
		StripeSecretKey:     "sk_test_123",
		StripeWebhookSecret: "whsec_123",
	}

	client := NewStripeClient(cfg)
	if client == nil {
		t.Fatal("NewStripeClient returned nil")
	}

	if client.secretKey != cfg.StripeSecretKey {
		t.Errorf("secretKey = %s, want %s", client.secretKey, cfg.StripeSecretKey)
	}
}

func TestParseSignatureHeader(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected map[string]string
	}{
		{
			name:   "valid header",
			header: "t=1234567890,v1=abc123,v0=def456",
			expected: map[string]string{
				"t":  "1234567890",
				"v1": "abc123",
				"v0": "def456",
			},
		},
		{
			name:     "empty header",
			header:   "",
			expected: map[string]string{},
		},
		{
			name:   "single value",
			header: "t=1234567890",
			expected: map[string]string{
				"t": "1234567890",
			},
		},
		{
			name:     "malformed",
			header:   "invalid",
			expected: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseSignatureHeader(tt.header)
			for k, v := range tt.expected {
				if result[k] != v {
					t.Errorf("result[%s] = %s, want %s", k, result[k], v)
				}
			}
		})
	}
}

func TestMapPriceIDToPlan(t *testing.T) {
	tests := []struct {
		priceID  string
		expected string
	}{
		{PriceIDProMonthly, "pro"},
		{PriceIDProYearly, "pro"},
		{PriceIDEnterpriseMonthly, "enterprise"},
		{PriceIDEnterpriseYearly, "enterprise"},
		{"unknown_price_id", "free"},
		{"", "free"},
	}

	for _, tt := range tests {
		t.Run(tt.priceID, func(t *testing.T) {
			result := mapPriceIDToPlan(tt.priceID)
			if result != tt.expected {
				t.Errorf("mapPriceIDToPlan(%s) = %s, want %s", tt.priceID, result, tt.expected)
			}
		})
	}
}

func TestStripeClient_VerifyWebhook_InvalidSignature(t *testing.T) {
	client := &StripeClient{
		webhookSecret: "whsec_test_secret",
	}

	payload := []byte(`{"type":"test"}`)

	// Invalid signature header
	_, err := client.VerifyWebhook(payload, "invalid")
	if err == nil {
		t.Error("Expected error for invalid signature header")
	}

	// Missing timestamp
	_, err = client.VerifyWebhook(payload, "v1=abc")
	if err == nil {
		t.Error("Expected error for missing timestamp")
	}

	// Missing v1 signature
	_, err = client.VerifyWebhook(payload, "t=1234567890")
	if err == nil {
		t.Error("Expected error for missing v1 signature")
	}
}

func TestStripeClient_VerifyWebhook_ValidSignature(t *testing.T) {
	webhookSecret := "whsec_test_secret"
	client := &StripeClient{
		webhookSecret: webhookSecret,
	}

	payload := []byte(`{"id":"evt_123","type":"checkout.session.completed","data":{},"created":1234567890}`)
	timestamp := fmt.Sprintf("%d", time.Now().Unix())

	// Compute valid signature
	signedPayload := timestamp + "." + string(payload)
	mac := hmac.New(sha256.New, []byte(webhookSecret))
	mac.Write([]byte(signedPayload))
	signature := hex.EncodeToString(mac.Sum(nil))

	header := fmt.Sprintf("t=%s,v1=%s", timestamp, signature)

	event, err := client.VerifyWebhook(payload, header)
	if err != nil {
		t.Fatalf("VerifyWebhook failed: %v", err)
	}

	if event.ID != "evt_123" {
		t.Errorf("event.ID = %s, want evt_123", event.ID)
	}
	if event.Type != "checkout.session.completed" {
		t.Errorf("event.Type = %s, want checkout.session.completed", event.Type)
	}
}

func TestStripeClient_VerifyWebhook_WrongSignature(t *testing.T) {
	client := &StripeClient{
		webhookSecret: "whsec_test_secret",
	}

	payload := []byte(`{"type":"test"}`)
	timestamp := fmt.Sprintf("%d", time.Now().Unix())

	// Wrong signature
	header := fmt.Sprintf("t=%s,v1=wrong_signature", timestamp)

	_, err := client.VerifyWebhook(payload, header)
	if err == nil {
		t.Error("Expected error for wrong signature")
	}
}

func TestWebhookEvent_JSON(t *testing.T) {
	event := WebhookEvent{
		ID:      "evt_123",
		Type:    "checkout.session.completed",
		Data:    json.RawMessage(`{"object":{"id":"cs_123"}}`),
		Created: 1234567890,
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded WebhookEvent
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.ID != event.ID {
		t.Errorf("ID = %s, want %s", decoded.ID, event.ID)
	}
	if decoded.Type != event.Type {
		t.Errorf("Type = %s, want %s", decoded.Type, event.Type)
	}
}

func TestCheckoutSession_Fields(t *testing.T) {
	session := CheckoutSession{
		ID:         "cs_123",
		URL:        "https://checkout.stripe.com/pay/cs_123",
		CustomerID: "cus_123",
	}

	if session.ID != "cs_123" {
		t.Errorf("ID = %s, want cs_123", session.ID)
	}
	if session.CustomerID != "cus_123" {
		t.Errorf("CustomerID = %s, want cus_123", session.CustomerID)
	}
}

func TestCustomer_Fields(t *testing.T) {
	customer := Customer{
		ID:    "cus_123",
		Email: "test@example.com",
		Name:  "Test User",
	}

	if customer.ID != "cus_123" {
		t.Errorf("ID = %s, want cus_123", customer.ID)
	}
	if customer.Email != "test@example.com" {
		t.Errorf("Email = %s, want test@example.com", customer.Email)
	}
}

func TestSubscription_Fields(t *testing.T) {
	sub := Subscription{
		ID:                "sub_123",
		CustomerID:        "cus_123",
		Status:            "active",
		CurrentPeriodEnd:  1234567890,
		CancelAtPeriodEnd: false,
		Plan:              "price_pro_monthly",
	}

	if sub.ID != "sub_123" {
		t.Errorf("ID = %s, want sub_123", sub.ID)
	}
	if sub.Status != "active" {
		t.Errorf("Status = %s, want active", sub.Status)
	}
}

func TestReadRequestBody(t *testing.T) {
	body := []byte(`{"test":"data"}`)
	req := httptest.NewRequest("POST", "/", bytes.NewReader(body))

	readBody, err := ReadRequestBody(req)
	if err != nil {
		t.Fatalf("ReadRequestBody failed: %v", err)
	}

	if !bytes.Equal(readBody, body) {
		t.Errorf("body = %s, want %s", readBody, body)
	}

	// Body should still be readable after ReadRequestBody
	secondRead, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("Second read failed: %v", err)
	}

	if !bytes.Equal(secondRead, body) {
		t.Errorf("second read = %s, want %s", secondRead, body)
	}
}

func TestStripeClient_CreateCustomer(t *testing.T) {
	server, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/customers" {
			t.Errorf("path = %s, want /customers", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("method = %s, want POST", r.Method)
		}

		// Verify auth header
		username, _, ok := r.BasicAuth()
		if !ok || username != "sk_test_123" {
			t.Errorf("missing or incorrect basic auth")
		}

		// Verify form data
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.Form.Get("email") != "test@example.com" {
			t.Errorf("email = %s, want test@example.com", r.Form.Get("email"))
		}
		if r.Form.Get("name") != "Test User" {
			t.Errorf("name = %s, want Test User", r.Form.Get("name"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Customer{
			ID:    "cus_123",
			Email: "test@example.com",
			Name:  "Test User",
		})
	})
	defer server.Close()

	customer, err := client.CreateCustomer(context.Background(), "test@example.com", "Test User")
	if err != nil {
		t.Fatalf("CreateCustomer failed: %v", err)
	}

	if customer.ID != "cus_123" {
		t.Errorf("ID = %s, want cus_123", customer.ID)
	}
	if customer.Email != "test@example.com" {
		t.Errorf("Email = %s, want test@example.com", customer.Email)
	}
}

func TestStripeClient_CreateCustomer_Error(t *testing.T) {
	server, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]string{
				"message": "Invalid email address",
				"type":    "invalid_request_error",
			},
		})
	})
	defer server.Close()

	_, err := client.CreateCustomer(context.Background(), "invalid", "Test")
	if err == nil {
		t.Fatal("Expected error for invalid email")
	}
	if err.Error() != "Stripe API error: Invalid email address" {
		t.Errorf("error = %v", err)
	}
}

func TestStripeClient_CreateCheckoutSession(t *testing.T) {
	server, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/checkout/sessions" {
			t.Errorf("path = %s, want /checkout/sessions", r.URL.Path)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.Form.Get("mode") != "subscription" {
			t.Errorf("mode = %s, want subscription", r.Form.Get("mode"))
		}
		if r.Form.Get("customer") != "cus_123" {
			t.Errorf("customer = %s, want cus_123", r.Form.Get("customer"))
		}
		if r.Form.Get("line_items[0][price]") != PriceIDProMonthly {
			t.Errorf("price = %s, want %s", r.Form.Get("line_items[0][price]"), PriceIDProMonthly)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(CheckoutSession{
			ID:         "cs_test_123",
			URL:        "https://checkout.stripe.com/pay/cs_test_123",
			CustomerID: "cus_123",
		})
	})
	defer server.Close()

	session, err := client.CreateCheckoutSession(
		context.Background(),
		"cus_123",
		PriceIDProMonthly,
		"https://example.com/success",
		"https://example.com/cancel",
	)
	if err != nil {
		t.Fatalf("CreateCheckoutSession failed: %v", err)
	}

	if session.ID != "cs_test_123" {
		t.Errorf("ID = %s, want cs_test_123", session.ID)
	}
	if session.URL != "https://checkout.stripe.com/pay/cs_test_123" {
		t.Errorf("URL = %s", session.URL)
	}
}

func TestStripeClient_CreateCheckoutSession_NoCustomer(t *testing.T) {
	server, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		// customer should not be set
		if r.Form.Get("customer") != "" {
			t.Errorf("customer should be empty, got %s", r.Form.Get("customer"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(CheckoutSession{
			ID:  "cs_test_123",
			URL: "https://checkout.stripe.com/pay/cs_test_123",
		})
	})
	defer server.Close()

	session, err := client.CreateCheckoutSession(
		context.Background(),
		"", // no customer
		PriceIDProMonthly,
		"https://example.com/success",
		"https://example.com/cancel",
	)
	if err != nil {
		t.Fatalf("CreateCheckoutSession failed: %v", err)
	}
	if session.ID != "cs_test_123" {
		t.Errorf("ID = %s, want cs_test_123", session.ID)
	}
}

func TestStripeClient_CreatePortalSession(t *testing.T) {
	server, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/billing_portal/sessions" {
			t.Errorf("path = %s, want /billing_portal/sessions", r.URL.Path)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.Form.Get("customer") != "cus_123" {
			t.Errorf("customer = %s, want cus_123", r.Form.Get("customer"))
		}
		if r.Form.Get("return_url") != "https://example.com/billing" {
			t.Errorf("return_url = %s", r.Form.Get("return_url"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"url": "https://billing.stripe.com/session/test_123",
		})
	})
	defer server.Close()

	url, err := client.CreatePortalSession(context.Background(), "cus_123", "https://example.com/billing")
	if err != nil {
		t.Fatalf("CreatePortalSession failed: %v", err)
	}

	if url != "https://billing.stripe.com/session/test_123" {
		t.Errorf("url = %s", url)
	}
}

func TestStripeClient_GetSubscription(t *testing.T) {
	server, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/subscriptions/sub_123" {
			t.Errorf("path = %s, want /subscriptions/sub_123", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("method = %s, want GET", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Subscription{
			ID:                "sub_123",
			CustomerID:        "cus_123",
			Status:            "active",
			CurrentPeriodEnd:  1234567890,
			CancelAtPeriodEnd: false,
			Plan:              PriceIDProMonthly,
		})
	})
	defer server.Close()

	sub, err := client.GetSubscription(context.Background(), "sub_123")
	if err != nil {
		t.Fatalf("GetSubscription failed: %v", err)
	}

	if sub.ID != "sub_123" {
		t.Errorf("ID = %s, want sub_123", sub.ID)
	}
	if sub.Status != "active" {
		t.Errorf("Status = %s, want active", sub.Status)
	}
	if sub.CustomerID != "cus_123" {
		t.Errorf("CustomerID = %s, want cus_123", sub.CustomerID)
	}
}

func TestStripeClient_CancelSubscription(t *testing.T) {
	server, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/subscriptions/sub_123" {
			t.Errorf("path = %s, want /subscriptions/sub_123", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("method = %s, want POST", r.Method)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.Form.Get("cancel_at_period_end") != "true" {
			t.Errorf("cancel_at_period_end = %s, want true", r.Form.Get("cancel_at_period_end"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Subscription{
			ID:                "sub_123",
			CancelAtPeriodEnd: true,
		})
	})
	defer server.Close()

	err := client.CancelSubscription(context.Background(), "sub_123")
	if err != nil {
		t.Fatalf("CancelSubscription failed: %v", err)
	}
}

func TestWebhookEventData_CheckoutCompleted(t *testing.T) {
	eventData := WebhookEventData{
		Object: json.RawMessage(`{"customer":"cus_123","subscription":"sub_123","client_reference_id":"user_123"}`),
	}
	eventDataJSON, _ := json.Marshal(eventData)

	event := &WebhookEvent{
		ID:   "evt_123",
		Type: "checkout.session.completed",
		Data: eventDataJSON,
	}

	// Verify event structure is correct
	if event.Type != "checkout.session.completed" {
		t.Errorf("Type = %s, want checkout.session.completed", event.Type)
	}
	if event.ID != "evt_123" {
		t.Errorf("ID = %s, want evt_123", event.ID)
	}

	// Parse and verify data
	var data WebhookEventData
	if err := json.Unmarshal(event.Data, &data); err != nil {
		t.Fatalf("Failed to unmarshal event data: %v", err)
	}

	var session struct {
		Customer          string `json:"customer"`
		Subscription      string `json:"subscription"`
		ClientReferenceID string `json:"client_reference_id"`
	}
	if err := json.Unmarshal(data.Object, &session); err != nil {
		t.Fatalf("Failed to unmarshal session: %v", err)
	}
	if session.Customer != "cus_123" {
		t.Errorf("Customer = %s, want cus_123", session.Customer)
	}
	if session.Subscription != "sub_123" {
		t.Errorf("Subscription = %s, want sub_123", session.Subscription)
	}
}

func TestWebhookEventData_SubscriptionCreated(t *testing.T) {
	eventData := WebhookEventData{
		Object: json.RawMessage(`{"id":"sub_123","customer":"cus_123","status":"active","items":{"data":[{"price":{"id":"price_pro_monthly"}}]}}`),
	}
	eventDataJSON, _ := json.Marshal(eventData)

	event := &WebhookEvent{
		ID:   "evt_123",
		Type: "customer.subscription.created",
		Data: eventDataJSON,
	}

	// Verify event structure
	if event.Type != "customer.subscription.created" {
		t.Errorf("Type = %s, want customer.subscription.created", event.Type)
	}

	// Parse and verify data
	var data WebhookEventData
	if err := json.Unmarshal(event.Data, &data); err != nil {
		t.Fatalf("Failed to unmarshal event data: %v", err)
	}

	var sub struct {
		ID       string `json:"id"`
		Customer string `json:"customer"`
		Status   string `json:"status"`
	}
	if err := json.Unmarshal(data.Object, &sub); err != nil {
		t.Fatalf("Failed to unmarshal subscription: %v", err)
	}
	if sub.ID != "sub_123" {
		t.Errorf("ID = %s, want sub_123", sub.ID)
	}
	if sub.Status != "active" {
		t.Errorf("Status = %s, want active", sub.Status)
	}
}

func TestWebhookEventData_SubscriptionDeleted(t *testing.T) {
	eventData := WebhookEventData{
		Object: json.RawMessage(`{"customer":"cus_123"}`),
	}
	eventDataJSON, _ := json.Marshal(eventData)

	event := &WebhookEvent{
		ID:   "evt_123",
		Type: "customer.subscription.deleted",
		Data: eventDataJSON,
	}

	if event.Type != "customer.subscription.deleted" {
		t.Errorf("Type = %s, want customer.subscription.deleted", event.Type)
	}
}

func TestStripeClient_ProcessWebhookEvent_PaymentSucceeded(t *testing.T) {
	client := &StripeClient{}

	eventData := WebhookEventData{
		Object: json.RawMessage(`{}`),
	}
	eventDataJSON, _ := json.Marshal(eventData)

	event := &WebhookEvent{
		ID:   "evt_123",
		Type: "invoice.payment_succeeded",
		Data: eventDataJSON,
	}

	// Should succeed (does nothing)
	err := client.ProcessWebhookEvent(context.Background(), nil, event)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestStripeClient_ProcessWebhookEvent_PaymentFailed(t *testing.T) {
	client := &StripeClient{}

	eventData := WebhookEventData{
		Object: json.RawMessage(`{}`),
	}
	eventDataJSON, _ := json.Marshal(eventData)

	event := &WebhookEvent{
		ID:   "evt_123",
		Type: "invoice.payment_failed",
		Data: eventDataJSON,
	}

	// Should succeed (does nothing)
	err := client.ProcessWebhookEvent(context.Background(), nil, event)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestStripeClient_ProcessWebhookEvent_UnhandledEvent(t *testing.T) {
	client := &StripeClient{}

	eventData := WebhookEventData{
		Object: json.RawMessage(`{}`),
	}
	eventDataJSON, _ := json.Marshal(eventData)

	event := &WebhookEvent{
		ID:   "evt_123",
		Type: "some.unhandled.event",
		Data: eventDataJSON,
	}

	// Should succeed (ignores unhandled events)
	err := client.ProcessWebhookEvent(context.Background(), nil, event)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestPriceIDConstants(t *testing.T) {
	if PriceIDProMonthly != "price_pro_monthly" {
		t.Errorf("PriceIDProMonthly = %s", PriceIDProMonthly)
	}
	if PriceIDProYearly != "price_pro_yearly" {
		t.Errorf("PriceIDProYearly = %s", PriceIDProYearly)
	}
	if PriceIDEnterpriseMonthly != "price_enterprise_monthly" {
		t.Errorf("PriceIDEnterpriseMonthly = %s", PriceIDEnterpriseMonthly)
	}
	if PriceIDEnterpriseYearly != "price_enterprise_yearly" {
		t.Errorf("PriceIDEnterpriseYearly = %s", PriceIDEnterpriseYearly)
	}
}

// Webhook handler tests with mock DB

func TestProcessWebhookEvent_CheckoutCompleted_WithDB(t *testing.T) {
	client := &StripeClient{}
	db := newMockDB()

	eventData := WebhookEventData{
		Object: json.RawMessage(`{"customer":"cus_123","subscription":"sub_123","client_reference_id":"user_456"}`),
	}
	eventDataJSON, _ := json.Marshal(eventData)

	event := &WebhookEvent{
		ID:   "evt_123",
		Type: "checkout.session.completed",
		Data: eventDataJSON,
	}

	err := client.ProcessWebhookEvent(context.Background(), db, event)
	if err != nil {
		t.Fatalf("ProcessWebhookEvent failed: %v", err)
	}

	if len(db.updateStripeCalls) != 1 {
		t.Fatalf("expected 1 UpdateUserStripe call, got %d", len(db.updateStripeCalls))
	}

	call := db.updateStripeCalls[0]
	if call.id != "user_456" {
		t.Errorf("id = %s, want user_456", call.id)
	}
	if call.customerID != "cus_123" {
		t.Errorf("customerID = %s, want cus_123", call.customerID)
	}
	if call.subscriptionID != "sub_123" {
		t.Errorf("subscriptionID = %s, want sub_123", call.subscriptionID)
	}
}

func TestProcessWebhookEvent_CheckoutCompleted_NoClientRef(t *testing.T) {
	client := &StripeClient{}
	db := newMockDB()

	eventData := WebhookEventData{
		Object: json.RawMessage(`{"customer":"cus_123","subscription":"sub_123"}`),
	}
	eventDataJSON, _ := json.Marshal(eventData)

	event := &WebhookEvent{
		ID:   "evt_123",
		Type: "checkout.session.completed",
		Data: eventDataJSON,
	}

	err := client.ProcessWebhookEvent(context.Background(), db, event)
	if err != nil {
		t.Fatalf("ProcessWebhookEvent failed: %v", err)
	}

	// Should not call UpdateUserStripe when no client_reference_id
	if len(db.updateStripeCalls) != 0 {
		t.Errorf("expected 0 UpdateUserStripe calls, got %d", len(db.updateStripeCalls))
	}
}

func TestProcessWebhookEvent_SubscriptionCreated_WithDB(t *testing.T) {
	client := &StripeClient{}
	db := newMockDB()
	db.users["cus_123"] = &database.User{ID: "user_456"}

	eventData := WebhookEventData{
		Object: json.RawMessage(`{"id":"sub_123","customer":"cus_123","status":"active","items":{"data":[{"price":{"id":"price_pro_monthly"}}]}}`),
	}
	eventDataJSON, _ := json.Marshal(eventData)

	event := &WebhookEvent{
		ID:   "evt_123",
		Type: "customer.subscription.created",
		Data: eventDataJSON,
	}

	err := client.ProcessWebhookEvent(context.Background(), db, event)
	if err != nil {
		t.Fatalf("ProcessWebhookEvent failed: %v", err)
	}

	if len(db.updatePlanCalls) != 1 {
		t.Fatalf("expected 1 UpdateUserPlan call, got %d", len(db.updatePlanCalls))
	}

	call := db.updatePlanCalls[0]
	if call.id != "user_456" {
		t.Errorf("id = %s, want user_456", call.id)
	}
	if call.plan != "pro" {
		t.Errorf("plan = %s, want pro", call.plan)
	}
}

func TestProcessWebhookEvent_SubscriptionCreated_Trialing(t *testing.T) {
	client := &StripeClient{}
	db := newMockDB()
	db.users["cus_123"] = &database.User{ID: "user_456"}

	eventData := WebhookEventData{
		Object: json.RawMessage(`{"id":"sub_123","customer":"cus_123","status":"trialing","items":{"data":[{"price":{"id":"price_enterprise_yearly"}}]}}`),
	}
	eventDataJSON, _ := json.Marshal(eventData)

	event := &WebhookEvent{
		ID:   "evt_123",
		Type: "customer.subscription.created",
		Data: eventDataJSON,
	}

	err := client.ProcessWebhookEvent(context.Background(), db, event)
	if err != nil {
		t.Fatalf("ProcessWebhookEvent failed: %v", err)
	}

	if len(db.updatePlanCalls) != 1 {
		t.Fatalf("expected 1 UpdateUserPlan call, got %d", len(db.updatePlanCalls))
	}

	if db.updatePlanCalls[0].plan != "enterprise" {
		t.Errorf("plan = %s, want enterprise", db.updatePlanCalls[0].plan)
	}
}

func TestProcessWebhookEvent_SubscriptionCreated_InactiveStatus(t *testing.T) {
	client := &StripeClient{}
	db := newMockDB()
	db.users["cus_123"] = &database.User{ID: "user_456"}

	eventData := WebhookEventData{
		Object: json.RawMessage(`{"id":"sub_123","customer":"cus_123","status":"past_due","items":{"data":[{"price":{"id":"price_pro_monthly"}}]}}`),
	}
	eventDataJSON, _ := json.Marshal(eventData)

	event := &WebhookEvent{
		ID:   "evt_123",
		Type: "customer.subscription.created",
		Data: eventDataJSON,
	}

	err := client.ProcessWebhookEvent(context.Background(), db, event)
	if err != nil {
		t.Fatalf("ProcessWebhookEvent failed: %v", err)
	}

	// Should not update plan for inactive status
	if len(db.updatePlanCalls) != 0 {
		t.Errorf("expected 0 UpdateUserPlan calls, got %d", len(db.updatePlanCalls))
	}
}

func TestProcessWebhookEvent_SubscriptionCreated_UserNotFound(t *testing.T) {
	client := &StripeClient{}
	db := newMockDB() // empty users

	eventData := WebhookEventData{
		Object: json.RawMessage(`{"id":"sub_123","customer":"cus_unknown","status":"active","items":{"data":[{"price":{"id":"price_pro_monthly"}}]}}`),
	}
	eventDataJSON, _ := json.Marshal(eventData)

	event := &WebhookEvent{
		ID:   "evt_123",
		Type: "customer.subscription.created",
		Data: eventDataJSON,
	}

	// Should not error when user is not found
	err := client.ProcessWebhookEvent(context.Background(), db, event)
	if err != nil {
		t.Fatalf("ProcessWebhookEvent failed: %v", err)
	}
}

func TestProcessWebhookEvent_SubscriptionUpdated_WithDB(t *testing.T) {
	client := &StripeClient{}
	db := newMockDB()
	db.users["cus_123"] = &database.User{ID: "user_456"}

	eventData := WebhookEventData{
		Object: json.RawMessage(`{"id":"sub_123","customer":"cus_123","status":"active","items":{"data":[{"price":{"id":"price_pro_yearly"}}]}}`),
	}
	eventDataJSON, _ := json.Marshal(eventData)

	event := &WebhookEvent{
		ID:   "evt_123",
		Type: "customer.subscription.updated",
		Data: eventDataJSON,
	}

	err := client.ProcessWebhookEvent(context.Background(), db, event)
	if err != nil {
		t.Fatalf("ProcessWebhookEvent failed: %v", err)
	}

	if len(db.updatePlanCalls) != 1 {
		t.Fatalf("expected 1 UpdateUserPlan call, got %d", len(db.updatePlanCalls))
	}

	if db.updatePlanCalls[0].plan != "pro" {
		t.Errorf("plan = %s, want pro", db.updatePlanCalls[0].plan)
	}
}

func TestProcessWebhookEvent_SubscriptionDeleted_WithDB(t *testing.T) {
	client := &StripeClient{}
	db := newMockDB()
	db.users["cus_123"] = &database.User{ID: "user_456"}

	eventData := WebhookEventData{
		Object: json.RawMessage(`{"customer":"cus_123"}`),
	}
	eventDataJSON, _ := json.Marshal(eventData)

	event := &WebhookEvent{
		ID:   "evt_123",
		Type: "customer.subscription.deleted",
		Data: eventDataJSON,
	}

	err := client.ProcessWebhookEvent(context.Background(), db, event)
	if err != nil {
		t.Fatalf("ProcessWebhookEvent failed: %v", err)
	}

	if len(db.updatePlanCalls) != 1 {
		t.Fatalf("expected 1 UpdateUserPlan call, got %d", len(db.updatePlanCalls))
	}

	call := db.updatePlanCalls[0]
	if call.id != "user_456" {
		t.Errorf("id = %s, want user_456", call.id)
	}
	if call.plan != "free" {
		t.Errorf("plan = %s, want free", call.plan)
	}
}

func TestProcessWebhookEvent_SubscriptionDeleted_UserNotFound(t *testing.T) {
	client := &StripeClient{}
	db := newMockDB() // empty users

	eventData := WebhookEventData{
		Object: json.RawMessage(`{"customer":"cus_unknown"}`),
	}
	eventDataJSON, _ := json.Marshal(eventData)

	event := &WebhookEvent{
		ID:   "evt_123",
		Type: "customer.subscription.deleted",
		Data: eventDataJSON,
	}

	// Should not error when user is not found
	err := client.ProcessWebhookEvent(context.Background(), db, event)
	if err != nil {
		t.Fatalf("ProcessWebhookEvent failed: %v", err)
	}
}

func TestProcessWebhookEvent_InvalidJSON(t *testing.T) {
	client := &StripeClient{}
	db := newMockDB()

	event := &WebhookEvent{
		ID:   "evt_123",
		Type: "checkout.session.completed",
		Data: json.RawMessage(`invalid json`),
	}

	err := client.ProcessWebhookEvent(context.Background(), db, event)
	if err == nil {
		t.Fatal("Expected error for invalid JSON")
	}
}

func TestProcessWebhookEvent_CheckoutCompleted_InvalidObjectJSON(t *testing.T) {
	client := &StripeClient{}
	db := newMockDB()

	eventData := WebhookEventData{
		Object: json.RawMessage(`invalid`),
	}
	eventDataJSON, _ := json.Marshal(eventData)

	event := &WebhookEvent{
		ID:   "evt_123",
		Type: "checkout.session.completed",
		Data: eventDataJSON,
	}

	err := client.ProcessWebhookEvent(context.Background(), db, event)
	if err == nil {
		t.Fatal("Expected error for invalid object JSON")
	}
}

func TestProcessWebhookEvent_SubscriptionCreated_NoItems(t *testing.T) {
	client := &StripeClient{}
	db := newMockDB()
	db.users["cus_123"] = &database.User{ID: "user_456"}

	eventData := WebhookEventData{
		Object: json.RawMessage(`{"id":"sub_123","customer":"cus_123","status":"active","items":{"data":[]}}`),
	}
	eventDataJSON, _ := json.Marshal(eventData)

	event := &WebhookEvent{
		ID:   "evt_123",
		Type: "customer.subscription.created",
		Data: eventDataJSON,
	}

	// Should not error when items array is empty
	err := client.ProcessWebhookEvent(context.Background(), db, event)
	if err != nil {
		t.Fatalf("ProcessWebhookEvent failed: %v", err)
	}

	// Should not update plan when no items
	if len(db.updatePlanCalls) != 0 {
		t.Errorf("expected 0 UpdateUserPlan calls, got %d", len(db.updatePlanCalls))
	}
}

// HTTP error handling tests

func TestStripeClient_HTTPError_StatusCode(t *testing.T) {
	server, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{}`))
	})
	defer server.Close()

	_, err := client.CreateCustomer(context.Background(), "test@example.com", "Test")
	if err == nil {
		t.Fatal("Expected error for 500 status")
	}
	if err.Error() != "Stripe API error: status 500" {
		t.Errorf("error = %v", err)
	}
}

func TestStripeClient_HTTPError_InvalidJSON(t *testing.T) {
	server, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`invalid json`))
	})
	defer server.Close()

	_, err := client.CreateCustomer(context.Background(), "test@example.com", "Test")
	if err == nil {
		t.Fatal("Expected error for invalid JSON response")
	}
}

func TestStripeClient_WithHTTPClient(t *testing.T) {
	cfg := &config.Config{
		StripeSecretKey:     "sk_test_123",
		StripeWebhookSecret: "whsec_123",
	}

	client := NewStripeClient(cfg)

	// Create a custom HTTP client
	customClient := &http.Client{Timeout: 5 * time.Second}
	client.WithHTTPClient(customClient)

	if client.httpClient != customClient {
		t.Error("WithHTTPClient did not set the custom client")
	}
}

func TestStripeClient_WithBaseURL(t *testing.T) {
	cfg := &config.Config{
		StripeSecretKey:     "sk_test_123",
		StripeWebhookSecret: "whsec_123",
	}

	client := NewStripeClient(cfg)
	client.WithBaseURL("http://localhost:8080")

	if client.baseURL != "http://localhost:8080" {
		t.Errorf("baseURL = %s, want http://localhost:8080", client.baseURL)
	}
}

func TestNewStripeClient_BaseURL(t *testing.T) {
	cfg := &config.Config{
		StripeSecretKey:     "sk_test_123",
		StripeWebhookSecret: "whsec_123",
	}

	client := NewStripeClient(cfg)

	if client.baseURL != stripeAPIBase {
		t.Errorf("baseURL = %s, want %s", client.baseURL, stripeAPIBase)
	}
}

// Benchmarks

func BenchmarkParseSignatureHeader(b *testing.B) {
	header := "t=1234567890,v1=abc123def456,v0=xyz789"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parseSignatureHeader(header)
	}
}

func BenchmarkVerifyWebhook(b *testing.B) {
	webhookSecret := "whsec_test_secret"
	client := &StripeClient{
		webhookSecret: webhookSecret,
	}

	payload := []byte(`{"id":"evt_123","type":"checkout.session.completed","data":{},"created":1234567890}`)
	timestamp := fmt.Sprintf("%d", time.Now().Unix())

	signedPayload := timestamp + "." + string(payload)
	mac := hmac.New(sha256.New, []byte(webhookSecret))
	mac.Write([]byte(signedPayload))
	signature := hex.EncodeToString(mac.Sum(nil))

	header := fmt.Sprintf("t=%s,v1=%s", timestamp, signature)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.VerifyWebhook(payload, header)
	}
}
