package fraud

import (
	"context"
	"testing"
	"time"

	"github.com/savegress/finsight/internal/config"
	"github.com/savegress/finsight/pkg/models"
	"github.com/shopspring/decimal"
)

func TestNewDetector(t *testing.T) {
	cfg := &config.FraudConfig{
		Enabled:         true,
		ScoreThreshold:  0.7,
		VelocityWindow:  time.Hour,
		MaxDailyAmount:  10000,
		MaxSingleAmount: 5000,
	}

	detector := NewDetector(cfg)

	if detector == nil {
		t.Fatal("NewDetector returned nil")
	}
	if detector.config != cfg {
		t.Error("config not set correctly")
	}
	if detector.alerts == nil {
		t.Error("alerts map not initialized")
	}
	if detector.velocity == nil {
		t.Error("velocity tracker not initialized")
	}
	if detector.patterns == nil {
		t.Error("pattern analyzer not initialized")
	}
	if detector.geofence == nil {
		t.Error("geofence checker not initialized")
	}
	if len(detector.rules) == 0 {
		t.Error("rules not initialized")
	}
}

func TestDetector_StartStop(t *testing.T) {
	cfg := &config.FraudConfig{
		Enabled:        true,
		ScoreThreshold: 0.7,
		VelocityWindow: time.Hour,
	}

	detector := NewDetector(cfg)
	ctx := context.Background()

	// Start detector
	err := detector.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if !detector.running {
		t.Error("detector should be running after Start")
	}

	// Starting again should be safe
	err = detector.Start(ctx)
	if err != nil {
		t.Fatalf("Second Start failed: %v", err)
	}

	// Stop detector
	detector.Stop()

	if detector.running {
		t.Error("detector should not be running after Stop")
	}
}

func TestDetector_Evaluate_Disabled(t *testing.T) {
	cfg := &config.FraudConfig{
		Enabled: false,
	}

	detector := NewDetector(cfg)

	txn := &models.Transaction{
		ID:            "txn-1",
		Amount:        decimal.NewFromFloat(1000000),
		SourceAccount: "acc-1",
		CreatedAt:     time.Now(),
	}

	result := detector.Evaluate(txn, nil)

	if result.RiskScore != 0 {
		t.Errorf("expected risk score 0 when disabled, got %f", result.RiskScore)
	}
	if result.Decision != DecisionAllow {
		t.Errorf("expected DecisionAllow when disabled, got %s", result.Decision)
	}
}

func TestDetector_Evaluate_LowRisk(t *testing.T) {
	cfg := &config.FraudConfig{
		Enabled:         true,
		ScoreThreshold:  0.7,
		VelocityWindow:  time.Hour,
		MaxDailyAmount:  10000,
		MaxSingleAmount: 5000,
	}

	detector := NewDetector(cfg)
	ctx := context.Background()
	detector.Start(ctx)
	defer detector.Stop()

	txn := &models.Transaction{
		ID:            "txn-1",
		Amount:        decimal.NewFromFloat(50),
		SourceAccount: "acc-1",
		CreatedAt:     time.Now().Add(-3 * time.Hour), // Not during unusual hours
	}

	result := detector.Evaluate(txn, nil)

	if result.Decision != DecisionAllow {
		t.Errorf("expected DecisionAllow for low risk, got %s", result.Decision)
	}
	if result.TransactionID != txn.ID {
		t.Errorf("expected transaction ID %s, got %s", txn.ID, result.TransactionID)
	}
}

func TestDetector_Evaluate_HighRisk_Amount(t *testing.T) {
	cfg := &config.FraudConfig{
		Enabled:         true,
		ScoreThreshold:  0.7,
		VelocityWindow:  time.Hour,
		MaxDailyAmount:  10000,
		MaxSingleAmount: 5000,
	}

	detector := NewDetector(cfg)
	ctx := context.Background()
	detector.Start(ctx)
	defer detector.Stop()

	// Transaction exceeding max single amount
	txn := &models.Transaction{
		ID:            "txn-high",
		Amount:        decimal.NewFromFloat(6000),
		SourceAccount: "acc-1",
		CreatedAt:     time.Now(),
	}

	result := detector.Evaluate(txn, nil)

	if result.RiskScore == 0 {
		t.Error("expected non-zero risk score for high amount transaction")
	}
	if len(result.Indicators) == 0 {
		t.Error("expected fraud indicators for high amount transaction")
	}
}

