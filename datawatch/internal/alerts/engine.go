package alerts

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MetricProvider provides metric values for alert evaluation
type MetricProvider interface {
	GetMetricValue(ctx context.Context, metric string) (float64, error)
	GetMetricBaseline(ctx context.Context, metric string, window time.Duration) (float64, error)
}

// Engine manages alert rules and evaluations
type Engine struct {
	config         *Config
	rules          map[string]*Rule
	alerts         map[string]*Alert
	channels       map[string]*Channel
	notifiers      map[ChannelType]Notifier
	metricProvider MetricProvider
	mu             sync.RWMutex
	running        bool
	stopCh         chan struct{}
	alertCh        chan *Alert
}

// Notifier interface for sending notifications
type Notifier interface {
	Send(ctx context.Context, alert *Alert, channel *Channel) error
}

// NewEngine creates a new alerts engine
func NewEngine(cfg *Config) *Engine {
	e := &Engine{
		config:    cfg,
		rules:     make(map[string]*Rule),
		alerts:    make(map[string]*Alert),
		channels:  make(map[string]*Channel),
		notifiers: make(map[ChannelType]Notifier),
		stopCh:    make(chan struct{}),
		alertCh:   make(chan *Alert, 100),
	}

	// Initialize default notifiers
	e.notifiers[ChannelTypeSlack] = &SlackNotifier{}
	e.notifiers[ChannelTypeEmail] = &EmailNotifier{}
	e.notifiers[ChannelTypePagerDuty] = &PagerDutyNotifier{}
	e.notifiers[ChannelTypeWebhook] = &WebhookNotifier{}
	e.notifiers[ChannelTypeTeams] = &TeamsNotifier{}
	e.notifiers[ChannelTypeDiscord] = &DiscordNotifier{}

	return e
}

// SetMetricProvider sets the metric provider
func (e *Engine) SetMetricProvider(provider MetricProvider) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.metricProvider = provider
}

// Start starts the alerts engine
func (e *Engine) Start(ctx context.Context) error {
	e.mu.Lock()
	if e.running {
		e.mu.Unlock()
		return nil
	}
	e.running = true
	e.mu.Unlock()

	go e.evaluationLoop(ctx)
	go e.processAlerts(ctx)

	return nil
}

// Stop stops the alerts engine
func (e *Engine) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.running {
		close(e.stopCh)
		e.running = false
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
		case <-ticker.C:
			e.evaluateAllRules(ctx)
		}
	}
}

func (e *Engine) evaluateAllRules(ctx context.Context) {
	e.mu.RLock()
	rules := make([]*Rule, 0, len(e.rules))
	for _, rule := range e.rules {
		if rule.Enabled {
			rules = append(rules, rule)
		}
	}
	provider := e.metricProvider
	e.mu.RUnlock()

	if provider == nil {
		return
	}

	for _, rule := range rules {
		result := e.evaluateRule(ctx, rule, provider)
		if result.Triggered {
			e.fireAlert(rule, result)
		} else {
			e.resolveAlertIfExists(rule.ID)
		}
	}
}

func (e *Engine) evaluateRule(ctx context.Context, rule *Rule, provider MetricProvider) *EvaluationResult {
	result := &EvaluationResult{
		RuleID:      rule.ID,
		Threshold:   rule.Condition.Threshold,
		EvaluatedAt: time.Now(),
	}

	value, err := provider.GetMetricValue(ctx, rule.Metric)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	result.Value = value

	// Handle relative comparisons
	if rule.Condition.CompareWith != "" {
		var baselineWindow time.Duration
		switch rule.Condition.CompareWith {
		case "previous_hour":
			baselineWindow = time.Hour
		case "same_hour_last_week":
			baselineWindow = 7 * 24 * time.Hour
		case "baseline":
			baselineWindow = 24 * time.Hour
		}

		baseline, err := provider.GetMetricBaseline(ctx, rule.Metric, baselineWindow)
		if err != nil {
			result.Error = err.Error()
			return result
		}

		// Calculate change percent
		if baseline > 0 {
			changePercent := ((value - baseline) / baseline) * 100
			result.Value = changePercent
		}
	}

	// Evaluate condition
	result.Triggered = e.checkCondition(result.Value, rule.Condition)

	return result
}

