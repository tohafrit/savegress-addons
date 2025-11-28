package aml

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/savegress/finsight/pkg/models"
	"github.com/shopspring/decimal"
)

// Engine manages AML detection, monitoring, and reporting
type Engine struct {
	config           *Config
	alerts           map[string]*AMLAlert
	cases            map[string]*AMLCase
	sars             map[string]*SuspiciousActivityReport
	ctrs             map[string]*CurrencyTransactionReport
	customerProfiles map[string]*CustomerRiskProfile
	watchlistMgr     *WatchlistManager
	scenarioMgr      *ScenarioManager
	mu               sync.RWMutex
	running          bool
	stopCh           chan struct{}
	alertCh          chan *AMLAlert
}

// Config holds AML engine configuration
type Config struct {
	Enabled              bool            `json:"enabled"`
	CTRThreshold         decimal.Decimal `json:"ctr_threshold"`          // Default $10,000
	StructuringThreshold decimal.Decimal `json:"structuring_threshold"`  // Default $9,000
	StructuringWindow    time.Duration   `json:"structuring_window"`     // Default 1 day
	RiskScoreThreshold   float64         `json:"risk_score_threshold"`   // Default 0.7
	HighRiskCountries    []string        `json:"high_risk_countries"`
	WatchlistSources     []string        `json:"watchlist_sources"`
	AlertRetentionDays   int             `json:"alert_retention_days"`
}

// NewEngine creates a new AML engine
func NewEngine(config *Config) *Engine {
	if config.CTRThreshold.IsZero() {
		config.CTRThreshold = decimal.NewFromInt(10000)
	}
	if config.StructuringThreshold.IsZero() {
		config.StructuringThreshold = decimal.NewFromInt(9000)
	}
	if config.StructuringWindow == 0 {
		config.StructuringWindow = 24 * time.Hour
	}
	if config.RiskScoreThreshold == 0 {
		config.RiskScoreThreshold = 0.7
	}

	return &Engine{
		config:           config,
		alerts:           make(map[string]*AMLAlert),
		cases:            make(map[string]*AMLCase),
		sars:             make(map[string]*SuspiciousActivityReport),
		ctrs:             make(map[string]*CurrencyTransactionReport),
		customerProfiles: make(map[string]*CustomerRiskProfile),
		watchlistMgr:     NewWatchlistManager(),
		scenarioMgr:      NewScenarioManager(config),
		stopCh:           make(chan struct{}),
		alertCh:          make(chan *AMLAlert, 100),
	}
}

// Start starts the AML engine
func (e *Engine) Start(ctx context.Context) error {
	e.mu.Lock()
	if e.running {
		e.mu.Unlock()
		return nil
	}
	e.running = true
	e.mu.Unlock()

	go e.processAlerts(ctx)
	go e.periodicReview(ctx)

	return nil
}

// Stop stops the AML engine
func (e *Engine) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.running {
		close(e.stopCh)
		e.running = false
	}
}

func (e *Engine) processAlerts(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-e.stopCh:
			return
		case alert := <-e.alertCh:
			e.mu.Lock()
			e.alerts[alert.ID] = alert
			e.mu.Unlock()
		}
	}
}

func (e *Engine) periodicReview(ctx context.Context) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-e.stopCh:
			return
		case <-ticker.C:
			e.reviewExpiringKYC()
			e.cleanupOldAlerts()
		}
	}
}

