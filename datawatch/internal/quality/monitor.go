package quality

import (
	"context"
	"fmt"
	"regexp"
	"sync"
	"time"
)

// Monitor monitors data quality in real-time
type Monitor struct {
	config     *Config
	rules      map[string]*Rule
	violations map[string]*Violation
	scores     map[string]*Score
	stats      map[string]*TableStats
	mu         sync.RWMutex
	running    bool
	stopCh     chan struct{}
	violationCh chan *Violation
}

// NewMonitor creates a new quality monitor
func NewMonitor(cfg *Config) *Monitor {
	m := &Monitor{
		config:      cfg,
		rules:       make(map[string]*Rule),
		violations:  make(map[string]*Violation),
		scores:      make(map[string]*Score),
		stats:       make(map[string]*TableStats),
		stopCh:      make(chan struct{}),
		violationCh: make(chan *Violation, 1000),
	}

	if cfg.DefaultRules {
		m.initializeDefaultRules()
	}

	return m
}

func (m *Monitor) initializeDefaultRules() {
	// These are generic rules that can apply to many tables
	defaultRules := []*Rule{
		{
			ID:          "default_pk_not_null",
			Name:        "Primary Key Not Null",
			Description: "Primary key fields should not be null",
			Type:        RuleTypeCompleteness,
			Condition:   "not_null",
			Threshold:   100.0,
			Severity:    "critical",
			Enabled:     true,
		},
		{
			ID:          "default_email_format",
			Name:        "Email Format Validation",
			Description: "Email fields should have valid format",
			Type:        RuleTypeValidity,
			Condition:   "email_format",
			Threshold:   95.0,
			Severity:    "medium",
			Enabled:     true,
		},
		{
			ID:          "default_positive_amounts",
			Name:        "Positive Amounts",
			Description: "Amount fields should be positive",
			Type:        RuleTypeValidity,
			Condition:   "positive",
			Threshold:   99.0,
			Severity:    "high",
			Enabled:     true,
		},
	}

	for _, rule := range defaultRules {
		rule.CreatedAt = time.Now()
		rule.UpdatedAt = time.Now()
		m.rules[rule.ID] = rule
	}
}

// Start starts the quality monitor
func (m *Monitor) Start(ctx context.Context) error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return nil
	}
	m.running = true
	m.mu.Unlock()

	go m.processViolations(ctx)
	go m.calculateScoresPeriodically(ctx)

	return nil
}

// Stop stops the quality monitor
func (m *Monitor) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.running {
		close(m.stopCh)
		m.running = false
	}
}

func (m *Monitor) processViolations(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		case violation := <-m.violationCh:
			m.mu.Lock()
			m.violations[violation.ID] = violation
			m.mu.Unlock()
		}
	}
}

func (m *Monitor) calculateScoresPeriodically(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.recalculateAllScores()
		}
	}
}

func (m *Monitor) recalculateAllScores() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Group violations by table
	tableViolations := make(map[string][]*Violation)
	for _, v := range m.violations {
		if !v.Acknowledged {
			tableViolations[v.Table] = append(tableViolations[v.Table], v)
		}
	}

	// Calculate score for each table
	for table, stats := range m.stats {
		if stats.TotalRecords == 0 {
			continue
		}

		violations := tableViolations[table]
		violationCount := len(violations)

		// Simple score calculation based on violation rate
		violationRate := float64(violationCount) / float64(stats.TotalRecords)
		score := 100.0 * (1.0 - violationRate)
		if score < 0 {
			score = 0
		}

		m.scores[table] = &Score{
			Table:          table,
			OverallScore:   score,
			Completeness:   m.calculateCompletenessScore(table, violations),
			Validity:       m.calculateValidityScore(table, violations),
			Freshness:      m.calculateFreshnessScore(stats),
			Consistency:    100.0, // Default
			Uniqueness:     100.0, // Default
			TotalRecords:   stats.TotalRecords,
			InvalidRecords: int64(violationCount),
			ValidRecords:   stats.TotalRecords - int64(violationCount),
			CalculatedAt:   time.Now(),
		}
	}
}

