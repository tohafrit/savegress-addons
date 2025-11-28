package reconciliation

import (
	"testing"
	"time"

	"github.com/savegress/finsight/pkg/models"
	"github.com/shopspring/decimal"
)

func TestExactMatcher_NewExactMatcher(t *testing.T) {
	matcher := NewExactMatcher()
	if matcher == nil {
		t.Fatal("NewExactMatcher returned nil")
	}
}

func TestExactMatcher_Name(t *testing.T) {
	matcher := NewExactMatcher()
	if matcher.Name() != "exact" {
		t.Errorf("expected name 'exact', got %s", matcher.Name())
	}
}

func TestExactMatcher_Match_ExternalID(t *testing.T) {
	matcher := NewExactMatcher()

	source := &models.Transaction{
		ID:         "source-1",
		ExternalID: "ext-123",
		Amount:     decimal.NewFromFloat(100),
		CreatedAt:  time.Now(),
	}

	target := &models.Transaction{
		ID:         "target-1",
		ExternalID: "ext-123",
		Amount:     decimal.NewFromFloat(100),
		CreatedAt:  time.Now(),
	}

	result := matcher.Match(source, target)

	if !result.Matched {
		t.Error("expected match by external ID")
	}
	// The implementation uses 0.95 for amount+date match, 1.0 for external ID only
	// Since both external ID and amount+date match, the second condition sets it to 0.95
	if result.Confidence < 0.95 {
		t.Errorf("expected confidence >= 0.95, got %f", result.Confidence)
	}
	if result.MatchType != "exact" {
		t.Errorf("expected match type 'exact', got %s", result.MatchType)
	}
}

func TestExactMatcher_Match_AmountAndDate(t *testing.T) {
	matcher := NewExactMatcher()

	now := time.Now()
	source := &models.Transaction{
		ID:        "source-1",
		Amount:    decimal.NewFromFloat(100),
		CreatedAt: now,
	}

	target := &models.Transaction{
		ID:        "target-1",
		Amount:    decimal.NewFromFloat(100),
		CreatedAt: now,
	}

	result := matcher.Match(source, target)

	if !result.Matched {
		t.Error("expected match by amount and date")
	}
	if result.Confidence != 0.95 {
		t.Errorf("expected confidence 0.95 for amount+date match, got %f", result.Confidence)
	}
}

func TestExactMatcher_Match_NoMatch(t *testing.T) {
	matcher := NewExactMatcher()

	now := time.Now()
	source := &models.Transaction{
		ID:        "source-1",
		Amount:    decimal.NewFromFloat(100),
		CreatedAt: now,
	}

	target := &models.Transaction{
		ID:        "target-1",
		Amount:    decimal.NewFromFloat(200), // Different amount
		CreatedAt: now,
	}

	result := matcher.Match(source, target)

	if result.Matched {
		t.Error("expected no match for different amounts")
	}
}

func TestExactMatcher_Match_DifferentDates(t *testing.T) {
	matcher := NewExactMatcher()

	now := time.Now()
	source := &models.Transaction{
		ID:        "source-1",
		Amount:    decimal.NewFromFloat(100),
		CreatedAt: now,
	}

	target := &models.Transaction{
		ID:        "target-1",
		Amount:    decimal.NewFromFloat(100),
		CreatedAt: now.Add(-48 * time.Hour), // Different date
	}

	result := matcher.Match(source, target)

	if result.Matched {
		t.Error("expected no match for different dates")
	}
}