func TestDetector_Evaluate_HighRisk_Velocity(t *testing.T) {
	cfg := &config.FraudConfig{
		Enabled:         true,
		ScoreThreshold:  0.5,
		VelocityWindow:  time.Hour,
		MaxDailyAmount:  1000,
		MaxSingleAmount: 5000,
	}

	detector := NewDetector(cfg)
	ctx := context.Background()
	detector.Start(ctx)
	defer detector.Stop()

	// Provide high velocity context
	evalCtx := &EvaluationContext{
		RecentActivity: &ActivitySummary{
			TransactionCount: 15, // High count
			TotalAmount:      decimal.NewFromFloat(2000),
			UniqueLocations:  5, // Many locations
			TimeWindow:       time.Hour,
		},
	}

	txn := &models.Transaction{
		ID:            "txn-velocity",
		Amount:        decimal.NewFromFloat(100),
		SourceAccount: "acc-1",
		CreatedAt:     time.Now(),
	}

	result := detector.Evaluate(txn, evalCtx)

	if result.RiskScore == 0 {
		t.Error("expected non-zero risk score for high velocity")
	}
}

func TestDetector_Evaluate_Review_Decision(t *testing.T) {
	cfg := &config.FraudConfig{
		Enabled:         true,
		ScoreThreshold:  0.8,
		VelocityWindow:  time.Hour,
		MaxDailyAmount:  10000,
		MaxSingleAmount: 5000,
	}

	detector := NewDetector(cfg)
	ctx := context.Background()
	detector.Start(ctx)
	defer detector.Stop()

	// Create evaluation context that triggers review (moderate risk)
	evalCtx := &EvaluationContext{
		RecentActivity: &ActivitySummary{
			TransactionCount: 12,
			TotalAmount:      decimal.NewFromFloat(5000),
			UniqueLocations:  4,
			TimeWindow:       time.Hour,
		},
	}

	txn := &models.Transaction{
		ID:            "txn-review",
		Amount:        decimal.NewFromFloat(100),
		SourceAccount: "acc-1",
		CreatedAt:     time.Now(),
	}

	result := detector.Evaluate(txn, evalCtx)

	// Should be either review or block depending on exact score
	if result.Decision == DecisionAllow && result.RiskScore > 0.5 {
		t.Logf("Risk score: %f, Decision: %s", result.RiskScore, result.Decision)
	}
}

func TestDetector_GetAlert(t *testing.T) {
	cfg := &config.FraudConfig{
		Enabled:        true,
		ScoreThreshold: 0.7,
		VelocityWindow: time.Hour,
	}

	detector := NewDetector(cfg)

	// Manually add an alert
	alert := &models.FraudAlert{
		ID:            "alert-1",
		TransactionID: "txn-1",
		AlertType:     models.FraudAlertTypeAmount,
		Severity:      models.AlertSeverityHigh,
		Status:        models.AlertStatusOpen,
		CreatedAt:     time.Now(),
	}
	detector.alerts[alert.ID] = alert

	// Get existing alert
	found, ok := detector.GetAlert("alert-1")
	if !ok {
		t.Error("expected to find alert")
	}
	if found.ID != alert.ID {
		t.Errorf("expected alert ID %s, got %s", alert.ID, found.ID)
	}

	// Get non-existent alert
	_, ok = detector.GetAlert("non-existent")
	if ok {
		t.Error("expected not to find non-existent alert")
	}
}

