package alerts

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/savegress/iotsense/internal/config"
	"github.com/savegress/iotsense/pkg/models"
)

// Engine evaluates alert rules and manages alerts
type Engine struct {
	config     *config.AlertsConfig
	rules      map[string]*models.AlertRule
	alerts     map[string]*models.Alert
	cooldowns  map[string]time.Time // rule:device -> last alert time
	notifiers  []Notifier
	mu         sync.RWMutex
	running    bool
	stopCh     chan struct{}
	dataCh     chan *models.TelemetryPoint
	onAlert    func(*models.Alert)
}

// Notifier sends alert notifications
type Notifier interface {
	Name() string
	Notify(alert *models.Alert) error
}

// NewEngine creates a new alerting engine
func NewEngine(cfg *config.AlertsConfig) *Engine {
	return &Engine{
		config:    cfg,
		rules:     make(map[string]*models.AlertRule),
		alerts:    make(map[string]*models.Alert),
		cooldowns: make(map[string]time.Time),
		stopCh:    make(chan struct{}),
		dataCh:    make(chan *models.TelemetryPoint, 10000),
	}
}

// Start starts the alerting engine
func (e *Engine) Start(ctx context.Context) error {
	e.mu.Lock()
	if e.running {
		e.mu.Unlock()
		return nil
	}
	e.running = true
	e.mu.Unlock()

	go e.evaluationLoop(ctx)
	return nil
}

// Stop stops the alerting engine
func (e *Engine) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.running {
		close(e.stopCh)
		e.running = false
	}
}

// SetAlertCallback sets a callback for new alerts
func (e *Engine) SetAlertCallback(cb func(*models.Alert)) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.onAlert = cb
}

// AddNotifier adds a notifier
func (e *Engine) AddNotifier(n Notifier) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.notifiers = append(e.notifiers, n)
}

// ProcessTelemetry processes a telemetry point for alerting
func (e *Engine) ProcessTelemetry(point *models.TelemetryPoint) {
	select {
	case e.dataCh <- point:
	default:
		// Drop if buffer full
	}
}

func (e *Engine) evaluationLoop(ctx context.Context) {
	ticker := time.NewTicker(e.config.EvaluationInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-e.stopCh:
			return
		case point := <-e.dataCh:
			e.evaluatePoint(point)
		case <-ticker.C:
			e.cleanupExpiredAlerts()
		}
	}
}

func (e *Engine) evaluatePoint(point *models.TelemetryPoint) {
	e.mu.RLock()
	rules := make([]*models.AlertRule, 0, len(e.rules))
	for _, rule := range e.rules {
		rules = append(rules, rule)
	}
	e.mu.RUnlock()

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		if rule.Metric != point.Metric {
			continue
		}
		if !e.deviceMatchesRule(point.DeviceID, rule) {
			continue
		}

		if e.evaluateCondition(point, rule) {
			e.triggerAlert(point, rule)
		}
	}
}

func (e *Engine) deviceMatchesRule(deviceID string, rule *models.AlertRule) bool {
	// If no specific devices or groups, applies to all
	if len(rule.DeviceIDs) == 0 && len(rule.GroupIDs) == 0 {
		return true
	}

	// Check device IDs
	for _, id := range rule.DeviceIDs {
		if id == deviceID {
			return true
		}
	}

	// Group matching would need device registry integration
	return false
}

func (e *Engine) evaluateCondition(point *models.TelemetryPoint, rule *models.AlertRule) bool {
	value, ok := toFloat64(point.Value)
	if !ok {
		return false
	}

	threshold, ok := toFloat64(rule.Condition.Value)
	if !ok {
		return false
	}

	switch rule.Condition.Operator {
	case "gt":
		return value > threshold
	case "gte":
		return value >= threshold
	case "lt":
		return value < threshold
	case "lte":
		return value <= threshold
	case "eq":
		return value == threshold
	case "neq":
		return value != threshold
	}

	// Range condition
	if rule.Condition.Type == "range" {
		minVal, _ := toFloat64(rule.Condition.MinValue)
		maxVal, _ := toFloat64(rule.Condition.MaxValue)
		return value < minVal || value > maxVal
	}

	return false
}

func (e *Engine) triggerAlert(point *models.TelemetryPoint, rule *models.AlertRule) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Check cooldown
	cooldownKey := rule.ID + ":" + point.DeviceID
	if lastAlert, ok := e.cooldowns[cooldownKey]; ok {
		if time.Since(lastAlert) < rule.Cooldown {
			return
		}
	}

	// Check max active alerts
	activeCount := 0
	for _, alert := range e.alerts {
		if alert.Status == models.AlertStatusActive {
			activeCount++
		}
	}
	if activeCount >= e.config.MaxActiveAlerts {
		return
	}

	// Create alert
	alert := &models.Alert{
		ID:           uuid.New().String(),
		DeviceID:     point.DeviceID,
		RuleID:       rule.ID,
		Severity:     rule.Severity,
		Type:         models.AlertTypeThreshold,
		Title:        rule.Name,
		Message:      formatAlertMessage(point, rule),
		Status:       models.AlertStatusActive,
		TriggerValue: point.Value,
		Threshold:    rule.Condition.Value,
		Metric:       point.Metric,
		CreatedAt:    time.Now(),
	}

	e.alerts[alert.ID] = alert
	e.cooldowns[cooldownKey] = time.Now()

	// Notify
	for _, notifier := range e.notifiers {
		go notifier.Notify(alert)
	}

	// Callback
	if e.onAlert != nil {
		go e.onAlert(alert)
	}
}