func (m *Monitor) calculateCompletenessScore(table string, violations []*Violation) float64 {
	var completenessViolations int
	for _, v := range violations {
		if v.Type == RuleTypeCompleteness {
			completenessViolations++
		}
	}
	if stats, ok := m.stats[table]; ok && stats.TotalRecords > 0 {
		return 100.0 * (1.0 - float64(completenessViolations)/float64(stats.TotalRecords))
	}
	return 100.0
}

func (m *Monitor) calculateValidityScore(table string, violations []*Violation) float64 {
	var validityViolations int
	for _, v := range violations {
		if v.Type == RuleTypeValidity {
			validityViolations++
		}
	}
	if stats, ok := m.stats[table]; ok && stats.TotalRecords > 0 {
		return 100.0 * (1.0 - float64(validityViolations)/float64(stats.TotalRecords))
	}
	return 100.0
}

func (m *Monitor) calculateFreshnessScore(stats *TableStats) float64 {
	if stats.LastRecordTime.IsZero() {
		return 0.0
	}
	age := time.Since(stats.LastRecordTime)
	// Consider fresh if within 5 minutes
	if age < 5*time.Minute {
		return 100.0
	}
	// Linearly decrease to 0 at 1 hour
	if age > time.Hour {
		return 0.0
	}
	return 100.0 * (1.0 - age.Minutes()/60.0)
}

// ValidateRecord validates a record against applicable rules
func (m *Monitor) ValidateRecord(table string, record map[string]interface{}) *ValidationResult {
	m.mu.RLock()
	rules := m.getRulesForTable(table)
	m.mu.RUnlock()

	result := &ValidationResult{
		Valid:       true,
		Table:       table,
		ValidatedAt: time.Now(),
	}

	var violations []*Violation

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		violation := m.checkRule(rule, table, record)
		if violation != nil {
			violations = append(violations, violation)
			result.Valid = false
		}
	}

	result.Violations = violations
	if len(violations) > 0 {
		result.Score = 100.0 * (1.0 - float64(len(violations))/float64(len(rules)))
	} else {
		result.Score = 100.0
	}

	// Record violations
	for _, v := range violations {
		select {
		case m.violationCh <- v:
		default:
			// Channel full, drop violation
		}
	}

	// Update stats
	m.updateStats(table, record)

	return result
}

func (m *Monitor) getRulesForTable(table string) []*Rule {
	var rules []*Rule
	for _, rule := range m.rules {
		if rule.Table == "" || rule.Table == table {
			rules = append(rules, rule)
		}
	}
	return rules
}

func (m *Monitor) checkRule(rule *Rule, table string, record map[string]interface{}) *Violation {
	field := rule.Field
	if field == "" {
		return nil // Rule needs a field to validate
	}

	value, exists := record[field]

	switch rule.Condition {
	case "not_null":
		if !exists || value == nil {
			return m.createViolation(rule, table, record, nil, value, "Field is null or missing")
		}

	case "not_empty":
		if !exists || value == nil {
			return m.createViolation(rule, table, record, "non-empty", value, "Field is null or missing")
		}
		if str, ok := value.(string); ok && str == "" {
			return m.createViolation(rule, table, record, "non-empty", value, "Field is empty")
		}

	case "positive":
		if exists && value != nil {
			if num, ok := toFloat(value); ok && num <= 0 {
				return m.createViolation(rule, table, record, "> 0", value, "Value should be positive")
			}
		}

	case "in_range":
		if exists && value != nil {
			if num, ok := toFloat(value); ok {
				min, _ := rule.Parameters["min"].(float64)
				max, _ := rule.Parameters["max"].(float64)
				if num < min || num > max {
					return m.createViolation(rule, table, record, fmt.Sprintf("[%v, %v]", min, max), value, "Value out of range")
				}
			}
		}

	case "in_set":
		if exists && value != nil {
			if values, ok := rule.Parameters["values"].([]interface{}); ok {
				found := false
				for _, v := range values {
					if v == value {
						found = true
						break
					}
				}
				if !found {
					return m.createViolation(rule, table, record, values, value, "Value not in allowed set")
				}
			}
		}

	case "email_format":
		if exists && value != nil {
			if str, ok := value.(string); ok {
				if !isValidEmail(str) {
					return m.createViolation(rule, table, record, "valid email", value, "Invalid email format")
				}
			}
		}

	case "regex":
		if exists && value != nil {
			if str, ok := value.(string); ok {
				if pattern, ok := rule.Parameters["pattern"].(string); ok {
					if matched, _ := regexp.MatchString(pattern, str); !matched {
						return m.createViolation(rule, table, record, pattern, value, "Value does not match pattern")
					}
				}
			}
		}
	}

	return nil
}

