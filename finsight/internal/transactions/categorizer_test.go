package transactions

import (
	"context"
	"testing"
	"time"

	"github.com/savegress/finsight/internal/config"
	"github.com/savegress/finsight/pkg/models"
	"github.com/shopspring/decimal"
)

func TestNewCategorizer(t *testing.T) {
	c := NewCategorizer()

	if c == nil {
		t.Fatal("expected non-nil categorizer")
	}
	if len(c.mccCategories) == 0 {
		t.Error("MCC categories should be initialized")
	}
	if len(c.merchantPatterns) == 0 {
		t.Error("merchant patterns should be initialized")
	}
	if len(c.descriptionRules) == 0 {
		t.Error("description rules should be initialized")
	}
}

func TestCategorizer_Categorize_ByMCC(t *testing.T) {
	c := NewCategorizer()

	tests := []struct {
		name     string
		mcc      string
		expected string
	}{
		{"grocery store 5411", "5411", CategoryGroceries},
		{"restaurant 5812", "5812", CategoryRestaurants},
		{"gas station 5541", "5541", CategoryTransportation},
		{"utilities 4900", "4900", CategoryUtilities},
		{"movie theater 7832", "7832", CategoryEntertainment},
		{"department store 5311", "5311", CategoryShopping},
		{"pharmacy 5912", "5912", CategoryHealthcare},
		{"airline 4511", "4511", CategoryTravel},
		{"school 8211", "8211", CategoryEducation},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			txn := &models.Transaction{
				Merchant: &models.Merchant{
					MCC: tt.mcc,
				},
			}
			got := c.Categorize(txn)
			if got != tt.expected {
				t.Errorf("Categorize(MCC=%s) = %s, want %s", tt.mcc, got, tt.expected)
			}
		})
	}
}

func TestCategorizer_Categorize_ByMerchantName(t *testing.T) {
	c := NewCategorizer()

	tests := []struct {
		name         string
		merchantName string
		expected     string
	}{
		{"walmart", "WALMART", CategoryGroceries},
		{"target", "Target Store #123", CategoryGroceries},
		{"starbucks", "STARBUCKS #12345", CategoryRestaurants},
		{"uber", "UBER TRIP", CategoryTransportation},
		{"netflix", "Netflix.com", CategoryEntertainment},
		{"amazon", "AMAZON MARKETPLACE", CategoryShopping},
		{"walgreens", "WALGREENS #12345", CategoryHealthcare},
		{"airbnb", "Airbnb Booking", CategoryTravel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			txn := &models.Transaction{
				Merchant: &models.Merchant{
					Name: tt.merchantName,
				},
			}
			got := c.Categorize(txn)
			if got != tt.expected {
				t.Errorf("Categorize(Merchant=%s) = %s, want %s", tt.merchantName, got, tt.expected)
			}
		})
	}
}

func TestCategorizer_Categorize_ByDescription(t *testing.T) {
	c := NewCategorizer()

	tests := []struct {
		name        string
		description string
		expected    string
	}{
		{"payroll", "PAYROLL DIRECT DEPOSIT", CategoryIncome},
		{"salary", "Monthly salary payment", CategoryIncome},
		{"interest", "Interest payment", CategoryIncome},
		{"dividend", "Quarterly Dividend", CategoryInvestment},
		{"transfer", "Transfer from savings", CategoryTransfer},
		{"ach", "ACH Credit", CategoryTransfer},
		{"fee", "Monthly Fee", CategoryFees},
		{"overdraft", "Overdraft charge", CategoryFees},
		{"insurance premium", "Auto Insurance Premium", CategoryInsurance},
		{"tuition", "University Tuition", CategoryEducation},
		{"gym membership", "Gold's Gym Monthly Membership", CategoryHealthcare},
		{"parking", "PARKING GARAGE EXIT TOLL", CategoryTransportation},
		{"toll", "Toll road", CategoryTransportation},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			txn := &models.Transaction{
				Description: tt.description,
			}
			got := c.Categorize(txn)
			if got != tt.expected {
				t.Errorf("Categorize(Description=%s) = %s, want %s", tt.description, got, tt.expected)
			}
		})
	}
}

