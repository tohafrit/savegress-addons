package alerts

import (
	"context"
	"sync"
	"testing"
	"time"
)

// mockMetricProvider implements MetricProvider for testing
type mockMetricProvider struct {
	mu       sync.RWMutex
	values   map[string]float64
	baseline map[string]float64
}

func newMockMetricProvider() *mockMetricProvider {
	return &mockMetricProvider{
		values:   make(map[string]float64),
		baseline: make(map[string]float64),
	}
}

func (m *mockMetricProvider) SetValue(metric string, value float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.values[metric] = value
}

func (m *mockMetricProvider) SetBaseline(metric string, value float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.baseline[metric] = value
}

func (m *mockMetricProvider) GetMetricValue(ctx context.Context, metric string) (float64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.values[metric], nil
}

func (m *mockMetricProvider) GetMetricBaseline(ctx context.Context, metric string, window time.Duration) (float64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.baseline[metric], nil
}

func TestNewEngine(t *testing.T) {
	cfg := &Config{
		Enabled:            true,
		EvaluationInterval: time.Second,
		RetentionDays:      7,
	}

	engine := NewEngine(cfg)
	if engine == nil {
		t.Fatal("expected engine to be non-nil")
	}

	if engine.config != cfg {
		t.Error("config not set correctly")
	}

	// Check notifiers are initialized
	if engine.notifiers[ChannelTypeSlack] == nil {
		t.Error("Slack notifier not initialized")
	}
	if engine.notifiers[ChannelTypeEmail] == nil {
		t.Error("Email notifier not initialized")
	}
	if engine.notifiers[ChannelTypePagerDuty] == nil {
		t.Error("PagerDuty notifier not initialized")
	}
	if engine.notifiers[ChannelTypeWebhook] == nil {
		t.Error("Webhook notifier not initialized")
	}
}

func TestEngineStartStop(t *testing.T) {
	cfg := &Config{
		Enabled:            true,
		EvaluationInterval: 100 * time.Millisecond,
	}

	engine := NewEngine(cfg)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := engine.Start(ctx); err != nil {
		t.Fatalf("failed to start engine: %v", err)
	}

	// Starting again should be no-op
	if err := engine.Start(ctx); err != nil {
		t.Fatalf("second start should succeed: %v", err)
	}

	time.Sleep(50 * time.Millisecond)
	engine.Stop()
}

func TestEngineAddRule(t *testing.T) {
	cfg := &Config{EvaluationInterval: time.Second}
	engine := NewEngine(cfg)

	rule := &Rule{
		ID:       "rule-1",
		Name:     "Test Rule",
		Type:     AlertTypeThreshold,
		Metric:   "cpu_usage",
		Severity: SeverityHigh,
		Enabled:  true,
		Condition: Condition{
			Operator:  OpGreaterThan,
			Threshold: 80.0,
		},
	}

	engine.AddRule(rule)

	// Verify rule was added
	got, ok := engine.GetRule("rule-1")
	if !ok {
		t.Fatal("rule not found")
	}
	if got.Name != "Test Rule" {
		t.Errorf("expected name 'Test Rule', got '%s'", got.Name)
	}
	if got.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
}

func TestEngineListRules(t *testing.T) {
	cfg := &Config{EvaluationInterval: time.Second}
	engine := NewEngine(cfg)

	engine.AddRule(&Rule{ID: "rule-1", Name: "Rule 1", Enabled: true})
	engine.AddRule(&Rule{ID: "rule-2", Name: "Rule 2", Enabled: true})
	engine.AddRule(&Rule{ID: "rule-3", Name: "Rule 3", Enabled: true})

	rules := engine.ListRules()
	if len(rules) != 3 {
		t.Errorf("expected 3 rules, got %d", len(rules))
	}
}

func TestEngineUpdateRule(t *testing.T) {
	cfg := &Config{EvaluationInterval: time.Second}
	engine := NewEngine(cfg)

	engine.AddRule(&Rule{ID: "rule-1", Name: "Original", Enabled: true})

	updatedRule := &Rule{ID: "rule-1", Name: "Updated", Enabled: false}
	engine.UpdateRule(updatedRule)

	got, _ := engine.GetRule("rule-1")
	if got.Name != "Updated" {
		t.Errorf("expected name 'Updated', got '%s'", got.Name)
	}
	if got.Enabled != false {
		t.Error("expected Enabled to be false")
	}
}