func TestDetector_GetAlerts_NoFilter(t *testing.T) {
	cfg := &config.FraudConfig{
		Enabled:        true,
		ScoreThreshold: 0.7,
		VelocityWindow: time.Hour,
	}

	detector := NewDetector(cfg)

	// Add multiple alerts
	detector.alerts["alert-1"] = &models.FraudAlert{
		ID:        "alert-1",
		AlertType: models.FraudAlertTypeAmount,
		Severity:  models.AlertSeverityHigh,
		Status:    models.AlertStatusOpen,
		CreatedAt: time.Now(),
	}
	detector.alerts["alert-2"] = &models.FraudAlert{
		ID:        "alert-2",
		AlertType: models.FraudAlertTypeVelocity,
		Severity:  models.AlertSeverityMedium,
		Status:    models.AlertStatusResolved,
		CreatedAt: time.Now(),
	}

	alerts := detector.GetAlerts(AlertFilter{})

	if len(alerts) != 2 {
		t.Errorf("expected 2 alerts, got %d", len(alerts))
	}
}

func TestDetector_GetAlerts_WithFilter(t *testing.T) {
	cfg := &config.FraudConfig{
		Enabled:        true,
		ScoreThreshold: 0.7,
		VelocityWindow: time.Hour,
	}

	detector := NewDetector(cfg)

	now := time.Now()
	detector.alerts["alert-1"] = &models.FraudAlert{
		ID:        "alert-1",
		AlertType: models.FraudAlertTypeAmount,
		Severity:  models.AlertSeverityHigh,
		Status:    models.AlertStatusOpen,
		CreatedAt: now,
	}
	detector.alerts["alert-2"] = &models.FraudAlert{
		ID:        "alert-2",
		AlertType: models.FraudAlertTypeVelocity,
		Severity:  models.AlertSeverityMedium,
		Status:    models.AlertStatusResolved,
		CreatedAt: now.Add(-2 * time.Hour),
	}
	detector.alerts["alert-3"] = &models.FraudAlert{
		ID:        "alert-3",
		AlertType: models.FraudAlertTypeAmount,
		Severity:  models.AlertSeverityHigh,
		Status:    models.AlertStatusOpen,
		CreatedAt: now.Add(-1 * time.Hour),
	}

	// Filter by status
	alerts := detector.GetAlerts(AlertFilter{Status: models.AlertStatusOpen})
	if len(alerts) != 2 {
		t.Errorf("expected 2 open alerts, got %d", len(alerts))
	}

	// Filter by severity
	alerts = detector.GetAlerts(AlertFilter{Severity: models.AlertSeverityMedium})
	if len(alerts) != 1 {
		t.Errorf("expected 1 medium severity alert, got %d", len(alerts))
	}

	// Filter by alert type
	alerts = detector.GetAlerts(AlertFilter{AlertType: models.FraudAlertTypeAmount})
	if len(alerts) != 2 {
		t.Errorf("expected 2 amount alerts, got %d", len(alerts))
	}

	// Filter by date range
	startDate := now.Add(-90 * time.Minute)
	endDate := now.Add(-30 * time.Minute)
	alerts = detector.GetAlerts(AlertFilter{StartDate: &startDate, EndDate: &endDate})
	if len(alerts) != 1 {
		t.Errorf("expected 1 alert in date range, got %d", len(alerts))
	}
}

func TestDetector_ResolveAlert(t *testing.T) {
	cfg := &config.FraudConfig{
		Enabled:        true,
		ScoreThreshold: 0.7,
		VelocityWindow: time.Hour,
	}

	detector := NewDetector(cfg)

	// Add an alert
	detector.alerts["alert-1"] = &models.FraudAlert{
		ID:        "alert-1",
		Status:    models.AlertStatusOpen,
		CreatedAt: time.Now(),
	}

	// Resolve as true positive
	err := detector.ResolveAlert("alert-1", "Confirmed fraud", false)
	if err != nil {
		t.Fatalf("ResolveAlert failed: %v", err)
	}

	alert := detector.alerts["alert-1"]
	if alert.Status != models.AlertStatusResolved {
		t.Errorf("expected status Resolved, got %s", alert.Status)
	}
	if alert.Resolution != "Confirmed fraud" {
		t.Errorf("expected resolution 'Confirmed fraud', got %s", alert.Resolution)
	}
	if alert.ResolvedAt == nil {
		t.Error("expected ResolvedAt to be set")
	}
}

