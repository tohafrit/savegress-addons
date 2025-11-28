package fraud

import (
	"context"
	"sync"
	"time"

	"github.com/savegress/finsight/internal/config"
	"github.com/savegress/finsight/pkg/models"
	"github.com/shopspring/decimal"
)

// Detector detects fraudulent transactions
type Detector struct {
	config     *config.FraudConfig
	rules      []Rule
	alerts     map[string]*models.FraudAlert
	velocity   *VelocityTracker
	patterns   *PatternAnalyzer
	geofence   *GeofenceChecker
	mu         sync.RWMutex
	running    bool
	stopCh     chan struct{}
	alertCh    chan *models.FraudAlert
}

// Rule defines a fraud detection rule
type Rule interface {
	Name() string
	Evaluate(txn *models.Transaction, ctx *EvaluationContext) *RuleResult
	Priority() int
}

// RuleResult contains the result of rule evaluation
type RuleResult struct {
	Triggered   bool
	Score       float64
	Indicators  []models.FraudIndicator
	Description string
}

// EvaluationContext provides context for rule evaluation
type EvaluationContext struct {
	AccountHistory   []*models.Transaction
	RecentActivity   *ActivitySummary
	AccountProfile   *AccountProfile
	DeviceInfo       *DeviceInfo
	GeoLocation      *GeoLocation
}

// ActivitySummary summarizes recent activity
type ActivitySummary struct {
	TransactionCount int
	TotalAmount      decimal.Decimal
	UniqueLocations  int
	UniqueMerchants  int
	TimeWindow       time.Duration
}

// AccountProfile contains account behavior profile
type AccountProfile struct {
	AvgTransactionAmount decimal.Decimal
	AvgDailyTransactions float64
	TypicalLocations     []string
	TypicalMerchants     []string
	TypicalHours         []int
	RiskLevel            string
}

// DeviceInfo contains device information
type DeviceInfo struct {
	DeviceID    string
	DeviceType  string
	OS          string
	Browser     string
	IsKnown     bool
	IsTrusted   bool
	FirstSeen   time.Time
}

// GeoLocation contains geolocation data
type GeoLocation struct {
	Country   string
	City      string
	Latitude  float64
	Longitude float64
	IPAddress string
}

// NewDetector creates a new fraud detector
func NewDetector(cfg *config.FraudConfig) *Detector {
	d := &Detector{
		config:   cfg,
		alerts:   make(map[string]*models.FraudAlert),
		velocity: NewVelocityTracker(cfg.VelocityWindow),
		patterns: NewPatternAnalyzer(),
		geofence: NewGeofenceChecker(),
		stopCh:   make(chan struct{}),
		alertCh:  make(chan *models.FraudAlert, 100),
	}
	d.initializeRules()
	return d
}

func (d *Detector) initializeRules() {
	d.rules = []Rule{
		NewAmountRule(d.config.MaxSingleAmount),
		NewVelocityRule(d.config.VelocityWindow, d.config.MaxDailyAmount),
		NewGeolocationRule(d.geofence),
		NewPatternRule(d.patterns),
		NewTimeRule(),
		NewMerchantRule(),
	}
}

// Start starts the fraud detector
func (d *Detector) Start(ctx context.Context) error {
	d.mu.Lock()
	if d.running {
		d.mu.Unlock()
		return nil
	}
	d.running = true
	d.mu.Unlock()

	go d.processAlerts(ctx)
	return nil
}

// Stop stops the fraud detector
func (d *Detector) Stop() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.running {
		close(d.stopCh)
		d.running = false
	}
}

func (d *Detector) processAlerts(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-d.stopCh:
			return
		case alert := <-d.alertCh:
			d.mu.Lock()
			d.alerts[alert.ID] = alert
			d.mu.Unlock()
		}
	}
}

// Evaluate evaluates a transaction for fraud
func (d *Detector) Evaluate(txn *models.Transaction, evalCtx *EvaluationContext) *EvaluationResult {
	if !d.config.Enabled {
		return &EvaluationResult{
			RiskScore: 0,
			Decision:  DecisionAllow,
		}
	}

	result := &EvaluationResult{
		TransactionID: txn.ID,
		Timestamp:     time.Now(),
	}

	var totalScore float64
	var indicators []models.FraudIndicator

	// Evaluate all rules
	for _, rule := range d.rules {
		ruleResult := rule.Evaluate(txn, evalCtx)
		if ruleResult.Triggered {
			totalScore += ruleResult.Score
			indicators = append(indicators, ruleResult.Indicators...)
		}
	}

	// Normalize score to 0-1 range
	result.RiskScore = normalizeScore(totalScore)
	result.Indicators = indicators

	// Make decision based on score
	if result.RiskScore >= d.config.ScoreThreshold {
		result.Decision = DecisionBlock
		result.Reason = "High risk score exceeds threshold"

		// Create alert
		alert := d.createAlert(txn, result)
		d.alertCh <- alert
	} else if result.RiskScore >= d.config.ScoreThreshold*0.7 {
		result.Decision = DecisionReview
		result.Reason = "Moderate risk score requires review"

		// Create alert for review
		alert := d.createAlert(txn, result)
		alert.Severity = models.AlertSeverityMedium
		d.alertCh <- alert
	} else {
		result.Decision = DecisionAllow
	}

	// Update velocity tracker
	d.velocity.Record(txn)

	return result
}