func TestCategorizer_Categorize_ByTransactionType(t *testing.T) {
	c := NewCategorizer()

	tests := []struct {
		name     string
		txnType  models.TransactionType
		expected string
	}{
		{"credit", models.TransactionTypeCredit, CategoryIncome},
		{"transfer", models.TransactionTypeTransfer, CategoryTransfer},
		{"fee", models.TransactionTypeFee, CategoryFees},
		{"interest", models.TransactionTypeInterest, CategoryIncome},
		{"refund", models.TransactionTypeRefund, CategoryOther},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			txn := &models.Transaction{
				Type: tt.txnType,
			}
			got := c.Categorize(txn)
			if got != tt.expected {
				t.Errorf("Categorize(Type=%s) = %s, want %s", tt.txnType, got, tt.expected)
			}
		})
	}
}

func TestCategorizer_Categorize_Priority(t *testing.T) {
	c := NewCategorizer()

	// MCC takes priority over merchant name
	txn := &models.Transaction{
		Merchant: &models.Merchant{
			MCC:  "5812", // Restaurant
			Name: "Amazon", // Would match shopping
		},
	}
	got := c.Categorize(txn)
	if got != CategoryRestaurants {
		t.Errorf("MCC should take priority, got %s, want %s", got, CategoryRestaurants)
	}
}

func TestCategorizer_Categorize_DefaultOther(t *testing.T) {
	c := NewCategorizer()

	txn := &models.Transaction{
		Description: "Random unknown transaction",
	}
	got := c.Categorize(txn)
	if got != CategoryOther {
		t.Errorf("Categorize unknown = %s, want %s", got, CategoryOther)
	}
}

func TestCategorizer_Categorize_NilMerchant(t *testing.T) {
	c := NewCategorizer()

	txn := &models.Transaction{
		Merchant:    nil,
		Description: "PAYROLL",
	}

	// Should not panic
	got := c.Categorize(txn)
	if got != CategoryIncome {
		t.Errorf("got %s, want %s", got, CategoryIncome)
	}
}

func TestCategorizer_AddMCCMapping(t *testing.T) {
	c := NewCategorizer()

	// Add custom mapping
	c.AddMCCMapping("9999", "custom_category")

	txn := &models.Transaction{
		Merchant: &models.Merchant{
			MCC: "9999",
		},
	}
	got := c.Categorize(txn)
	if got != "custom_category" {
		t.Errorf("custom MCC = %s, want custom_category", got)
	}
}

func TestCategorizer_AddMerchantPattern(t *testing.T) {
	c := NewCategorizer()

	// Add custom pattern
	err := c.AddMerchantPattern(`(?i)custom\s*merchant`, "custom_category")
	if err != nil {
		t.Fatalf("AddMerchantPattern failed: %v", err)
	}

	txn := &models.Transaction{
		Merchant: &models.Merchant{
			Name: "CUSTOM MERCHANT #123",
		},
	}
	got := c.Categorize(txn)
	if got != "custom_category" {
		t.Errorf("custom pattern = %s, want custom_category", got)
	}
}

func TestCategorizer_AddMerchantPattern_InvalidRegex(t *testing.T) {
	c := NewCategorizer()

	err := c.AddMerchantPattern(`[invalid`, "category")
	if err == nil {
		t.Error("expected error for invalid regex")
	}
}

func TestCategorizer_AddDescriptionRule(t *testing.T) {
	c := NewCategorizer()

	// Add custom rule with high priority
	err := c.AddDescriptionRule(`(?i)custom\s*rule`, "custom_category", 200)
	if err != nil {
		t.Fatalf("AddDescriptionRule failed: %v", err)
	}

	txn := &models.Transaction{
		Description: "Custom Rule Transaction",
	}
	got := c.Categorize(txn)
	if got != "custom_category" {
		t.Errorf("custom rule = %s, want custom_category", got)
	}
}

func TestCategorizer_AddDescriptionRule_InvalidRegex(t *testing.T) {
	c := NewCategorizer()

	err := c.AddDescriptionRule(`[invalid`, "category", 100)
	if err == nil {
		t.Error("expected error for invalid regex")
	}
}

func TestCategorizer_DescriptionRule_Priority(t *testing.T) {
	c := NewCategorizer()

	// Description matches both payroll (100) and fee (70)
	txn := &models.Transaction{
		Description: "PAYROLL FEE ADJUSTMENT",
	}

	// Higher priority rule (payroll = 100) should win
	got := c.Categorize(txn)
	if got != CategoryIncome {
		t.Errorf("higher priority rule should win, got %s, want %s", got, CategoryIncome)
	}
}

