package alerts

import (
	"context"
	"testing"
	"time"

	"github.com/savegress/iotsense/internal/config"
	"github.com/savegress/iotsense/pkg/models"
)

func TestNewEngine(t *testing.T) {
	cfg := &config.AlertsConfig{
		EvaluationInterval: time.Second,
		MaxActiveAlerts:    100,
	}

	engine := NewEngine(cfg)

	if engine == nil {
		t.Fatal("expected engine to be created")
	}

	if engine.config != cfg {
		t.Error("config not set correctly")
	}

	if engine.rules == nil {
		t.Error("rules map not initialized")
	}

	if engine.alerts == nil {
		t.Error("alerts map not initialized")
	}
}

func TestEngine_StartStop(t *testing.T) {
	cfg := &config.AlertsConfig{
		EvaluationInterval: time.Millisecond * 100,
		MaxActiveAlerts:    100,
	}
	engine := NewEngine(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start engine
	err := engine.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start engine: %v", err)
	}

	if !engine.running {
		t.Error("engine should be running")
	}

	// Starting again should be no-op
	err = engine.Start(ctx)
	if err != nil {
		t.Fatalf("second start should not fail: %v", err)
	}

	// Stop engine
	engine.Stop()

	if engine.running {
		t.Error("engine should not be running after stop")
	}

	// Stopping again should be safe
	engine.Stop()
}

func TestEngine_SetAlertCallback(t *testing.T) {
	cfg := &config.AlertsConfig{
		EvaluationInterval: time.Second,
	}
	engine := NewEngine(cfg)

	engine.SetAlertCallback(func(alert *models.Alert) {
		// Callback set
	})

	if engine.onAlert == nil {
		t.Error("callback should be set")
	}
}

func TestEngine_AddNotifier(t *testing.T) {
	cfg := &config.AlertsConfig{
		EvaluationInterval: time.Second,
	}
	engine := NewEngine(cfg)

	notifier := &mockNotifier{name: "test"}
	engine.AddNotifier(notifier)

	if len(engine.notifiers) != 1 {
		t.Errorf("expected 1 notifier, got %d", len(engine.notifiers))
	}

	if engine.notifiers[0].Name() != "test" {
		t.Error("notifier name mismatch")
	}
}

type mockNotifier struct {
	name      string
	notified  bool
	lastAlert *models.Alert
}

func (n *mockNotifier) Name() string { return n.name }

func (n *mockNotifier) Notify(alert *models.Alert) error {
	n.notified = true
	n.lastAlert = alert
	return nil
}

func TestEngine_CreateRule(t *testing.T) {
	cfg := &config.AlertsConfig{}
	engine := NewEngine(cfg)

	rule := &models.AlertRule{
		Name:     "High Temperature",
		Metric:   "temperature",
		Enabled:  true,
		Severity: models.AlertSeverityCritical,
		Condition: models.RuleCondition{
			Operator: "gt",
			Value:    80.0,
		},
	}

	err := engine.CreateRule(rule)
	if err != nil {
		t.Fatalf("failed to create rule: %v", err)
	}

	if rule.ID == "" {
		t.Error("expected rule ID to be set")
	}

	if rule.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}

	// Verify rule was stored
	retrieved, ok := engine.GetRule(rule.ID)
	if !ok {
		t.Error("expected to find created rule")
	}

	if retrieved.Name != "High Temperature" {
		t.Error("rule name mismatch")
	}
}

func TestEngine_CreateRule_WithExistingID(t *testing.T) {
	cfg := &config.AlertsConfig{}
	engine := NewEngine(cfg)

	rule := &models.AlertRule{
		ID:     "existing-id",
		Name:   "Test Rule",
		Metric: "test",
	}

	engine.CreateRule(rule)

	if rule.ID != "existing-id" {
		t.Error("existing ID should be preserved")
	}
}

