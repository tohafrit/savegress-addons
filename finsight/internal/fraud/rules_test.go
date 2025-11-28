package fraud

import (
	"testing"
	"time"

	"github.com/savegress/finsight/pkg/models"
	"github.com/shopspring/decimal"
)

func TestAmountRule_NewAmountRule(t *testing.T) {
	rule := NewAmountRule(5000)
	if rule == nil {
		t.Fatal("NewAmountRule returned nil")
	}
	if rule.maxAmount != 5000 {
		t.Errorf("expected maxAmount 5000, got %f", rule.maxAmount)
	}
}

func TestAmountRule_Name(t *testing.T) {
	rule := NewAmountRule(5000)
	if rule.Name() != "amount" {
		t.Errorf("expected name 'amount', got %s", rule.Name())
	}
}

func TestAmountRule_Priority(t *testing.T) {
	rule := NewAmountRule(5000)
	if rule.Priority() != 100 {
		t.Errorf("expected priority 100, got %d", rule.Priority())
	}
}

func TestAmountRule_Evaluate_Normal(t *testing.T) {
	rule := NewAmountRule(5000)
	txn := &models.Transaction{
		ID:     "txn-1",
		Amount: decimal.NewFromFloat(100),
	}

	result := rule.Evaluate(txn, nil)

	if result.Triggered {
		t.Error("expected rule not to trigger for normal amount")
	}
	if result.Score != 0 {
		t.Errorf("expected score 0, got %f", result.Score)
	}
}

func TestAmountRule_Evaluate_ExceedsMax(t *testing.T) {
	rule := NewAmountRule(5000)
	txn := &models.Transaction{
		ID:     "txn-1",
		Amount: decimal.NewFromFloat(6000),
	}

	result := rule.Evaluate(txn, nil)

	if !result.Triggered {
		t.Error("expected rule to trigger for excessive amount")
	}
	if result.Score != 3.0 {
		t.Errorf("expected score 3.0, got %f", result.Score)
	}
	if len(result.Indicators) == 0 {
		t.Error("expected indicators to be set")
	}
	if result.Indicators[0].Type != "amount_anomaly" {
		t.Errorf("expected indicator type 'amount_anomaly', got %s", result.Indicators[0].Type)
	}
}

func TestAmountRule_Evaluate_DeviationFromAverage(t *testing.T) {
	rule := NewAmountRule(10000)
	txn := &models.Transaction{
		ID:     "txn-1",
		Amount: decimal.NewFromFloat(600), // 6x average
	}

	ctx := &EvaluationContext{
		AccountProfile: &AccountProfile{
			AvgTransactionAmount: decimal.NewFromFloat(100),
		},
	}

	result := rule.Evaluate(txn, ctx)

	if !result.Triggered {
		t.Error("expected rule to trigger for high deviation")
	}
	// Score should include deviation indicator
	if result.Score < 2.0 {
		t.Errorf("expected score >= 2.0, got %f", result.Score)
	}
}

func TestAmountRule_Evaluate_NormalDeviation(t *testing.T) {
	rule := NewAmountRule(10000)
	txn := &models.Transaction{
		ID:     "txn-1",
		Amount: decimal.NewFromFloat(200), // 2x average (normal)
	}

	ctx := &EvaluationContext{
		AccountProfile: &AccountProfile{
			AvgTransactionAmount: decimal.NewFromFloat(100),
		},
	}

	result := rule.Evaluate(txn, ctx)

	// Deviation of 2x should not trigger
	hasDeviationIndicator := false
	for _, ind := range result.Indicators {
		if ind.Description == "Amount significantly higher than average" {
			hasDeviationIndicator = true
		}
	}
	if hasDeviationIndicator {
		t.Error("expected no deviation indicator for 2x average")
	}
}

func TestAmountRule_Evaluate_ZeroAverage(t *testing.T) {
	rule := NewAmountRule(10000)
	txn := &models.Transaction{
		ID:     "txn-1",
		Amount: decimal.NewFromFloat(100),
	}

	ctx := &EvaluationContext{
		AccountProfile: &AccountProfile{
			AvgTransactionAmount: decimal.NewFromFloat(0), // Zero average
		},
	}

	result := rule.Evaluate(txn, ctx)

	// Should not crash and should not add deviation indicator
	hasDeviationIndicator := false
	for _, ind := range result.Indicators {
		if ind.Description == "Amount significantly higher than average" {
			hasDeviationIndicator = true
		}
	}
	if hasDeviationIndicator {
		t.Error("should not add deviation indicator when average is zero")
	}
}