func TestExactMatcher_FindDifferences(t *testing.T) {
	matcher := NewExactMatcher()

	source := &models.Transaction{
		ID:       "source-1",
		Amount:   decimal.NewFromFloat(100),
		Currency: "USD",
		Status:   models.TransactionStatusCompleted,
	}

	target := &models.Transaction{
		ID:       "target-1",
		Amount:   decimal.NewFromFloat(100.50),
		Currency: "EUR",
		Status:   models.TransactionStatusPending,
	}

	diffs := matcher.findDifferences(source, target)

	if len(diffs) != 3 {
		t.Errorf("expected 3 differences, got %d", len(diffs))
	}

	// Check for amount diff
	foundAmount := false
	foundCurrency := false
	foundStatus := false
	for _, diff := range diffs {
		if diff.Field == "amount" {
			foundAmount = true
			if diff.Severity != "error" {
				t.Error("amount difference should be error severity")
			}
		}
		if diff.Field == "currency" {
			foundCurrency = true
			if diff.Severity != "error" {
				t.Error("currency difference should be error severity")
			}
		}
		if diff.Field == "status" {
			foundStatus = true
			if diff.Severity != "warning" {
				t.Error("status difference should be warning severity")
			}
		}
	}

	if !foundAmount {
		t.Error("expected amount difference")
	}
	if !foundCurrency {
		t.Error("expected currency difference")
	}
	if !foundStatus {
		t.Error("expected status difference")
	}
}

func TestFuzzyMatcher_NewFuzzyMatcher(t *testing.T) {
	matcher := NewFuzzyMatcher(0.01, 24*time.Hour)
	if matcher == nil {
		t.Fatal("NewFuzzyMatcher returned nil")
	}
	if matcher.amountTolerance != 0.01 {
		t.Errorf("expected amount tolerance 0.01, got %f", matcher.amountTolerance)
	}
	if matcher.dateTolerance != 24*time.Hour {
		t.Errorf("expected date tolerance 24h, got %s", matcher.dateTolerance)
	}
}

func TestFuzzyMatcher_Name(t *testing.T) {
	matcher := NewFuzzyMatcher(0.01, 24*time.Hour)
	if matcher.Name() != "fuzzy" {
		t.Errorf("expected name 'fuzzy', got %s", matcher.Name())
	}
}

func TestFuzzyMatcher_Match_WithinTolerance(t *testing.T) {
	matcher := NewFuzzyMatcher(0.05, 48*time.Hour) // 5% tolerance, 2 days

	now := time.Now()
	source := &models.Transaction{
		ID:        "source-1",
		Amount:    decimal.NewFromFloat(100),
		CreatedAt: now,
	}

	target := &models.Transaction{
		ID:        "target-1",
		Amount:    decimal.NewFromFloat(103), // 3% difference
		CreatedAt: now.Add(24 * time.Hour),   // 1 day difference
	}

	result := matcher.Match(source, target)

	if !result.Matched {
		t.Error("expected fuzzy match within tolerance")
	}
	if result.Confidence == 0 {
		t.Error("expected non-zero confidence")
	}
	if result.MatchType != "fuzzy" {
		t.Errorf("expected match type 'fuzzy', got %s", result.MatchType)
	}
}

func TestFuzzyMatcher_Match_OutsideAmountTolerance(t *testing.T) {
	matcher := NewFuzzyMatcher(0.05, 48*time.Hour) // 5% tolerance

	now := time.Now()
	source := &models.Transaction{
		ID:        "source-1",
		Amount:    decimal.NewFromFloat(100),
		CreatedAt: now,
	}

	target := &models.Transaction{
		ID:        "target-1",
		Amount:    decimal.NewFromFloat(110), // 10% difference
		CreatedAt: now,
	}

	result := matcher.Match(source, target)

	if result.Matched {
		t.Error("expected no match outside amount tolerance")
	}
}

func TestFuzzyMatcher_Match_OutsideDateTolerance(t *testing.T) {
	matcher := NewFuzzyMatcher(0.05, 24*time.Hour) // 1 day tolerance

	now := time.Now()
	source := &models.Transaction{
		ID:        "source-1",
		Amount:    decimal.NewFromFloat(100),
		CreatedAt: now,
	}

	target := &models.Transaction{
		ID:        "target-1",
		Amount:    decimal.NewFromFloat(100),
		CreatedAt: now.Add(48 * time.Hour), // 2 days difference
	}

	result := matcher.Match(source, target)

	if result.Matched {
		t.Error("expected no match outside date tolerance")
	}
}