// AnalyzeTransaction analyzes a transaction for AML compliance
func (e *Engine) AnalyzeTransaction(ctx context.Context, txn *models.Transaction) (*AnalysisResult, error) {
	if !e.config.Enabled {
		return &AnalysisResult{Decision: "allow"}, nil
	}

	result := &AnalysisResult{
		TransactionID: txn.ID,
		Timestamp:     time.Now(),
	}

	// Get customer profile
	profile := e.getOrCreateProfile(txn.SourceAccount)

	// Run scenarios
	scenarioResults := e.scenarioMgr.Evaluate(txn, profile)

	// Aggregate results
	var totalScore float64
	var indicators []AlertIndicator

	for _, sr := range scenarioResults {
		if sr.Triggered {
			totalScore += sr.Score
			indicators = append(indicators, AlertIndicator{
				Type:        sr.ScenarioType,
				Description: sr.Description,
				Score:       sr.Score,
				Evidence:    sr.Evidence,
			})
		}
	}

	result.RiskScore = normalizeScore(totalScore)
	result.Indicators = indicators

	// Check for CTR requirement
	if txn.Type == models.TransactionTypeDebit || txn.Type == models.TransactionTypeCredit {
		if txn.Amount.GreaterThanOrEqual(e.config.CTRThreshold) {
			result.CTRRequired = true
		}
	}

	// Make decision
	if result.RiskScore >= e.config.RiskScoreThreshold {
		result.Decision = "block"
		result.Reason = "High AML risk score"

		// Create alert
		alert := e.createAlert(txn, profile, result, indicators)
		e.alertCh <- alert
	} else if result.RiskScore >= e.config.RiskScoreThreshold*0.7 {
		result.Decision = "review"
		result.Reason = "Moderate AML risk - requires review"

		alert := e.createAlert(txn, profile, result, indicators)
		alert.Severity = models.AlertSeverityMedium
		e.alertCh <- alert
	} else {
		result.Decision = "allow"
	}

	// Update customer profile
	e.updateProfileFromTransaction(profile, txn)

	return result, nil
}

// AnalysisResult contains the AML analysis result
type AnalysisResult struct {
	TransactionID string           `json:"transaction_id"`
	RiskScore     float64          `json:"risk_score"`
	Decision      string           `json:"decision"` // allow, review, block
	Reason        string           `json:"reason,omitempty"`
	Indicators    []AlertIndicator `json:"indicators,omitempty"`
	CTRRequired   bool             `json:"ctr_required"`
	Timestamp     time.Time        `json:"timestamp"`
}

func (e *Engine) getOrCreateProfile(customerID string) *CustomerRiskProfile {
	e.mu.Lock()
	defer e.mu.Unlock()

	profile, ok := e.customerProfiles[customerID]
	if !ok {
		profile = &CustomerRiskProfile{
			CustomerID:   customerID,
			RiskLevel:    RiskLevelLow,
			RiskScore:    0,
			RiskFactors:  []RiskFactor{},
			KYCStatus:    "pending",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
			TransactionProfile: &TransactionProfile{},
		}
		e.customerProfiles[customerID] = profile
	}
	return profile
}

func (e *Engine) updateProfileFromTransaction(profile *CustomerRiskProfile, txn *models.Transaction) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if profile.TransactionProfile == nil {
		profile.TransactionProfile = &TransactionProfile{}
	}

	// Update average transaction size
	profile.TransactionProfile.AverageTransactionSize = profile.TransactionProfile.AverageTransactionSize.
		Add(txn.Amount).
		Div(decimal.NewFromInt(2))

	profile.UpdatedAt = time.Now()
}

func (e *Engine) createAlert(txn *models.Transaction, profile *CustomerRiskProfile, result *AnalysisResult, indicators []AlertIndicator) *AMLAlert {
	alertType := AlertTypeUnusualPattern
	if len(indicators) > 0 {
		alertType = AMLAlertType(indicators[0].Type)
	}

	severity := models.AlertSeverityLow
	if result.RiskScore >= 0.9 {
		severity = models.AlertSeverityCritical
	} else if result.RiskScore >= 0.7 {
		severity = models.AlertSeverityHigh
	} else if result.RiskScore >= 0.5 {
		severity = models.AlertSeverityMedium
	}

	return &AMLAlert{
		ID:           generateID("alert"),
		CustomerID:   txn.SourceAccount,
		AlertType:    alertType,
		Severity:     severity,
		Status:       models.AlertStatusOpen,
		Title:        fmt.Sprintf("AML Alert: %s", alertType),
		Description:  result.Reason,
		RiskScore:    result.RiskScore,
		Indicators:   indicators,
		Transactions: []string{txn.ID},
		CreatedAt:    time.Now(),
	}
}