func TestVelocityRule_NewVelocityRule(t *testing.T) {
	rule := NewVelocityRule(time.Hour, 10000)
	if rule == nil {
		t.Fatal("NewVelocityRule returned nil")
	}
	if rule.window != time.Hour {
		t.Errorf("expected window 1h, got %s", rule.window)
	}
	if rule.maxDaily != 10000 {
		t.Errorf("expected maxDaily 10000, got %f", rule.maxDaily)
	}
}

func TestVelocityRule_Name(t *testing.T) {
	rule := NewVelocityRule(time.Hour, 10000)
	if rule.Name() != "velocity" {
		t.Errorf("expected name 'velocity', got %s", rule.Name())
	}
}

func TestVelocityRule_Priority(t *testing.T) {
	rule := NewVelocityRule(time.Hour, 10000)
	if rule.Priority() != 90 {
		t.Errorf("expected priority 90, got %d", rule.Priority())
	}
}

func TestVelocityRule_Evaluate_NoContext(t *testing.T) {
	rule := NewVelocityRule(time.Hour, 10000)
	txn := &models.Transaction{
		ID:     "txn-1",
		Amount: decimal.NewFromFloat(100),
	}

	result := rule.Evaluate(txn, nil)

	if result.Triggered {
		t.Error("expected rule not to trigger without context")
	}
}

func TestVelocityRule_Evaluate_NoActivity(t *testing.T) {
	rule := NewVelocityRule(time.Hour, 10000)
	txn := &models.Transaction{
		ID:     "txn-1",
		Amount: decimal.NewFromFloat(100),
	}

	ctx := &EvaluationContext{}

	result := rule.Evaluate(txn, ctx)

	if result.Triggered {
		t.Error("expected rule not to trigger without activity")
	}
}

func TestVelocityRule_Evaluate_HighTransactionCount(t *testing.T) {
	rule := NewVelocityRule(time.Hour, 10000)
	txn := &models.Transaction{
		ID:     "txn-1",
		Amount: decimal.NewFromFloat(100),
	}

	ctx := &EvaluationContext{
		RecentActivity: &ActivitySummary{
			TransactionCount: 15, // High count
			TotalAmount:      decimal.NewFromFloat(1000),
			TimeWindow:       time.Hour,
		},
	}

	result := rule.Evaluate(txn, ctx)

	if !result.Triggered {
		t.Error("expected rule to trigger for high transaction count")
	}
	if result.Score < 2.0 {
		t.Errorf("expected score >= 2.0, got %f", result.Score)
	}
}

func TestVelocityRule_Evaluate_ExceedsDailyAmount(t *testing.T) {
	rule := NewVelocityRule(time.Hour, 1000)
	txn := &models.Transaction{
		ID:     "txn-1",
		Amount: decimal.NewFromFloat(100),
	}

	ctx := &EvaluationContext{
		RecentActivity: &ActivitySummary{
			TransactionCount: 5,
			TotalAmount:      decimal.NewFromFloat(1500), // Exceeds maxDaily
			TimeWindow:       time.Hour,
		},
	}

	result := rule.Evaluate(txn, ctx)

	if !result.Triggered {
		t.Error("expected rule to trigger for exceeding daily amount")
	}
}

func TestVelocityRule_Evaluate_MultipleLocations(t *testing.T) {
	rule := NewVelocityRule(time.Hour, 10000)
	txn := &models.Transaction{
		ID:     "txn-1",
		Amount: decimal.NewFromFloat(100),
	}

	ctx := &EvaluationContext{
		RecentActivity: &ActivitySummary{
			TransactionCount: 5,
			TotalAmount:      decimal.NewFromFloat(500),
			UniqueLocations:  5, // Many locations
			TimeWindow:       time.Hour,
		},
	}

	result := rule.Evaluate(txn, ctx)

	if !result.Triggered {
		t.Error("expected rule to trigger for multiple locations")
	}
}

func TestGeolocationRule_NewGeolocationRule(t *testing.T) {
	geofence := NewGeofenceChecker()
	rule := NewGeolocationRule(geofence)
	if rule == nil {
		t.Fatal("NewGeolocationRule returned nil")
	}
}

func TestGeolocationRule_Name(t *testing.T) {
	rule := NewGeolocationRule(NewGeofenceChecker())
	if rule.Name() != "geolocation" {
		t.Errorf("expected name 'geolocation', got %s", rule.Name())
	}
}