func (m *Monitor) createViolation(rule *Rule, table string, record map[string]interface{}, expected, actual interface{}, message string) *Violation {
	var recordID string
	if id, ok := record["id"]; ok {
		recordID = fmt.Sprintf("%v", id)
	}

	return &Violation{
		ID:          fmt.Sprintf("vio_%d", time.Now().UnixNano()),
		RuleID:      rule.ID,
		RuleName:    rule.Name,
		Table:       table,
		Field:       rule.Field,
		RecordID:    recordID,
		Type:        rule.Type,
		Severity:    rule.Severity,
		Message:     message,
		ExpectedVal: expected,
		ActualVal:   actual,
		DetectedAt:  time.Now(),
	}
}

func (m *Monitor) updateStats(table string, record map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()

	stats, ok := m.stats[table]
	if !ok {
		stats = &TableStats{
			Table:         table,
			NullCounts:    make(map[string]int64),
			UniqueValues:  make(map[string]int64),
			InvalidCounts: make(map[string]int64),
		}
		m.stats[table] = stats
	}

	stats.TotalRecords++
	stats.LastRecordTime = time.Now()

	for field, value := range record {
		if value == nil {
			stats.NullCounts[field]++
		}
	}
}

// AddRule adds a quality rule
func (m *Monitor) AddRule(rule *Rule) {
	m.mu.Lock()
	defer m.mu.Unlock()

	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()
	m.rules[rule.ID] = rule
}

// GetRule returns a rule by ID
func (m *Monitor) GetRule(id string) (*Rule, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	rule, ok := m.rules[id]
	return rule, ok
}

// ListRules returns all rules
func (m *Monitor) ListRules() []*Rule {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rules := make([]*Rule, 0, len(m.rules))
	for _, rule := range m.rules {
		rules = append(rules, rule)
	}
	return rules
}

// DeleteRule deletes a rule
func (m *Monitor) DeleteRule(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.rules, id)
}

// GetScore returns the quality score for a table
func (m *Monitor) GetScore(table string) (*Score, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	score, ok := m.scores[table]
	return score, ok
}

// GetOverallScore returns the overall quality score
func (m *Monitor) GetOverallScore() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.scores) == 0 {
		return 100.0
	}

	var total float64
	for _, score := range m.scores {
		total += score.OverallScore
	}
	return total / float64(len(m.scores))
}

// GetViolations returns violations with filters
func (m *Monitor) GetViolations(filter ViolationFilter) []*Violation {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var results []*Violation
	for _, v := range m.violations {
		if m.matchesFilter(v, filter) {
			results = append(results, v)
		}
	}
	return results
}

// ViolationFilter defines filters for violations
type ViolationFilter struct {
	Table        string
	Field        string
	Type         RuleType
	Severity     string
	Acknowledged *bool
	StartTime    *time.Time
	EndTime      *time.Time
	Limit        int
}

