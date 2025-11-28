package alerts

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/savegress/iotsense/internal/config"
	"github.com/savegress/iotsense/pkg/models"
)

func TestNewSlackNotifier(t *testing.T) {
	cfg := config.SlackConfig{
		WebhookURL: "https://hooks.slack.com/test",
		Channel:    "#alerts",
	}

	notifier := NewSlackNotifier(cfg)

	if notifier == nil {
		t.Fatal("expected notifier to be created")
	}
	if notifier.webhookURL != cfg.WebhookURL {
		t.Errorf("expected webhook URL %s, got %s", cfg.WebhookURL, notifier.webhookURL)
	}
	if notifier.channel != cfg.Channel {
		t.Errorf("expected channel %s, got %s", cfg.Channel, notifier.channel)
	}
	if notifier.client == nil {
		t.Error("expected HTTP client to be created")
	}
}

func TestSlackNotifier_Name(t *testing.T) {
	notifier := NewSlackNotifier(config.SlackConfig{})
	if notifier.Name() != "slack" {
		t.Errorf("expected name 'slack', got '%s'", notifier.Name())
	}
}

func TestSlackNotifier_Notify_EmptyURL(t *testing.T) {
	notifier := NewSlackNotifier(config.SlackConfig{})
	alert := &models.Alert{
		Title:   "Test Alert",
		Message: "Test message",
	}

	err := notifier.Notify(alert)
	if err != nil {
		t.Errorf("expected no error for empty URL, got: %v", err)
	}
}

func TestSlackNotifier_Notify_Success(t *testing.T) {
	var receivedPayload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedPayload)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := NewSlackNotifier(config.SlackConfig{
		WebhookURL: server.URL,
		Channel:    "#test",
	})

	alert := &models.Alert{
		ID:           "alert-1",
		Title:        "High Temperature",
		Message:      "Temperature exceeded threshold",
		Severity:     models.AlertSeverityCritical,
		DeviceID:     "device-1",
		Metric:       "temperature",
		TriggerValue: 95.0,
		Threshold:    80.0,
		CreatedAt:    time.Now(),
	}

	err := notifier.Notify(alert)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if receivedPayload["channel"] != "#test" {
		t.Errorf("expected channel #test, got %v", receivedPayload["channel"])
	}
}

func TestSlackNotifier_Notify_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	notifier := NewSlackNotifier(config.SlackConfig{
		WebhookURL: server.URL,
	})

	alert := &models.Alert{
		Title:   "Test Alert",
		Message: "Test message",
	}

	err := notifier.Notify(alert)
	if err == nil {
		t.Error("expected error for server error response")
	}
}

func TestSlackNotifier_Notify_AllSeverities(t *testing.T) {
	severities := []models.AlertSeverity{
		models.AlertSeverityCritical,
		models.AlertSeverityError,
		models.AlertSeverityWarning,
		models.AlertSeverityInfo,
	}

	expectedColors := map[models.AlertSeverity]string{
		models.AlertSeverityCritical: "#FF0000",
		models.AlertSeverityError:    "#FF6600",
		models.AlertSeverityWarning:  "#FFCC00",
		models.AlertSeverityInfo:     "#36A64F",
	}

	for _, severity := range severities {
		t.Run(string(severity), func(t *testing.T) {
			var receivedPayload map[string]interface{}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				json.NewDecoder(r.Body).Decode(&receivedPayload)
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			notifier := NewSlackNotifier(config.SlackConfig{
				WebhookURL: server.URL,
			})

			alert := &models.Alert{
				Title:    "Test Alert",
				Severity: severity,
			}

			err := notifier.Notify(alert)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			attachments := receivedPayload["attachments"].([]interface{})
			attachment := attachments[0].(map[string]interface{})
			if attachment["color"] != expectedColors[severity] {
				t.Errorf("expected color %s for severity %s, got %s",
					expectedColors[severity], severity, attachment["color"])
			}
		})
	}
}

func TestNewWebhookNotifier(t *testing.T) {
	cfg := config.WebhookConfig{
		URL: "https://example.com/webhook",
		Headers: map[string]string{
			"Authorization": "Bearer token",
		},
	}

	notifier := NewWebhookNotifier(cfg)

	if notifier == nil {
		t.Fatal("expected notifier to be created")
	}
	if notifier.url != cfg.URL {
		t.Errorf("expected URL %s, got %s", cfg.URL, notifier.url)
	}
	if len(notifier.headers) != 1 {
		t.Errorf("expected 1 header, got %d", len(notifier.headers))
	}
	if notifier.client == nil {
		t.Error("expected HTTP client to be created")
	}
}

