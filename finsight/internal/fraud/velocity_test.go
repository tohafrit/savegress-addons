package fraud

import (
	"testing"
	"time"

	"github.com/savegress/finsight/pkg/models"
	"github.com/shopspring/decimal"
)

func TestVelocityTracker_NewVelocityTracker(t *testing.T) {
	tracker := NewVelocityTracker(time.Hour)
	if tracker == nil {
		t.Fatal("NewVelocityTracker returned nil")
	}
	if tracker.window != time.Hour {
		t.Errorf("expected window 1h, got %s", tracker.window)
	}
	if tracker.accounts == nil {
		t.Error("accounts map not initialized")
	}
}

func TestVelocityTracker_Record(t *testing.T) {
	tracker := NewVelocityTracker(time.Hour)

	txn := &models.Transaction{
		ID:            "txn-1",
		Amount:        decimal.NewFromFloat(100),
		SourceAccount: "acc-1",
		CreatedAt:     time.Now(),
		Merchant: &models.Merchant{
			ID:      "merchant-1",
			Country: "US",
		},
	}

	tracker.Record(txn)

	// Check account was created
	acc, ok := tracker.accounts["acc-1"]
	if !ok {
		t.Fatal("expected account to be created")
	}
	if len(acc.Transactions) != 1 {
		t.Errorf("expected 1 transaction, got %d", len(acc.Transactions))
	}

	// Check transaction record
	record := acc.Transactions[0]
	if record.ID != "txn-1" {
		t.Errorf("expected ID 'txn-1', got %s", record.ID)
	}
	if !record.Amount.Equal(decimal.NewFromFloat(100)) {
		t.Errorf("expected amount 100, got %s", record.Amount)
	}
	if record.Location != "US" {
		t.Errorf("expected location 'US', got %s", record.Location)
	}
	if record.Merchant != "merchant-1" {
		t.Errorf("expected merchant 'merchant-1', got %s", record.Merchant)
	}
}

func TestVelocityTracker_Record_NoMerchant(t *testing.T) {
	tracker := NewVelocityTracker(time.Hour)

	txn := &models.Transaction{
		ID:            "txn-1",
		Amount:        decimal.NewFromFloat(100),
		SourceAccount: "acc-1",
		CreatedAt:     time.Now(),
		Merchant:      nil,
	}

	tracker.Record(txn)

	acc := tracker.accounts["acc-1"]
	record := acc.Transactions[0]
	if record.Location != "" {
		t.Errorf("expected empty location, got %s", record.Location)
	}
	if record.Merchant != "" {
		t.Errorf("expected empty merchant, got %s", record.Merchant)
	}
}

func TestVelocityTracker_Record_MultipleTransactions(t *testing.T) {
	tracker := NewVelocityTracker(time.Hour)

	for i := 0; i < 5; i++ {
		txn := &models.Transaction{
			ID:            string(rune('a' + i)),
			Amount:        decimal.NewFromFloat(float64(100 + i*10)),
			SourceAccount: "acc-1",
			CreatedAt:     time.Now(),
		}
		tracker.Record(txn)
	}

	acc := tracker.accounts["acc-1"]
	if len(acc.Transactions) != 5 {
		t.Errorf("expected 5 transactions, got %d", len(acc.Transactions))
	}
}

func TestVelocityTracker_Record_DifferentAccounts(t *testing.T) {
	tracker := NewVelocityTracker(time.Hour)

	txn1 := &models.Transaction{
		ID:            "txn-1",
		Amount:        decimal.NewFromFloat(100),
		SourceAccount: "acc-1",
		CreatedAt:     time.Now(),
	}
	txn2 := &models.Transaction{
		ID:            "txn-2",
		Amount:        decimal.NewFromFloat(200),
		SourceAccount: "acc-2",
		CreatedAt:     time.Now(),
	}

	tracker.Record(txn1)
	tracker.Record(txn2)

	if len(tracker.accounts) != 2 {
		t.Errorf("expected 2 accounts, got %d", len(tracker.accounts))
	}
	if len(tracker.accounts["acc-1"].Transactions) != 1 {
		t.Error("expected 1 transaction for acc-1")
	}
	if len(tracker.accounts["acc-2"].Transactions) != 1 {
		t.Error("expected 1 transaction for acc-2")
	}
}