func TestCategory_Constants(t *testing.T) {
	categories := []string{
		CategoryGroceries,
		CategoryRestaurants,
		CategoryTransportation,
		CategoryUtilities,
		CategoryEntertainment,
		CategoryShopping,
		CategoryHealthcare,
		CategoryTravel,
		CategoryFees,
		CategoryTransfer,
		CategoryIncome,
		CategoryInvestment,
		CategoryInsurance,
		CategoryEducation,
		CategorySubscription,
		CategoryOther,
	}

	for _, cat := range categories {
		if cat == "" {
			t.Error("category constant should not be empty")
		}
	}
}

func TestCategorizationRule_Struct(t *testing.T) {
	c := NewCategorizer()

	if len(c.descriptionRules) == 0 {
		t.Fatal("expected description rules")
	}

	rule := c.descriptionRules[0]
	if rule.Pattern == nil {
		t.Error("Pattern should be set")
	}
	if rule.Category == "" {
		t.Error("Category should be set")
	}
	if rule.Priority == 0 {
		t.Error("Priority should be set")
	}
}

// Aggregator tests

func TestNewAggregator(t *testing.T) {
	a := NewAggregator()

	if a == nil {
		t.Fatal("expected non-nil aggregator")
	}
	if a.byType == nil {
		t.Error("byType should be initialized")
	}
	if a.byCategory == nil {
		t.Error("byCategory should be initialized")
	}
	if a.byStatus == nil {
		t.Error("byStatus should be initialized")
	}
	if len(a.hourlyVolume) != 24 {
		t.Errorf("hourlyVolume len = %d, want 24", len(a.hourlyVolume))
	}
}

func TestAggregator_Add(t *testing.T) {
	a := NewAggregator()

	txn := &models.Transaction{
		ID:        "TXN-001",
		Type:      models.TransactionTypeDebit,
		Status:    models.TransactionStatusPending,
		Amount:    decimal.NewFromFloat(100.50),
		Category:  CategoryShopping,
		CreatedAt: time.Now(),
	}

	a.Add(txn)

	if a.totalCount != 1 {
		t.Errorf("totalCount = %d, want 1", a.totalCount)
	}
	if !a.totalVolume.Equal(decimal.NewFromFloat(100.50)) {
		t.Errorf("totalVolume = %s, want 100.50", a.totalVolume)
	}
	if a.byType[string(models.TransactionTypeDebit)] == nil {
		t.Fatal("byType should have debit entry")
	}
	if a.byType[string(models.TransactionTypeDebit)].Count != 1 {
		t.Error("byType debit count should be 1")
	}
	if a.byCategory[CategoryShopping] == nil {
		t.Fatal("byCategory should have shopping entry")
	}
	if a.byCategory[CategoryShopping].Count != 1 {
		t.Error("byCategory shopping count should be 1")
	}
	if a.byStatus[string(models.TransactionStatusPending)] != 1 {
		t.Error("byStatus pending should be 1")
	}
}

func TestAggregator_Add_MultipleTransactions(t *testing.T) {
	a := NewAggregator()

	txns := []*models.Transaction{
		{Type: models.TransactionTypeDebit, Amount: decimal.NewFromFloat(100), Category: CategoryShopping, Status: models.TransactionStatusPending, CreatedAt: time.Now()},
		{Type: models.TransactionTypeDebit, Amount: decimal.NewFromFloat(50), Category: CategoryShopping, Status: models.TransactionStatusCompleted, CreatedAt: time.Now()},
		{Type: models.TransactionTypeCredit, Amount: decimal.NewFromFloat(200), Category: CategoryIncome, Status: models.TransactionStatusCompleted, CreatedAt: time.Now()},
	}

	for _, txn := range txns {
		a.Add(txn)
	}

	if a.totalCount != 3 {
		t.Errorf("totalCount = %d, want 3", a.totalCount)
	}
	if !a.totalVolume.Equal(decimal.NewFromFloat(350)) {
		t.Errorf("totalVolume = %s, want 350", a.totalVolume)
	}

	// Check type averages
	debitStats := a.byType[string(models.TransactionTypeDebit)]
	if debitStats.Count != 2 {
		t.Errorf("debit count = %d, want 2", debitStats.Count)
	}
	// Average should be 75 (100+50)/2
	expectedAvg := decimal.NewFromFloat(75)
	if !debitStats.Average.Equal(expectedAvg) {
		t.Errorf("debit average = %s, want %s", debitStats.Average, expectedAvg)
	}
}

func TestAggregator_Add_NoCategory(t *testing.T) {
	a := NewAggregator()

	txn := &models.Transaction{
		Type:      models.TransactionTypeDebit,
		Amount:    decimal.NewFromFloat(100),
		Category:  "", // Empty category
		CreatedAt: time.Now(),
	}

	a.Add(txn)

	// Should not panic and should not add to byCategory
	if len(a.byCategory) != 0 {
		t.Error("byCategory should be empty for transactions without category")
	}
}

