package alerts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/savegress/iotsense/internal/config"
	"github.com/savegress/iotsense/pkg/models"
)

// SlackNotifier sends alerts to Slack
type SlackNotifier struct {
	webhookURL string
	channel    string
	client     *http.Client
}

// NewSlackNotifier creates a new Slack notifier
func NewSlackNotifier(cfg config.SlackConfig) *SlackNotifier {
	return &SlackNotifier{
		webhookURL: cfg.WebhookURL,
		channel:    cfg.Channel,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Name returns the notifier name
func (n *SlackNotifier) Name() string {
	return "slack"
}

// Notify sends an alert to Slack
func (n *SlackNotifier) Notify(alert *models.Alert) error {
	if n.webhookURL == "" {
		return nil
	}

	color := getSlackColor(alert.Severity)

	payload := map[string]interface{}{
		"channel": n.channel,
		"attachments": []map[string]interface{}{
			{
				"color":  color,
				"title":  fmt.Sprintf("[%s] %s", alert.Severity, alert.Title),
				"text":   alert.Message,
				"fields": []map[string]interface{}{
					{"title": "Device", "value": alert.DeviceID, "short": true},
					{"title": "Metric", "value": alert.Metric, "short": true},
					{"title": "Value", "value": fmt.Sprintf("%v", alert.TriggerValue), "short": true},
					{"title": "Threshold", "value": fmt.Sprintf("%v", alert.Threshold), "short": true},
				},
				"footer": "IoTSense Alert",
				"ts":     alert.CreatedAt.Unix(),
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := n.client.Post(n.webhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("slack returned status %d", resp.StatusCode)
	}

	return nil
}

func getSlackColor(severity models.AlertSeverity) string {
	switch severity {
	case models.AlertSeverityCritical:
		return "#FF0000"
	case models.AlertSeverityError:
		return "#FF6600"
	case models.AlertSeverityWarning:
		return "#FFCC00"
	default:
		return "#36A64F"
	}
}

// WebhookNotifier sends alerts to a webhook
type WebhookNotifier struct {
	url     string
	headers map[string]string
	client  *http.Client
}

// NewWebhookNotifier creates a new webhook notifier
func NewWebhookNotifier(cfg config.WebhookConfig) *WebhookNotifier {
	return &WebhookNotifier{
		url:     cfg.URL,
		headers: cfg.Headers,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Name returns the notifier name
func (n *WebhookNotifier) Name() string {
	return "webhook"
}

// Notify sends an alert to the webhook
func (n *WebhookNotifier) Notify(alert *models.Alert) error {
	if n.url == "" {
		return nil
	}

	payload := WebhookPayload{
		EventType: "alert",
		Alert:     alert,
		Timestamp: time.Now(),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", n.url, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	for key, value := range n.headers {
		req.Header.Set(key, value)
	}

	resp, err := n.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// WebhookPayload represents the webhook payload
type WebhookPayload struct {
	EventType string        `json:"event_type"`
	Alert     *models.Alert `json:"alert"`
	Timestamp time.Time     `json:"timestamp"`
}

// ConsoleNotifier logs alerts to console (for testing)
type ConsoleNotifier struct{}

// NewConsoleNotifier creates a new console notifier
func NewConsoleNotifier() *ConsoleNotifier {
	return &ConsoleNotifier{}
}

// Name returns the notifier name
func (n *ConsoleNotifier) Name() string {
	return "console"
}

// Notify logs the alert
func (n *ConsoleNotifier) Notify(alert *models.Alert) error {
	fmt.Printf("[ALERT] [%s] %s: %s (Device: %s, Metric: %s, Value: %v)\n",
		alert.Severity, alert.Title, alert.Message, alert.DeviceID, alert.Metric, alert.TriggerValue)
	return nil
}