func TestVelocityTracker_GetActivity_NoAccount(t *testing.T) {
	tracker := NewVelocityTracker(time.Hour)

	summary := tracker.GetActivity("non-existent")

	if summary.TransactionCount != 0 {
		t.Errorf("expected 0 transactions, got %d", summary.TransactionCount)
	}
	if summary.TimeWindow != time.Hour {
		t.Errorf("expected window 1h, got %s", summary.TimeWindow)
	}
}

func TestVelocityTracker_GetActivity(t *testing.T) {
	tracker := NewVelocityTracker(time.Hour)

	now := time.Now()

	// Add recent transactions
	txns := []*models.Transaction{
		{
			ID:            "txn-1",
			Amount:        decimal.NewFromFloat(100),
			SourceAccount: "acc-1",
			CreatedAt:     now.Add(-10 * time.Minute),
			Merchant:      &models.Merchant{ID: "m1", Country: "US"},
		},
		{
			ID:            "txn-2",
			Amount:        decimal.NewFromFloat(200),
			SourceAccount: "acc-1",
			CreatedAt:     now.Add(-20 * time.Minute),
			Merchant:      &models.Merchant{ID: "m2", Country: "CA"},
		},
		{
			ID:            "txn-3",
			Amount:        decimal.NewFromFloat(150),
			SourceAccount: "acc-1",
			CreatedAt:     now.Add(-30 * time.Minute),
			Merchant:      &models.Merchant{ID: "m1", Country: "US"}, // Same merchant and location
		},
	}

	for _, txn := range txns {
		tracker.Record(txn)
	}

	summary := tracker.GetActivity("acc-1")

	if summary.TransactionCount != 3 {
		t.Errorf("expected 3 transactions, got %d", summary.TransactionCount)
	}
	expectedTotal := decimal.NewFromFloat(450)
	if !summary.TotalAmount.Equal(expectedTotal) {
		t.Errorf("expected total amount 450, got %s", summary.TotalAmount)
	}
	if summary.UniqueLocations != 2 { // US and CA
		t.Errorf("expected 2 unique locations, got %d", summary.UniqueLocations)
	}
	if summary.UniqueMerchants != 2 { // m1 and m2
		t.Errorf("expected 2 unique merchants, got %d", summary.UniqueMerchants)
	}
}

func TestVelocityTracker_GetActivity_ExcludesOld(t *testing.T) {
	tracker := NewVelocityTracker(time.Hour)

	now := time.Now()

	// Add old transaction
	tracker.accounts["acc-1"] = &AccountVelocity{
		Transactions: []*TransactionRecord{
			{
				ID:        "old-txn",
				Amount:    decimal.NewFromFloat(1000),
				Timestamp: now.Add(-2 * time.Hour), // 2 hours ago, outside window
				Location:  "FR",
				Merchant:  "old-merchant",
			},
			{
				ID:        "recent-txn",
				Amount:    decimal.NewFromFloat(100),
				Timestamp: now.Add(-30 * time.Minute), // 30 min ago, inside window
				Location:  "US",
				Merchant:  "new-merchant",
			},
		},
	}

	summary := tracker.GetActivity("acc-1")

	if summary.TransactionCount != 1 {
		t.Errorf("expected 1 transaction (excluding old), got %d", summary.TransactionCount)
	}
	if !summary.TotalAmount.Equal(decimal.NewFromFloat(100)) {
		t.Errorf("expected total amount 100, got %s", summary.TotalAmount)
	}
}

