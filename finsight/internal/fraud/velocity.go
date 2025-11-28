package fraud

import (
	"sync"
	"time"

	"github.com/savegress/finsight/pkg/models"
	"github.com/shopspring/decimal"
)

// VelocityTracker tracks transaction velocity per account
type VelocityTracker struct {
	window   time.Duration
	accounts map[string]*AccountVelocity
	mu       sync.RWMutex
}

// AccountVelocity tracks velocity for a single account
type AccountVelocity struct {
	Transactions []*TransactionRecord
	LastCleanup  time.Time
}

// TransactionRecord is a minimal transaction record for velocity tracking
type TransactionRecord struct {
	ID        string
	Amount    decimal.Decimal
	Timestamp time.Time
	Location  string
	Merchant  string
}

// NewVelocityTracker creates a new velocity tracker
func NewVelocityTracker(window time.Duration) *VelocityTracker {
	return &VelocityTracker{
		window:   window,
		accounts: make(map[string]*AccountVelocity),
	}
}

// Record records a transaction for velocity tracking
func (v *VelocityTracker) Record(txn *models.Transaction) {
	v.mu.Lock()
	defer v.mu.Unlock()

	accountID := txn.SourceAccount

	if _, ok := v.accounts[accountID]; !ok {
		v.accounts[accountID] = &AccountVelocity{
			Transactions: make([]*TransactionRecord, 0),
			LastCleanup:  time.Now(),
		}
	}

	acc := v.accounts[accountID]

	// Clean old records if needed
	if time.Since(acc.LastCleanup) > v.window/2 {
		v.cleanOldRecords(acc)
		acc.LastCleanup = time.Now()
	}

	// Add new record
	location := ""
	merchant := ""
	if txn.Merchant != nil {
		location = txn.Merchant.Country
		merchant = txn.Merchant.ID
	}

	acc.Transactions = append(acc.Transactions, &TransactionRecord{
		ID:        txn.ID,
		Amount:    txn.Amount,
		Timestamp: txn.CreatedAt,
		Location:  location,
		Merchant:  merchant,
	})
}

// GetActivity returns recent activity summary for an account
func (v *VelocityTracker) GetActivity(accountID string) *ActivitySummary {
	v.mu.RLock()
	defer v.mu.RUnlock()

	acc, ok := v.accounts[accountID]
	if !ok {
		return &ActivitySummary{
			TimeWindow: v.window,
		}
	}

	cutoff := time.Now().Add(-v.window)
	summary := &ActivitySummary{
		TimeWindow: v.window,
	}

	locations := make(map[string]bool)
	merchants := make(map[string]bool)

	for _, txn := range acc.Transactions {
		if txn.Timestamp.After(cutoff) {
			summary.TransactionCount++
			summary.TotalAmount = summary.TotalAmount.Add(txn.Amount)

			if txn.Location != "" {
				locations[txn.Location] = true
			}
			if txn.Merchant != "" {
				merchants[txn.Merchant] = true
			}
		}
	}

	summary.UniqueLocations = len(locations)
	summary.UniqueMerchants = len(merchants)

	return summary
}

// GetRecentTransactions returns recent transactions for an account
func (v *VelocityTracker) GetRecentTransactions(accountID string, limit int) []*TransactionRecord {
	v.mu.RLock()
	defer v.mu.RUnlock()

	acc, ok := v.accounts[accountID]
	if !ok {
		return nil
	}

	cutoff := time.Now().Add(-v.window)
	var recent []*TransactionRecord

	for i := len(acc.Transactions) - 1; i >= 0 && len(recent) < limit; i-- {
		txn := acc.Transactions[i]
		if txn.Timestamp.After(cutoff) {
			recent = append(recent, txn)
		}
	}

	return recent
}

func (v *VelocityTracker) cleanOldRecords(acc *AccountVelocity) {
	cutoff := time.Now().Add(-v.window)
	var valid []*TransactionRecord

	for _, txn := range acc.Transactions {
		if txn.Timestamp.After(cutoff) {
			valid = append(valid, txn)
		}
	}

	acc.Transactions = valid
}

// Reset resets all velocity tracking
func (v *VelocityTracker) Reset() {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.accounts = make(map[string]*AccountVelocity)
}

// PatternAnalyzer analyzes transaction patterns
type PatternAnalyzer struct {
	accountPatterns map[string]*AccountPattern
	mu              sync.RWMutex
}

// AccountPattern contains learned patterns for an account
type AccountPattern struct {
	AvgAmount        decimal.Decimal
	StdDevAmount     float64
	TypicalHours     []int
	TypicalDays      []time.Weekday
	TypicalMerchants []string
	TypicalAmounts   []decimal.Decimal
	LastUpdated      time.Time
}