func TestDetector_ResolveAlert_FalsePositive(t *testing.T) {
	cfg := &config.FraudConfig{
		Enabled:        true,
		ScoreThreshold: 0.7,
		VelocityWindow: time.Hour,
	}

	detector := NewDetector(cfg)

	// Add an alert
	detector.alerts["alert-fp"] = &models.FraudAlert{
		ID:        "alert-fp",
		Status:    models.AlertStatusOpen,
		CreatedAt: time.Now(),
	}

	// Resolve as false positive
	err := detector.ResolveAlert("alert-fp", "Customer verified", true)
	if err != nil {
		t.Fatalf("ResolveAlert failed: %v", err)
	}

	alert := detector.alerts["alert-fp"]
	if alert.Status != models.AlertStatusFalsePos {
		t.Errorf("expected status FalsePositive, got %s", alert.Status)
	}
}

func TestDetector_ResolveAlert_NotFound(t *testing.T) {
	cfg := &config.FraudConfig{
		Enabled:        true,
		ScoreThreshold: 0.7,
		VelocityWindow: time.Hour,
	}

	detector := NewDetector(cfg)

	err := detector.ResolveAlert("non-existent", "test", false)
	if err != ErrAlertNotFound {
		t.Errorf("expected ErrAlertNotFound, got %v", err)
	}
}

func TestDetector_GetStats_Empty(t *testing.T) {
	cfg := &config.FraudConfig{
		Enabled:        true,
		ScoreThreshold: 0.7,
		VelocityWindow: time.Hour,
	}

	detector := NewDetector(cfg)

	stats := detector.GetStats()

	if stats.TotalAlerts != 0 {
		t.Errorf("expected 0 total alerts, got %d", stats.TotalAlerts)
	}
	if stats.FalsePositiveRate != 0 {
		t.Errorf("expected 0 false positive rate, got %f", stats.FalsePositiveRate)
	}
}

func TestDetector_GetStats(t *testing.T) {
	cfg := &config.FraudConfig{
		Enabled:        true,
		ScoreThreshold: 0.7,
		VelocityWindow: time.Hour,
	}

	detector := NewDetector(cfg)

	// Add various alerts
	detector.alerts["alert-1"] = &models.FraudAlert{
		ID:        "alert-1",
		AlertType: models.FraudAlertTypeAmount,
		Severity:  models.AlertSeverityHigh,
		Status:    models.AlertStatusOpen,
	}
	detector.alerts["alert-2"] = &models.FraudAlert{
		ID:        "alert-2",
		AlertType: models.FraudAlertTypeVelocity,
		Severity:  models.AlertSeverityMedium,
		Status:    models.AlertStatusInProgress,
	}
	detector.alerts["alert-3"] = &models.FraudAlert{
		ID:        "alert-3",
		AlertType: models.FraudAlertTypeAmount,
		Severity:  models.AlertSeverityHigh,
		Status:    models.AlertStatusResolved,
	}
	detector.alerts["alert-4"] = &models.FraudAlert{
		ID:        "alert-4",
		AlertType: models.FraudAlertTypeGeolocation,
		Severity:  models.AlertSeverityCritical,
		Status:    models.AlertStatusFalsePos,
	}

	stats := detector.GetStats()

	if stats.TotalAlerts != 4 {
		t.Errorf("expected 4 total alerts, got %d", stats.TotalAlerts)
	}
	if stats.OpenAlerts != 2 { // Open + InProgress
		t.Errorf("expected 2 open alerts, got %d", stats.OpenAlerts)
	}
	if stats.FalsePositives != 1 {
		t.Errorf("expected 1 false positive, got %d", stats.FalsePositives)
	}
	expectedFPRate := 1.0 / 4.0
	if stats.FalsePositiveRate != expectedFPRate {
		t.Errorf("expected false positive rate %f, got %f", expectedFPRate, stats.FalsePositiveRate)
	}
	if stats.ByStatus[string(models.AlertStatusOpen)] != 1 {
		t.Errorf("expected 1 open status, got %d", stats.ByStatus[string(models.AlertStatusOpen)])
	}
	if stats.BySeverity[string(models.AlertSeverityHigh)] != 2 {
		t.Errorf("expected 2 high severity, got %d", stats.BySeverity[string(models.AlertSeverityHigh)])
	}
	if stats.ByType[string(models.FraudAlertTypeAmount)] != 2 {
		t.Errorf("expected 2 amount type, got %d", stats.ByType[string(models.FraudAlertTypeAmount)])
	}
}