func TestEngine_UpdateRule(t *testing.T) {
	cfg := &config.AlertsConfig{}
	engine := NewEngine(cfg)

	rule := &models.AlertRule{
		Name:   "Test Rule",
		Metric: "temperature",
	}
	engine.CreateRule(rule)

	// Update rule
	rule.Name = "Updated Rule"
	err := engine.UpdateRule(rule)
	if err != nil {
		t.Fatalf("failed to update rule: %v", err)
	}

	retrieved, _ := engine.GetRule(rule.ID)
	if retrieved.Name != "Updated Rule" {
		t.Error("rule not updated")
	}

	if retrieved.UpdatedAt.Before(retrieved.CreatedAt) {
		t.Error("UpdatedAt should be after CreatedAt")
	}
}

func TestEngine_UpdateRule_NotFound(t *testing.T) {
	cfg := &config.AlertsConfig{}
	engine := NewEngine(cfg)

	rule := &models.AlertRule{
		ID:   "nonexistent",
		Name: "Test",
	}

	err := engine.UpdateRule(rule)

	if err != ErrRuleNotFound {
		t.Errorf("expected ErrRuleNotFound, got %v", err)
	}
}

func TestEngine_DeleteRule(t *testing.T) {
	cfg := &config.AlertsConfig{}
	engine := NewEngine(cfg)

	rule := &models.AlertRule{
		Name:   "Test Rule",
		Metric: "temperature",
	}
	engine.CreateRule(rule)

	err := engine.DeleteRule(rule.ID)
	if err != nil {
		t.Fatalf("failed to delete rule: %v", err)
	}

	_, ok := engine.GetRule(rule.ID)
	if ok {
		t.Error("rule should be deleted")
	}
}

func TestEngine_DeleteRule_NotFound(t *testing.T) {
	cfg := &config.AlertsConfig{}
	engine := NewEngine(cfg)

	err := engine.DeleteRule("nonexistent")

	if err != ErrRuleNotFound {
		t.Errorf("expected ErrRuleNotFound, got %v", err)
	}
}

func TestEngine_ListRules(t *testing.T) {
	cfg := &config.AlertsConfig{}
	engine := NewEngine(cfg)

	engine.CreateRule(&models.AlertRule{Name: "Rule 1", Metric: "temp"})
	engine.CreateRule(&models.AlertRule{Name: "Rule 2", Metric: "humidity"})
	engine.CreateRule(&models.AlertRule{Name: "Rule 3", Metric: "pressure"})

	rules := engine.ListRules()

	if len(rules) != 3 {
		t.Errorf("expected 3 rules, got %d", len(rules))
	}
}