func (e *Engine) checkCondition(value float64, condition Condition) bool {
	threshold := condition.Threshold
	if condition.ChangePercent > 0 {
		threshold = condition.ChangePercent
	}

	switch condition.Operator {
	case OpGreaterThan:
		return value > threshold
	case OpGreaterThanEqual:
		return value >= threshold
	case OpLessThan:
		return value < threshold
	case OpLessThanEqual:
		return value <= threshold
	case OpEqual:
		return value == threshold
	case OpNotEqual:
		return value != threshold
	}
	return false
}

func (e *Engine) fireAlert(rule *Rule, result *EvaluationResult) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Check if alert already exists for this rule
	for _, alert := range e.alerts {
		if alert.RuleID == rule.ID && alert.Status == StatusOpen {
			return // Already firing
		}
	}

	alert := &Alert{
		ID:             fmt.Sprintf("alert_%d", time.Now().UnixNano()),
		RuleID:         rule.ID,
		RuleName:       rule.Name,
		Type:           rule.Type,
		Severity:       rule.Severity,
		Status:         StatusOpen,
		Title:          fmt.Sprintf("[%s] %s", rule.Severity, rule.Name),
		Message:        e.formatAlertMessage(rule, result),
		Metric:         rule.Metric,
		CurrentValue:   result.Value,
		ThresholdValue: result.Threshold,
		Labels:         rule.Labels,
		Annotations:    rule.Annotations,
		FiredAt:        time.Now(),
	}

	e.alerts[alert.ID] = alert

	// Send to notification channel
	select {
	case e.alertCh <- alert:
	default:
	}
}

func (e *Engine) formatAlertMessage(rule *Rule, result *EvaluationResult) string {
	return fmt.Sprintf(
		"Alert '%s' triggered: %s is %.2f (threshold: %s %.2f)",
		rule.Name,
		rule.Metric,
		result.Value,
		rule.Condition.Operator,
		rule.Condition.Threshold,
	)
}

func (e *Engine) resolveAlertIfExists(ruleID string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	for _, alert := range e.alerts {
		if alert.RuleID == ruleID && alert.Status == StatusOpen {
			now := time.Now()
			alert.Status = StatusResolved
			alert.ResolvedAt = &now
			alert.ResolvedBy = "system"
		}
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
			e.sendNotifications(ctx, alert)
		}
	}
}

func (e *Engine) sendNotifications(ctx context.Context, alert *Alert) {
	e.mu.RLock()
	rule, exists := e.rules[alert.RuleID]
	if !exists {
		e.mu.RUnlock()
		return
	}
	channelIDs := rule.Channels
	e.mu.RUnlock()

	for _, channelID := range channelIDs {
		e.mu.RLock()
		channel, ok := e.channels[channelID]
		e.mu.RUnlock()

		if !ok || !channel.Enabled {
			continue
		}

		notifier := e.notifiers[channel.Type]
		if notifier == nil {
			continue
		}

		record := NotificationRecord{
			Channel: channelID,
			SentAt:  time.Now(),
		}

		if err := notifier.Send(ctx, alert, channel); err != nil {
			record.Success = false
			record.Error = err.Error()
		} else {
			record.Success = true
		}

		e.mu.Lock()
		if a, ok := e.alerts[alert.ID]; ok {
			a.NotifiedAt = append(a.NotifiedAt, record)
		}
		e.mu.Unlock()
	}
}

// AddRule adds an alert rule
func (e *Engine) AddRule(rule *Rule) {
	e.mu.Lock()
	defer e.mu.Unlock()

	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()
	e.rules[rule.ID] = rule
}

// GetRule returns a rule by ID
func (e *Engine) GetRule(id string) (*Rule, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	rule, ok := e.rules[id]
	return rule, ok
}

// ListRules returns all rules
func (e *Engine) ListRules() []*Rule {
	e.mu.RLock()
	defer e.mu.RUnlock()

	rules := make([]*Rule, 0, len(e.rules))
	for _, rule := range e.rules {
		rules = append(rules, rule)
	}
	return rules
}

// UpdateRule updates a rule
func (e *Engine) UpdateRule(rule *Rule) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if existing, ok := e.rules[rule.ID]; ok {
		rule.CreatedAt = existing.CreatedAt
	}
	rule.UpdatedAt = time.Now()
	e.rules[rule.ID] = rule
}

// DeleteRule deletes a rule
func (e *Engine) DeleteRule(id string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.rules, id)
}

// AddChannel adds a notification channel
func (e *Engine) AddChannel(channel *Channel) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.channels[channel.ID] = channel
}