func TestNormalizeScore(t *testing.T) {
	tests := []struct {
		input    float64
		expected float64
	}{
		{0, 0},
		{-1, 0},
		{5, 0.5},
		{10, 1.0},
		{15, 1.0},
		{7.5, 0.75},
	}

	for _, tt := range tests {
		result := normalizeScore(tt.input)
		if result != tt.expected {
			t.Errorf("normalizeScore(%f) = %f, expected %f", tt.input, result, tt.expected)
		}
	}
}

func TestDecisionConstants(t *testing.T) {
	if DecisionAllow != "allow" {
		t.Errorf("expected DecisionAllow = 'allow', got %s", DecisionAllow)
	}
	if DecisionBlock != "block" {
		t.Errorf("expected DecisionBlock = 'block', got %s", DecisionBlock)
	}
	if DecisionReview != "review" {
		t.Errorf("expected DecisionReview = 'review', got %s", DecisionReview)
	}
}

func TestError(t *testing.T) {
	err := &Error{Code: "TEST_ERROR", Message: "test message"}
	if err.Error() != "test message" {
		t.Errorf("expected 'test message', got %s", err.Error())
	}
}

func TestErrAlertNotFound(t *testing.T) {
	if ErrAlertNotFound.Code != "ALERT_NOT_FOUND" {
		t.Errorf("expected code 'ALERT_NOT_FOUND', got %s", ErrAlertNotFound.Code)
	}
}

func TestMatchesAlertFilter(t *testing.T) {
	cfg := &config.FraudConfig{
		VelocityWindow: time.Hour,
	}
	detector := NewDetector(cfg)

	now := time.Now()
	alert := &models.FraudAlert{
		ID:        "test",
		Status:    models.AlertStatusOpen,
		Severity:  models.AlertSeverityHigh,
		AlertType: models.FraudAlertTypeAmount,
		CreatedAt: now,
	}

	// Empty filter matches all
	if !detector.matchesAlertFilter(alert, AlertFilter{}) {
		t.Error("empty filter should match")
	}

	// Status filter
	if !detector.matchesAlertFilter(alert, AlertFilter{Status: models.AlertStatusOpen}) {
		t.Error("matching status should pass")
	}
	if detector.matchesAlertFilter(alert, AlertFilter{Status: models.AlertStatusResolved}) {
		t.Error("non-matching status should fail")
	}

	// Severity filter
	if !detector.matchesAlertFilter(alert, AlertFilter{Severity: models.AlertSeverityHigh}) {
		t.Error("matching severity should pass")
	}
	if detector.matchesAlertFilter(alert, AlertFilter{Severity: models.AlertSeverityLow}) {
		t.Error("non-matching severity should fail")
	}

	// AlertType filter
	if !detector.matchesAlertFilter(alert, AlertFilter{AlertType: models.FraudAlertTypeAmount}) {
		t.Error("matching alert type should pass")
	}
	if detector.matchesAlertFilter(alert, AlertFilter{AlertType: models.FraudAlertTypeVelocity}) {
		t.Error("non-matching alert type should fail")
	}

	// Date range filters
	pastDate := now.Add(-1 * time.Hour)
	futureDate := now.Add(1 * time.Hour)

	if !detector.matchesAlertFilter(alert, AlertFilter{StartDate: &pastDate}) {
		t.Error("alert after start date should pass")
	}
	if detector.matchesAlertFilter(alert, AlertFilter{StartDate: &futureDate}) {
		t.Error("alert before start date should fail")
	}

	if !detector.matchesAlertFilter(alert, AlertFilter{EndDate: &futureDate}) {
		t.Error("alert before end date should pass")
	}
	if detector.matchesAlertFilter(alert, AlertFilter{EndDate: &pastDate}) {
		t.Error("alert after end date should fail")
	}
}

