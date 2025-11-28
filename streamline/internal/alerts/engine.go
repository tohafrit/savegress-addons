package alerts

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/savegress/streamline/internal/connectors"
)

// Engine manages alert rules, evaluates conditions, and dispatches notifications
type Engine struct {
	rules      map[string]*AlertRule
	alerts     map[string]*Alert
	channels   map[string]*NotificationChannel
	notifiers  map[ActionType]Notifier
	inventory  map[string]*InventorySnapshot // key: productID:channelID
	salesData  map[string][]*SalesMetric     // key: productID:channelID
	mu         sync.RWMutex
	running    bool
	stopCh     chan struct{}
	alertCh    chan *Alert
	eventCh    chan *InventoryEvent
}

// InventoryEvent represents an inventory change event
type InventoryEvent struct {
	ProductID  string
	SKU        string
	ChannelID  string
	OldQty     int
	NewQty     int
	Timestamp  time.Time
}

// Notifier interface for sending notifications
type Notifier interface {
	Send(ctx context.Context, alert *Alert, action AlertAction) error
	Type() ActionType
}

// NewEngine creates a new alert engine
func NewEngine() *Engine {
	return &Engine{
		rules:     make(map[string]*AlertRule),
		alerts:    make(map[string]*Alert),
		channels:  make(map[string]*NotificationChannel),
		notifiers: make(map[ActionType]Notifier),
		inventory: make(map[string]*InventorySnapshot),
		salesData: make(map[string][]*SalesMetric),
		stopCh:    make(chan struct{}),
		alertCh:   make(chan *Alert, 100),
		eventCh:   make(chan *InventoryEvent, 1000),
	}
}

// RegisterNotifier registers a notifier for an action type
func (e *Engine) RegisterNotifier(notifier Notifier) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.notifiers[notifier.Type()] = notifier
}

// Start starts the alert engine
func (e *Engine) Start(ctx context.Context) error {
	e.mu.Lock()
	if e.running {
		e.mu.Unlock()
		return nil
	}
	e.running = true
	e.mu.Unlock()

	go e.processAlerts(ctx)
	go e.processInventoryEvents(ctx)
	go e.periodicEvaluation(ctx)

	return nil
}

// Stop stops the alert engine
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
			e.dispatchAlert(ctx, alert)
		}
	}
}

func (e *Engine) processInventoryEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-e.stopCh:
			return
		case event := <-e.eventCh:
			e.evaluateInventoryEvent(event)
		}
	}
}