func TestAggregator_Remove(t *testing.T) {
	a := NewAggregator()

	txn := &models.Transaction{
		Type:      models.TransactionTypeDebit,
		Status:    models.TransactionStatusPending,
		Amount:    decimal.NewFromFloat(100),
		Category:  CategoryShopping,
		CreatedAt: time.Now(),
	}

	a.Add(txn)
	a.Remove(txn)

	if a.totalCount != 0 {
		t.Errorf("totalCount after remove = %d, want 0", a.totalCount)
	}
	if !a.totalVolume.Equal(decimal.Zero) {
		t.Errorf("totalVolume after remove = %s, want 0", a.totalVolume)
	}
}

func TestAggregator_Remove_SetsAverageZero(t *testing.T) {
	a := NewAggregator()

	txn := &models.Transaction{
		Type:      models.TransactionTypeDebit,
		Amount:    decimal.NewFromFloat(100),
		CreatedAt: time.Now(),
	}

	a.Add(txn)
	a.Remove(txn)

	stats := a.byType[string(models.TransactionTypeDebit)]
	if stats.Count != 0 {
		t.Errorf("count after remove = %d, want 0", stats.Count)
	}
	if !stats.Average.Equal(decimal.Zero) {
		t.Errorf("average after remove = %s, want 0", stats.Average)
	}
}

func TestAggregator_Remove_NonExistent(t *testing.T) {
	a := NewAggregator()

	txn := &models.Transaction{
		Type:      models.TransactionTypeDebit,
		Amount:    decimal.NewFromFloat(100),
		Category:  CategoryShopping,
		Status:    models.TransactionStatusPending,
		CreatedAt: time.Now(),
	}

	// Remove without adding first - should not panic
	a.Remove(txn)

	if a.totalCount != -1 {
		t.Logf("totalCount after remove without add = %d", a.totalCount)
	}
}

func TestAggregator_GetStats(t *testing.T) {
	a := NewAggregator()

	txns := []*models.Transaction{
		{Type: models.TransactionTypeDebit, Amount: decimal.NewFromFloat(100), Category: CategoryShopping, Status: models.TransactionStatusPending, CreatedAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)},
		{Type: models.TransactionTypeCredit, Amount: decimal.NewFromFloat(200), Category: CategoryIncome, Status: models.TransactionStatusCompleted, CreatedAt: time.Date(2024, 1, 15, 14, 0, 0, 0, time.UTC)},
	}

	for _, txn := range txns {
		a.Add(txn)
	}

	stats := a.GetStats()

	if stats.TotalCount != 2 {
		t.Errorf("TotalCount = %d, want 2", stats.TotalCount)
	}
	if !stats.TotalVolume.Equal(decimal.NewFromFloat(300)) {
		t.Errorf("TotalVolume = %s, want 300", stats.TotalVolume)
	}
	// Average = 300/2 = 150
	if !stats.AverageAmount.Equal(decimal.NewFromFloat(150)) {
		t.Errorf("AverageAmount = %s, want 150", stats.AverageAmount)
	}
	if len(stats.ByType) != 2 {
		t.Errorf("ByType len = %d, want 2", len(stats.ByType))
	}
	if len(stats.ByCategory) != 2 {
		t.Errorf("ByCategory len = %d, want 2", len(stats.ByCategory))
	}
}

func TestAggregator_GetStats_Empty(t *testing.T) {
	a := NewAggregator()

	stats := a.GetStats()

	if stats.TotalCount != 0 {
		t.Errorf("TotalCount = %d, want 0", stats.TotalCount)
	}
	if !stats.AverageAmount.Equal(decimal.Zero) {
		t.Errorf("AverageAmount = %s, want 0", stats.AverageAmount)
	}
}

func TestAggregator_GetHourlyVolume(t *testing.T) {
	a := NewAggregator()

	txn := &models.Transaction{
		Type:      models.TransactionTypeDebit,
		Amount:    decimal.NewFromFloat(100),
		CreatedAt: time.Now(),
	}
	a.Add(txn)

	hourlyStats := a.GetHourlyVolume()

	if len(hourlyStats) != 24 {
		t.Errorf("hourlyStats len = %d, want 24", len(hourlyStats))
	}
}

