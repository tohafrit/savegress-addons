package transactions

import (
	"testing"

	"github.com/savegress/finsight/pkg/models"
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