// ScreenCustomer screens a customer against watchlists
func (e *Engine) ScreenCustomer(ctx context.Context, customerID string, name string, dob string, country string) ([]WatchlistMatch, error) {
	matches := e.watchlistMgr.Screen(name, dob, country)

	// Update customer profile with matches
	e.mu.Lock()
	profile, ok := e.customerProfiles[customerID]
	if ok {
		profile.WatchlistMatches = append(profile.WatchlistMatches, matches...)
		if len(matches) > 0 {
			profile.RiskLevel = RiskLevelHigh
			profile.RiskScore = 0.9
		}
	}
	e.mu.Unlock()

	// Generate alerts for matches
	for _, match := range matches {
		alert := &AMLAlert{
			ID:          generateID("alert"),
			CustomerID:  customerID,
			AlertType:   AlertTypeWatchlistMatch,
			Severity:    models.AlertSeverityCritical,
			Status:      models.AlertStatusOpen,
			Title:       fmt.Sprintf("Watchlist Match: %s", match.WatchlistType),
			Description: fmt.Sprintf("Customer matched against %s watchlist: %s (score: %.2f)", match.WatchlistType, match.MatchedName, match.MatchScore),
			RiskScore:   match.MatchScore,
			CreatedAt:   time.Now(),
		}
		e.alertCh <- alert
	}

	return matches, nil
}

// CreateSAR creates a new Suspicious Activity Report
func (e *Engine) CreateSAR(sar *SuspiciousActivityReport) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if sar.ID == "" {
		sar.ID = generateID("sar")
	}
	sar.Status = SARStatusDraft
	sar.CreatedAt = time.Now()
	sar.UpdatedAt = time.Now()

	e.sars[sar.ID] = sar
	return nil
}

// GetSAR retrieves a SAR by ID
func (e *Engine) GetSAR(id string) (*SuspiciousActivityReport, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	sar, ok := e.sars[id]
	return sar, ok
}

// ListSARs returns SARs matching the filter
func (e *Engine) ListSARs(filter SARFilter) []*SuspiciousActivityReport {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var results []*SuspiciousActivityReport
	for _, sar := range e.sars {
		if matchesSARFilter(sar, filter) {
			results = append(results, sar)
		}
	}

	// Sort by created date descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].CreatedAt.After(results[j].CreatedAt)
	})

	return results
}

// SARFilter defines filters for SAR queries
type SARFilter struct {
	Status    SARStatus
	StartDate *time.Time
	EndDate   *time.Time
	Limit     int
}

func matchesSARFilter(sar *SuspiciousActivityReport, filter SARFilter) bool {
	if filter.Status != "" && sar.Status != filter.Status {
		return false
	}
	if filter.StartDate != nil && sar.CreatedAt.Before(*filter.StartDate) {
		return false
	}
	if filter.EndDate != nil && sar.CreatedAt.After(*filter.EndDate) {
		return false
	}
	return true
}

// SubmitSAR submits a SAR for review
func (e *Engine) SubmitSAR(id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	sar, ok := e.sars[id]
	if !ok {
		return fmt.Errorf("SAR not found: %s", id)
	}

	if sar.Status != SARStatusDraft {
		return fmt.Errorf("SAR must be in draft status to submit")
	}

	sar.Status = SARStatusPending
	sar.UpdatedAt = time.Now()
	return nil
}

// ApproveSAR approves a SAR
func (e *Engine) ApproveSAR(id, approver string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	sar, ok := e.sars[id]
	if !ok {
		return fmt.Errorf("SAR not found: %s", id)
	}

	if sar.Status != SARStatusPending {
		return fmt.Errorf("SAR must be pending review to approve")
	}

	sar.Status = SARStatusApproved
	sar.ApprovedBy = approver
	sar.UpdatedAt = time.Now()
	return nil
}