func TestAggregator_GetDailyVolume(t *testing.T) {
	a := NewAggregator()

	date := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	txn := &models.Transaction{
		Type:      models.TransactionTypeDebit,
		Amount:    decimal.NewFromFloat(100),
		CreatedAt: date,
	}
	a.Add(txn)

	startDate := time.Date(2024, 1, 14, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC)

	dailyStats := a.GetDailyVolume(startDate, endDate)

	if len(dailyStats) != 3 { // 14, 15, 16
		t.Errorf("dailyStats len = %d, want 3", len(dailyStats))
	}

	// Check that Jan 15 has volume
	for _, ds := range dailyStats {
		if ds.Date.Day() == 15 && ds.Date.Month() == 1 {
			if !ds.Volume.Equal(decimal.NewFromFloat(100)) {
				t.Errorf("Jan 15 volume = %s, want 100", ds.Volume)
			}
		}
	}
}

func TestAggregator_Reset(t *testing.T) {
	a := NewAggregator()

	txn := &models.Transaction{
		Type:      models.TransactionTypeDebit,
		Amount:    decimal.NewFromFloat(100),
		Category:  CategoryShopping,
		CreatedAt: time.Now(),
	}
	a.Add(txn)

	a.Reset()

	if a.totalCount != 0 {
		t.Errorf("totalCount after reset = %d, want 0", a.totalCount)
	}
	if !a.totalVolume.Equal(decimal.Zero) {
		t.Errorf("totalVolume after reset = %s, want 0", a.totalVolume)
	}
	if len(a.byType) != 0 {
		t.Errorf("byType len after reset = %d, want 0", len(a.byType))
	}
}

// Engine tests

func TestNewEngine(t *testing.T) {
	cfg := &config.TransactionsConfig{
		BatchSize:             100,
		ProcessInterval:       5 * time.Second,
		CategorizationEnabled: true,
	}

	e := NewEngine(cfg)

	if e == nil {
		t.Fatal("expected non-nil engine")
	}
	if e.transactions == nil {
		t.Error("transactions map should be initialized")
	}
	if e.accounts == nil {
		t.Error("accounts map should be initialized")
	}
	if e.categorizer == nil {
		t.Error("categorizer should be initialized")
	}
	if e.aggregator == nil {
		t.Error("aggregator should be initialized")
	}
}

func TestEngine_StartStop(t *testing.T) {
	cfg := &config.TransactionsConfig{
		ProcessInterval: 100 * time.Millisecond,
	}
	e := NewEngine(cfg)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start
	err := e.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if !e.running {
		t.Error("engine should be running")
	}

	// Start again - should be idempotent
	err = e.Start(ctx)
	if err != nil {
		t.Fatalf("second Start failed: %v", err)
	}

	// Stop
	e.Stop()
	if e.running {
		t.Error("engine should not be running after Stop")
	}

	// Stop again - should be idempotent
	e.Stop()
}

func TestEngine_ProcessTransaction(t *testing.T) {
	cfg := &config.TransactionsConfig{
		CategorizationEnabled: true,
	}
	e := NewEngine(cfg)
	ctx := context.Background()

	txn := &models.Transaction{
		ID:          "TXN-001",
		Type:        models.TransactionTypeDebit,
		Amount:      decimal.NewFromFloat(100),
		Description: "WALMART STORE",
		CreatedAt:   time.Now(),
	}

	err := e.ProcessTransaction(ctx, txn)

	if err != nil {
		t.Fatalf("ProcessTransaction failed: %v", err)
	}
	if txn.Category == "" {
		t.Error("Category should be set when categorization enabled")
	}

	// Verify stored
	got, ok := e.GetTransaction("TXN-001")
	if !ok {
		t.Fatal("transaction should be stored")
	}
	if got.ID != "TXN-001" {
		t.Errorf("ID = %s, want TXN-001", got.ID)
	}
}

func TestEngine_ProcessTransaction_NoCategorization(t *testing.T) {
	cfg := &config.TransactionsConfig{
		CategorizationEnabled: false,
	}
	e := NewEngine(cfg)
	ctx := context.Background()

	txn := &models.Transaction{
		ID:          "TXN-001",
		Type:        models.TransactionTypeDebit,
		Amount:      decimal.NewFromFloat(100),
		Description: "WALMART STORE",
	}

	err := e.ProcessTransaction(ctx, txn)

	if err != nil {
		t.Fatalf("ProcessTransaction failed: %v", err)
	}
	if txn.Category != "" {
		t.Errorf("Category should be empty when categorization disabled, got %s", txn.Category)
	}
}