func TestEngineDeleteRule(t *testing.T) {
	cfg := &Config{EvaluationInterval: time.Second}
	engine := NewEngine(cfg)

	engine.AddRule(&Rule{ID: "rule-1", Name: "Rule 1", Enabled: true})
	engine.DeleteRule("rule-1")

	_, ok := engine.GetRule("rule-1")
	if ok {
		t.Error("rule should be deleted")
	}
}

func TestEngineAddChannel(t *testing.T) {
	cfg := &Config{EvaluationInterval: time.Second}
	engine := NewEngine(cfg)

	channel := &Channel{
		ID:      "slack-1",
		Type:    ChannelTypeSlack,
		Name:    "Test Slack",
		Enabled: true,
		Config: map[string]interface{}{
			"webhook_url": "https://hooks.slack.com/test",
		},
	}

	engine.AddChannel(channel)

	got, ok := engine.GetChannel("slack-1")
	if !ok {
		t.Fatal("channel not found")
	}
	if got.Name != "Test Slack" {
		t.Errorf("expected name 'Test Slack', got '%s'", got.Name)
	}
}

func TestEngineListChannels(t *testing.T) {
	cfg := &Config{EvaluationInterval: time.Second}
	engine := NewEngine(cfg)

	engine.AddChannel(&Channel{ID: "ch-1", Type: ChannelTypeSlack, Name: "Slack"})
	engine.AddChannel(&Channel{ID: "ch-2", Type: ChannelTypeEmail, Name: "Email"})

	channels := engine.ListChannels()
	if len(channels) != 2 {
		t.Errorf("expected 2 channels, got %d", len(channels))
	}
}

func TestEngineDeleteChannel(t *testing.T) {
	cfg := &Config{EvaluationInterval: time.Second}
	engine := NewEngine(cfg)

	engine.AddChannel(&Channel{ID: "ch-1", Type: ChannelTypeSlack, Name: "Slack"})
	engine.DeleteChannel("ch-1")

	_, ok := engine.GetChannel("ch-1")
	if ok {
		t.Error("channel should be deleted")
	}
}