func TestWebhookNotifier_Name(t *testing.T) {
	notifier := NewWebhookNotifier(config.WebhookConfig{})
	if notifier.Name() != "webhook" {
		t.Errorf("expected name 'webhook', got '%s'", notifier.Name())
	}
}

func TestWebhookNotifier_Notify_EmptyURL(t *testing.T) {
	notifier := NewWebhookNotifier(config.WebhookConfig{})
	alert := &models.Alert{
		Title:   "Test Alert",
		Message: "Test message",
	}

	err := notifier.Notify(alert)
	if err != nil {
		t.Errorf("expected no error for empty URL, got: %v", err)
	}
}

func TestWebhookNotifier_Notify_Success(t *testing.T) {
	var receivedPayload WebhookPayload
	var receivedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header
		json.NewDecoder(r.Body).Decode(&receivedPayload)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := NewWebhookNotifier(config.WebhookConfig{
		URL: server.URL,
		Headers: map[string]string{
			"X-Custom-Header": "custom-value",
			"Authorization":   "Bearer test-token",
		},
	})

	alert := &models.Alert{
		ID:           "alert-1",
		Title:        "High Temperature",
		Message:      "Temperature exceeded threshold",
		Severity:     models.AlertSeverityCritical,
		DeviceID:     "device-1",
		Metric:       "temperature",
		TriggerValue: 95.0,
	}

	err := notifier.Notify(alert)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if receivedPayload.EventType != "alert" {
		t.Errorf("expected event type 'alert', got %s", receivedPayload.EventType)
	}
	if receivedPayload.Alert.ID != "alert-1" {
		t.Errorf("expected alert ID 'alert-1', got %s", receivedPayload.Alert.ID)
	}
	if receivedHeaders.Get("X-Custom-Header") != "custom-value" {
		t.Errorf("expected custom header 'custom-value', got %s", receivedHeaders.Get("X-Custom-Header"))
	}
	if receivedHeaders.Get("Authorization") != "Bearer test-token" {
		t.Errorf("expected authorization header")
	}
	if receivedHeaders.Get("Content-Type") != "application/json" {
		t.Errorf("expected content-type application/json, got %s", receivedHeaders.Get("Content-Type"))
	}
}

func TestWebhookNotifier_Notify_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	notifier := NewWebhookNotifier(config.WebhookConfig{
		URL: server.URL,
	})

	alert := &models.Alert{
		Title:   "Test Alert",
		Message: "Test message",
	}

	err := notifier.Notify(alert)
	if err == nil {
		t.Error("expected error for server error response")
	}
}

func TestNewConsoleNotifier(t *testing.T) {
	notifier := NewConsoleNotifier()
	if notifier == nil {
		t.Fatal("expected notifier to be created")
	}
}

func TestConsoleNotifier_Name(t *testing.T) {
	notifier := NewConsoleNotifier()
	if notifier.Name() != "console" {
		t.Errorf("expected name 'console', got '%s'", notifier.Name())
	}
}

func TestConsoleNotifier_Notify(t *testing.T) {
	notifier := NewConsoleNotifier()
	alert := &models.Alert{
		ID:           "alert-1",
		Title:        "Test Alert",
		Message:      "Test message",
		Severity:     models.AlertSeverityWarning,
		DeviceID:     "device-1",
		Metric:       "cpu_usage",
		TriggerValue: 95.0,
	}

	err := notifier.Notify(alert)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetSlackColor(t *testing.T) {
	tests := []struct {
		severity models.AlertSeverity
		expected string
	}{
		{models.AlertSeverityCritical, "#FF0000"},
		{models.AlertSeverityError, "#FF6600"},
		{models.AlertSeverityWarning, "#FFCC00"},
		{models.AlertSeverityInfo, "#36A64F"},
		{"unknown", "#36A64F"}, // default
	}

	for _, test := range tests {
		t.Run(string(test.severity), func(t *testing.T) {
			color := getSlackColor(test.severity)
			if color != test.expected {
				t.Errorf("expected color %s for severity %s, got %s",
					test.expected, test.severity, color)
			}
		})
	}
}