func TestGeolocationRule_Priority(t *testing.T) {
	rule := NewGeolocationRule(NewGeofenceChecker())
	if rule.Priority() != 80 {
		t.Errorf("expected priority 80, got %d", rule.Priority())
	}
}

func TestGeolocationRule_Evaluate_NoContext(t *testing.T) {
	rule := NewGeolocationRule(NewGeofenceChecker())
	txn := &models.Transaction{ID: "txn-1"}

	result := rule.Evaluate(txn, nil)
	if result.Triggered {
		t.Error("expected rule not to trigger without context")
	}
}

func TestGeolocationRule_Evaluate_NoGeoLocation(t *testing.T) {
	rule := NewGeolocationRule(NewGeofenceChecker())
	txn := &models.Transaction{ID: "txn-1"}
	ctx := &EvaluationContext{}

	result := rule.Evaluate(txn, ctx)
	if result.Triggered {
		t.Error("expected rule not to trigger without geolocation")
	}
}

func TestGeolocationRule_Evaluate_HighRiskCountry(t *testing.T) {
	rule := NewGeolocationRule(NewGeofenceChecker())
	txn := &models.Transaction{ID: "txn-1"}

	ctx := &EvaluationContext{
		GeoLocation: &GeoLocation{
			Country: "NG", // Nigeria - high risk
			City:    "Lagos",
		},
	}

	result := rule.Evaluate(txn, ctx)

	if !result.Triggered {
		t.Error("expected rule to trigger for high-risk country")
	}
	if result.Score < 2.0 {
		t.Errorf("expected score >= 2.0, got %f", result.Score)
	}
}

func TestGeolocationRule_Evaluate_NormalCountry(t *testing.T) {
	rule := NewGeolocationRule(NewGeofenceChecker())
	txn := &models.Transaction{ID: "txn-1"}

	ctx := &EvaluationContext{
		GeoLocation: &GeoLocation{
			Country: "US",
			City:    "New York",
		},
	}

	result := rule.Evaluate(txn, ctx)

	// Should not trigger for normal country without profile
	hasHighRiskIndicator := false
	for _, ind := range result.Indicators {
		if ind.Description == "Transaction from high-risk country" {
			hasHighRiskIndicator = true
		}
	}
	if hasHighRiskIndicator {
		t.Error("expected no high-risk indicator for US")
	}
}

func TestGeolocationRule_Evaluate_UnusualLocation(t *testing.T) {
	rule := NewGeolocationRule(NewGeofenceChecker())
	txn := &models.Transaction{ID: "txn-1"}

	ctx := &EvaluationContext{
		GeoLocation: &GeoLocation{
			Country: "JP", // Japan - not typical
			City:    "Tokyo",
		},
		AccountProfile: &AccountProfile{
			TypicalLocations: []string{"US", "CA", "MX"},
		},
	}

	result := rule.Evaluate(txn, ctx)

	if !result.Triggered {
		t.Error("expected rule to trigger for unusual location")
	}
}

func TestGeolocationRule_Evaluate_TypicalLocation(t *testing.T) {
	rule := NewGeolocationRule(NewGeofenceChecker())
	txn := &models.Transaction{ID: "txn-1"}

	ctx := &EvaluationContext{
		GeoLocation: &GeoLocation{
			Country: "US",
			City:    "Los Angeles",
		},
		AccountProfile: &AccountProfile{
			TypicalLocations: []string{"US", "CA"},
		},
	}

	result := rule.Evaluate(txn, ctx)

	// Should not have unusual location indicator
	hasUnusualIndicator := false
	for _, ind := range result.Indicators {
		if ind.Description == "Transaction from unusual location" {
			hasUnusualIndicator = true
		}
	}
	if hasUnusualIndicator {
		t.Error("expected no unusual location indicator for typical location")
	}
}

func TestGeolocationRule_Evaluate_ImpossibleTravel(t *testing.T) {
	rule := NewGeolocationRule(NewGeofenceChecker())

	now := time.Now()
	previousTxn := &models.Transaction{
		ID:        "txn-prev",
		CreatedAt: now.Add(-30 * time.Minute), // 30 minutes ago
		Merchant:  &models.Merchant{Country: "US"},
	}

	txn := &models.Transaction{
		ID:        "txn-1",
		CreatedAt: now,
		Merchant:  &models.Merchant{Country: "JP"}, // Japan - impossible in 30 min
	}

	ctx := &EvaluationContext{
		GeoLocation: &GeoLocation{
			Country: "JP",
			City:    "Tokyo",
		},
		AccountHistory: []*models.Transaction{previousTxn},
	}

	result := rule.Evaluate(txn, ctx)

	if !result.Triggered {
		t.Error("expected rule to trigger for impossible travel")
	}
	if result.Score < 4.0 {
		t.Errorf("expected score >= 4.0 for impossible travel, got %f", result.Score)
	}
}