func TestFuzzyMatcher_Match_ZeroAmount(t *testing.T) {
	matcher := NewFuzzyMatcher(0.05, 24*time.Hour)

	source := &models.Transaction{
		ID:        "source-1",
		Amount:    decimal.NewFromFloat(0),
		CreatedAt: time.Now(),
	}

	target := &models.Transaction{
		ID:        "target-1",
		Amount:    decimal.NewFromFloat(100),
		CreatedAt: time.Now(),
	}

	result := matcher.Match(source, target)

	// Should handle zero amount gracefully
	if result.Matched {
		t.Error("expected no match with zero source amount")
	}
}

func TestFuzzyMatcher_Match_WithDifferences(t *testing.T) {
	matcher := NewFuzzyMatcher(0.1, 48*time.Hour) // 10% tolerance

	now := time.Now()
	source := &models.Transaction{
		ID:        "source-1",
		Amount:    decimal.NewFromFloat(100),
		CreatedAt: now,
	}

	target := &models.Transaction{
		ID:        "target-1",
		Amount:    decimal.NewFromFloat(105),        // 5% difference
		CreatedAt: now.Add(25 * time.Hour),          // Next day
	}

	result := matcher.Match(source, target)

	if !result.Matched {
		t.Error("expected fuzzy match")
	}
	if len(result.Differences) != 2 {
		t.Errorf("expected 2 differences (amount and date), got %d", len(result.Differences))
	}
}

func TestFuzzyMatcher_Match_NegativeTimeDiff(t *testing.T) {
	matcher := NewFuzzyMatcher(0.05, 48*time.Hour)

	now := time.Now()
	source := &models.Transaction{
		ID:        "source-1",
		Amount:    decimal.NewFromFloat(100),
		CreatedAt: now.Add(24 * time.Hour), // Source is in future
	}

	target := &models.Transaction{
		ID:        "target-1",
		Amount:    decimal.NewFromFloat(100),
		CreatedAt: now, // Target is in past
	}

	result := matcher.Match(source, target)

	// Should still match even with negative time diff
	if !result.Matched {
		t.Error("expected match even with negative time diff")
	}
}

func TestReferenceIDMatcher_NewReferenceIDMatcher(t *testing.T) {
	matcher := NewReferenceIDMatcher()
	if matcher == nil {
		t.Fatal("NewReferenceIDMatcher returned nil")
	}
}

func TestReferenceIDMatcher_Name(t *testing.T) {
	matcher := NewReferenceIDMatcher()
	if matcher.Name() != "reference" {
		t.Errorf("expected name 'reference', got %s", matcher.Name())
	}
}

func TestReferenceIDMatcher_Match_ExternalID(t *testing.T) {
	matcher := NewReferenceIDMatcher()

	source := &models.Transaction{
		ID:         "source-1",
		ExternalID: "ext-123",
		Amount:     decimal.NewFromFloat(100),
	}

	target := &models.Transaction{
		ID:         "target-1",
		ExternalID: "ext-123",
		Amount:     decimal.NewFromFloat(100),
	}

	result := matcher.Match(source, target)

	if !result.Matched {
		t.Error("expected match by external ID")
	}
	if result.Confidence != 0.98 {
		t.Errorf("expected confidence 0.98, got %f", result.Confidence)
	}
}

func TestReferenceIDMatcher_Match_ContainedID(t *testing.T) {
	matcher := NewReferenceIDMatcher()

	source := &models.Transaction{
		ID:     "txn-123",
		Amount: decimal.NewFromFloat(100),
	}

	target := &models.Transaction{
		ID:     "prefix-txn-123-suffix",
		Amount: decimal.NewFromFloat(100),
	}

	result := matcher.Match(source, target)

	if !result.Matched {
		t.Error("expected match by contained ID")
	}
	if result.Confidence != 0.85 {
		t.Errorf("expected confidence 0.85, got %f", result.Confidence)
	}
}

func TestReferenceIDMatcher_Match_MetadataReference(t *testing.T) {
	matcher := NewReferenceIDMatcher()

	source := &models.Transaction{
		ID:       "source-1",
		Amount:   decimal.NewFromFloat(100),
		Metadata: map[string]string{"reference_id": "ref-xyz"},
	}

	target := &models.Transaction{
		ID:       "target-1",
		Amount:   decimal.NewFromFloat(100),
		Metadata: map[string]string{"reference_id": "ref-xyz"},
	}

	result := matcher.Match(source, target)

	if !result.Matched {
		t.Error("expected match by metadata reference")
	}
	if result.Confidence != 0.95 {
		t.Errorf("expected confidence 0.95, got %f", result.Confidence)
	}
}