func (e *Engine) cleanupExpiredAlerts() {
	e.mu.Lock()
	defer e.mu.Unlock()

	retention := time.Duration(e.config.RetentionDays) * 24 * time.Hour
	cutoff := time.Now().Add(-retention)

	for id, alert := range e.alerts {
		if alert.Status == models.AlertStatusResolved && alert.CreatedAt.Before(cutoff) {
			delete(e.alerts, id)
		}
	}
}

// Rule Management

// CreateRule creates an alert rule
func (e *Engine) CreateRule(rule *models.AlertRule) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if rule.ID == "" {
		rule.ID = uuid.New().String()
	}

	now := time.Now()
	rule.CreatedAt = now
	rule.UpdatedAt = now

	e.rules[rule.ID] = rule
	return nil
}

// UpdateRule updates an alert rule
func (e *Engine) UpdateRule(rule *models.AlertRule) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, ok := e.rules[rule.ID]; !ok {
		return ErrRuleNotFound
	}

	rule.UpdatedAt = time.Now()
	e.rules[rule.ID] = rule
	return nil
}

// DeleteRule deletes an alert rule
func (e *Engine) DeleteRule(id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, ok := e.rules[id]; !ok {
		return ErrRuleNotFound
	}

	delete(e.rules, id)
	return nil
}

// GetRule retrieves a rule by ID
func (e *Engine) GetRule(id string) (*models.AlertRule, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	rule, ok := e.rules[id]
	return rule, ok
}

// ListRules lists all rules
func (e *Engine) ListRules() []*models.AlertRule {
	e.mu.RLock()
	defer e.mu.RUnlock()

	rules := make([]*models.AlertRule, 0, len(e.rules))
	for _, rule := range e.rules {
		rules = append(rules, rule)
	}
	return rules
}

// Alert Management

// GetAlert retrieves an alert by ID
func (e *Engine) GetAlert(id string) (*models.Alert, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	alert, ok := e.alerts[id]
	return alert, ok
}

// ListAlerts lists alerts with optional filter
func (e *Engine) ListAlerts(filter AlertFilter) []*models.Alert {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var results []*models.Alert
	for _, alert := range e.alerts {
		if e.matchesFilter(alert, filter) {
			results = append(results, alert)
		}
	}
	return results
}

// AlertFilter defines filters for alert queries
type AlertFilter struct {
	DeviceID string
	RuleID   string
	Severity models.AlertSeverity
	Status   models.AlertStatus
	Limit    int
}

func (e *Engine) matchesFilter(alert *models.Alert, filter AlertFilter) bool {
	if filter.DeviceID != "" && alert.DeviceID != filter.DeviceID {
		return false
	}
	if filter.RuleID != "" && alert.RuleID != filter.RuleID {
		return false
	}
	if filter.Severity != "" && alert.Severity != filter.Severity {
		return false
	}
	if filter.Status != "" && alert.Status != filter.Status {
		return false
	}
	return true
}

// AcknowledgeAlert acknowledges an alert
func (e *Engine) AcknowledgeAlert(id, user string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	alert, ok := e.alerts[id]
	if !ok {
		return ErrAlertNotFound
	}

	now := time.Now()
	alert.Status = models.AlertStatusAcknowledged
	alert.AckedBy = user
	alert.AckedAt = &now

	return nil
}

// ResolveAlert resolves an alert
func (e *Engine) ResolveAlert(id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	alert, ok := e.alerts[id]
	if !ok {
		return ErrAlertNotFound
	}

	now := time.Now()
	alert.Status = models.AlertStatusResolved
	alert.ResolvedAt = &now

	return nil
}

// GetStats returns alerting statistics
func (e *Engine) GetStats() *AlertStats {
	e.mu.RLock()
	defer e.mu.RUnlock()

	stats := &AlertStats{
		TotalRules:     len(e.rules),
		EnabledRules:   0,
		TotalAlerts:    len(e.alerts),
		BySeverity:     make(map[string]int),
		ByStatus:       make(map[string]int),
	}

	for _, rule := range e.rules {
		if rule.Enabled {
			stats.EnabledRules++
		}
	}

	for _, alert := range e.alerts {
		stats.BySeverity[string(alert.Severity)]++
		stats.ByStatus[string(alert.Status)]++

		if alert.Status == models.AlertStatusActive {
			stats.ActiveAlerts++
		}
	}

	return stats
}

// AlertStats contains alerting statistics
type AlertStats struct {
	TotalRules   int            `json:"total_rules"`
	EnabledRules int            `json:"enabled_rules"`
	TotalAlerts  int            `json:"total_alerts"`
	ActiveAlerts int            `json:"active_alerts"`
	BySeverity   map[string]int `json:"by_severity"`
	ByStatus     map[string]int `json:"by_status"`
}

// Helper functions

func toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case int32:
		return float64(val), true
	default:
		return 0, false
	}
}

func formatAlertMessage(point *models.TelemetryPoint, rule *models.AlertRule) string {
	return "Alert triggered: " + rule.Name + " on metric " + point.Metric
}

// Errors
var (
	ErrRuleNotFound  = &Error{Code: "RULE_NOT_FOUND", Message: "Alert rule not found"}
	ErrAlertNotFound = &Error{Code: "ALERT_NOT_FOUND", Message: "Alert not found"}
)

// Error represents an alerting error
type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string {
	return e.Message
}