func TestGeolocationRule_Evaluate_PossibleTravel(t *testing.T) {
	rule := NewGeolocationRule(NewGeofenceChecker())

	now := time.Now()
	previousTxn := &models.Transaction{
		ID:        "txn-prev",
		CreatedAt: now.Add(-24 * time.Hour), // 24 hours ago
		Merchant:  &models.Merchant{Country: "US"},
	}

	txn := &models.Transaction{
		ID:        "txn-1",
		CreatedAt: now,
		Merchant:  &models.Merchant{Country: "JP"}, // Japan - possible in 24h
	}

	ctx := &EvaluationContext{
		GeoLocation: &GeoLocation{
			Country: "JP",
			City:    "Tokyo",
		},
		AccountHistory: []*models.Transaction{previousTxn},
	}

	result := rule.Evaluate(txn, ctx)

	// Should not have impossible travel indicator
	hasImpossibleTravel := false
	for _, ind := range result.Indicators {
		if ind.Description == "Impossible travel detected" {
			hasImpossibleTravel = true
		}
	}
	if hasImpossibleTravel {
		t.Error("expected no impossible travel indicator for 24h between countries")
	}
}

func TestPatternRule_NewPatternRule(t *testing.T) {
	analyzer := NewPatternAnalyzer()
	rule := NewPatternRule(analyzer)
	if rule == nil {
		t.Fatal("NewPatternRule returned nil")
	}
}

func TestPatternRule_Name(t *testing.T) {
	rule := NewPatternRule(NewPatternAnalyzer())
	if rule.Name() != "pattern" {
		t.Errorf("expected name 'pattern', got %s", rule.Name())
	}
}

func TestPatternRule_Priority(t *testing.T) {
	rule := NewPatternRule(NewPatternAnalyzer())
	if rule.Priority() != 70 {
		t.Errorf("expected priority 70, got %d", rule.Priority())
	}
}

func TestPatternRule_Evaluate_NoContext(t *testing.T) {
	rule := NewPatternRule(NewPatternAnalyzer())
	txn := &models.Transaction{
		ID:     "txn-1",
		Amount: decimal.NewFromFloat(100),
	}

	result := rule.Evaluate(txn, nil)
	if result.Triggered {
		t.Error("expected rule not to trigger without context")
	}
}

func TestPatternRule_Evaluate_RoundAmount(t *testing.T) {
	rule := NewPatternRule(NewPatternAnalyzer())
	txn := &models.Transaction{
		ID:     "txn-1",
		Amount: decimal.NewFromFloat(500), // Round amount > 100
	}

	ctx := &EvaluationContext{
		AccountHistory: []*models.Transaction{},
	}

	result := rule.Evaluate(txn, ctx)

	// Should have round amount indicator (with low score)
	hasRoundIndicator := false
	for _, ind := range result.Indicators {
		if ind.Description == "Large round amount transaction" {
			hasRoundIndicator = true
		}
	}
	if !hasRoundIndicator {
		t.Error("expected round amount indicator")
	}
}

func TestPatternRule_Evaluate_RepeatedAmounts(t *testing.T) {
	rule := NewPatternRule(NewPatternAnalyzer())
	txn := &models.Transaction{
		ID:     "txn-1",
		Amount: decimal.NewFromFloat(99.99),
	}

	ctx := &EvaluationContext{
		AccountHistory: []*models.Transaction{
			{Amount: decimal.NewFromFloat(99.99)},
			{Amount: decimal.NewFromFloat(99.99)},
			{Amount: decimal.NewFromFloat(99.99)},
		},
	}

	result := rule.Evaluate(txn, ctx)

	if !result.Triggered {
		t.Error("expected rule to trigger for repeated amounts")
	}
}