// FileSAR files an approved SAR with FinCEN
func (e *Engine) FileSAR(ctx context.Context, id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	sar, ok := e.sars[id]
	if !ok {
		return fmt.Errorf("SAR not found: %s", id)
	}

	if sar.Status != SARStatusApproved {
		return fmt.Errorf("SAR must be approved to file")
	}

	// In production, this would submit to FinCEN BSA E-Filing System
	sar.Status = SARStatusFiled
	now := time.Now()
	sar.FiledAt = &now
	sar.BSAIdentifier = generateBSAID()
	sar.UpdatedAt = time.Now()

	return nil
}

// CreateCTR creates a new Currency Transaction Report
func (e *Engine) CreateCTR(ctr *CurrencyTransactionReport) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if ctr.ID == "" {
		ctr.ID = generateID("ctr")
	}
	ctr.Status = CTRStatusPending
	ctr.CreatedAt = time.Now()

	e.ctrs[ctr.ID] = ctr
	return nil
}

// GetCTR retrieves a CTR by ID
func (e *Engine) GetCTR(id string) (*CurrencyTransactionReport, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	ctr, ok := e.ctrs[id]
	return ctr, ok
}

// FileCTR files a CTR with FinCEN
func (e *Engine) FileCTR(ctx context.Context, id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	ctr, ok := e.ctrs[id]
	if !ok {
		return fmt.Errorf("CTR not found: %s", id)
	}

	// In production, this would submit to FinCEN
	ctr.Status = CTRStatusFiled
	now := time.Now()
	ctr.FiledAt = &now
	ctr.BSAIdentifier = generateBSAID()

	return nil
}

// CreateCase creates a new AML investigation case
func (e *Engine) CreateCase(amlCase *AMLCase) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if amlCase.ID == "" {
		amlCase.ID = generateID("case")
	}
	if amlCase.CaseNumber == "" {
		amlCase.CaseNumber = generateCaseNumber()
	}
	amlCase.Status = CaseStatusOpen
	amlCase.CreatedAt = time.Now()
	amlCase.UpdatedAt = time.Now()

	// Add initial event
	amlCase.Timeline = append(amlCase.Timeline, CaseEvent{
		ID:          generateID("event"),
		Type:        "case_created",
		Description: "Case created",
		Timestamp:   time.Now(),
	})

	e.cases[amlCase.ID] = amlCase
	return nil
}

// GetCase retrieves a case by ID
func (e *Engine) GetCase(id string) (*AMLCase, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	c, ok := e.cases[id]
	return c, ok
}

// ListCases returns cases matching the filter
func (e *Engine) ListCases(filter CaseFilter) []*AMLCase {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var results []*AMLCase
	for _, c := range e.cases {
		if matchesCaseFilter(c, filter) {
			results = append(results, c)
		}
	}
	return results
}

// CaseFilter defines filters for case queries
type CaseFilter struct {
	Status     CaseStatus
	AssignedTo string
	CustomerID string
	Priority   string
	StartDate  *time.Time
	EndDate    *time.Time
	Limit      int
}

func matchesCaseFilter(c *AMLCase, filter CaseFilter) bool {
	if filter.Status != "" && c.Status != filter.Status {
		return false
	}
	if filter.AssignedTo != "" && c.AssignedTo != filter.AssignedTo {
		return false
	}
	if filter.CustomerID != "" && c.CustomerID != filter.CustomerID {
		return false
	}
	if filter.Priority != "" && c.Priority != filter.Priority {
		return false
	}
	return true
}

// AssignCase assigns a case to an investigator
func (e *Engine) AssignCase(id, assignee string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	c, ok := e.cases[id]
	if !ok {
		return fmt.Errorf("case not found: %s", id)
	}

	c.AssignedTo = assignee
	c.Status = CaseStatusInProgress
	c.UpdatedAt = time.Now()

	c.Timeline = append(c.Timeline, CaseEvent{
		ID:          generateID("event"),
		Type:        "case_assigned",
		Description: fmt.Sprintf("Case assigned to %s", assignee),
		Actor:       assignee,
		Timestamp:   time.Now(),
	})

	return nil
}