func TestEngine_ProcessTransaction_PreserveCategory(t *testing.T) {
	cfg := &config.TransactionsConfig{
		CategorizationEnabled: true,
	}
	e := NewEngine(cfg)
	ctx := context.Background()

	txn := &models.Transaction{
		ID:       "TXN-001",
		Category: "custom_category", // Pre-set category
	}

	err := e.ProcessTransaction(ctx, txn)

	if err != nil {
		t.Fatalf("ProcessTransaction failed: %v", err)
	}
	if txn.Category != "custom_category" {
		t.Errorf("Category should be preserved, got %s", txn.Category)
	}
}

func TestEngine_GetTransaction_NotFound(t *testing.T) {
	cfg := &config.TransactionsConfig{}
	e := NewEngine(cfg)

	_, ok := e.GetTransaction("NONEXISTENT")
	if ok {
		t.Error("should not find nonexistent transaction")
	}
}

func TestEngine_GetTransactions(t *testing.T) {
	cfg := &config.TransactionsConfig{}
	e := NewEngine(cfg)
	ctx := context.Background()

	txns := []*models.Transaction{
		{ID: "TXN-001", Type: models.TransactionTypeDebit, Category: CategoryShopping, Status: models.TransactionStatusPending, Amount: decimal.NewFromFloat(100), CreatedAt: time.Now()},
		{ID: "TXN-002", Type: models.TransactionTypeCredit, Category: CategoryIncome, Status: models.TransactionStatusCompleted, Amount: decimal.NewFromFloat(200), CreatedAt: time.Now()},
		{ID: "TXN-003", Type: models.TransactionTypeDebit, Category: CategoryShopping, Status: models.TransactionStatusPending, Amount: decimal.NewFromFloat(50), CreatedAt: time.Now()},
	}

	for _, txn := range txns {
		e.ProcessTransaction(ctx, txn)
	}

	// Filter by type
	debitTxns := e.GetTransactions(TransactionFilter{Type: models.TransactionTypeDebit})
	if len(debitTxns) != 2 {
		t.Errorf("debit count = %d, want 2", len(debitTxns))
	}

	// Filter by status
	pendingTxns := e.GetTransactions(TransactionFilter{Status: models.TransactionStatusPending})
	if len(pendingTxns) != 2 {
		t.Errorf("pending count = %d, want 2", len(pendingTxns))
	}

	// Filter by category
	shoppingTxns := e.GetTransactions(TransactionFilter{Category: CategoryShopping})
	if len(shoppingTxns) != 2 {
		t.Errorf("shopping count = %d, want 2", len(shoppingTxns))
	}
}

func TestEngine_GetTransactions_AmountFilters(t *testing.T) {
	cfg := &config.TransactionsConfig{}
	e := NewEngine(cfg)
	ctx := context.Background()

	txns := []*models.Transaction{
		{ID: "TXN-001", Amount: decimal.NewFromFloat(50), CreatedAt: time.Now()},
		{ID: "TXN-002", Amount: decimal.NewFromFloat(100), CreatedAt: time.Now()},
		{ID: "TXN-003", Amount: decimal.NewFromFloat(200), CreatedAt: time.Now()},
	}

	for _, txn := range txns {
		e.ProcessTransaction(ctx, txn)
	}

	minAmount := decimal.NewFromFloat(75)
	filtered := e.GetTransactions(TransactionFilter{MinAmount: &minAmount})
	if len(filtered) != 2 {
		t.Errorf("min amount filter count = %d, want 2", len(filtered))
	}

	maxAmount := decimal.NewFromFloat(150)
	filtered = e.GetTransactions(TransactionFilter{MaxAmount: &maxAmount})
	if len(filtered) != 2 {
		t.Errorf("max amount filter count = %d, want 2", len(filtered))
	}
}

func TestEngine_GetTransactions_DateFilters(t *testing.T) {
	cfg := &config.TransactionsConfig{}
	e := NewEngine(cfg)
	ctx := context.Background()

	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	tomorrow := now.Add(24 * time.Hour)

	txn := &models.Transaction{
		ID:        "TXN-001",
		Amount:    decimal.NewFromFloat(100),
		CreatedAt: now,
	}
	e.ProcessTransaction(ctx, txn)

	// Start date filter
	filtered := e.GetTransactions(TransactionFilter{StartDate: &yesterday})
	if len(filtered) != 1 {
		t.Errorf("start date filter count = %d, want 1", len(filtered))
	}

	filtered = e.GetTransactions(TransactionFilter{StartDate: &tomorrow})
	if len(filtered) != 0 {
		t.Errorf("future start date filter count = %d, want 0", len(filtered))
	}

	// End date filter
	filtered = e.GetTransactions(TransactionFilter{EndDate: &tomorrow})
	if len(filtered) != 1 {
		t.Errorf("end date filter count = %d, want 1", len(filtered))
	}

	filtered = e.GetTransactions(TransactionFilter{EndDate: &yesterday})
	if len(filtered) != 0 {
		t.Errorf("past end date filter count = %d, want 0", len(filtered))
	}
}