func TestPatternRule_Evaluate_CardTesting(t *testing.T) {
	rule := NewPatternRule(NewPatternAnalyzer())
	txn := &models.Transaction{
		ID:     "txn-1",
		Amount: decimal.NewFromFloat(1.00), // Small amount
	}

	ctx := &EvaluationContext{
		AccountHistory: []*models.Transaction{
			{Amount: decimal.NewFromFloat(1.00)},
			{Amount: decimal.NewFromFloat(2.00)},
			{Amount: decimal.NewFromFloat(0.50)},
		},
	}

	result := rule.Evaluate(txn, ctx)

	if !result.Triggered {
		t.Error("expected rule to trigger for card testing pattern")
	}
	if result.Score < 3.0 {
		t.Errorf("expected score >= 3.0 for card testing, got %f", result.Score)
	}
}

func TestTimeRule_NewTimeRule(t *testing.T) {
	rule := NewTimeRule()
	if rule == nil {
		t.Fatal("NewTimeRule returned nil")
	}
}

func TestTimeRule_Name(t *testing.T) {
	rule := NewTimeRule()
	if rule.Name() != "time" {
		t.Errorf("expected name 'time', got %s", rule.Name())
	}
}

func TestTimeRule_Priority(t *testing.T) {
	rule := NewTimeRule()
	if rule.Priority() != 60 {
		t.Errorf("expected priority 60, got %d", rule.Priority())
	}
}

func TestTimeRule_Evaluate_NormalHours(t *testing.T) {
	rule := NewTimeRule()

	// Create transaction at 10 AM
	txn := &models.Transaction{
		ID:        "txn-1",
		CreatedAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
	}

	result := rule.Evaluate(txn, nil)

	// Should not trigger for normal business hours
	hasUnusualHours := false
	for _, ind := range result.Indicators {
		if ind.Description == "Transaction during unusual hours" {
			hasUnusualHours = true
		}
	}
	if hasUnusualHours {
		t.Error("expected no unusual hours indicator at 10 AM")
	}
}

func TestTimeRule_Evaluate_UnusualHours(t *testing.T) {
	rule := NewTimeRule()

	// Create transaction at 3 AM
	txn := &models.Transaction{
		ID:        "txn-1",
		CreatedAt: time.Date(2024, 1, 15, 3, 0, 0, 0, time.UTC),
	}

	result := rule.Evaluate(txn, nil)

	if !result.Triggered {
		t.Error("expected rule to trigger at 3 AM")
	}
}

func TestTimeRule_Evaluate_OutsideTypicalHours(t *testing.T) {
	rule := NewTimeRule()

	// Create transaction at 8 PM when typical hours are 9 AM - 5 PM
	txn := &models.Transaction{
		ID:        "txn-1",
		CreatedAt: time.Date(2024, 1, 15, 20, 0, 0, 0, time.UTC), // 8 PM
	}

	ctx := &EvaluationContext{
		AccountProfile: &AccountProfile{
			TypicalHours: []int{9, 10, 11, 12, 13, 14, 15, 16, 17},
		},
	}

	result := rule.Evaluate(txn, ctx)

	if !result.Triggered {
		t.Error("expected rule to trigger outside typical hours")
	}
}

func TestTimeRule_Evaluate_WithinTypicalHours(t *testing.T) {
	rule := NewTimeRule()

	// Create transaction at 2 PM when typical hours include 14
	txn := &models.Transaction{
		ID:        "txn-1",
		CreatedAt: time.Date(2024, 1, 15, 14, 0, 0, 0, time.UTC),
	}

	ctx := &EvaluationContext{
		AccountProfile: &AccountProfile{
			TypicalHours: []int{9, 10, 11, 12, 13, 14, 15, 16, 17},
		},
	}

	result := rule.Evaluate(txn, ctx)

	// Should not have outside typical hours indicator
	hasOutsideTypical := false
	for _, ind := range result.Indicators {
		if ind.Description == "Transaction outside typical hours" {
			hasOutsideTypical = true
		}
	}
	if hasOutsideTypical {
		t.Error("expected no outside typical hours indicator during typical hours")
	}
}

func TestMerchantRule_NewMerchantRule(t *testing.T) {
	rule := NewMerchantRule()
	if rule == nil {
		t.Fatal("NewMerchantRule returned nil")
	}
}

func TestMerchantRule_Name(t *testing.T) {
	rule := NewMerchantRule()
	if rule.Name() != "merchant" {
		t.Errorf("expected name 'merchant', got %s", rule.Name())
	}
}

func TestMerchantRule_Priority(t *testing.T) {
	rule := NewMerchantRule()
	if rule.Priority() != 50 {
		t.Errorf("expected priority 50, got %d", rule.Priority())
	}
}