// AddCaseNote adds a note to a case
func (e *Engine) AddCaseNote(id, actor, note string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	c, ok := e.cases[id]
	if !ok {
		return fmt.Errorf("case not found: %s", id)
	}

	c.Timeline = append(c.Timeline, CaseEvent{
		ID:          generateID("event"),
		Type:        "note_added",
		Description: note,
		Actor:       actor,
		Timestamp:   time.Now(),
	})

	c.UpdatedAt = time.Now()
	return nil
}

// CloseCase closes a case
func (e *Engine) CloseCase(id, actor, reason string, sarRequired bool) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	c, ok := e.cases[id]
	if !ok {
		return fmt.Errorf("case not found: %s", id)
	}

	c.Status = CaseStatusClosed
	c.ClosureReason = reason
	c.SARRequired = sarRequired
	now := time.Now()
	c.ClosedAt = &now
	c.UpdatedAt = now

	c.Timeline = append(c.Timeline, CaseEvent{
		ID:          generateID("event"),
		Type:        "case_closed",
		Description: fmt.Sprintf("Case closed: %s", reason),
		Actor:       actor,
		Timestamp:   now,
	})

	return nil
}

// GetAlert retrieves an alert by ID
func (e *Engine) GetAlert(id string) (*AMLAlert, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	alert, ok := e.alerts[id]
	return alert, ok
}

// ListAlerts returns alerts matching the filter
func (e *Engine) ListAlerts(filter AlertFilter) []*AMLAlert {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var results []*AMLAlert
	for _, alert := range e.alerts {
		if matchesAlertFilter(alert, filter) {
			results = append(results, alert)
		}
	}
	return results
}

// AlertFilter defines filters for alert queries
type AlertFilter struct {
	Status     models.AlertStatus
	AlertType  AMLAlertType
	CustomerID string
	StartDate  *time.Time
	EndDate    *time.Time
	Limit      int
}

func matchesAlertFilter(alert *AMLAlert, filter AlertFilter) bool {
	if filter.Status != "" && alert.Status != filter.Status {
		return false
	}
	if filter.AlertType != "" && alert.AlertType != filter.AlertType {
		return false
	}
	if filter.CustomerID != "" && alert.CustomerID != filter.CustomerID {
		return false
	}
	return true
}

// ResolveAlert resolves an alert
func (e *Engine) ResolveAlert(id, resolution string, createCase bool) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	alert, ok := e.alerts[id]
	if !ok {
		return fmt.Errorf("alert not found: %s", id)
	}

	alert.Resolution = resolution
	now := time.Now()
	alert.ResolvedAt = &now
	alert.Status = models.AlertStatusResolved

	// Optionally create a case from the alert
	if createCase {
		amlCase := &AMLCase{
			ID:         generateID("case"),
			CaseNumber: generateCaseNumber(),
			Status:     CaseStatusOpen,
			CustomerID: alert.CustomerID,
			Type:       "alert_driven",
			Alerts:     []string{alert.ID},
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		e.cases[amlCase.ID] = amlCase
		alert.CaseID = amlCase.ID
	}

	return nil
}

// GetCustomerProfile retrieves a customer risk profile
func (e *Engine) GetCustomerProfile(customerID string) (*CustomerRiskProfile, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	profile, ok := e.customerProfiles[customerID]
	return profile, ok
}

// UpdateCustomerRisk updates a customer's risk level
func (e *Engine) UpdateCustomerRisk(customerID string, level RiskLevel, factors []RiskFactor) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	profile, ok := e.customerProfiles[customerID]
	if !ok {
		return fmt.Errorf("customer profile not found: %s", customerID)
	}

	profile.RiskLevel = level
	profile.RiskFactors = factors
	profile.UpdatedAt = time.Now()

	// Calculate risk score from factors
	var totalScore float64
	for _, factor := range factors {
		totalScore += factor.Score * factor.Weight
	}
	profile.RiskScore = totalScore

	return nil
}