func TestReferenceIDMatcher_Match_NoMatch(t *testing.T) {
	matcher := NewReferenceIDMatcher()

	source := &models.Transaction{
		ID:       "source-1",
		Amount:   decimal.NewFromFloat(100),
		Metadata: map[string]string{"reference_id": "ref-abc"},
	}

	target := &models.Transaction{
		ID:       "target-1",
		Amount:   decimal.NewFromFloat(100),
		Metadata: map[string]string{"reference_id": "ref-xyz"},
	}

	result := matcher.Match(source, target)

	if result.Matched {
		t.Error("expected no match for different references")
	}
}

func TestReferenceIDMatcher_Match_NilMetadata(t *testing.T) {
	matcher := NewReferenceIDMatcher()

	source := &models.Transaction{
		ID:       "source-1",
		Amount:   decimal.NewFromFloat(100),
		Metadata: nil,
	}

	target := &models.Transaction{
		ID:       "target-1",
		Amount:   decimal.NewFromFloat(100),
		Metadata: nil,
	}

	result := matcher.Match(source, target)

	// Should not crash and should not match
	if result.Matched {
		t.Error("expected no match with nil metadata")
	}
}

func TestReferenceIDMatcher_Match_EmptyReference(t *testing.T) {
	matcher := NewReferenceIDMatcher()

	source := &models.Transaction{
		ID:       "source-1",
		Amount:   decimal.NewFromFloat(100),
		Metadata: map[string]string{"reference_id": ""},
	}

	target := &models.Transaction{
		ID:       "target-1",
		Amount:   decimal.NewFromFloat(100),
		Metadata: map[string]string{"reference_id": ""},
	}

	result := matcher.Match(source, target)

	if result.Matched {
		t.Error("expected no match with empty reference IDs")
	}
}

func TestReferenceIDMatcher_FindDifferences(t *testing.T) {
	matcher := NewReferenceIDMatcher()

	source := &models.Transaction{
		ID:     "source-1",
		Amount: decimal.NewFromFloat(100),
		Status: models.TransactionStatusCompleted,
	}

	target := &models.Transaction{
		ID:     "target-1",
		Amount: decimal.NewFromFloat(95),
		Status: models.TransactionStatusPending,
	}

	diffs := matcher.findDifferences(source, target)

	if len(diffs) != 2 {
		t.Errorf("expected 2 differences, got %d", len(diffs))
	}
}

func TestMultiMatcher_NewMultiMatcher(t *testing.T) {
	exact := NewExactMatcher()
	fuzzy := NewFuzzyMatcher(0.01, 24*time.Hour)

	matcher := NewMultiMatcher(exact, fuzzy)
	if matcher == nil {
		t.Fatal("NewMultiMatcher returned nil")
	}
	if len(matcher.matchers) != 2 {
		t.Errorf("expected 2 matchers, got %d", len(matcher.matchers))
	}
}

func TestMultiMatcher_Name(t *testing.T) {
	matcher := NewMultiMatcher()
	if matcher.Name() != "multi" {
		t.Errorf("expected name 'multi', got %s", matcher.Name())
	}
}

func TestMultiMatcher_Match_Combined(t *testing.T) {
	exact := NewExactMatcher()
	reference := NewReferenceIDMatcher()

	matcher := NewMultiMatcher(exact, reference)

	now := time.Now()
	source := &models.Transaction{
		ID:         "source-1",
		ExternalID: "ext-123",
		Amount:     decimal.NewFromFloat(100),
		CreatedAt:  now,
	}

	target := &models.Transaction{
		ID:         "target-1",
		ExternalID: "ext-123",
		Amount:     decimal.NewFromFloat(100),
		CreatedAt:  now,
	}

	result := matcher.Match(source, target)

	if !result.Matched {
		t.Error("expected match from combined matchers")
	}
	// Match type should be combination
	if result.MatchType != "exact+reference" {
		t.Errorf("expected match type 'exact+reference', got %s", result.MatchType)
	}
}