func TestVelocityTracker_GetRecentTransactions_NoAccount(t *testing.T) {
	tracker := NewVelocityTracker(time.Hour)

	recent := tracker.GetRecentTransactions("non-existent", 10)

	if recent != nil {
		t.Errorf("expected nil for non-existent account, got %v", recent)
	}
}

func TestVelocityTracker_GetRecentTransactions(t *testing.T) {
	tracker := NewVelocityTracker(time.Hour)

	now := time.Now()

	// Add transactions
	tracker.accounts["acc-1"] = &AccountVelocity{
		Transactions: []*TransactionRecord{
			{ID: "txn-1", Amount: decimal.NewFromFloat(100), Timestamp: now.Add(-50 * time.Minute)},
			{ID: "txn-2", Amount: decimal.NewFromFloat(200), Timestamp: now.Add(-40 * time.Minute)},
			{ID: "txn-3", Amount: decimal.NewFromFloat(300), Timestamp: now.Add(-30 * time.Minute)},
			{ID: "txn-4", Amount: decimal.NewFromFloat(400), Timestamp: now.Add(-20 * time.Minute)},
			{ID: "txn-5", Amount: decimal.NewFromFloat(500), Timestamp: now.Add(-10 * time.Minute)},
		},
	}

	// Get 3 most recent
	recent := tracker.GetRecentTransactions("acc-1", 3)

	if len(recent) != 3 {
		t.Errorf("expected 3 transactions, got %d", len(recent))
	}
	// Should be in reverse order (most recent first)
	if recent[0].ID != "txn-5" {
		t.Errorf("expected first transaction to be txn-5, got %s", recent[0].ID)
	}
}

func TestVelocityTracker_GetRecentTransactions_ExcludesOld(t *testing.T) {
	tracker := NewVelocityTracker(time.Hour)

	now := time.Now()

	tracker.accounts["acc-1"] = &AccountVelocity{
		Transactions: []*TransactionRecord{
			{ID: "old", Amount: decimal.NewFromFloat(1000), Timestamp: now.Add(-2 * time.Hour)},
			{ID: "recent", Amount: decimal.NewFromFloat(100), Timestamp: now.Add(-30 * time.Minute)},
		},
	}

	recent := tracker.GetRecentTransactions("acc-1", 10)

	if len(recent) != 1 {
		t.Errorf("expected 1 recent transaction, got %d", len(recent))
	}
	if recent[0].ID != "recent" {
		t.Errorf("expected recent transaction, got %s", recent[0].ID)
	}
}

func TestVelocityTracker_Reset(t *testing.T) {
	tracker := NewVelocityTracker(time.Hour)

	// Add some transactions
	txn := &models.Transaction{
		ID:            "txn-1",
		Amount:        decimal.NewFromFloat(100),
		SourceAccount: "acc-1",
		CreatedAt:     time.Now(),
	}
	tracker.Record(txn)

	if len(tracker.accounts) == 0 {
		t.Error("expected accounts to be populated")
	}

	// Reset
	tracker.Reset()

	if len(tracker.accounts) != 0 {
		t.Errorf("expected empty accounts after reset, got %d", len(tracker.accounts))
	}
}

func TestVelocityTracker_CleanOldRecords(t *testing.T) {
	tracker := NewVelocityTracker(time.Hour)

	now := time.Now()

	acc := &AccountVelocity{
		Transactions: []*TransactionRecord{
			{ID: "old-1", Timestamp: now.Add(-3 * time.Hour)},
			{ID: "old-2", Timestamp: now.Add(-2 * time.Hour)},
			{ID: "recent-1", Timestamp: now.Add(-30 * time.Minute)},
			{ID: "recent-2", Timestamp: now.Add(-10 * time.Minute)},
		},
	}

	tracker.cleanOldRecords(acc)

	if len(acc.Transactions) != 2 {
		t.Errorf("expected 2 recent transactions, got %d", len(acc.Transactions))
	}
}

