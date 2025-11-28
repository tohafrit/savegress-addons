package dashboard

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// AlertRule defines an alerting rule
type AlertRule struct {
	Name        string
	Condition   AlertCondition
	Duration    time.Duration
	Severity    AlertSeverity
	Annotations map[string]string
}

// AlertCondition evaluates a condition
type AlertCondition func(metrics interface{}) bool

// AlertSeverity represents alert severity
type AlertSeverity string

const (
	SeverityWarning  AlertSeverity = "warning"
	SeverityError    AlertSeverity = "error"
	SeverityCritical AlertSeverity = "critical"
)

// Alert represents an active alert
type Alert struct {
	Rule        AlertRule
	FiredAt     time.Time
	Value       interface{}
	Message     string
	Severity    AlertSeverity
	Annotations map[string]string
}

// AlertChannel sends alerts to external systems
type AlertChannel interface {
	Send(alert Alert) error
}

// AlertManager manages alerting rules and channels
type AlertManager struct {
	rules        []AlertRule
	channels     []AlertChannel
	evaluator    *RuleEvaluator
	state        sync.Map // map[string]*AlertState
	stopChan     chan struct{}
	wg           sync.WaitGroup
	provider     MetricsProvider
}

// AlertState tracks the state of an alert
type AlertState struct {
	FirstFired time.Time
	LastFired  time.Time
	Count      int
	Active     bool
}

// NewAlertManager creates a new alert manager
func NewAlertManager(provider MetricsProvider) *AlertManager {
	return &AlertManager{
		rules:     []AlertRule{},
		channels:  []AlertChannel{},
		evaluator: NewRuleEvaluator(),
		provider:  provider,
		stopChan:  make(chan struct{}),
	}
}

// AddRule adds an alerting rule
func (am *AlertManager) AddRule(rule AlertRule) {
	am.rules = append(am.rules, rule)
}

// AddChannel adds an alert channel
func (am *AlertManager) AddChannel(channel AlertChannel) {
	am.channels = append(am.channels, channel)
}

// Start starts the alert manager
func (am *AlertManager) Start() {
	am.wg.Add(1)
	go am.evaluationLoop()
}

// Stop stops the alert manager
func (am *AlertManager) Stop() {
	close(am.stopChan)
	am.wg.Wait()
}

// evaluationLoop continuously evaluates rules
func (am *AlertManager) evaluationLoop() {
	defer am.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-am.stopChan:
			return
		case <-ticker.C:
			if am.provider != nil {
				metrics := am.provider.GetMetrics()
				am.Evaluate(metrics)
			}
		}
	}
}

// Evaluate evaluates all rules against current metrics
func (am *AlertManager) Evaluate(metrics interface{}) {
	for _, rule := range am.rules {
		shouldFire := rule.Condition(metrics)

		if shouldFire {
			am.handleAlert(rule, metrics)
		} else {
			am.clearAlert(rule)
		}
	}
}

// handleAlert handles a firing alert
func (am *AlertManager) handleAlert(rule AlertRule, metrics interface{}) {
	key := rule.Name

	stateI, _ := am.state.LoadOrStore(key, &AlertState{})
	state := stateI.(*AlertState)

	now := time.Now()

	if !state.Active {
		// First time firing
		state.FirstFired = now
		state.Active = true
		state.Count = 1
	} else {
		// Already firing, check duration
		if now.Sub(state.FirstFired) < rule.Duration {
			return // Not long enough yet
		}
		state.Count++
	}

	state.LastFired = now

	// Fire alert
	alert := Alert{
		Rule:        rule,
		FiredAt:     now,
		Message:     fmt.Sprintf("Alert: %s", rule.Name),
		Severity:    rule.Severity,
		Annotations: rule.Annotations,
		Value:       metrics,
	}

	// Send to all channels
	for _, channel := range am.channels {
		go func(ch AlertChannel) {
			if err := ch.Send(alert); err != nil {
				// Log error (simplified)
				fmt.Printf("Failed to send alert: %v\n", err)
			}
		}(channel)
	}
}

// clearAlert clears an alert
func (am *AlertManager) clearAlert(rule AlertRule) {
	key := rule.Name

	stateI, ok := am.state.Load(key)
	if !ok {
		return
	}

	state := stateI.(*AlertState)
	state.Active = false
}

// GetActiveAlerts returns all active alerts
func (am *AlertManager) GetActiveAlerts() []Alert {
	var alerts []Alert

	am.state.Range(func(key, value interface{}) bool {
		state := value.(*AlertState)
		if state.Active {
			// Find corresponding rule
			ruleName := key.(string)
			for _, rule := range am.rules {
				if rule.Name == ruleName {
					alerts = append(alerts, Alert{
						Rule:     rule,
						FiredAt:  state.FirstFired,
						Message:  fmt.Sprintf("Alert: %s", rule.Name),
						Severity: rule.Severity,
					})
					break
				}
			}
		}
		return true
	})

	return alerts
}

// RuleEvaluator evaluates rule conditions
type RuleEvaluator struct{}

// NewRuleEvaluator creates a new rule evaluator
func NewRuleEvaluator() *RuleEvaluator {
	return &RuleEvaluator{}
}

// Evaluate evaluates a condition string (simplified)
func (re *RuleEvaluator) Evaluate(condition string, metrics interface{}) bool {
	// This is a simplified implementation
	// In production, use a proper expression evaluator
	return false
}

// Alert Channels

// SlackChannel sends alerts to Slack
type SlackChannel struct {
	WebhookURL string
}

// Send sends an alert to Slack
func (sc *SlackChannel) Send(alert Alert) error {
	payload := map[string]interface{}{
		"text": fmt.Sprintf("[%s] %s", alert.Severity, alert.Message),
		"attachments": []map[string]interface{}{
			{
				"color": sc.severityColor(alert.Severity),
				"fields": []map[string]interface{}{
					{
						"title": "Rule",
						"value": alert.Rule.Name,
						"short": true,
					},
					{
						"title": "Fired At",
						"value": alert.FiredAt.Format(time.RFC3339),
						"short": true,
					},
				},
			},
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := http.Post(sc.WebhookURL, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack returned status %d", resp.StatusCode)
	}

	return nil
}

func (sc *SlackChannel) severityColor(severity AlertSeverity) string {
	switch severity {
	case SeverityWarning:
		return "warning"
	case SeverityError:
		return "danger"
	case SeverityCritical:
		return "#ff0000"
	default:
		return "good"
	}
}

// WebhookChannel sends alerts to a generic webhook
type WebhookChannel struct {
	URL     string
	Headers map[string]string
}

// Send sends an alert to a webhook
func (wc *WebhookChannel) Send(alert Alert) error {
	data, err := json.Marshal(alert)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", wc.URL, bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range wc.Headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// LogChannel logs alerts to stdout
type LogChannel struct{}

// Send logs an alert
func (lc *LogChannel) Send(alert Alert) error {
	fmt.Printf("[ALERT] [%s] %s - %s\n",
		alert.Severity,
		alert.Rule.Name,
		alert.Message,
	)
	return nil
}