func TestEngine_GetTransactions_AccountFilter(t *testing.T) {
	cfg := &config.TransactionsConfig{}
	e := NewEngine(cfg)
	ctx := context.Background()

	txns := []*models.Transaction{
		{ID: "TXN-001", SourceAccount: "ACC-001", DestAccount: "ACC-002", Amount: decimal.NewFromFloat(100), CreatedAt: time.Now()},
		{ID: "TXN-002", SourceAccount: "ACC-002", DestAccount: "ACC-003", Amount: decimal.NewFromFloat(50), CreatedAt: time.Now()},
		{ID: "TXN-003", SourceAccount: "ACC-003", DestAccount: "ACC-001", Amount: decimal.NewFromFloat(75), CreatedAt: time.Now()},
	}

	for _, txn := range txns {
		e.ProcessTransaction(ctx, txn)
	}

	// ACC-001 appears in TXN-001 (source) and TXN-003 (dest)
	filtered := e.GetTransactions(TransactionFilter{AccountID: "ACC-001"})
	if len(filtered) != 2 {
		t.Errorf("account filter count = %d, want 2", len(filtered))
	}
}

func TestEngine_GetTransactions_MerchantFilter(t *testing.T) {
	cfg := &config.TransactionsConfig{}
	e := NewEngine(cfg)
	ctx := context.Background()

	txns := []*models.Transaction{
		{ID: "TXN-001", Merchant: &models.Merchant{ID: "M-001"}, Amount: decimal.NewFromFloat(100), CreatedAt: time.Now()},
		{ID: "TXN-002", Merchant: &models.Merchant{ID: "M-002"}, Amount: decimal.NewFromFloat(50), CreatedAt: time.Now()},
		{ID: "TXN-003", Merchant: nil, Amount: decimal.NewFromFloat(75), CreatedAt: time.Now()},
	}

	for _, txn := range txns {
		e.ProcessTransaction(ctx, txn)
	}

	filtered := e.GetTransactions(TransactionFilter{MerchantID: "M-001"})
	if len(filtered) != 1 {
		t.Errorf("merchant filter count = %d, want 1", len(filtered))
	}
}

func TestEngine_CreateAccount(t *testing.T) {
	cfg := &config.TransactionsConfig{}
	e := NewEngine(cfg)

	acc := &models.Account{
		ID:            "ACC-001",
		AccountNumber: "1234567890",
		Balance:       decimal.NewFromFloat(1000),
		AvailableBal:  decimal.NewFromFloat(1000),
	}

	err := e.CreateAccount(acc)

	if err != nil {
		t.Fatalf("CreateAccount failed: %v", err)
	}
	if acc.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}

	got, ok := e.GetAccount("ACC-001")
	if !ok {
		t.Fatal("account should be stored")
	}
	if got.AccountNumber != "1234567890" {
		t.Errorf("AccountNumber = %s, want 1234567890", got.AccountNumber)
	}
}

func TestEngine_GetAccount_NotFound(t *testing.T) {
	cfg := &config.TransactionsConfig{}
	e := NewEngine(cfg)

	_, ok := e.GetAccount("NONEXISTENT")
	if ok {
		t.Error("should not find nonexistent account")
	}
}

func TestEngine_UpdateAccountBalances_Debit(t *testing.T) {
	cfg := &config.TransactionsConfig{}
	e := NewEngine(cfg)
	ctx := context.Background()

	acc := &models.Account{
		ID:           "ACC-001",
		Balance:      decimal.NewFromFloat(1000),
		AvailableBal: decimal.NewFromFloat(1000),
	}
	e.CreateAccount(acc)

	txn := &models.Transaction{
		ID:            "TXN-001",
		Type:          models.TransactionTypeDebit,
		SourceAccount: "ACC-001",
		Amount:        decimal.NewFromFloat(100),
		CreatedAt:     time.Now(),
	}
	e.ProcessTransaction(ctx, txn)

	got, _ := e.GetAccount("ACC-001")
	if !got.Balance.Equal(decimal.NewFromFloat(900)) {
		t.Errorf("Balance = %s, want 900", got.Balance)
	}
}