func (e *Engine) reviewExpiringKYC() {
	e.mu.Lock()
	defer e.mu.Unlock()

	now := time.Now()
	thirtyDays := now.Add(30 * 24 * time.Hour)

	for _, profile := range e.customerProfiles {
		if profile.KYCNextReview != nil && profile.KYCNextReview.Before(thirtyDays) {
			// Create alert for expiring KYC
			alert := &AMLAlert{
				ID:          generateID("alert"),
				CustomerID:  profile.CustomerID,
				AlertType:   AlertTypeKYCExpiring,
				Severity:    models.AlertSeverityMedium,
				Status:      models.AlertStatusOpen,
				Title:       "KYC Review Due",
				Description: fmt.Sprintf("Customer %s KYC review due by %s", profile.CustomerID, profile.KYCNextReview.Format("2006-01-02")),
				CreatedAt:   now,
			}
			e.alerts[alert.ID] = alert
		}
	}
}

func (e *Engine) cleanupOldAlerts() {
	if e.config.AlertRetentionDays <= 0 {
		return
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	cutoff := time.Now().AddDate(0, 0, -e.config.AlertRetentionDays)

	for id, alert := range e.alerts {
		if alert.Status == models.AlertStatusResolved && alert.ResolvedAt != nil && alert.ResolvedAt.Before(cutoff) {
			delete(e.alerts, id)
		}
	}
}

// GetStats returns AML statistics
func (e *Engine) GetStats() *AMLStats {
	e.mu.RLock()
	defer e.mu.RUnlock()

	stats := &AMLStats{
		AlertsByType:  make(map[string]int),
		CasesByStatus: make(map[string]int),
	}

	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)

	for _, alert := range e.alerts {
		stats.TotalAlerts++
		stats.AlertsByType[string(alert.AlertType)]++
		if alert.Status == models.AlertStatusOpen {
			stats.OpenAlerts++
		}
		if alert.CreatedAt.After(thirtyDaysAgo) {
			stats.Last30Days.NewAlerts++
		}
		if alert.ResolvedAt != nil && alert.ResolvedAt.After(thirtyDaysAgo) {
			stats.Last30Days.ResolvedAlerts++
		}
	}

	for _, c := range e.cases {
		stats.TotalCases++
		stats.CasesByStatus[string(c.Status)]++
		if c.Status != CaseStatusClosed {
			stats.OpenCases++
		}
		if c.CreatedAt.After(thirtyDaysAgo) {
			stats.Last30Days.NewCases++
		}
		if c.ClosedAt != nil && c.ClosedAt.After(thirtyDaysAgo) {
			stats.Last30Days.ClosedCases++
		}
	}

	for _, sar := range e.sars {
		if sar.Status == SARStatusFiled {
			stats.SARsFiled++
			if sar.FiledAt != nil && sar.FiledAt.After(thirtyDaysAgo) {
				stats.Last30Days.SARsFiled++
			}
		}
	}

	for _, ctr := range e.ctrs {
		if ctr.Status == CTRStatusFiled {
			stats.CTRsFiled++
		}
	}

	for _, profile := range e.customerProfiles {
		if len(profile.WatchlistMatches) > 0 {
			stats.WatchlistMatches += len(profile.WatchlistMatches)
		}
		if profile.RiskLevel == RiskLevelHigh || profile.RiskLevel == RiskLevelCritical {
			stats.HighRiskCustomers++
		}
	}

	return stats
}

// Helper functions
func normalizeScore(score float64) float64 {
	if score <= 0 {
		return 0
	}
	if score >= 10 {
		return 1.0
	}
	return score / 10.0
}

func generateID(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}

func generateCaseNumber() string {
	return fmt.Sprintf("AML-%s-%04d", time.Now().Format("200601"), time.Now().UnixNano()%10000)
}

func generateBSAID() string {
	return fmt.Sprintf("BSA%d", time.Now().UnixNano())
}