func (e *Engine) periodicEvaluation(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
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

// OnInventoryChange handles inventory change notifications
func (e *Engine) OnInventoryChange(productID, sku, channelID string, oldQty, newQty int) {
	event := &InventoryEvent{
		ProductID: productID,
		SKU:       sku,
		ChannelID: channelID,
		OldQty:    oldQty,
		NewQty:    newQty,
		Timestamp: time.Now(),
	}

	select {
	case e.eventCh <- event:
	default:
		// Channel full, drop event
	}

	// Update snapshot
	key := fmt.Sprintf("%s:%s", productID, channelID)
	e.mu.Lock()
	e.inventory[key] = &InventorySnapshot{
		ProductID: productID,
		SKU:       sku,
		ChannelID: channelID,
		Quantity:  newQty,
		Timestamp: time.Now(),
	}
	e.mu.Unlock()
}

// OnSale records a sale for analysis
func (e *Engine) OnSale(productID, sku, channelID string, units int, revenue float64) {
	key := fmt.Sprintf("%s:%s", productID, channelID)
	metric := &SalesMetric{
		ProductID:  productID,
		SKU:        sku,
		ChannelID:  channelID,
		Units:      units,
		Revenue:    revenue,
		Period:     time.Now().Truncate(time.Hour),
		PeriodType: "hourly",
	}

	e.mu.Lock()
	e.salesData[key] = append(e.salesData[key], metric)
	// Keep only last 7 days of data
	cutoff := time.Now().Add(-7 * 24 * time.Hour)
	filtered := make([]*SalesMetric, 0)
	for _, m := range e.salesData[key] {
		if m.Period.After(cutoff) {
			filtered = append(filtered, m)
		}
	}
	e.salesData[key] = filtered
	e.mu.Unlock()

	// Evaluate sales-related rules
	e.evaluateSalesRules(productID, channelID)
}

func (e *Engine) evaluateInventoryEvent(event *InventoryEvent) {
	e.mu.RLock()
	rules := make([]*AlertRule, 0)
	for _, rule := range e.rules {
		if !rule.Enabled {
			continue
		}
		switch rule.Type {
		case AlertTypeLowStock, AlertTypeOutOfStock, AlertTypeOverstock, AlertTypeInventoryMismatch:
			if e.ruleAppliesToChannel(rule, event.ChannelID) {
				rules = append(rules, rule)
			}
		}
	}
	e.mu.RUnlock()

	for _, rule := range rules {
		if e.shouldFireAlert(rule, event) {
			e.fireAlert(rule, event)
		}
	}
}

func (e *Engine) ruleAppliesToChannel(rule *AlertRule, channelID string) bool {
	if len(rule.Channels) == 0 {
		return true // Applies to all channels
	}
	for _, ch := range rule.Channels {
		if ch == channelID {
			return true
		}
	}
	return false
}

func (e *Engine) shouldFireAlert(rule *AlertRule, event *InventoryEvent) bool {
	// Check cooldown
	if rule.LastFiredAt != nil && time.Since(*rule.LastFiredAt) < rule.Cooldown {
		return false
	}

	// Check schedule
	if rule.Schedule != nil && !e.isScheduleActive(rule.Schedule) {
		return false
	}

	// Evaluate condition
	switch rule.Type {
	case AlertTypeLowStock:
		return event.NewQty <= rule.Condition.Threshold && event.NewQty > 0

	case AlertTypeOutOfStock:
		return event.NewQty == 0 && event.OldQty > 0

	case AlertTypeOverstock:
		return event.NewQty >= rule.Condition.Threshold

	case AlertTypeInventoryMismatch:
		change := float64(event.NewQty-event.OldQty) / float64(max(event.OldQty, 1)) * 100
		return absFloat(change) >= rule.Condition.PercentChange
	}

	return false
}

func (e *Engine) isScheduleActive(schedule *AlertSchedule) bool {
	loc, err := time.LoadLocation(schedule.Timezone)
	if err != nil {
		loc = time.UTC
	}

	now := time.Now().In(loc)
	weekday := int(now.Weekday())

	// Check active days
	activeDay := false
	for _, day := range schedule.ActiveDays {
		if day == weekday {
			activeDay = true
			break
		}
	}
	if !activeDay {
		return false
	}

	// Check active hours
	currentTime := now.Format("15:04")
	if schedule.ActiveStart != "" && currentTime < schedule.ActiveStart {
		return false
	}
	if schedule.ActiveEnd != "" && currentTime > schedule.ActiveEnd {
		return false
	}

	return true
}

func (e *Engine) fireAlert(rule *AlertRule, event *InventoryEvent) {
	alert := &Alert{
		ID:        fmt.Sprintf("alert_%d", time.Now().UnixNano()),
		RuleID:    rule.ID,
		RuleName:  rule.Name,
		Type:      rule.Type,
		Severity:  e.getSeverity(rule, event),
		Status:    AlertStatusOpen,
		Title:     e.formatAlertTitle(rule, event),
		Message:   e.formatAlertMessage(rule, event),
		ChannelID: event.ChannelID,
		ProductID: event.ProductID,
		ProductSKU: event.SKU,
		Data: map[string]interface{}{
			"old_quantity": event.OldQty,
			"new_quantity": event.NewQty,
			"threshold":    rule.Condition.Threshold,
		},
		FiredAt: time.Now(),
	}

	e.mu.Lock()
	e.alerts[alert.ID] = alert
	now := time.Now()
	rule.LastFiredAt = &now
	e.mu.Unlock()

	select {
	case e.alertCh <- alert:
	default:
	}
}

func (e *Engine) getSeverity(rule *AlertRule, event *InventoryEvent) AlertSeverity {
	switch rule.Type {
	case AlertTypeOutOfStock:
		return SeverityCritical
	case AlertTypeLowStock:
		if event.NewQty <= rule.Condition.Threshold/2 {
			return SeverityHigh
		}
		return SeverityMedium
	case AlertTypeOverstock:
		return SeverityLow
	case AlertTypeSyncError, AlertTypeChannelDown:
		return SeverityHigh
	default:
		return SeverityMedium
	}
}

func (e *Engine) formatAlertTitle(rule *AlertRule, event *InventoryEvent) string {
	switch rule.Type {
	case AlertTypeLowStock:
		return fmt.Sprintf("Low Stock Alert: %s", event.SKU)
	case AlertTypeOutOfStock:
		return fmt.Sprintf("Out of Stock: %s", event.SKU)
	case AlertTypeOverstock:
		return fmt.Sprintf("Overstock Alert: %s", event.SKU)
	case AlertTypeInventoryMismatch:
		return fmt.Sprintf("Inventory Mismatch: %s", event.SKU)
	default:
		return fmt.Sprintf("[%s] %s", rule.Type, rule.Name)
	}
}

func (e *Engine) formatAlertMessage(rule *AlertRule, event *InventoryEvent) string {
	switch rule.Type {
	case AlertTypeLowStock:
		return fmt.Sprintf("Product %s has low stock. Current quantity: %d (threshold: %d)",
			event.SKU, event.NewQty, rule.Condition.Threshold)
	case AlertTypeOutOfStock:
		return fmt.Sprintf("Product %s is now out of stock on channel %s. Previous quantity: %d",
			event.SKU, event.ChannelID, event.OldQty)
	case AlertTypeOverstock:
		return fmt.Sprintf("Product %s exceeds overstock threshold. Current quantity: %d (threshold: %d)",
			event.SKU, event.NewQty, rule.Condition.Threshold)
	case AlertTypeInventoryMismatch:
		change := float64(event.NewQty-event.OldQty) / float64(max(event.OldQty, 1)) * 100
		return fmt.Sprintf("Significant inventory change detected for %s: %d -> %d (%.1f%% change)",
			event.SKU, event.OldQty, event.NewQty, change)
	default:
		return rule.Description
	}
}

func (e *Engine) dispatchAlert(ctx context.Context, alert *Alert) {
	e.mu.RLock()
	rule, ok := e.rules[alert.RuleID]
	if !ok {
		e.mu.RUnlock()
		return
	}
	actions := rule.Actions
	e.mu.RUnlock()

	for _, action := range actions {
		e.mu.RLock()
		notifier, ok := e.notifiers[action.Type]
		e.mu.RUnlock()

		if !ok {
			continue
		}

		go func(n Notifier, a AlertAction) {
			if err := n.Send(ctx, alert, a); err != nil {
				// Log error
			}
		}(notifier, action)
	}
}

func (e *Engine) evaluateSalesRules(productID, channelID string) {
	e.mu.RLock()
	rules := make([]*AlertRule, 0)
	for _, rule := range e.rules {
		if !rule.Enabled {
			continue
		}
		switch rule.Type {
		case AlertTypeSalesSpike, AlertTypeSalesDrop:
			if e.ruleAppliesToChannel(rule, channelID) {
				rules = append(rules, rule)
			}
		}
	}

	key := fmt.Sprintf("%s:%s", productID, channelID)
	salesData := e.salesData[key]
	e.mu.RUnlock()

	if len(salesData) < 2 {
		return
	}

	// Calculate recent vs historical sales
	recent := 0
	historical := 0
	now := time.Now()
	recentCutoff := now.Add(-24 * time.Hour)
	historicalCutoff := now.Add(-7 * 24 * time.Hour)

	for _, m := range salesData {
		if m.Period.After(recentCutoff) {
			recent += m.Units
		} else if m.Period.After(historicalCutoff) {
			historical += m.Units
		}
	}

	// Calculate average daily historical sales
	avgDaily := float64(historical) / 6.0 // 6 days excluding recent

	for _, rule := range rules {
		change := (float64(recent) - avgDaily) / maxFloat(avgDaily, 1) * 100

		switch rule.Type {
		case AlertTypeSalesSpike:
			if change >= rule.Condition.PercentChange {
				e.fireSalesAlert(rule, productID, channelID, recent, int(avgDaily), change)
			}
		case AlertTypeSalesDrop:
			if change <= -rule.Condition.PercentChange {
				e.fireSalesAlert(rule, productID, channelID, recent, int(avgDaily), change)
			}
		}
	}
}

func (e *Engine) fireSalesAlert(rule *AlertRule, productID, channelID string, recent, historical int, change float64) {
	// Check cooldown
	if rule.LastFiredAt != nil && time.Since(*rule.LastFiredAt) < rule.Cooldown {
		return
	}

	var title, message string
	var severity AlertSeverity

	if rule.Type == AlertTypeSalesSpike {
		title = fmt.Sprintf("Sales Spike Detected: %s", productID)
		message = fmt.Sprintf("Sales for product %s increased by %.1f%%. Recent: %d units, Average: %d units",
			productID, change, recent, historical)
		severity = SeverityMedium
	} else {
		title = fmt.Sprintf("Sales Drop Detected: %s", productID)
		message = fmt.Sprintf("Sales for product %s decreased by %.1f%%. Recent: %d units, Average: %d units",
			productID, absFloat(change), recent, historical)
		severity = SeverityHigh
	}

	alert := &Alert{
		ID:        fmt.Sprintf("alert_%d", time.Now().UnixNano()),
		RuleID:    rule.ID,
		RuleName:  rule.Name,
		Type:      rule.Type,
		Severity:  severity,
		Status:    AlertStatusOpen,
		Title:     title,
		Message:   message,
		ChannelID: channelID,
		ProductID: productID,
		Data: map[string]interface{}{
			"recent_sales":     recent,
			"historical_sales": historical,
			"percent_change":   change,
		},
		FiredAt: time.Now(),
	}

	e.mu.Lock()
	e.alerts[alert.ID] = alert
	now := time.Now()
	rule.LastFiredAt = &now
	e.mu.Unlock()

	select {
	case e.alertCh <- alert:
	default:
	}
}

func (e *Engine) evaluateAllRules(ctx context.Context) {
	e.mu.RLock()
	snapshots := make([]*InventorySnapshot, 0, len(e.inventory))
	for _, s := range e.inventory {
		snapshots = append(snapshots, s)
	}
	e.mu.RUnlock()

	for _, snapshot := range snapshots {
		event := &InventoryEvent{
			ProductID: snapshot.ProductID,
			SKU:       snapshot.SKU,
			ChannelID: snapshot.ChannelID,
			OldQty:    snapshot.Quantity,
			NewQty:    snapshot.Quantity,
			Timestamp: snapshot.Timestamp,
		}
		e.evaluateInventoryEvent(event)
	}
}

// AddRule adds an alert rule
func (e *Engine) AddRule(rule *AlertRule) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if rule.ID == "" {
		rule.ID = fmt.Sprintf("rule_%d", time.Now().UnixNano())
	}
	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()

	e.rules[rule.ID] = rule
	return nil
}