func TestMerchantRule_Evaluate_NoMerchant(t *testing.T) {
	rule := NewMerchantRule()
	txn := &models.Transaction{
		ID:       "txn-1",
		Merchant: nil,
	}

	result := rule.Evaluate(txn, nil)
	if result.Triggered {
		t.Error("expected rule not to trigger without merchant")
	}
}

func TestMerchantRule_Evaluate_HighRiskMCC(t *testing.T) {
	rule := NewMerchantRule()

	// Test with gambling MCC
	txn := &models.Transaction{
		ID: "txn-1",
		Merchant: &models.Merchant{
			ID:   "merchant-1",
			Name: "Casino Online",
			MCC:  "7995", // Betting/Casino Gambling
		},
	}

	result := rule.Evaluate(txn, nil)

	if !result.Triggered {
		t.Error("expected rule to trigger for high-risk MCC")
	}
	if result.Score < 2.0 {
		t.Errorf("expected score >= 2.0, got %f", result.Score)
	}
}

func TestMerchantRule_Evaluate_NormalMCC(t *testing.T) {
	rule := NewMerchantRule()

	txn := &models.Transaction{
		ID: "txn-1",
		Merchant: &models.Merchant{
			ID:   "merchant-1",
			Name: "Grocery Store",
			MCC:  "5411", // Grocery stores
		},
	}

	result := rule.Evaluate(txn, nil)

	// Should not trigger for normal MCC without profile
	hasHighRiskMCC := false
	for _, ind := range result.Indicators {
		if ind.Description == "Transaction with high-risk merchant category" {
			hasHighRiskMCC = true
		}
	}
	if hasHighRiskMCC {
		t.Error("expected no high-risk MCC indicator for grocery store")
	}
}

func TestMerchantRule_Evaluate_NewMerchant(t *testing.T) {
	rule := NewMerchantRule()

	txn := &models.Transaction{
		ID: "txn-1",
		Merchant: &models.Merchant{
			ID:   "new-merchant",
			Name: "New Shop",
			MCC:  "5411",
		},
	}

	ctx := &EvaluationContext{
		AccountProfile: &AccountProfile{
			TypicalMerchants: []string{
				"merchant-1", "merchant-2", "merchant-3",
				"merchant-4", "merchant-5", "merchant-6",
			},
		},
	}

	result := rule.Evaluate(txn, ctx)

	if !result.Triggered {
		t.Error("expected rule to trigger for new merchant")
	}
}

func TestMerchantRule_Evaluate_KnownMerchant(t *testing.T) {
	rule := NewMerchantRule()

	txn := &models.Transaction{
		ID: "txn-1",
		Merchant: &models.Merchant{
			ID:   "merchant-1",
			Name: "Known Store",
			MCC:  "5411",
		},
	}

	ctx := &EvaluationContext{
		AccountProfile: &AccountProfile{
			TypicalMerchants: []string{
				"merchant-1", "merchant-2", "merchant-3",
				"merchant-4", "merchant-5", "merchant-6",
			},
		},
	}

	result := rule.Evaluate(txn, ctx)

	// Should not have new merchant indicator
	hasNewMerchant := false
	for _, ind := range result.Indicators {
		if ind.Description == "First transaction with this merchant" {
			hasNewMerchant = true
		}
	}
	if hasNewMerchant {
		t.Error("expected no new merchant indicator for known merchant")
	}
}

func TestIsRoundAmount(t *testing.T) {
	tests := []struct {
		amount   float64
		expected bool
	}{
		{100, true},
		{200, true},
		{500, true},
		{1000, true},
		{50, true},
		{150, true},
		{99.99, false},
		{123.45, false},
		{75, false},
		{250, true},
	}

	for _, tt := range tests {
		result := isRoundAmount(tt.amount)
		if result != tt.expected {
			t.Errorf("isRoundAmount(%f) = %v, expected %v", tt.amount, result, tt.expected)
		}
	}
}

func TestAllHighRiskMCCs(t *testing.T) {
	rule := NewMerchantRule()
	highRiskMCCs := []string{"5967", "5966", "7995", "5962", "4829", "6051"}

	for _, mcc := range highRiskMCCs {
		txn := &models.Transaction{
			ID:       "txn-1",
			Merchant: &models.Merchant{MCC: mcc},
		}

		result := rule.Evaluate(txn, nil)

		if !result.Triggered {
			t.Errorf("expected rule to trigger for high-risk MCC %s", mcc)
		}
	}
}