// EvaluationResult contains the fraud evaluation result
type EvaluationResult struct {
	TransactionID string                   `json:"transaction_id"`
	RiskScore     float64                  `json:"risk_score"`
	Decision      Decision                 `json:"decision"`
	Reason        string                   `json:"reason,omitempty"`
	Indicators    []models.FraudIndicator  `json:"indicators,omitempty"`
	Timestamp     time.Time                `json:"timestamp"`
}

// Decision represents a fraud decision
type Decision string

const (
	DecisionAllow  Decision = "allow"
	DecisionBlock  Decision = "block"
	DecisionReview Decision = "review"
)

func (d *Detector) createAlert(txn *models.Transaction, result *EvaluationResult) *models.FraudAlert {
	alertType := models.FraudAlertTypeAmount
	if len(result.Indicators) > 0 {
		// Use the type from the highest scoring indicator
		alertType = models.FraudAlertType(result.Indicators[0].Type)
	}

	severity := models.AlertSeverityLow
	if result.RiskScore >= 0.9 {
		severity = models.AlertSeverityCritical
	} else if result.RiskScore >= 0.7 {
		severity = models.AlertSeverityHigh
	} else if result.RiskScore >= 0.5 {
		severity = models.AlertSeverityMedium
	}

	now := time.Now()
	return &models.FraudAlert{
		ID:            generateAlertID(),
		TransactionID: txn.ID,
		AlertType:     alertType,
		Severity:      severity,
		RiskScore:     result.RiskScore,
		Indicators:    result.Indicators,
		Status:        models.AlertStatusOpen,
		CreatedAt:     now,
	}
}

// GetAlert retrieves an alert by ID
func (d *Detector) GetAlert(id string) (*models.FraudAlert, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	alert, ok := d.alerts[id]
	return alert, ok
}

// GetAlerts retrieves alerts with filters
func (d *Detector) GetAlerts(filter AlertFilter) []*models.FraudAlert {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var results []*models.FraudAlert
	for _, alert := range d.alerts {
		if d.matchesAlertFilter(alert, filter) {
			results = append(results, alert)
		}
	}
	return results
}

// AlertFilter defines filters for alert queries
type AlertFilter struct {
	Status    models.AlertStatus
	Severity  models.AlertSeverity
	AlertType models.FraudAlertType
	StartDate *time.Time
	EndDate   *time.Time
	Limit     int
}

func (d *Detector) matchesAlertFilter(alert *models.FraudAlert, filter AlertFilter) bool {
	if filter.Status != "" && alert.Status != filter.Status {
		return false
	}
	if filter.Severity != "" && alert.Severity != filter.Severity {
		return false
	}
	if filter.AlertType != "" && alert.AlertType != filter.AlertType {
		return false
	}
	if filter.StartDate != nil && alert.CreatedAt.Before(*filter.StartDate) {
		return false
	}
	if filter.EndDate != nil && alert.CreatedAt.After(*filter.EndDate) {
		return false
	}
	return true
}

// ResolveAlert resolves an alert
func (d *Detector) ResolveAlert(id string, resolution string, falsePositive bool) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	alert, ok := d.alerts[id]
	if !ok {
		return ErrAlertNotFound
	}

	now := time.Now()
	alert.Resolution = resolution
	alert.ResolvedAt = &now

	if falsePositive {
		alert.Status = models.AlertStatusFalsePos
	} else {
		alert.Status = models.AlertStatusResolved
	}

	return nil
}

// GetStats returns fraud detection statistics
func (d *Detector) GetStats() *FraudStats {
	d.mu.RLock()
	defer d.mu.RUnlock()

	stats := &FraudStats{
		ByStatus:   make(map[string]int),
		BySeverity: make(map[string]int),
		ByType:     make(map[string]int),
	}

	for _, alert := range d.alerts {
		stats.TotalAlerts++
		stats.ByStatus[string(alert.Status)]++
		stats.BySeverity[string(alert.Severity)]++
		stats.ByType[string(alert.AlertType)]++

		if alert.Status == models.AlertStatusOpen || alert.Status == models.AlertStatusInProgress {
			stats.OpenAlerts++
		}
		if alert.Status == models.AlertStatusFalsePos {
			stats.FalsePositives++
		}
	}

	if stats.TotalAlerts > 0 {
		stats.FalsePositiveRate = float64(stats.FalsePositives) / float64(stats.TotalAlerts)
	}

	return stats
}

// FraudStats contains fraud detection statistics
type FraudStats struct {
	TotalAlerts       int                `json:"total_alerts"`
	OpenAlerts        int                `json:"open_alerts"`
	FalsePositives    int                `json:"false_positives"`
	FalsePositiveRate float64            `json:"false_positive_rate"`
	ByStatus          map[string]int     `json:"by_status"`
	BySeverity        map[string]int     `json:"by_severity"`
	ByType            map[string]int     `json:"by_type"`
}

func normalizeScore(score float64) float64 {
	// Use sigmoid-like normalization
	if score <= 0 {
		return 0
	}
	if score >= 10 {
		return 1.0
	}
	return score / 10.0
}

func generateAlertID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}

// Errors
var (
	ErrAlertNotFound = &Error{Code: "ALERT_NOT_FOUND", Message: "Alert not found"}
)

// Error represents a fraud detection error
type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string {
	return e.Message
}