// GetChannel returns a channel by ID
func (e *Engine) GetChannel(id string) (*Channel, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	ch, ok := e.channels[id]
	return ch, ok
}

// ListChannels returns all channels
func (e *Engine) ListChannels() []*Channel {
	e.mu.RLock()
	defer e.mu.RUnlock()

	channels := make([]*Channel, 0, len(e.channels))
	for _, ch := range e.channels {
		channels = append(channels, ch)
	}
	return channels
}

// DeleteChannel deletes a channel
func (e *Engine) DeleteChannel(id string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.channels, id)
}

// FireManualAlert fires an alert manually
func (e *Engine) FireManualAlert(alert *Alert) {
	e.mu.Lock()
	alert.ID = fmt.Sprintf("alert_%d", time.Now().UnixNano())
	alert.FiredAt = time.Now()
	alert.Status = StatusOpen
	e.alerts[alert.ID] = alert
	e.mu.Unlock()

	select {
	case e.alertCh <- alert:
	default:
	}
}

// GetAlert returns an alert by ID
func (e *Engine) GetAlert(id string) (*Alert, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	alert, ok := e.alerts[id]
	return alert, ok
}

// GetAlerts returns alerts with filters
func (e *Engine) GetAlerts(filter AlertFilter) []*Alert {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var results []*Alert
	for _, alert := range e.alerts {
		if e.matchesAlertFilter(alert, filter) {
			results = append(results, alert)
		}
	}
	return results
}

// AlertFilter defines filters for alerts
type AlertFilter struct {
	Status    Status
	Severity  Severity
	Type      AlertType
	RuleID    string
	StartTime *time.Time
	EndTime   *time.Time
	Limit     int
}

func (e *Engine) matchesAlertFilter(alert *Alert, filter AlertFilter) bool {
	if filter.Status != "" && alert.Status != filter.Status {
		return false
	}
	if filter.Severity != "" && alert.Severity != filter.Severity {
		return false
	}
	if filter.Type != "" && alert.Type != filter.Type {
		return false
	}
	if filter.RuleID != "" && alert.RuleID != filter.RuleID {
		return false
	}
	if filter.StartTime != nil && alert.FiredAt.Before(*filter.StartTime) {
		return false
	}
	if filter.EndTime != nil && alert.FiredAt.After(*filter.EndTime) {
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
		return fmt.Errorf("alert not found: %s", id)
	}

	now := time.Now()
	alert.Status = StatusAcknowledged
	alert.AcknowledgedAt = &now
	alert.AcknowledgedBy = user

	return nil
}

// ResolveAlert resolves an alert
func (e *Engine) ResolveAlert(id, user string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	alert, ok := e.alerts[id]
	if !ok {
		return fmt.Errorf("alert not found: %s", id)
	}

	now := time.Now()
	alert.Status = StatusResolved
	alert.ResolvedAt = &now
	alert.ResolvedBy = user

	return nil
}

// SnoozeAlert snoozes an alert
func (e *Engine) SnoozeAlert(id string, duration time.Duration) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	alert, ok := e.alerts[id]
	if !ok {
		return fmt.Errorf("alert not found: %s", id)
	}

	until := time.Now().Add(duration)
	alert.Status = StatusSnoozed
	alert.SnoozedUntil = &until

	return nil
}

// GetSummary returns an alert summary
func (e *Engine) GetSummary() *AlertSummary {
	e.mu.RLock()
	defer e.mu.RUnlock()

	summary := &AlertSummary{
		BySeverity: make(map[string]int),
		ByType:     make(map[string]int),
	}

	hourAgo := time.Now().Add(-time.Hour)

	for _, alert := range e.alerts {
		summary.TotalAlerts++
		summary.BySeverity[string(alert.Severity)]++
		summary.ByType[string(alert.Type)]++

		switch alert.Status {
		case StatusOpen:
			summary.OpenAlerts++
		case StatusAcknowledged:
			summary.AckedAlerts++
		case StatusResolved:
			summary.ResolvedAlerts++
		}

		if alert.FiredAt.After(hourAgo) {
			summary.RecentFired++
		}
	}

	return summary
}

// GetHistory returns alert history for a rule
func (e *Engine) GetHistory(ruleID string, limit int) []*Alert {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var history []*Alert
	for _, alert := range e.alerts {
		if alert.RuleID == ruleID {
			history = append(history, alert)
		}
	}

	// Sort by time descending and limit
	if len(history) > limit && limit > 0 {
		history = history[:limit]
	}

	return history
}