// NewPatternAnalyzer creates a new pattern analyzer
func NewPatternAnalyzer() *PatternAnalyzer {
	return &PatternAnalyzer{
		accountPatterns: make(map[string]*AccountPattern),
	}
}

// Learn learns patterns from historical transactions
func (p *PatternAnalyzer) Learn(accountID string, transactions []*models.Transaction) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(transactions) == 0 {
		return
	}

	pattern := &AccountPattern{
		LastUpdated: time.Now(),
	}

	// Calculate average amount
	var total decimal.Decimal
	hourCounts := make(map[int]int)
	dayCounts := make(map[time.Weekday]int)
	merchantCounts := make(map[string]int)

	for _, txn := range transactions {
		total = total.Add(txn.Amount)

		hour := txn.CreatedAt.Hour()
		hourCounts[hour]++

		day := txn.CreatedAt.Weekday()
		dayCounts[day]++

		if txn.Merchant != nil {
			merchantCounts[txn.Merchant.ID]++
		}
	}

	pattern.AvgAmount = total.Div(decimal.NewFromInt(int64(len(transactions))))

	// Find typical hours (top 5)
	pattern.TypicalHours = topNKeys(hourCounts, 5)

	// Find typical days
	for day, count := range dayCounts {
		if count > len(transactions)/10 {
			pattern.TypicalDays = append(pattern.TypicalDays, day)
		}
	}

	// Find typical merchants (top 10)
	for merchant, count := range merchantCounts {
		if count >= 2 {
			pattern.TypicalMerchants = append(pattern.TypicalMerchants, merchant)
			if len(pattern.TypicalMerchants) >= 10 {
				break
			}
		}
	}

	p.accountPatterns[accountID] = pattern
}

// GetPattern returns the learned pattern for an account
func (p *PatternAnalyzer) GetPattern(accountID string) *AccountPattern {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.accountPatterns[accountID]
}

func topNKeys(m map[int]int, n int) []int {
	if len(m) == 0 {
		return nil
	}

	type kv struct {
		Key   int
		Value int
	}

	var sorted []kv
	for k, v := range m {
		sorted = append(sorted, kv{k, v})
	}

	// Simple bubble sort for small n
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].Value > sorted[i].Value {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	result := make([]int, 0, n)
	for i := 0; i < n && i < len(sorted); i++ {
		result = append(result, sorted[i].Key)
	}
	return result
}

// GeofenceChecker checks geolocation rules
type GeofenceChecker struct {
	highRiskCountries map[string]bool
	blockedCountries  map[string]bool
}

// NewGeofenceChecker creates a new geofence checker
func NewGeofenceChecker() *GeofenceChecker {
	g := &GeofenceChecker{
		highRiskCountries: make(map[string]bool),
		blockedCountries:  make(map[string]bool),
	}
	g.initializeDefaultRules()
	return g
}

func (g *GeofenceChecker) initializeDefaultRules() {
	// High-risk countries (simplified list for demonstration)
	highRisk := []string{
		"NG", // Nigeria
		"RU", // Russia
		"UA", // Ukraine
		"RO", // Romania
		"ID", // Indonesia
		"PH", // Philippines
		"VN", // Vietnam
	}

	for _, code := range highRisk {
		g.highRiskCountries[code] = true
	}

	// Blocked countries (sanctions, etc.)
	blocked := []string{
		"KP", // North Korea
		"IR", // Iran
		"SY", // Syria
		"CU", // Cuba
	}

	for _, code := range blocked {
		g.blockedCountries[code] = true
	}
}

// IsHighRiskCountry checks if a country is high-risk
func (g *GeofenceChecker) IsHighRiskCountry(countryCode string) bool {
	return g.highRiskCountries[countryCode]
}

// IsBlockedCountry checks if a country is blocked
func (g *GeofenceChecker) IsBlockedCountry(countryCode string) bool {
	return g.blockedCountries[countryCode]
}

// AddHighRiskCountry adds a country to the high-risk list
func (g *GeofenceChecker) AddHighRiskCountry(countryCode string) {
	g.highRiskCountries[countryCode] = true
}

// AddBlockedCountry adds a country to the blocked list
func (g *GeofenceChecker) AddBlockedCountry(countryCode string) {
	g.blockedCountries[countryCode] = true
}

// RemoveHighRiskCountry removes a country from the high-risk list
func (g *GeofenceChecker) RemoveHighRiskCountry(countryCode string) {
	delete(g.highRiskCountries, countryCode)
}

// RemoveBlockedCountry removes a country from the blocked list
func (g *GeofenceChecker) RemoveBlockedCountry(countryCode string) {
	delete(g.blockedCountries, countryCode)
}
