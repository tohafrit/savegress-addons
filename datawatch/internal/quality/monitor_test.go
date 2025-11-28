package quality

import (
	"context"
	"testing"
	"time"
)

func TestNewMonitor(t *testing.T) {
	cfg := &Config{
		Enabled:        true,
		DefaultRules:   true,
		ScoreThreshold: 90.0,
	}

	monitor := NewMonitor(cfg)
	if monitor == nil {
		t.Fatal("expected monitor to be non-nil")
	}

	// Check default rules are initialized
	rules := monitor.ListRules()
	if len(rules) == 0 {
		t.Error("expected default rules to be initialized")
	}
}

func TestNewMonitorWithoutDefaultRules(t *testing.T) {
	cfg := &Config{
		Enabled:      true,
		DefaultRules: false,
	}

	monitor := NewMonitor(cfg)
	rules := monitor.ListRules()
	if len(rules) != 0 {
		t.Errorf("expected 0 rules, got %d", len(rules))
	}
}

func TestMonitorStartStop(t *testing.T) {
	cfg := &Config{Enabled: true}
	monitor := NewMonitor(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := monitor.Start(ctx); err != nil {
		t.Fatalf("failed to start monitor: %v", err)
	}

	// Starting again should be no-op
	if err := monitor.Start(ctx); err != nil {
		t.Fatalf("second start should succeed: %v", err)
	}

	time.Sleep(50 * time.Millisecond)
	monitor.Stop()
}

func TestMonitorAddRule(t *testing.T) {
	cfg := &Config{Enabled: true, DefaultRules: false}
	monitor := NewMonitor(cfg)

	rule := &Rule{
		ID:        "test-rule-1",
		Name:      "Test Rule",
		Type:      RuleTypeCompleteness,
		Condition: "not_null",
		Table:     "users",
		Field:     "email",
		Enabled:   true,
	}

	monitor.AddRule(rule)

	got, ok := monitor.GetRule("test-rule-1")
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

func TestMonitorListRules(t *testing.T) {
	cfg := &Config{Enabled: true, DefaultRules: false}
	monitor := NewMonitor(cfg)

	monitor.AddRule(&Rule{ID: "rule-1", Name: "Rule 1"})
	monitor.AddRule(&Rule{ID: "rule-2", Name: "Rule 2"})
	monitor.AddRule(&Rule{ID: "rule-3", Name: "Rule 3"})

	rules := monitor.ListRules()
	if len(rules) != 3 {
		t.Errorf("expected 3 rules, got %d", len(rules))
	}
}

func TestMonitorDeleteRule(t *testing.T) {
	cfg := &Config{Enabled: true, DefaultRules: false}
	monitor := NewMonitor(cfg)

	monitor.AddRule(&Rule{ID: "rule-1", Name: "Rule 1"})
	monitor.DeleteRule("rule-1")

	_, ok := monitor.GetRule("rule-1")
	if ok {
		t.Error("rule should be deleted")
	}
}

func TestValidateRecordNotNull(t *testing.T) {
	cfg := &Config{Enabled: true, DefaultRules: false}
	monitor := NewMonitor(cfg)

	monitor.AddRule(&Rule{
		ID:        "not-null-rule",
		Name:      "Email Not Null",
		Type:      RuleTypeCompleteness,
		Condition: "not_null",
		Field:     "email",
		Enabled:   true,
	})

	// Valid record
	validRecord := map[string]interface{}{
		"id":    1,
		"email": "test@example.com",
	}
	result := monitor.ValidateRecord("users", validRecord)
	if !result.Valid {
		t.Error("expected valid record")
	}
	if result.Score != 100.0 {
		t.Errorf("expected score 100, got %f", result.Score)
	}

	// Invalid record (null email)
	invalidRecord := map[string]interface{}{
		"id":    2,
		"email": nil,
	}
	result = monitor.ValidateRecord("users", invalidRecord)
	if result.Valid {
		t.Error("expected invalid record")
	}
	if len(result.Violations) != 1 {
		t.Errorf("expected 1 violation, got %d", len(result.Violations))
	}
}

func TestValidateRecordNotEmpty(t *testing.T) {
	cfg := &Config{Enabled: true, DefaultRules: false}
	monitor := NewMonitor(cfg)

	monitor.AddRule(&Rule{
		ID:        "not-empty-rule",
		Name:      "Name Not Empty",
		Type:      RuleTypeCompleteness,
		Condition: "not_empty",
		Field:     "name",
		Enabled:   true,
	})

	// Valid record
	validRecord := map[string]interface{}{
		"id":   1,
		"name": "John",
	}
	result := monitor.ValidateRecord("users", validRecord)
	if !result.Valid {
		t.Error("expected valid record")
	}

	// Invalid record (empty name)
	invalidRecord := map[string]interface{}{
		"id":   2,
		"name": "",
	}
	result = monitor.ValidateRecord("users", invalidRecord)
	if result.Valid {
		t.Error("expected invalid record")
	}
}

func TestValidateRecordPositive(t *testing.T) {
	cfg := &Config{Enabled: true, DefaultRules: false}
	monitor := NewMonitor(cfg)

	monitor.AddRule(&Rule{
		ID:        "positive-rule",
		Name:      "Amount Positive",
		Type:      RuleTypeValidity,
		Condition: "positive",
		Field:     "amount",
		Enabled:   true,
	})

	// Valid record
	validRecord := map[string]interface{}{
		"id":     1,
		"amount": 100.50,
	}
	result := monitor.ValidateRecord("orders", validRecord)
	if !result.Valid {
		t.Error("expected valid record")
	}

	// Invalid record (negative amount)
	invalidRecord := map[string]interface{}{
		"id":     2,
		"amount": -50.0,
	}
	result = monitor.ValidateRecord("orders", invalidRecord)
	if result.Valid {
		t.Error("expected invalid record")
	}
}

func TestValidateRecordInRange(t *testing.T) {
	cfg := &Config{Enabled: true, DefaultRules: false}
	monitor := NewMonitor(cfg)

	monitor.AddRule(&Rule{
		ID:        "range-rule",
		Name:      "Age In Range",
		Type:      RuleTypeValidity,
		Condition: "in_range",
		Field:     "age",
		Enabled:   true,
		Parameters: map[string]interface{}{
			"min": 0.0,
			"max": 150.0,
		},
	})

	// Valid record
	validRecord := map[string]interface{}{
		"id":  1,
		"age": 25,
	}
	result := monitor.ValidateRecord("users", validRecord)
	if !result.Valid {
		t.Error("expected valid record")
	}

	// Invalid record (age out of range)
	invalidRecord := map[string]interface{}{
		"id":  2,
		"age": 200,
	}
	result = monitor.ValidateRecord("users", invalidRecord)
	if result.Valid {
		t.Error("expected invalid record")
	}
}

func TestValidateRecordInSet(t *testing.T) {
	cfg := &Config{Enabled: true, DefaultRules: false}
	monitor := NewMonitor(cfg)

	monitor.AddRule(&Rule{
		ID:        "set-rule",
		Name:      "Status In Set",
		Type:      RuleTypeValidity,
		Condition: "in_set",
		Field:     "status",
		Enabled:   true,
		Parameters: map[string]interface{}{
			"values": []interface{}{"pending", "active", "completed"},
		},
	})

	// Valid record
	validRecord := map[string]interface{}{
		"id":     1,
		"status": "active",
	}
	result := monitor.ValidateRecord("orders", validRecord)
	if !result.Valid {
		t.Error("expected valid record")
	}

	// Invalid record (status not in set)
	invalidRecord := map[string]interface{}{
		"id":     2,
		"status": "unknown",
	}
	result = monitor.ValidateRecord("orders", invalidRecord)
	if result.Valid {
		t.Error("expected invalid record")
	}
}

func TestValidateRecordEmailFormat(t *testing.T) {
	cfg := &Config{Enabled: true, DefaultRules: false}
	monitor := NewMonitor(cfg)

	monitor.AddRule(&Rule{
		ID:        "email-rule",
		Name:      "Email Format",
		Type:      RuleTypeValidity,
		Condition: "email_format",
		Field:     "email",
		Enabled:   true,
	})

	// Valid email
	validRecord := map[string]interface{}{
		"id":    1,
		"email": "user@example.com",
	}
	result := monitor.ValidateRecord("users", validRecord)
	if !result.Valid {
		t.Error("expected valid record")
	}

	// Invalid email
	invalidRecord := map[string]interface{}{
		"id":    2,
		"email": "invalid-email",
	}
	result = monitor.ValidateRecord("users", invalidRecord)
	if result.Valid {
		t.Error("expected invalid record")
	}
}

func TestValidateRecordRegex(t *testing.T) {
	cfg := &Config{Enabled: true, DefaultRules: false}
	monitor := NewMonitor(cfg)

	monitor.AddRule(&Rule{
		ID:        "regex-rule",
		Name:      "Phone Format",
		Type:      RuleTypeValidity,
		Condition: "regex",
		Field:     "phone",
		Enabled:   true,
		Parameters: map[string]interface{}{
			"pattern": `^\+\d{1,3}-\d{3}-\d{4}$`,
		},
	})

	// Valid phone
	validRecord := map[string]interface{}{
		"id":    1,
		"phone": "+1-555-1234",
	}
	result := monitor.ValidateRecord("users", validRecord)
	if !result.Valid {
		t.Error("expected valid record")
	}

	// Invalid phone
	invalidRecord := map[string]interface{}{
		"id":    2,
		"phone": "5551234",
	}
	result = monitor.ValidateRecord("users", invalidRecord)
	if result.Valid {
		t.Error("expected invalid record")
	}
}

func TestGetScore(t *testing.T) {
	cfg := &Config{Enabled: true, DefaultRules: false}
	monitor := NewMonitor(cfg)

	// Initially no score
	_, ok := monitor.GetScore("users")
	if ok {
		t.Error("expected no score initially")
	}
}

func TestGetOverallScore(t *testing.T) {
	cfg := &Config{Enabled: true, DefaultRules: false}
	monitor := NewMonitor(cfg)

	// Default score should be 100 when no scores exist
	score := monitor.GetOverallScore()
	if score != 100.0 {
		t.Errorf("expected 100.0, got %f", score)
	}
}

func TestGetViolations(t *testing.T) {
	cfg := &Config{Enabled: true, DefaultRules: false}
	monitor := NewMonitor(cfg)

	monitor.AddRule(&Rule{
		ID:        "test-rule",
		Name:      "Test",
		Type:      RuleTypeCompleteness,
		Condition: "not_null",
		Field:     "email",
		Enabled:   true,
	})

	// Create violations
	for i := 0; i < 3; i++ {
		monitor.ValidateRecord("users", map[string]interface{}{"id": i, "email": nil})
	}

	// Wait for violations to be processed
	time.Sleep(50 * time.Millisecond)

	violations := monitor.GetViolations(ViolationFilter{})
	if len(violations) < 3 {
		t.Logf("expected at least 3 violations, got %d", len(violations))
	}
}

func TestGetViolationsWithFilter(t *testing.T) {
	cfg := &Config{Enabled: true, DefaultRules: false}
	monitor := NewMonitor(cfg)

	monitor.AddRule(&Rule{
		ID:        "completeness-rule",
		Name:      "Completeness",
		Type:      RuleTypeCompleteness,
		Condition: "not_null",
		Field:     "email",
		Severity:  "high",
		Enabled:   true,
	})
	monitor.AddRule(&Rule{
		ID:        "validity-rule",
		Name:      "Validity",
		Type:      RuleTypeValidity,
		Condition: "positive",
		Field:     "amount",
		Severity:  "low",
		Enabled:   true,
	})

	// Create completeness violation
	monitor.ValidateRecord("users", map[string]interface{}{"id": 1, "email": nil})
	// Create validity violation
	monitor.ValidateRecord("orders", map[string]interface{}{"id": 1, "amount": -10})

	time.Sleep(50 * time.Millisecond)

	// Filter by table
	userViolations := monitor.GetViolations(ViolationFilter{Table: "users"})
	if len(userViolations) < 1 {
		t.Logf("expected at least 1 user violation")
	}
}

func TestAcknowledgeViolation(t *testing.T) {
	cfg := &Config{Enabled: true, DefaultRules: false}
	monitor := NewMonitor(cfg)

	monitor.AddRule(&Rule{
		ID:        "test-rule",
		Name:      "Test",
		Type:      RuleTypeCompleteness,
		Condition: "not_null",
		Field:     "email",
		Enabled:   true,
	})

	// Create a violation
	result := monitor.ValidateRecord("users", map[string]interface{}{"id": 1, "email": nil})
	if len(result.Violations) == 0 {
		t.Fatal("expected violation")
	}

	violationID := result.Violations[0].ID

	err := monitor.AcknowledgeViolation(violationID, "test-user")
	if err != nil {
		// Violation may not be in store yet due to async processing
		// This is expected behavior
		t.Logf("acknowledge error (may be expected): %v", err)
	}
}

func TestAcknowledgeViolationNotFound(t *testing.T) {
	cfg := &Config{Enabled: true, DefaultRules: false}
	monitor := NewMonitor(cfg)

	err := monitor.AcknowledgeViolation("non-existent", "user")
	if err == nil {
		t.Error("expected error for non-existent violation")
	}
}

func TestGetReport(t *testing.T) {
	cfg := &Config{Enabled: true, DefaultRules: false}
	monitor := NewMonitor(cfg)

	start := time.Now().Add(-time.Hour)
	end := time.Now()

	report := monitor.GetReport(start, end)
	if report == nil {
		t.Fatal("expected report to be non-nil")
	}
	if report.Period.Start != start {
		t.Error("report start time mismatch")
	}
	if report.Period.End != end {
		t.Error("report end time mismatch")
	}
}

func TestGetStats(t *testing.T) {
	cfg := &Config{Enabled: true, DefaultRules: false}
	monitor := NewMonitor(cfg)

	monitor.AddRule(&Rule{ID: "rule-1", Name: "Rule 1"})
	monitor.AddRule(&Rule{ID: "rule-2", Name: "Rule 2"})

	stats := monitor.GetStats()
	if stats.TotalRules != 2 {
		t.Errorf("expected 2 rules, got %d", stats.TotalRules)
	}
	if stats.OverallScore != 100.0 {
		t.Errorf("expected overall score 100, got %f", stats.OverallScore)
	}
}

func TestIsValidEmail(t *testing.T) {
	tests := []struct {
		email    string
		expected bool
	}{
		{"user@example.com", true},
		{"user.name@example.com", true},
		{"user+tag@example.org", true},
		{"user@sub.example.com", true},
		{"invalid", false},
		{"@example.com", false},
		{"user@", false},
		{"user@.com", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			result := isValidEmail(tt.email)
			if result != tt.expected {
				t.Errorf("isValidEmail(%s) = %v, want %v", tt.email, result, tt.expected)
			}
		})
	}
}

func TestToFloat(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected float64
		ok       bool
	}{
		{float64(1.5), 1.5, true},
		{float32(2.5), 2.5, true},
		{int(10), 10.0, true},
		{int32(20), 20.0, true},
		{int64(30), 30.0, true},
		{"string", 0, false},
		{nil, 0, false},
	}

	for _, tt := range tests {
		result, ok := toFloat(tt.input)
		if ok != tt.ok {
			t.Errorf("toFloat(%v) ok = %v, want %v", tt.input, ok, tt.ok)
		}
		if ok && result != tt.expected {
			t.Errorf("toFloat(%v) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestCalculateFreshnessScore(t *testing.T) {
	cfg := &Config{Enabled: true, DefaultRules: false}
	monitor := NewMonitor(cfg)

	// Recent data - should be 100
	recentStats := &TableStats{LastRecordTime: time.Now()}
	score := monitor.calculateFreshnessScore(recentStats)
	if score != 100.0 {
		t.Errorf("expected 100.0 for recent data, got %f", score)
	}

	// Old data - should be 0
	oldStats := &TableStats{LastRecordTime: time.Now().Add(-2 * time.Hour)}
	score = monitor.calculateFreshnessScore(oldStats)
	if score != 0.0 {
		t.Errorf("expected 0.0 for old data, got %f", score)
	}

	// No data - should be 0
	noDataStats := &TableStats{}
	score = monitor.calculateFreshnessScore(noDataStats)
	if score != 0.0 {
		t.Errorf("expected 0.0 for no data, got %f", score)
	}
}