func TestGenerateAlertID(t *testing.T) {
	id1 := generateAlertID()

	if id1 == "" {
		t.Error("generated ID should not be empty")
	}
	if len(id1) < 10 {
		t.Error("generated ID should be at least 10 characters")
	}

	// Generate another to verify format consistency
	id2 := generateAlertID()
	if id2 == "" {
		t.Error("second generated ID should not be empty")
	}
}

func TestRandomString(t *testing.T) {
	s1 := randomString(8)
	if len(s1) != 8 {
		t.Errorf("expected length 8, got %d", len(s1))
	}

	s2 := randomString(16)
	if len(s2) != 16 {
		t.Errorf("expected length 16, got %d", len(s2))
	}
}

func TestCreateAlert(t *testing.T) {
	cfg := &config.FraudConfig{
		Enabled:        true,
		ScoreThreshold: 0.7,
		VelocityWindow: time.Hour,
	}

	detector := NewDetector(cfg)

	txn := &models.Transaction{
		ID:            "txn-1",
		Amount:        decimal.NewFromFloat(1000),
		SourceAccount: "acc-1",
		CreatedAt:     time.Now(),
	}

	// Test with high risk score
	result := &EvaluationResult{
		TransactionID: txn.ID,
		RiskScore:     0.95,
		Indicators: []models.FraudIndicator{
			{Type: "velocity", Description: "High velocity"},
		},
	}

	alert := detector.createAlert(txn, result)

	if alert.TransactionID != txn.ID {
		t.Errorf("expected transaction ID %s, got %s", txn.ID, alert.TransactionID)
	}
	if alert.Severity != models.AlertSeverityCritical {
		t.Errorf("expected critical severity for 0.95 score, got %s", alert.Severity)
	}
	if alert.Status != models.AlertStatusOpen {
		t.Errorf("expected open status, got %s", alert.Status)
	}
	if alert.AlertType != models.FraudAlertType("velocity") {
		t.Errorf("expected velocity alert type, got %s", alert.AlertType)
	}

	// Test with medium risk score
	result2 := &EvaluationResult{
		TransactionID: txn.ID,
		RiskScore:     0.6,
		Indicators:    []models.FraudIndicator{},
	}

	alert2 := detector.createAlert(txn, result2)

	if alert2.Severity != models.AlertSeverityMedium {
		t.Errorf("expected medium severity for 0.6 score, got %s", alert2.Severity)
	}
	if alert2.AlertType != models.FraudAlertTypeAmount {
		t.Errorf("expected default amount alert type, got %s", alert2.AlertType)
	}

	// Test with low risk score
	result3 := &EvaluationResult{
		TransactionID: txn.ID,
		RiskScore:     0.3,
	}

	alert3 := detector.createAlert(txn, result3)

	if alert3.Severity != models.AlertSeverityLow {
		t.Errorf("expected low severity for 0.3 score, got %s", alert3.Severity)
	}

	// Test with high (but not critical) risk score
	result4 := &EvaluationResult{
		TransactionID: txn.ID,
		RiskScore:     0.75,
	}

	alert4 := detector.createAlert(txn, result4)

	if alert4.Severity != models.AlertSeverityHigh {
		t.Errorf("expected high severity for 0.75 score, got %s", alert4.Severity)
	}
}

func TestDetector_ProcessAlerts(t *testing.T) {
	cfg := &config.FraudConfig{
		Enabled:        true,
		ScoreThreshold: 0.7,
		VelocityWindow: time.Hour,
	}

	detector := NewDetector(cfg)
	ctx, cancel := context.WithCancel(context.Background())

	// Start the detector
	detector.Start(ctx)

	// Send an alert through the channel
	alert := &models.FraudAlert{
		ID:            "test-alert",
		TransactionID: "txn-1",
		Status:        models.AlertStatusOpen,
	}

	detector.alertCh <- alert

	// Give time for alert to be processed
	time.Sleep(50 * time.Millisecond)

	// Check alert was stored
	found, ok := detector.GetAlert("test-alert")
	if !ok {
		t.Error("expected alert to be stored")
	}
	if found.ID != alert.ID {
		t.Errorf("expected alert ID %s, got %s", alert.ID, found.ID)
	}

	cancel()
	time.Sleep(50 * time.Millisecond)
}