// WatchlistManager manages watchlist screening
type WatchlistManager struct {
	entries map[WatchlistType][]WatchlistEntry
	mu      sync.RWMutex
}

// WatchlistEntry represents an entry in a watchlist
type WatchlistEntry struct {
	ID       string        `json:"id"`
	Type     WatchlistType `json:"type"`
	Name     string        `json:"name"`
	Aliases  []string      `json:"aliases,omitempty"`
	DOB      string        `json:"dob,omitempty"`
	Country  string        `json:"country,omitempty"`
	Programs []string      `json:"programs,omitempty"`
}

// NewWatchlistManager creates a new watchlist manager
func NewWatchlistManager() *WatchlistManager {
	return &WatchlistManager{
		entries: make(map[WatchlistType][]WatchlistEntry),
	}
}

// Screen screens a name against watchlists
func (m *WatchlistManager) Screen(name, dob, country string) []WatchlistMatch {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var matches []WatchlistMatch
	nameLower := strings.ToLower(name)

	for listType, entries := range m.entries {
		for _, entry := range entries {
			score := m.calculateMatchScore(nameLower, entry)
			if score >= 0.8 {
				matches = append(matches, WatchlistMatch{
					ID:            generateID("match"),
					WatchlistType: listType,
					MatchedName:   entry.Name,
					MatchScore:    score,
					MatchType:     m.getMatchType(score),
					ListEntryID:   entry.ID,
					Status:        "pending_review",
					MatchedAt:     time.Now(),
				})
			}
		}
	}

	return matches
}

func (m *WatchlistManager) calculateMatchScore(name string, entry WatchlistEntry) float64 {
	entryNameLower := strings.ToLower(entry.Name)

	// Exact match
	if name == entryNameLower {
		return 1.0
	}

	// Check aliases
	for _, alias := range entry.Aliases {
		if name == strings.ToLower(alias) {
			return 0.95
		}
	}

	// Simple fuzzy match (Levenshtein would be better)
	if strings.Contains(name, entryNameLower) || strings.Contains(entryNameLower, name) {
		return 0.85
	}

	return 0
}

func (m *WatchlistManager) getMatchType(score float64) string {
	if score >= 0.99 {
		return "exact"
	} else if score >= 0.95 {
		return "alias"
	}
	return "fuzzy"
}

// LoadWatchlist loads entries into a watchlist
func (m *WatchlistManager) LoadWatchlist(listType WatchlistType, entries []WatchlistEntry) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries[listType] = entries
}

// ScenarioManager manages AML detection scenarios
type ScenarioManager struct {
	config    *Config
	scenarios []Scenario
}

// Scenario represents an AML detection scenario
type Scenario interface {
	Name() string
	Evaluate(txn *models.Transaction, profile *CustomerRiskProfile) *ScenarioResult
}

// ScenarioResult contains the result of scenario evaluation
type ScenarioResult struct {
	ScenarioType string                 `json:"scenario_type"`
	Triggered    bool                   `json:"triggered"`
	Score        float64                `json:"score"`
	Description  string                 `json:"description,omitempty"`
	Evidence     map[string]interface{} `json:"evidence,omitempty"`
}

// NewScenarioManager creates a new scenario manager
func NewScenarioManager(config *Config) *ScenarioManager {
	mgr := &ScenarioManager{config: config}
	mgr.initializeScenarios()
	return mgr
}

func (m *ScenarioManager) initializeScenarios() {
	m.scenarios = []Scenario{
		&StructuringScenario{threshold: m.config.StructuringThreshold, window: m.config.StructuringWindow},
		&HighRiskCountryScenario{countries: m.config.HighRiskCountries},
		&RoundAmountScenario{},
		&RapidMovementScenario{},
		&UnusualVolumeScenario{},
	}
}

// Evaluate evaluates all scenarios
func (m *ScenarioManager) Evaluate(txn *models.Transaction, profile *CustomerRiskProfile) []*ScenarioResult {
	var results []*ScenarioResult
	for _, scenario := range m.scenarios {
		result := scenario.Evaluate(txn, profile)
		results = append(results, result)
	}
	return results
}