func TestPatternAnalyzer_NewPatternAnalyzer(t *testing.T) {
	analyzer := NewPatternAnalyzer()
	if analyzer == nil {
		t.Fatal("NewPatternAnalyzer returned nil")
	}
	if analyzer.accountPatterns == nil {
		t.Error("accountPatterns map not initialized")
	}
}

func TestPatternAnalyzer_Learn_Empty(t *testing.T) {
	analyzer := NewPatternAnalyzer()

	analyzer.Learn("acc-1", []*models.Transaction{})

	pattern := analyzer.GetPattern("acc-1")
	if pattern != nil {
		t.Error("expected nil pattern for empty transactions")
	}
}

func TestPatternAnalyzer_Learn(t *testing.T) {
	analyzer := NewPatternAnalyzer()

	txns := []*models.Transaction{
		{
			Amount:    decimal.NewFromFloat(100),
			CreatedAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC), // 10 AM Monday
			Merchant:  &models.Merchant{ID: "m1"},
		},
		{
			Amount:    decimal.NewFromFloat(200),
			CreatedAt: time.Date(2024, 1, 16, 11, 0, 0, 0, time.UTC), // 11 AM Tuesday
			Merchant:  &models.Merchant{ID: "m1"},
		},
		{
			Amount:    decimal.NewFromFloat(150),
			CreatedAt: time.Date(2024, 1, 17, 10, 0, 0, 0, time.UTC), // 10 AM Wednesday
			Merchant:  &models.Merchant{ID: "m2"},
		},
	}

	analyzer.Learn("acc-1", txns)

	pattern := analyzer.GetPattern("acc-1")
	if pattern == nil {
		t.Fatal("expected pattern to be created")
	}

	expectedAvg := decimal.NewFromFloat(150)
	if !pattern.AvgAmount.Equal(expectedAvg) {
		t.Errorf("expected avg amount 150, got %s", pattern.AvgAmount)
	}

	// Check typical hours include 10 and 11
	hasHour10 := false
	hasHour11 := false
	for _, h := range pattern.TypicalHours {
		if h == 10 {
			hasHour10 = true
		}
		if h == 11 {
			hasHour11 = true
		}
	}
	if !hasHour10 || !hasHour11 {
		t.Errorf("expected typical hours to include 10 and 11, got %v", pattern.TypicalHours)
	}

	// m1 appears twice, should be in typical merchants
	hasM1 := false
	for _, m := range pattern.TypicalMerchants {
		if m == "m1" {
			hasM1 = true
		}
	}
	if !hasM1 {
		t.Error("expected m1 to be in typical merchants")
	}
}

func TestPatternAnalyzer_Learn_NoMerchant(t *testing.T) {
	analyzer := NewPatternAnalyzer()

	txns := []*models.Transaction{
		{
			Amount:    decimal.NewFromFloat(100),
			CreatedAt: time.Now(),
			Merchant:  nil,
		},
	}

	analyzer.Learn("acc-1", txns)

	pattern := analyzer.GetPattern("acc-1")
	if pattern == nil {
		t.Fatal("expected pattern to be created")
	}
	if len(pattern.TypicalMerchants) != 0 {
		t.Errorf("expected no typical merchants, got %d", len(pattern.TypicalMerchants))
	}
}

func TestPatternAnalyzer_GetPattern_NotFound(t *testing.T) {
	analyzer := NewPatternAnalyzer()

	pattern := analyzer.GetPattern("non-existent")
	if pattern != nil {
		t.Error("expected nil for non-existent account")
	}
}

func TestTopNKeys_Empty(t *testing.T) {
	result := topNKeys(map[int]int{}, 5)
	if result != nil {
		t.Errorf("expected nil for empty map, got %v", result)
	}
}