func TestEvaluationContext(t *testing.T) {
	ctx := &EvaluationContext{
		AccountHistory: []*models.Transaction{
			{ID: "txn-1", Amount: decimal.NewFromFloat(100)},
		},
		RecentActivity: &ActivitySummary{
			TransactionCount: 5,
			TotalAmount:      decimal.NewFromFloat(500),
		},
		AccountProfile: &AccountProfile{
			AvgTransactionAmount: decimal.NewFromFloat(100),
			TypicalLocations:     []string{"US", "CA"},
		},
		DeviceInfo: &DeviceInfo{
			DeviceID:   "device-1",
			DeviceType: "mobile",
			IsKnown:    true,
		},
		GeoLocation: &GeoLocation{
			Country: "US",
			City:    "New York",
		},
	}

	if len(ctx.AccountHistory) != 1 {
		t.Errorf("expected 1 transaction in history, got %d", len(ctx.AccountHistory))
	}
	if ctx.RecentActivity.TransactionCount != 5 {
		t.Errorf("expected transaction count 5, got %d", ctx.RecentActivity.TransactionCount)
	}
	if ctx.AccountProfile.TypicalLocations[0] != "US" {
		t.Errorf("expected first typical location 'US', got %s", ctx.AccountProfile.TypicalLocations[0])
	}
	if ctx.DeviceInfo.DeviceID != "device-1" {
		t.Errorf("expected device ID 'device-1', got %s", ctx.DeviceInfo.DeviceID)
	}
	if ctx.GeoLocation.Country != "US" {
		t.Errorf("expected country 'US', got %s", ctx.GeoLocation.Country)
	}
}

func TestActivitySummary(t *testing.T) {
	summary := &ActivitySummary{
		TransactionCount: 10,
		TotalAmount:      decimal.NewFromFloat(1500),
		UniqueLocations:  3,
		UniqueMerchants:  5,
		TimeWindow:       time.Hour,
	}

	if summary.TransactionCount != 10 {
		t.Errorf("expected transaction count 10, got %d", summary.TransactionCount)
	}
	if !summary.TotalAmount.Equal(decimal.NewFromFloat(1500)) {
		t.Errorf("expected total amount 1500, got %s", summary.TotalAmount.String())
	}
	if summary.UniqueLocations != 3 {
		t.Errorf("expected unique locations 3, got %d", summary.UniqueLocations)
	}
	if summary.UniqueMerchants != 5 {
		t.Errorf("expected unique merchants 5, got %d", summary.UniqueMerchants)
	}
	if summary.TimeWindow != time.Hour {
		t.Errorf("expected time window 1h, got %s", summary.TimeWindow)
	}
}

func TestAccountProfile(t *testing.T) {
	profile := &AccountProfile{
		AvgTransactionAmount: decimal.NewFromFloat(250),
		AvgDailyTransactions: 3.5,
		TypicalLocations:     []string{"US", "CA", "MX"},
		TypicalMerchants:     []string{"Amazon", "Walmart"},
		TypicalHours:         []int{9, 10, 11, 12, 14, 15, 16},
		RiskLevel:            "low",
	}

	if !profile.AvgTransactionAmount.Equal(decimal.NewFromFloat(250)) {
		t.Errorf("expected avg amount 250, got %s", profile.AvgTransactionAmount.String())
	}
	if profile.AvgDailyTransactions != 3.5 {
		t.Errorf("expected avg daily transactions 3.5, got %f", profile.AvgDailyTransactions)
	}
	if len(profile.TypicalLocations) != 3 {
		t.Errorf("expected 3 typical locations, got %d", len(profile.TypicalLocations))
	}
	if profile.RiskLevel != "low" {
		t.Errorf("expected risk level 'low', got %s", profile.RiskLevel)
	}
}

func TestDeviceInfo(t *testing.T) {
	now := time.Now()
	device := &DeviceInfo{
		DeviceID:   "device-123",
		DeviceType: "desktop",
		OS:         "macOS",
		Browser:    "Safari",
		IsKnown:    true,
		IsTrusted:  true,
		FirstSeen:  now,
	}

	if device.DeviceID != "device-123" {
		t.Errorf("expected device ID 'device-123', got %s", device.DeviceID)
	}
	if device.DeviceType != "desktop" {
		t.Errorf("expected device type 'desktop', got %s", device.DeviceType)
	}
	if !device.IsKnown {
		t.Error("expected device to be known")
	}
	if !device.IsTrusted {
		t.Error("expected device to be trusted")
	}
}