// StructuringScenario detects potential structuring
type StructuringScenario struct {
	threshold decimal.Decimal
	window    time.Duration
}

func (s *StructuringScenario) Name() string { return "structuring" }

func (s *StructuringScenario) Evaluate(txn *models.Transaction, profile *CustomerRiskProfile) *ScenarioResult {
	result := &ScenarioResult{ScenarioType: "structuring"}

	// Check if amount is just below CTR threshold
	ctrThreshold := decimal.NewFromInt(10000)
	if txn.Amount.GreaterThanOrEqual(s.threshold) && txn.Amount.LessThan(ctrThreshold) {
		result.Triggered = true
		result.Score = 3.0
		result.Description = "Transaction amount just below CTR reporting threshold"
		result.Evidence = map[string]interface{}{
			"amount":    txn.Amount.String(),
			"threshold": ctrThreshold.String(),
		}
	}

	return result
}

// HighRiskCountryScenario detects transactions to/from high-risk countries
type HighRiskCountryScenario struct {
	countries []string
}

func (s *HighRiskCountryScenario) Name() string { return "high_risk_country" }

func (s *HighRiskCountryScenario) Evaluate(txn *models.Transaction, profile *CustomerRiskProfile) *ScenarioResult {
	result := &ScenarioResult{ScenarioType: "high_risk_country"}

	if txn.Merchant != nil {
		for _, country := range s.countries {
			if strings.EqualFold(txn.Merchant.Country, country) {
				result.Triggered = true
				result.Score = 4.0
				result.Description = fmt.Sprintf("Transaction involves high-risk country: %s", country)
				result.Evidence = map[string]interface{}{
					"country": country,
				}
				break
			}
		}
	}

	return result
}

// RoundAmountScenario detects suspicious round amounts
type RoundAmountScenario struct{}

func (s *RoundAmountScenario) Name() string { return "round_amounts" }

func (s *RoundAmountScenario) Evaluate(txn *models.Transaction, profile *CustomerRiskProfile) *ScenarioResult {
	result := &ScenarioResult{ScenarioType: "round_amounts"}

	// Check for round amounts (multiples of $1000)
	remainder := txn.Amount.Mod(decimal.NewFromInt(1000))
	if txn.Amount.GreaterThan(decimal.NewFromInt(5000)) && remainder.IsZero() {
		result.Triggered = true
		result.Score = 1.5
		result.Description = "Suspiciously round transaction amount"
		result.Evidence = map[string]interface{}{
			"amount": txn.Amount.String(),
		}
	}

	return result
}

// RapidMovementScenario detects rapid fund movement
type RapidMovementScenario struct{}

func (s *RapidMovementScenario) Name() string { return "rapid_movement" }

func (s *RapidMovementScenario) Evaluate(txn *models.Transaction, profile *CustomerRiskProfile) *ScenarioResult {
	result := &ScenarioResult{ScenarioType: "rapid_movement"}
	// Would check account history for rapid in/out patterns
	return result
}

// UnusualVolumeScenario detects unusual transaction volume
type UnusualVolumeScenario struct{}

func (s *UnusualVolumeScenario) Name() string { return "unusual_volume" }

func (s *UnusualVolumeScenario) Evaluate(txn *models.Transaction, profile *CustomerRiskProfile) *ScenarioResult {
	result := &ScenarioResult{ScenarioType: "unusual_volume"}

	if profile.TransactionProfile != nil {
		expected := profile.TransactionProfile.ExpectedMonthlyVolume
		if !expected.IsZero() && txn.Amount.GreaterThan(expected.Mul(decimal.NewFromFloat(0.5))) {
			result.Triggered = true
			result.Score = 2.5
			result.Description = "Transaction significantly exceeds expected volume"
			result.Evidence = map[string]interface{}{
				"amount":   txn.Amount.String(),
				"expected": expected.String(),
			}
		}
	}

	return result
}