func TestEngine_ProcessTelemetry(t *testing.T) {
	cfg := &config.AlertsConfig{
		EvaluationInterval: time.Millisecond * 50,
		MaxActiveAlerts:    100,
	}
	engine := NewEngine(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	engine.Start(ctx)
	defer engine.Stop()

	// Create a rule
	rule := &models.AlertRule{
		Name:     "High Temperature",
		Metric:   "temperature",
		Enabled:  true,
		Severity: models.AlertSeverityCritical,
		Condition: models.RuleCondition{
			Operator: "gt",
			Value:    80.0,
		},
	}
	engine.CreateRule(rule)

	// Process telemetry that should trigger alert
	point := &models.TelemetryPoint{
		DeviceID:  "device-1",
		Metric:    "temperature",
		Value:     90.0,
		Timestamp: time.Now(),
	}

	engine.ProcessTelemetry(point)

	// Give time for async processing
	time.Sleep(100 * time.Millisecond)

	// Should have created an alert
	alerts := engine.ListAlerts(AlertFilter{})
	if len(alerts) != 1 {
		t.Errorf("expected 1 alert, got %d", len(alerts))
	}
}

func TestEngine_ProcessTelemetry_NoTrigger(t *testing.T) {
	cfg := &config.AlertsConfig{
		EvaluationInterval: time.Millisecond * 50,
		MaxActiveAlerts:    100,
	}
	engine := NewEngine(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	engine.Start(ctx)
	defer engine.Stop()

	// Create a rule
	rule := &models.AlertRule{
		Name:     "High Temperature",
		Metric:   "temperature",
		Enabled:  true,
		Severity: models.AlertSeverityCritical,
		Condition: models.RuleCondition{
			Operator: "gt",
			Value:    80.0,
		},
	}
	engine.CreateRule(rule)

	// Process telemetry that should NOT trigger alert (below threshold)
	point := &models.TelemetryPoint{
		DeviceID:  "device-1",
		Metric:    "temperature",
		Value:     70.0,
		Timestamp: time.Now(),
	}

	engine.ProcessTelemetry(point)

	time.Sleep(100 * time.Millisecond)

	alerts := engine.ListAlerts(AlertFilter{})
	if len(alerts) != 0 {
		t.Errorf("expected 0 alerts, got %d", len(alerts))
	}
}

func TestEngine_ProcessTelemetry_DisabledRule(t *testing.T) {
	cfg := &config.AlertsConfig{
		EvaluationInterval: time.Millisecond * 50,
		MaxActiveAlerts:    100,
	}
	engine := NewEngine(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	engine.Start(ctx)
	defer engine.Stop()

	// Create a disabled rule
	rule := &models.AlertRule{
		Name:     "High Temperature",
		Metric:   "temperature",
		Enabled:  false, // Disabled
		Severity: models.AlertSeverityCritical,
		Condition: models.RuleCondition{
			Operator: "gt",
			Value:    80.0,
		},
	}
	engine.CreateRule(rule)

	point := &models.TelemetryPoint{
		DeviceID:  "device-1",
		Metric:    "temperature",
		Value:     90.0,
		Timestamp: time.Now(),
	}

	engine.ProcessTelemetry(point)

	time.Sleep(100 * time.Millisecond)

	alerts := engine.ListAlerts(AlertFilter{})
	if len(alerts) != 0 {
		t.Errorf("expected 0 alerts for disabled rule, got %d", len(alerts))
	}
}

func TestEngine_ProcessTelemetry_DifferentOperators(t *testing.T) {
	tests := []struct {
		operator string
		value    float64
		expected bool
	}{
		{"gt", 90.0, true},  // 90 > 80
		{"gt", 80.0, false}, // 80 not > 80
		{"gte", 80.0, true}, // 80 >= 80
		{"lt", 70.0, true},  // 70 < 80
		{"lt", 80.0, false}, // 80 not < 80
		{"lte", 80.0, true}, // 80 <= 80
		{"eq", 80.0, true},  // 80 == 80
		{"eq", 81.0, false}, // 81 != 80
		{"neq", 81.0, true}, // 81 != 80
		{"neq", 80.0, false}, // 80 not != 80
	}

	for _, test := range tests {
		t.Run(test.operator, func(t *testing.T) {
			cfg := &config.AlertsConfig{
				EvaluationInterval: time.Millisecond * 50,
				MaxActiveAlerts:    100,
			}
			engine := NewEngine(cfg)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			engine.Start(ctx)
			defer engine.Stop()

			rule := &models.AlertRule{
				Name:     "Test",
				Metric:   "test",
				Enabled:  true,
				Severity: models.AlertSeverityWarning,
				Condition: models.RuleCondition{
					Operator: test.operator,
					Value:    80.0,
				},
			}
			engine.CreateRule(rule)

			point := &models.TelemetryPoint{
				DeviceID:  "device-1",
				Metric:    "test",
				Value:     test.value,
				Timestamp: time.Now(),
			}

			engine.ProcessTelemetry(point)
			time.Sleep(100 * time.Millisecond)

			alerts := engine.ListAlerts(AlertFilter{})
			if test.expected && len(alerts) == 0 {
				t.Errorf("expected alert for %s with value %f", test.operator, test.value)
			}
			if !test.expected && len(alerts) > 0 {
				t.Errorf("did not expect alert for %s with value %f", test.operator, test.value)
			}
		})
	}
}

func TestEngine_ProcessTelemetry_RangeCondition(t *testing.T) {
	cfg := &config.AlertsConfig{
		EvaluationInterval: time.Millisecond * 50,
		MaxActiveAlerts:    100,
	}
	engine := NewEngine(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	engine.Start(ctx)
	defer engine.Stop()

	rule := &models.AlertRule{
		Name:     "Out of Range",
		Metric:   "temperature",
		Enabled:  true,
		Severity: models.AlertSeverityWarning,
		Condition: models.RuleCondition{
			Type:     "range",
			Value:    0.0, // Required due to implementation checking Value before Type
			MinValue: 20.0,
			MaxValue: 80.0,
		},
	}
	engine.CreateRule(rule)

	// Value outside range should trigger
	point := &models.TelemetryPoint{
		DeviceID:  "device-1",
		Metric:    "temperature",
		Value:     90.0, // Above max
		Timestamp: time.Now(),
	}

	engine.ProcessTelemetry(point)
	time.Sleep(100 * time.Millisecond)

	alerts := engine.ListAlerts(AlertFilter{})
	if len(alerts) != 1 {
		t.Errorf("expected 1 alert for out-of-range value, got %d", len(alerts))
	}
}

func TestEngine_ProcessTelemetry_Cooldown(t *testing.T) {
	cfg := &config.AlertsConfig{
		EvaluationInterval: time.Millisecond * 50,
		MaxActiveAlerts:    100,
	}
	engine := NewEngine(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	engine.Start(ctx)
	defer engine.Stop()

	rule := &models.AlertRule{
		Name:     "Test",
		Metric:   "temperature",
		Enabled:  true,
		Severity: models.AlertSeverityCritical,
		Cooldown: time.Hour, // Long cooldown
		Condition: models.RuleCondition{
			Operator: "gt",
			Value:    80.0,
		},
	}
	engine.CreateRule(rule)

	// First point - should trigger
	point1 := &models.TelemetryPoint{
		DeviceID:  "device-1",
		Metric:    "temperature",
		Value:     90.0,
		Timestamp: time.Now(),
	}
	engine.ProcessTelemetry(point1)
	time.Sleep(100 * time.Millisecond)

	// Second point - should NOT trigger due to cooldown
	point2 := &models.TelemetryPoint{
		DeviceID:  "device-1",
		Metric:    "temperature",
		Value:     95.0,
		Timestamp: time.Now(),
	}
	engine.ProcessTelemetry(point2)
	time.Sleep(100 * time.Millisecond)

	alerts := engine.ListAlerts(AlertFilter{})
	if len(alerts) != 1 {
		t.Errorf("expected 1 alert due to cooldown, got %d", len(alerts))
	}
}

func TestEngine_ProcessTelemetry_DeviceSpecificRule(t *testing.T) {
	cfg := &config.AlertsConfig{
		EvaluationInterval: time.Millisecond * 50,
		MaxActiveAlerts:    100,
	}
	engine := NewEngine(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	engine.Start(ctx)
	defer engine.Stop()

	rule := &models.AlertRule{
		Name:      "Device Specific",
		Metric:    "temperature",
		Enabled:   true,
		DeviceIDs: []string{"device-1"}, // Only for device-1
		Severity:  models.AlertSeverityCritical,
		Condition: models.RuleCondition{
			Operator: "gt",
			Value:    80.0,
		},
	}
	engine.CreateRule(rule)

	// Point from device-1 should trigger
	point1 := &models.TelemetryPoint{
		DeviceID:  "device-1",
		Metric:    "temperature",
		Value:     90.0,
		Timestamp: time.Now(),
	}
	engine.ProcessTelemetry(point1)
	time.Sleep(100 * time.Millisecond)

	alerts := engine.ListAlerts(AlertFilter{DeviceID: "device-1"})
	if len(alerts) != 1 {
		t.Errorf("expected 1 alert for device-1, got %d", len(alerts))
	}

	// Point from device-2 should NOT trigger
	point2 := &models.TelemetryPoint{
		DeviceID:  "device-2",
		Metric:    "temperature",
		Value:     90.0,
		Timestamp: time.Now(),
	}
	engine.ProcessTelemetry(point2)
	time.Sleep(100 * time.Millisecond)

	alertsDevice2 := engine.ListAlerts(AlertFilter{DeviceID: "device-2"})
	if len(alertsDevice2) != 0 {
		t.Errorf("expected 0 alerts for device-2, got %d", len(alertsDevice2))
	}
}

func TestEngine_AcknowledgeAlert(t *testing.T) {
	cfg := &config.AlertsConfig{
		EvaluationInterval: time.Millisecond * 50,
		MaxActiveAlerts:    100,
	}
	engine := NewEngine(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	engine.Start(ctx)
	defer engine.Stop()

	// Create rule and trigger alert
	rule := &models.AlertRule{
		Name:     "Test",
		Metric:   "temperature",
		Enabled:  true,
		Severity: models.AlertSeverityCritical,
		Condition: models.RuleCondition{
			Operator: "gt",
			Value:    80.0,
		},
	}
	engine.CreateRule(rule)

	point := &models.TelemetryPoint{
		DeviceID: "device-1",
		Metric:   "temperature",
		Value:    90.0,
	}
	engine.ProcessTelemetry(point)
	time.Sleep(100 * time.Millisecond)

	alerts := engine.ListAlerts(AlertFilter{})
	if len(alerts) == 0 {
		t.Fatal("expected alert to be created")
	}

	alertID := alerts[0].ID

	err := engine.AcknowledgeAlert(alertID, "admin")
	if err != nil {
		t.Fatalf("failed to acknowledge alert: %v", err)
	}

	alert, _ := engine.GetAlert(alertID)
	if alert.Status != models.AlertStatusAcknowledged {
		t.Error("alert should be acknowledged")
	}

	if alert.AckedBy != "admin" {
		t.Error("AckedBy should be set")
	}

	if alert.AckedAt == nil {
		t.Error("AckedAt should be set")
	}
}

func TestEngine_AcknowledgeAlert_NotFound(t *testing.T) {
	cfg := &config.AlertsConfig{}
	engine := NewEngine(cfg)

	err := engine.AcknowledgeAlert("nonexistent", "admin")

	if err != ErrAlertNotFound {
		t.Errorf("expected ErrAlertNotFound, got %v", err)
	}
}

func TestEngine_ResolveAlert(t *testing.T) {
	cfg := &config.AlertsConfig{
		EvaluationInterval: time.Millisecond * 50,
		MaxActiveAlerts:    100,
	}
	engine := NewEngine(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	engine.Start(ctx)
	defer engine.Stop()

	rule := &models.AlertRule{
		Name:     "Test",
		Metric:   "temperature",
		Enabled:  true,
		Severity: models.AlertSeverityCritical,
		Condition: models.RuleCondition{
			Operator: "gt",
			Value:    80.0,
		},
	}
	engine.CreateRule(rule)

	point := &models.TelemetryPoint{
		DeviceID: "device-1",
		Metric:   "temperature",
		Value:    90.0,
	}
	engine.ProcessTelemetry(point)
	time.Sleep(100 * time.Millisecond)

	alerts := engine.ListAlerts(AlertFilter{})
	alertID := alerts[0].ID

	err := engine.ResolveAlert(alertID)
	if err != nil {
		t.Fatalf("failed to resolve alert: %v", err)
	}

	alert, _ := engine.GetAlert(alertID)
	if alert.Status != models.AlertStatusResolved {
		t.Error("alert should be resolved")
	}

	if alert.ResolvedAt == nil {
		t.Error("ResolvedAt should be set")
	}
}

func TestEngine_ResolveAlert_NotFound(t *testing.T) {
	cfg := &config.AlertsConfig{}
	engine := NewEngine(cfg)

	err := engine.ResolveAlert("nonexistent")

	if err != ErrAlertNotFound {
		t.Errorf("expected ErrAlertNotFound, got %v", err)
	}
}

func TestEngine_ListAlerts_Filters(t *testing.T) {
	cfg := &config.AlertsConfig{
		EvaluationInterval: time.Millisecond * 50,
		MaxActiveAlerts:    100,
	}
	engine := NewEngine(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	engine.Start(ctx)
	defer engine.Stop()

	// Create multiple rules with different severities
	engine.CreateRule(&models.AlertRule{
		Name:     "Critical",
		Metric:   "temp",
		Enabled:  true,
		Severity: models.AlertSeverityCritical,
		Condition: models.RuleCondition{
			Operator: "gt",
			Value:    80.0,
		},
	})
	engine.CreateRule(&models.AlertRule{
		Name:     "Warning",
		Metric:   "humidity",
		Enabled:  true,
		Severity: models.AlertSeverityWarning,
		Condition: models.RuleCondition{
			Operator: "gt",
			Value:    80.0,
		},
	})

	// Trigger alerts
	engine.ProcessTelemetry(&models.TelemetryPoint{DeviceID: "device-1", Metric: "temp", Value: 90.0})
	engine.ProcessTelemetry(&models.TelemetryPoint{DeviceID: "device-2", Metric: "humidity", Value: 90.0})
	time.Sleep(150 * time.Millisecond)

	// Filter by device
	alertsDevice1 := engine.ListAlerts(AlertFilter{DeviceID: "device-1"})
	if len(alertsDevice1) != 1 {
		t.Errorf("expected 1 alert for device-1, got %d", len(alertsDevice1))
	}

	// Filter by severity
	criticalAlerts := engine.ListAlerts(AlertFilter{Severity: models.AlertSeverityCritical})
	if len(criticalAlerts) != 1 {
		t.Errorf("expected 1 critical alert, got %d", len(criticalAlerts))
	}

	// Filter by status
	activeAlerts := engine.ListAlerts(AlertFilter{Status: models.AlertStatusActive})
	if len(activeAlerts) != 2 {
		t.Errorf("expected 2 active alerts, got %d", len(activeAlerts))
	}
}

func TestEngine_GetStats(t *testing.T) {
	cfg := &config.AlertsConfig{
		EvaluationInterval: time.Millisecond * 50,
		MaxActiveAlerts:    100,
	}
	engine := NewEngine(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	engine.Start(ctx)
	defer engine.Stop()

	// Create rules
	engine.CreateRule(&models.AlertRule{Name: "Rule 1", Metric: "temp", Enabled: true, Severity: models.AlertSeverityCritical, Condition: models.RuleCondition{Operator: "gt", Value: 80}})
	engine.CreateRule(&models.AlertRule{Name: "Rule 2", Metric: "humidity", Enabled: false})

	// Trigger alert
	engine.ProcessTelemetry(&models.TelemetryPoint{DeviceID: "device-1", Metric: "temp", Value: 90.0})
	time.Sleep(100 * time.Millisecond)

	stats := engine.GetStats()

	if stats.TotalRules != 2 {
		t.Errorf("expected 2 total rules, got %d", stats.TotalRules)
	}

	if stats.EnabledRules != 1 {
		t.Errorf("expected 1 enabled rule, got %d", stats.EnabledRules)
	}

	if stats.TotalAlerts != 1 {
		t.Errorf("expected 1 total alert, got %d", stats.TotalAlerts)
	}

	if stats.ActiveAlerts != 1 {
		t.Errorf("expected 1 active alert, got %d", stats.ActiveAlerts)
	}

	if stats.BySeverity["critical"] != 1 {
		t.Errorf("expected 1 critical alert, got %d", stats.BySeverity["critical"])
	}
}

func TestEngine_ToFloat64(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected float64
		ok       bool
	}{
		{float64(10.5), 10.5, true},
		{float32(10.5), 10.5, true},
		{int(10), 10.0, true},
		{int64(10), 10.0, true},
		{int32(10), 10.0, true},
		{"string", 0, false},
		{nil, 0, false},
	}

	for _, test := range tests {
		result, ok := toFloat64(test.input)
		if ok != test.ok {
			t.Errorf("toFloat64(%v) ok = %v, expected %v", test.input, ok, test.ok)
		}
		if ok && result != test.expected {
			t.Errorf("toFloat64(%v) = %f, expected %f", test.input, result, test.expected)
		}
	}
}

func TestErrRuleNotFound(t *testing.T) {
	if ErrRuleNotFound.Code != "RULE_NOT_FOUND" {
		t.Errorf("expected code 'RULE_NOT_FOUND', got %s", ErrRuleNotFound.Code)
	}

	if ErrRuleNotFound.Error() != "Alert rule not found" {
		t.Error("error message mismatch")
	}
}

func TestErrAlertNotFound(t *testing.T) {
	if ErrAlertNotFound.Code != "ALERT_NOT_FOUND" {
		t.Errorf("expected code 'ALERT_NOT_FOUND', got %s", ErrAlertNotFound.Code)
	}
}