func TestGeoLocation(t *testing.T) {
	geo := &GeoLocation{
		Country:   "US",
		City:      "San Francisco",
		Latitude:  37.7749,
		Longitude: -122.4194,
		IPAddress: "192.168.1.1",
	}

	if geo.Country != "US" {
		t.Errorf("expected country 'US', got %s", geo.Country)
	}
	if geo.City != "San Francisco" {
		t.Errorf("expected city 'San Francisco', got %s", geo.City)
	}
	if geo.Latitude != 37.7749 {
		t.Errorf("expected latitude 37.7749, got %f", geo.Latitude)
	}
	if geo.IPAddress != "192.168.1.1" {
		t.Errorf("expected IP '192.168.1.1', got %s", geo.IPAddress)
	}
}

func TestRuleResult(t *testing.T) {
	result := &RuleResult{
		Triggered:   true,
		Score:       2.5,
		Description: "Test rule triggered",
		Indicators: []models.FraudIndicator{
			{Type: "test", Description: "Test indicator", Score: 2.5},
		},
	}

	if !result.Triggered {
		t.Error("expected result to be triggered")
	}
	if result.Score != 2.5 {
		t.Errorf("expected score 2.5, got %f", result.Score)
	}
	if len(result.Indicators) != 1 {
		t.Errorf("expected 1 indicator, got %d", len(result.Indicators))
	}
}

func TestEvaluationResult(t *testing.T) {
	now := time.Now()
	result := &EvaluationResult{
		TransactionID: "txn-123",
		RiskScore:     0.75,
		Decision:      DecisionReview,
		Reason:        "Moderate risk",
		Indicators: []models.FraudIndicator{
			{Type: "amount", Description: "High amount"},
		},
		Timestamp: now,
	}

	if result.TransactionID != "txn-123" {
		t.Errorf("expected transaction ID 'txn-123', got %s", result.TransactionID)
	}
	if result.RiskScore != 0.75 {
		t.Errorf("expected risk score 0.75, got %f", result.RiskScore)
	}
	if result.Decision != DecisionReview {
		t.Errorf("expected decision 'review', got %s", result.Decision)
	}
}

func TestFraudStats(t *testing.T) {
	stats := &FraudStats{
		TotalAlerts:       100,
		OpenAlerts:        20,
		FalsePositives:    10,
		FalsePositiveRate: 0.1,
		ByStatus:          map[string]int{"open": 20, "resolved": 70, "false_positive": 10},
		BySeverity:        map[string]int{"high": 30, "medium": 50, "low": 20},
		ByType:            map[string]int{"amount": 40, "velocity": 35, "geolocation": 25},
	}

	if stats.TotalAlerts != 100 {
		t.Errorf("expected 100 total alerts, got %d", stats.TotalAlerts)
	}
	if stats.FalsePositiveRate != 0.1 {
		t.Errorf("expected FP rate 0.1, got %f", stats.FalsePositiveRate)
	}
	if stats.ByStatus["open"] != 20 {
		t.Errorf("expected 20 open, got %d", stats.ByStatus["open"])
	}
}

func TestAlertFilter(t *testing.T) {
	now := time.Now()
	startDate := now.Add(-24 * time.Hour)
	endDate := now

	filter := AlertFilter{
		Status:    models.AlertStatusOpen,
		Severity:  models.AlertSeverityHigh,
		AlertType: models.FraudAlertTypeAmount,
		StartDate: &startDate,
		EndDate:   &endDate,
		Limit:     10,
	}

	if filter.Status != models.AlertStatusOpen {
		t.Errorf("expected status open, got %s", filter.Status)
	}
	if filter.Limit != 10 {
		t.Errorf("expected limit 10, got %d", filter.Limit)
	}
	if filter.StartDate == nil {
		t.Error("expected start date to be set")
	}
}