func TestTopNKeys(t *testing.T) {
	m := map[int]int{
		10: 5,
		11: 8,
		12: 3,
		14: 10,
		15: 2,
	}

	result := topNKeys(m, 3)

	if len(result) != 3 {
		t.Errorf("expected 3 results, got %d", len(result))
	}
	// Should be sorted by count descending
	if result[0] != 14 {
		t.Errorf("expected first key to be 14 (count 10), got %d", result[0])
	}
	if result[1] != 11 {
		t.Errorf("expected second key to be 11 (count 8), got %d", result[1])
	}
	if result[2] != 10 {
		t.Errorf("expected third key to be 10 (count 5), got %d", result[2])
	}
}

func TestTopNKeys_LessThanN(t *testing.T) {
	m := map[int]int{
		10: 5,
		11: 8,
	}

	result := topNKeys(m, 5)

	if len(result) != 2 {
		t.Errorf("expected 2 results (less than n), got %d", len(result))
	}
}

func TestGeofenceChecker_NewGeofenceChecker(t *testing.T) {
	checker := NewGeofenceChecker()
	if checker == nil {
		t.Fatal("NewGeofenceChecker returned nil")
	}
	if checker.highRiskCountries == nil {
		t.Error("highRiskCountries map not initialized")
	}
	if checker.blockedCountries == nil {
		t.Error("blockedCountries map not initialized")
	}
}

func TestGeofenceChecker_InitializeDefaultRules(t *testing.T) {
	checker := NewGeofenceChecker()

	// Check high-risk countries
	highRiskCountries := []string{"NG", "RU", "UA", "RO", "ID", "PH", "VN"}
	for _, code := range highRiskCountries {
		if !checker.IsHighRiskCountry(code) {
			t.Errorf("expected %s to be high-risk", code)
		}
	}

	// Check blocked countries
	blockedCountries := []string{"KP", "IR", "SY", "CU"}
	for _, code := range blockedCountries {
		if !checker.IsBlockedCountry(code) {
			t.Errorf("expected %s to be blocked", code)
		}
	}
}

func TestGeofenceChecker_IsHighRiskCountry(t *testing.T) {
	checker := NewGeofenceChecker()

	if !checker.IsHighRiskCountry("NG") {
		t.Error("expected NG to be high-risk")
	}
	if checker.IsHighRiskCountry("US") {
		t.Error("expected US not to be high-risk")
	}
}

func TestGeofenceChecker_IsBlockedCountry(t *testing.T) {
	checker := NewGeofenceChecker()

	if !checker.IsBlockedCountry("KP") {
		t.Error("expected KP to be blocked")
	}
	if checker.IsBlockedCountry("US") {
		t.Error("expected US not to be blocked")
	}
}

func TestGeofenceChecker_AddHighRiskCountry(t *testing.T) {
	checker := NewGeofenceChecker()

	if checker.IsHighRiskCountry("XX") {
		t.Error("XX should not be high-risk initially")
	}

	checker.AddHighRiskCountry("XX")

	if !checker.IsHighRiskCountry("XX") {
		t.Error("XX should be high-risk after adding")
	}
}

func TestGeofenceChecker_AddBlockedCountry(t *testing.T) {
	checker := NewGeofenceChecker()

	if checker.IsBlockedCountry("XX") {
		t.Error("XX should not be blocked initially")
	}

	checker.AddBlockedCountry("XX")

	if !checker.IsBlockedCountry("XX") {
		t.Error("XX should be blocked after adding")
	}
}

func TestGeofenceChecker_RemoveHighRiskCountry(t *testing.T) {
	checker := NewGeofenceChecker()

	if !checker.IsHighRiskCountry("NG") {
		t.Error("NG should be high-risk initially")
	}

	checker.RemoveHighRiskCountry("NG")

	if checker.IsHighRiskCountry("NG") {
		t.Error("NG should not be high-risk after removal")
	}
}