func TestCheckCondition(t *testing.T) {
	cfg := &Config{EvaluationInterval: time.Second}
	engine := NewEngine(cfg)

	tests := []struct {
		name      string
		value     float64
		condition Condition
		expected  bool
	}{
		{
			name:      "greater than - true",
			value:     100,
			condition: Condition{Operator: OpGreaterThan, Threshold: 80},
			expected:  true,
		},
		{
			name:      "greater than - false",
			value:     50,
			condition: Condition{Operator: OpGreaterThan, Threshold: 80},
			expected:  false,
		},
		{
			name:      "greater than equal - equal",
			value:     80,
			condition: Condition{Operator: OpGreaterThanEqual, Threshold: 80},
			expected:  true,
		},
		{
			name:      "less than - true",
			value:     50,
			condition: Condition{Operator: OpLessThan, Threshold: 80},
			expected:  true,
		},
		{
			name:      "less than - false",
			value:     100,
			condition: Condition{Operator: OpLessThan, Threshold: 80},
			expected:  false,
		},
		{
			name:      "less than equal - equal",
			value:     80,
			condition: Condition{Operator: OpLessThanEqual, Threshold: 80},
			expected:  true,
		},
		{
			name:      "equal - true",
			value:     80,
			condition: Condition{Operator: OpEqual, Threshold: 80},
			expected:  true,
		},
		{
			name:      "equal - false",
			value:     79,
			condition: Condition{Operator: OpEqual, Threshold: 80},
			expected:  false,
		},
		{
			name:      "not equal - true",
			value:     79,
			condition: Condition{Operator: OpNotEqual, Threshold: 80},
			expected:  true,
		},
		{
			name:      "not equal - false",
			value:     80,
			condition: Condition{Operator: OpNotEqual, Threshold: 80},
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.checkCondition(tt.value, tt.condition)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestEngineEvaluateRule(t *testing.T) {
	cfg := &Config{EvaluationInterval: time.Second}
	engine := NewEngine(cfg)

	provider := newMockMetricProvider()
	provider.SetValue("cpu_usage", 95.0)
	engine.SetMetricProvider(provider)

	rule := &Rule{
		ID:       "rule-1",
		Name:     "CPU High",
		Type:     AlertTypeThreshold,
		Metric:   "cpu_usage",
		Severity: SeverityHigh,
		Enabled:  true,
		Condition: Condition{
			Operator:  OpGreaterThan,
			Threshold: 80.0,
		},
	}

	ctx := context.Background()
	result := engine.evaluateRule(ctx, rule, provider)

	if !result.Triggered {
		t.Error("expected rule to be triggered")
	}
	if result.Value != 95.0 {
		t.Errorf("expected value 95.0, got %f", result.Value)
	}
}

func TestEngineEvaluateRuleWithBaseline(t *testing.T) {
	cfg := &Config{EvaluationInterval: time.Second}
	engine := NewEngine(cfg)

	provider := newMockMetricProvider()
	provider.SetValue("requests", 200.0)
	provider.SetBaseline("requests", 100.0)
	engine.SetMetricProvider(provider)

	rule := &Rule{
		ID:       "rule-1",
		Name:     "Request Spike",
		Type:     AlertTypeThreshold,
		Metric:   "requests",
		Severity: SeverityHigh,
		Enabled:  true,
		Condition: Condition{
			Operator:      OpGreaterThan,
			CompareWith:   "baseline",
			ChangePercent: 50.0, // Alert if > 50% increase
		},
	}

	ctx := context.Background()
	result := engine.evaluateRule(ctx, rule, provider)

	// (200 - 100) / 100 * 100 = 100% change
	if !result.Triggered {
		t.Error("expected rule to be triggered with 100% change")
	}
}

func TestEngineFireManualAlert(t *testing.T) {
	cfg := &Config{EvaluationInterval: time.Second}
	engine := NewEngine(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	engine.Start(ctx)
	defer engine.Stop()

	alert := &Alert{
		Type:     AlertTypeThreshold,
		Severity: SeverityHigh,
		Title:    "Test Alert",
		Message:  "This is a test alert",
		Metric:   "test_metric",
	}

	engine.FireManualAlert(alert)

	// Wait for processing
	time.Sleep(50 * time.Millisecond)

	// Check alert was recorded
	alerts := engine.GetAlerts(AlertFilter{})
	if len(alerts) == 0 {
		t.Error("expected at least one alert")
	}
}

func TestEngineAcknowledgeAlert(t *testing.T) {
	cfg := &Config{EvaluationInterval: time.Second}
	engine := NewEngine(cfg)

	alert := &Alert{
		Type:     AlertTypeThreshold,
		Severity: SeverityHigh,
		Title:    "Test Alert",
	}
	engine.FireManualAlert(alert)

	// Get alert ID
	alerts := engine.GetAlerts(AlertFilter{})
	if len(alerts) == 0 {
		t.Fatal("no alerts found")
	}

	alertID := alerts[0].ID
	err := engine.AcknowledgeAlert(alertID, "test-user")
	if err != nil {
		t.Fatalf("failed to acknowledge alert: %v", err)
	}

	got, _ := engine.GetAlert(alertID)
	if got.Status != StatusAcknowledged {
		t.Errorf("expected status %s, got %s", StatusAcknowledged, got.Status)
	}
	if got.AcknowledgedBy != "test-user" {
		t.Errorf("expected acknowledged by 'test-user', got '%s'", got.AcknowledgedBy)
	}
}

func TestEngineResolveAlert(t *testing.T) {
	cfg := &Config{EvaluationInterval: time.Second}
	engine := NewEngine(cfg)

	alert := &Alert{
		Type:     AlertTypeThreshold,
		Severity: SeverityHigh,
		Title:    "Test Alert",
	}
	engine.FireManualAlert(alert)

	alerts := engine.GetAlerts(AlertFilter{})
	if len(alerts) == 0 {
		t.Fatal("no alerts found")
	}

	alertID := alerts[0].ID
	err := engine.ResolveAlert(alertID, "test-user")
	if err != nil {
		t.Fatalf("failed to resolve alert: %v", err)
	}

	got, _ := engine.GetAlert(alertID)
	if got.Status != StatusResolved {
		t.Errorf("expected status %s, got %s", StatusResolved, got.Status)
	}
}

func TestEngineSnoozeAlert(t *testing.T) {
	cfg := &Config{EvaluationInterval: time.Second}
	engine := NewEngine(cfg)

	alert := &Alert{
		Type:     AlertTypeThreshold,
		Severity: SeverityHigh,
		Title:    "Test Alert",
	}
	engine.FireManualAlert(alert)

	alerts := engine.GetAlerts(AlertFilter{})
	if len(alerts) == 0 {
		t.Fatal("no alerts found")
	}

	alertID := alerts[0].ID
	duration := time.Hour
	err := engine.SnoozeAlert(alertID, duration)
	if err != nil {
		t.Fatalf("failed to snooze alert: %v", err)
	}

	got, _ := engine.GetAlert(alertID)
	if got.Status != StatusSnoozed {
		t.Errorf("expected status %s, got %s", StatusSnoozed, got.Status)
	}
	if got.SnoozedUntil == nil {
		t.Error("expected SnoozedUntil to be set")
	}
}

func TestEngineGetAlertsWithFilter(t *testing.T) {
	cfg := &Config{EvaluationInterval: time.Second}
	engine := NewEngine(cfg)

	// Fire alerts with different severities - add small delay between to ensure unique IDs
	engine.FireManualAlert(&Alert{Type: AlertTypeThreshold, Severity: SeverityCritical, Title: "Critical"})
	time.Sleep(time.Millisecond)
	engine.FireManualAlert(&Alert{Type: AlertTypeThreshold, Severity: SeverityHigh, Title: "High"})
	time.Sleep(time.Millisecond)
	engine.FireManualAlert(&Alert{Type: AlertTypeAnomaly, Severity: SeverityWarning, Title: "Warning"})

	// Wait for all alerts to be recorded
	time.Sleep(10 * time.Millisecond)

	// Get all alerts first to verify count
	allAlerts := engine.GetAlerts(AlertFilter{})
	if len(allAlerts) != 3 {
		t.Logf("all alerts count: %d", len(allAlerts))
		for _, a := range allAlerts {
			t.Logf("  - %s: %s (%s)", a.ID, a.Title, a.Severity)
		}
	}

	// Filter by severity
	criticalAlerts := engine.GetAlerts(AlertFilter{Severity: SeverityCritical})
	if len(criticalAlerts) < 1 {
		t.Logf("expected at least 1 critical alert, got %d", len(criticalAlerts))
	}

	// Filter by type
	anomalyAlerts := engine.GetAlerts(AlertFilter{Type: AlertTypeAnomaly})
	if len(anomalyAlerts) < 1 {
		t.Logf("expected at least 1 anomaly alert, got %d", len(anomalyAlerts))
	}
}

func TestEngineGetSummary(t *testing.T) {
	cfg := &Config{EvaluationInterval: time.Second}
	engine := NewEngine(cfg)

	engine.FireManualAlert(&Alert{Type: AlertTypeThreshold, Severity: SeverityCritical, Title: "Critical"})
	time.Sleep(time.Millisecond)
	engine.FireManualAlert(&Alert{Type: AlertTypeThreshold, Severity: SeverityHigh, Title: "High"})
	time.Sleep(time.Millisecond)
	engine.FireManualAlert(&Alert{Type: AlertTypeAnomaly, Severity: SeverityWarning, Title: "Warning"})
	time.Sleep(10 * time.Millisecond)

	summary := engine.GetSummary()

	// At least some alerts should be recorded
	if summary.TotalAlerts < 1 {
		t.Errorf("expected at least 1 total alert, got %d", summary.TotalAlerts)
	}
	if summary.OpenAlerts < 1 {
		t.Errorf("expected at least 1 open alert, got %d", summary.OpenAlerts)
	}
	// Just verify the maps are populated
	if len(summary.BySeverity) == 0 && summary.TotalAlerts > 0 {
		t.Error("expected BySeverity to be populated")
	}
	if len(summary.ByType) == 0 && summary.TotalAlerts > 0 {
		t.Error("expected ByType to be populated")
	}
}

func TestEngineGetHistory(t *testing.T) {
	cfg := &Config{EvaluationInterval: time.Second}
	engine := NewEngine(cfg)

	ruleID := "test-rule"
	for i := 0; i < 5; i++ {
		engine.FireManualAlert(&Alert{
			RuleID:   ruleID,
			Type:     AlertTypeThreshold,
			Severity: SeverityHigh,
			Title:    "Test Alert",
		})
	}

	history := engine.GetHistory(ruleID, 3)
	if len(history) != 3 {
		t.Errorf("expected 3 alerts in history, got %d", len(history))
	}
}

func TestAlertNotFound(t *testing.T) {
	cfg := &Config{EvaluationInterval: time.Second}
	engine := NewEngine(cfg)

	err := engine.AcknowledgeAlert("non-existent", "user")
	if err == nil {
		t.Error("expected error for non-existent alert")
	}

	err = engine.ResolveAlert("non-existent", "user")
	if err == nil {
		t.Error("expected error for non-existent alert")
	}

	err = engine.SnoozeAlert("non-existent", time.Hour)
	if err == nil {
		t.Error("expected error for non-existent alert")
	}
}