// GetRule returns a rule by ID
func (e *Engine) GetRule(id string) (*AlertRule, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	rule, ok := e.rules[id]
	return rule, ok
}

// ListRules returns all rules
func (e *Engine) ListRules() []*AlertRule {
	e.mu.RLock()
	defer e.mu.RUnlock()

	rules := make([]*AlertRule, 0, len(e.rules))
	for _, r := range e.rules {
		rules = append(rules, r)
	}
	return rules
}

// UpdateRule updates a rule
func (e *Engine) UpdateRule(rule *AlertRule) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, ok := e.rules[rule.ID]; !ok {
		return fmt.Errorf("rule not found: %s", rule.ID)
	}

	rule.UpdatedAt = time.Now()
	e.rules[rule.ID] = rule
	return nil
}

// DeleteRule deletes a rule
func (e *Engine) DeleteRule(id string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.rules, id)
}

// GetAlert returns an alert by ID
func (e *Engine) GetAlert(id string) (*Alert, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	alert, ok := e.alerts[id]
	return alert, ok
}

// ListAlerts returns alerts matching the filter
func (e *Engine) ListAlerts(filter AlertFilter) []*Alert {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var results []*Alert
	for _, alert := range e.alerts {
		if e.matchesFilter(alert, filter) {
			results = append(results, alert)
		}
	}

	// Apply limit and offset
	if filter.Offset > 0 && filter.Offset < len(results) {
		results = results[filter.Offset:]
	}
	if filter.Limit > 0 && filter.Limit < len(results) {
		results = results[:filter.Limit]
	}

	return results
}