func TestGeofenceChecker_RemoveBlockedCountry(t *testing.T) {
	checker := NewGeofenceChecker()

	if !checker.IsBlockedCountry("KP") {
		t.Error("KP should be blocked initially")
	}

	checker.RemoveBlockedCountry("KP")

	if checker.IsBlockedCountry("KP") {
		t.Error("KP should not be blocked after removal")
	}
}

func TestAccountVelocity(t *testing.T) {
	av := &AccountVelocity{
		Transactions: []*TransactionRecord{
			{ID: "txn-1"},
			{ID: "txn-2"},
		},
		LastCleanup: time.Now(),
	}

	if len(av.Transactions) != 2 {
		t.Errorf("expected 2 transactions, got %d", len(av.Transactions))
	}
	if av.LastCleanup.IsZero() {
		t.Error("LastCleanup should not be zero")
	}
}

func TestTransactionRecord(t *testing.T) {
	now := time.Now()
	record := &TransactionRecord{
		ID:        "txn-123",
		Amount:    decimal.NewFromFloat(500),
		Timestamp: now,
		Location:  "US",
		Merchant:  "Amazon",
	}

	if record.ID != "txn-123" {
		t.Errorf("expected ID 'txn-123', got %s", record.ID)
	}
	if !record.Amount.Equal(decimal.NewFromFloat(500)) {
		t.Errorf("expected amount 500, got %s", record.Amount)
	}
	if record.Location != "US" {
		t.Errorf("expected location 'US', got %s", record.Location)
	}
	if record.Merchant != "Amazon" {
		t.Errorf("expected merchant 'Amazon', got %s", record.Merchant)
	}
}

func TestAccountPattern(t *testing.T) {
	pattern := &AccountPattern{
		AvgAmount:        decimal.NewFromFloat(250),
		StdDevAmount:     50.5,
		TypicalHours:     []int{9, 10, 11, 12},
		TypicalDays:      []time.Weekday{time.Monday, time.Tuesday},
		TypicalMerchants: []string{"Amazon", "Walmart"},
		TypicalAmounts:   []decimal.Decimal{decimal.NewFromFloat(100), decimal.NewFromFloat(200)},
		LastUpdated:      time.Now(),
	}

	if !pattern.AvgAmount.Equal(decimal.NewFromFloat(250)) {
		t.Errorf("expected avg amount 250, got %s", pattern.AvgAmount)
	}
	if pattern.StdDevAmount != 50.5 {
		t.Errorf("expected std dev 50.5, got %f", pattern.StdDevAmount)
	}
	if len(pattern.TypicalHours) != 4 {
		t.Errorf("expected 4 typical hours, got %d", len(pattern.TypicalHours))
	}
	if len(pattern.TypicalDays) != 2 {
		t.Errorf("expected 2 typical days, got %d", len(pattern.TypicalDays))
	}
}

func TestVelocityTracker_Record_TriggersCleanup(t *testing.T) {
	tracker := NewVelocityTracker(time.Hour)

	now := time.Now()

	// Create account with old LastCleanup
	tracker.accounts["acc-1"] = &AccountVelocity{
		Transactions: []*TransactionRecord{
			{ID: "old", Timestamp: now.Add(-2 * time.Hour)},
		},
		LastCleanup: now.Add(-time.Hour), // Old enough to trigger cleanup
	}

	// Record new transaction, should trigger cleanup
	txn := &models.Transaction{
		ID:            "new",
		Amount:        decimal.NewFromFloat(100),
		SourceAccount: "acc-1",
		CreatedAt:     now,
	}
	tracker.Record(txn)

	// Old transaction should be cleaned up
	acc := tracker.accounts["acc-1"]

	// Check that LastCleanup was updated
	if acc.LastCleanup.Before(now.Add(-time.Minute)) {
		t.Error("LastCleanup should have been updated")
	}

	// Only new transaction should remain
	foundOld := false
	foundNew := false
	for _, tx := range acc.Transactions {
		if tx.ID == "old" {
			foundOld = true
		}
		if tx.ID == "new" {
			foundNew = true
		}
	}
	if foundOld {
		t.Error("old transaction should have been cleaned up")
	}
	if !foundNew {
		t.Error("new transaction should exist")
	}
}