func TestEngine_UpdateAccountBalances_Credit(t *testing.T) {
	cfg := &config.TransactionsConfig{}
	e := NewEngine(cfg)
	ctx := context.Background()

	acc := &models.Account{
		ID:           "ACC-001",
		Balance:      decimal.NewFromFloat(1000),
		AvailableBal: decimal.NewFromFloat(1000),
	}
	e.CreateAccount(acc)

	txn := &models.Transaction{
		ID:            "TXN-001",
		Type:          models.TransactionTypeCredit,
		SourceAccount: "ACC-001",
		Amount:        decimal.NewFromFloat(100),
		CreatedAt:     time.Now(),
	}
	e.ProcessTransaction(ctx, txn)

	got, _ := e.GetAccount("ACC-001")
	if !got.Balance.Equal(decimal.NewFromFloat(1100)) {
		t.Errorf("Balance = %s, want 1100", got.Balance)
	}
}

func TestEngine_UpdateAccountBalances_Transfer(t *testing.T) {
	cfg := &config.TransactionsConfig{}
	e := NewEngine(cfg)
	ctx := context.Background()

	srcAcc := &models.Account{
		ID:           "ACC-001",
		Balance:      decimal.NewFromFloat(1000),
		AvailableBal: decimal.NewFromFloat(1000),
	}
	dstAcc := &models.Account{
		ID:           "ACC-002",
		Balance:      decimal.NewFromFloat(500),
		AvailableBal: decimal.NewFromFloat(500),
	}
	e.CreateAccount(srcAcc)
	e.CreateAccount(dstAcc)

	txn := &models.Transaction{
		ID:            "TXN-001",
		Type:          models.TransactionTypeTransfer,
		SourceAccount: "ACC-001",
		DestAccount:   "ACC-002",
		Amount:        decimal.NewFromFloat(100),
		CreatedAt:     time.Now(),
	}
	e.ProcessTransaction(ctx, txn)

	src, _ := e.GetAccount("ACC-001")
	dst, _ := e.GetAccount("ACC-002")

	if !src.Balance.Equal(decimal.NewFromFloat(900)) {
		t.Errorf("Source Balance = %s, want 900", src.Balance)
	}
	if !dst.Balance.Equal(decimal.NewFromFloat(600)) {
		t.Errorf("Dest Balance = %s, want 600", dst.Balance)
	}
}

func TestEngine_GetAccountTransactions(t *testing.T) {
	cfg := &config.TransactionsConfig{}
	e := NewEngine(cfg)
	ctx := context.Background()

	txns := []*models.Transaction{
		{ID: "TXN-001", SourceAccount: "ACC-001", Amount: decimal.NewFromFloat(100), CreatedAt: time.Now()},
		{ID: "TXN-002", SourceAccount: "ACC-002", Amount: decimal.NewFromFloat(50), CreatedAt: time.Now()},
	}

	for _, txn := range txns {
		e.ProcessTransaction(ctx, txn)
	}

	accTxns := e.GetAccountTransactions("ACC-001", 10)
	if len(accTxns) != 1 {
		t.Errorf("account transactions count = %d, want 1", len(accTxns))
	}
}

func TestEngine_GetStats(t *testing.T) {
	cfg := &config.TransactionsConfig{}
	e := NewEngine(cfg)
	ctx := context.Background()

	txn := &models.Transaction{
		ID:        "TXN-001",
		Type:      models.TransactionTypeDebit,
		Amount:    decimal.NewFromFloat(100),
		Category:  CategoryShopping,
		CreatedAt: time.Now(),
	}
	e.ProcessTransaction(ctx, txn)

	stats := e.GetStats()

	if stats.TotalCount != 1 {
		t.Errorf("TotalCount = %d, want 1", stats.TotalCount)
	}
	if !stats.TotalVolume.Equal(decimal.NewFromFloat(100)) {
		t.Errorf("TotalVolume = %s, want 100", stats.TotalVolume)
	}
}

func TestTransactionFilter_EmptyFilter(t *testing.T) {
	cfg := &config.TransactionsConfig{}
	e := NewEngine(cfg)
	ctx := context.Background()

	txn := &models.Transaction{
		ID:        "TXN-001",
		Amount:    decimal.NewFromFloat(100),
		CreatedAt: time.Now(),
	}
	e.ProcessTransaction(ctx, txn)

	// Empty filter should match all
	filtered := e.GetTransactions(TransactionFilter{})
	if len(filtered) != 1 {
		t.Errorf("empty filter count = %d, want 1", len(filtered))
	}
}