func (e *Engine) matchesFilter(alert *Alert, filter AlertFilter) bool {
	if filter.Status != "" && alert.Status != filter.Status {
		return false
	}
	if filter.Type != "" && alert.Type != filter.Type {
		return false
	}
	if filter.Severity != "" && alert.Severity != filter.Severity {
		return false
	}
	if filter.ChannelID != "" && alert.ChannelID != filter.ChannelID {
		return false
	}
	if filter.ProductID != "" && alert.ProductID != filter.ProductID {
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
	alert.Status = AlertStatusAcknowledged
	alert.AckedAt = &now
	alert.AckedBy = user

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
	alert.Status = AlertStatusResolved
	alert.ResolvedAt = &now
	alert.ResolvedBy = user

	return nil
}

// SnoozeAlert snoozes an alert
func (e *Engine) SnoozeAlert(id string, until time.Time) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	alert, ok := e.alerts[id]
	if !ok {
		return fmt.Errorf("alert not found: %s", id)
	}

	alert.Status = AlertStatusSnoozed
	alert.SnoozedUntil = &until

	return nil
}

// AddAlertNote adds a note to an alert
func (e *Engine) AddAlertNote(alertID, author, content string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	alert, ok := e.alerts[alertID]
	if !ok {
		return fmt.Errorf("alert not found: %s", alertID)
	}

	note := AlertNote{
		ID:        fmt.Sprintf("note_%d", time.Now().UnixNano()),
		Author:    author,
		Content:   content,
		CreatedAt: time.Now(),
	}

	alert.Notes = append(alert.Notes, note)
	return nil
}