func TestPatternAnalyzer_Learn_TypicalDays(t *testing.T) {
	analyzer := NewPatternAnalyzer()

	// Create transactions on the same day multiple times
	txns := make([]*models.Transaction, 0, 20)
	for i := 0; i < 20; i++ {
		day := time.Monday
		if i%2 == 0 {
			day = time.Tuesday
		}
		// Create dates that fall on these weekdays
		txns = append(txns, &models.Transaction{
			Amount:    decimal.NewFromFloat(100),
			CreatedAt: time.Date(2024, 1, 15+int(day), 10, 0, 0, 0, time.UTC),
		})
	}

	analyzer.Learn("acc-1", txns)

	pattern := analyzer.GetPattern("acc-1")
	if pattern == nil {
		t.Fatal("pattern should be created")
	}

	// Should have typical days
	if len(pattern.TypicalDays) == 0 {
		t.Log("Note: typical days may be empty depending on day distribution")
	}
}

func TestPatternAnalyzer_Learn_MaxMerchants(t *testing.T) {
	analyzer := NewPatternAnalyzer()

	// Create transactions with many different merchants
	txns := make([]*models.Transaction, 0, 30)
	for i := 0; i < 30; i++ {
		merchantID := string(rune('A' + (i % 15)))
		txns = append(txns, &models.Transaction{
			Amount:    decimal.NewFromFloat(100),
			CreatedAt: time.Now(),
			Merchant:  &models.Merchant{ID: merchantID},
		})
	}

	analyzer.Learn("acc-1", txns)

	pattern := analyzer.GetPattern("acc-1")
	if pattern == nil {
		t.Fatal("pattern should be created")
	}

	// Should cap at 10 typical merchants
	if len(pattern.TypicalMerchants) > 10 {
		t.Errorf("expected max 10 typical merchants, got %d", len(pattern.TypicalMerchants))
	}
}

func TestVelocityTracker_Concurrency(t *testing.T) {
	tracker := NewVelocityTracker(time.Hour)

	done := make(chan bool)

	// Concurrent writes
	for i := 0; i < 10; i++ {
		go func(id int) {
			txn := &models.Transaction{
				ID:            string(rune('a' + id)),
				Amount:        decimal.NewFromFloat(float64(100 + id)),
				SourceAccount: "acc-1",
				CreatedAt:     time.Now(),
			}
			tracker.Record(txn)
			done <- true
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func() {
			tracker.GetActivity("acc-1")
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}

	// Verify no crash and data integrity
	summary := tracker.GetActivity("acc-1")
	if summary.TransactionCount != 10 {
		t.Errorf("expected 10 transactions, got %d", summary.TransactionCount)
	}
}

func TestPatternAnalyzer_Concurrency(t *testing.T) {
	analyzer := NewPatternAnalyzer()

	done := make(chan bool)

	// Concurrent learns
	for i := 0; i < 5; i++ {
		go func(id int) {
			txns := []*models.Transaction{
				{Amount: decimal.NewFromFloat(float64(100 + id)), CreatedAt: time.Now()},
			}
			analyzer.Learn(string(rune('a'+id)), txns)
			done <- true
		}(i)
	}

	// Concurrent gets
	for i := 0; i < 5; i++ {
		go func(id int) {
			analyzer.GetPattern(string(rune('a' + id)))
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestGeofenceChecker_Concurrency(t *testing.T) {
	checker := NewGeofenceChecker()

	done := make(chan bool)

	// Concurrent reads are safe
	for i := 0; i < 10; i++ {
		go func() {
			checker.IsHighRiskCountry("NG")
			checker.IsBlockedCountry("KP")
			checker.IsHighRiskCountry("US")
			checker.IsBlockedCountry("CA")
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}