func (m *Monitor) matchesFilter(v *Violation, filter ViolationFilter) bool {
	if filter.Table != "" && v.Table != filter.Table {
		return false
	}
	if filter.Field != "" && v.Field != filter.Field {
		return false
	}
	if filter.Type != "" && v.Type != filter.Type {
		return false
	}
	if filter.Severity != "" && v.Severity != filter.Severity {
		return false
	}
	if filter.Acknowledged != nil && v.Acknowledged != *filter.Acknowledged {
		return false
	}
	if filter.StartTime != nil && v.DetectedAt.Before(*filter.StartTime) {
		return false
	}
	if filter.EndTime != nil && v.DetectedAt.After(*filter.EndTime) {
		return false
	}
	return true
}

// AcknowledgeViolation acknowledges a violation
func (m *Monitor) AcknowledgeViolation(id, user string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	v, ok := m.violations[id]
	if !ok {
		return fmt.Errorf("violation not found: %s", id)
	}

	now := time.Now()
	v.Acknowledged = true
	v.AckedBy = user
	v.AckedAt = &now

	return nil
}

// GetReport generates a data quality report
func (m *Monitor) GetReport(start, end time.Time) *Report {
	m.mu.RLock()
	defer m.mu.RUnlock()

	report := &Report{
		ID: fmt.Sprintf("report_%d", time.Now().UnixNano()),
		Period: ReportPeriod{
			Start: start,
			End:   end,
		},
		TableScores: make(map[string]*Score),
		GeneratedAt: time.Now(),
	}

	// Copy scores
	var totalScore float64
	for table, score := range m.scores {
		report.TableScores[table] = score
		totalScore += score.OverallScore
	}
	if len(m.scores) > 0 {
		report.OverallScore = totalScore / float64(len(m.scores))
	}

	// Count rules
	for _, rule := range m.rules {
		report.TotalRules++
		if rule.Enabled {
			report.PassingRules++ // Simplified
		}
	}

	// Collect violations in period
	for _, v := range m.violations {
		if v.DetectedAt.After(start) && v.DetectedAt.Before(end) {
			report.Violations = append(report.Violations, v)
		}
	}

	// Generate recommendations
	report.Recommendations = m.generateRecommendations()

	return report
}

func (m *Monitor) generateRecommendations() []string {
	var recommendations []string

	// Check for tables with low scores
	for table, score := range m.scores {
		if score.Completeness < 95 {
			recommendations = append(recommendations,
				fmt.Sprintf("Table '%s' has completeness issues (%.1f%%). Review null values.", table, score.Completeness))
		}
		if score.Validity < 95 {
			recommendations = append(recommendations,
				fmt.Sprintf("Table '%s' has validity issues (%.1f%%). Check data formats.", table, score.Validity))
		}
		if score.Freshness < 50 {
			recommendations = append(recommendations,
				fmt.Sprintf("Table '%s' has stale data. Last update was over 30 minutes ago.", table))
		}
	}

	return recommendations
}

// GetStats returns quality monitoring statistics
func (m *Monitor) GetStats() *MonitorStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := &MonitorStats{
		TotalRules:      len(m.rules),
		TotalViolations: len(m.violations),
		TablesMonitored: len(m.stats),
		ScoresByTable:   make(map[string]float64),
	}

	for table, score := range m.scores {
		stats.ScoresByTable[table] = score.OverallScore
	}

	for _, v := range m.violations {
		if !v.Acknowledged {
			stats.OpenViolations++
		}
	}

	stats.OverallScore = m.GetOverallScore()

	return stats
}

// MonitorStats contains quality monitor statistics
type MonitorStats struct {
	TotalRules      int                `json:"total_rules"`
	TotalViolations int                `json:"total_violations"`
	OpenViolations  int                `json:"open_violations"`
	TablesMonitored int                `json:"tables_monitored"`
	OverallScore    float64            `json:"overall_score"`
	ScoresByTable   map[string]float64 `json:"scores_by_table"`
}

// Helper functions
func toFloat(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int32:
		return float64(val), true
	case int64:
		return float64(val), true
	default:
		return 0, false
	}
}

func isValidEmail(email string) bool {
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	matched, _ := regexp.MatchString(pattern, email)
	return matched
}