func TestMultiMatcher_Match_NoMatch(t *testing.T) {
	exact := NewExactMatcher()
	matcher := NewMultiMatcher(exact)

	now := time.Now()
	source := &models.Transaction{
		ID:        "source-1",
		Amount:    decimal.NewFromFloat(100),
		CreatedAt: now,
	}

	target := &models.Transaction{
		ID:        "target-1",
		Amount:    decimal.NewFromFloat(999),
		CreatedAt: now.Add(-100 * 24 * time.Hour),
	}

	result := matcher.Match(source, target)

	if result.Matched {
		t.Error("expected no match")
	}
}

func TestMultiMatcher_Match_MergesDifferences(t *testing.T) {
	exact := NewExactMatcher()
	reference := NewReferenceIDMatcher()

	matcher := NewMultiMatcher(exact, reference)

	now := time.Now()
	source := &models.Transaction{
		ID:         "source-1",
		ExternalID: "ext-123",
		Amount:     decimal.NewFromFloat(100),
		Currency:   "USD",
		Status:     models.TransactionStatusCompleted,
		CreatedAt:  now,
	}

	target := &models.Transaction{
		ID:         "target-1",
		ExternalID: "ext-123",
		Amount:     decimal.NewFromFloat(100.50),
		Currency:   "EUR",
		Status:     models.TransactionStatusPending,
		CreatedAt:  now,
	}

	result := matcher.Match(source, target)

	// Should have merged differences without duplicates
	fieldsSeen := make(map[string]bool)
	for _, diff := range result.Differences {
		if fieldsSeen[diff.Field] {
			t.Errorf("duplicate difference for field %s", diff.Field)
		}
		fieldsSeen[diff.Field] = true
	}
}

func TestCompositeMatch(t *testing.T) {
	cm := &CompositeMatch{
		SourceID:   "source-1",
		TargetID:   "target-1",
		Matchers:   []string{"exact", "reference"},
		Confidence: 0.97,
		Differences: []Difference{
			{Field: "amount", Source: "100", Target: "100.01"},
		},
	}

	if cm.SourceID != "source-1" {
		t.Errorf("expected SourceID 'source-1', got %s", cm.SourceID)
	}
	if len(cm.Matchers) != 2 {
		t.Errorf("expected 2 matchers, got %d", len(cm.Matchers))
	}
	if cm.Confidence != 0.97 {
		t.Errorf("expected confidence 0.97, got %f", cm.Confidence)
	}
}

func TestMultiMatcher_Match_EmptyMatchers(t *testing.T) {
	matcher := NewMultiMatcher() // No matchers

	source := &models.Transaction{
		ID:        "source-1",
		Amount:    decimal.NewFromFloat(100),
		CreatedAt: time.Now(),
	}

	target := &models.Transaction{
		ID:        "target-1",
		Amount:    decimal.NewFromFloat(100),
		CreatedAt: time.Now(),
	}

	result := matcher.Match(source, target)

	if result.Matched {
		t.Error("expected no match with empty matchers")
	}
}

func TestFuzzyMatcher_Match_ConfidenceCalculation(t *testing.T) {
	matcher := NewFuzzyMatcher(0.10, 48*time.Hour) // 10% tolerance, 2 days

	now := time.Now()
	source := &models.Transaction{
		ID:        "source-1",
		Amount:    decimal.NewFromFloat(100),
		CreatedAt: now,
	}

	// Target is very close (high confidence)
	target := &models.Transaction{
		ID:        "target-1",
		Amount:    decimal.NewFromFloat(100.01), // 0.01% difference
		CreatedAt: now.Add(1 * time.Hour),       // 1 hour difference
	}

	result := matcher.Match(source, target)

	if !result.Matched {
		t.Error("expected match")
	}
	// Confidence should be high but not perfect (max 90% for fuzzy)
	if result.Confidence <= 0.8 {
		t.Errorf("expected high confidence, got %f", result.Confidence)
	}
	if result.Confidence > 0.9 {
		t.Errorf("confidence should not exceed 0.9 for fuzzy match, got %f", result.Confidence)
	}
}