// AddNotificationChannel adds a notification channel
func (e *Engine) AddNotificationChannel(channel *NotificationChannel) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if channel.ID == "" {
		channel.ID = fmt.Sprintf("channel_%d", time.Now().UnixNano())
	}
	channel.CreatedAt = time.Now()
	channel.UpdatedAt = time.Now()

	e.channels[channel.ID] = channel
	return nil
}

// GetNotificationChannel returns a notification channel by ID
func (e *Engine) GetNotificationChannel(id string) (*NotificationChannel, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	channel, ok := e.channels[id]
	return channel, ok
}

// ListNotificationChannels returns all notification channels
func (e *Engine) ListNotificationChannels() []*NotificationChannel {
	e.mu.RLock()
	defer e.mu.RUnlock()

	channels := make([]*NotificationChannel, 0, len(e.channels))
	for _, ch := range e.channels {
		channels = append(channels, ch)
	}
	return channels
}

// DeleteNotificationChannel deletes a notification channel
func (e *Engine) DeleteNotificationChannel(id string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.channels, id)
}

// GetStats returns alert statistics
func (e *Engine) GetStats() *AlertStats {
	e.mu.RLock()
	defer e.mu.RUnlock()

	stats := &AlertStats{
		TotalAlerts: len(e.alerts),
		ByType:      make(map[string]int),
		BySeverity:  make(map[string]int),
		ByChannel:   make(map[string]int),
	}

	now := time.Now()
	day := now.Add(-24 * time.Hour)
	week := now.Add(-7 * 24 * time.Hour)

	for _, alert := range e.alerts {
		switch alert.Status {
		case AlertStatusOpen:
			stats.OpenAlerts++
		case AlertStatusAcknowledged:
			stats.AckedAlerts++
		case AlertStatusResolved:
			stats.ResolvedAlerts++
		}

		stats.ByType[string(alert.Type)]++
		stats.BySeverity[string(alert.Severity)]++
		if alert.ChannelID != "" {
			stats.ByChannel[alert.ChannelID]++
		}

		if alert.FiredAt.After(day) {
			stats.Last24Hours++
		}
		if alert.FiredAt.After(week) {
			stats.Last7Days++
		}
	}

	return stats
}

// EvaluateConnectorStatus checks connector health and fires alerts
func (e *Engine) EvaluateConnectorStatus(connector connectors.Connector, err error) {
	if err == nil {
		return
	}

	e.mu.RLock()
	var matchingRule *AlertRule
	for _, rule := range e.rules {
		if rule.Enabled && rule.Type == AlertTypeChannelDown {
			matchingRule = rule
			break
		}
	}
	e.mu.RUnlock()

	if matchingRule == nil {
		return
	}

	alert := &Alert{
		ID:          fmt.Sprintf("alert_%d", time.Now().UnixNano()),
		RuleID:      matchingRule.ID,
		RuleName:    matchingRule.Name,
		Type:        AlertTypeChannelDown,
		Severity:    SeverityCritical,
		Status:      AlertStatusOpen,
		Title:       fmt.Sprintf("Channel Down: %s", connector.GetName()),
		Message:     fmt.Sprintf("Failed to connect to %s channel: %s", connector.GetType(), err.Error()),
		ChannelID:   string(connector.GetType()),
		ChannelName: connector.GetName(),
		Data: map[string]interface{}{
			"error": err.Error(),
		},
		FiredAt: time.Now(),
	}

	e.mu.Lock()
	e.alerts[alert.ID] = alert
	e.mu.Unlock()

	select {
	case e.alertCh <- alert:
	default:
	}
}

// Helper functions
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func absFloat(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}
