package monitor

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"
)

// AlertManager manages alert rules and notifications
type AlertManager struct {
	rules     map[string]*AlertRule
	alerts    map[string]*Alert
	notifiers map[string]AlertNotifier
	mu        sync.RWMutex
	running   bool
	stopCh    chan struct{}
	alertCh   chan *Alert
}

// AlertNotifier interface for sending alert notifications
type AlertNotifier interface {
	Send(ctx context.Context, alert *Alert) error
	Type() string
}

// NewAlertManager creates a new alert manager
func NewAlertManager() *AlertManager {
	return &AlertManager{
		rules:     make(map[string]*AlertRule),
		alerts:    make(map[string]*Alert),
		notifiers: make(map[string]AlertNotifier),
		stopCh:    make(chan struct{}),
		alertCh:   make(chan *Alert, 100),
	}
}

// RegisterNotifier registers a notification channel
func (m *AlertManager) RegisterNotifier(name string, notifier AlertNotifier) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.notifiers[name] = notifier
}

// Start starts the alert manager
func (m *AlertManager) Start(ctx context.Context) error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return nil
	}
	m.running = true
	m.mu.Unlock()

	go m.processAlerts(ctx)

	return nil
}

// Stop stops the alert manager
func (m *AlertManager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.running {
		close(m.stopCh)
		m.running = false
	}
}

func (m *AlertManager) processAlerts(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		case alert := <-m.alertCh:
			m.sendNotifications(ctx, alert)
		}
	}
}

func (m *AlertManager) sendNotifications(ctx context.Context, alert *Alert) {
	m.mu.RLock()
	rule, ok := m.rules[alert.RuleID]
	if !ok {
		m.mu.RUnlock()
		return
	}
	channels := rule.Channels
	m.mu.RUnlock()

	for _, channel := range channels {
		m.mu.RLock()
		notifier, ok := m.notifiers[channel]
		m.mu.RUnlock()

		if !ok {
			continue
		}

		go func(n AlertNotifier) {
			if err := n.Send(ctx, alert); err != nil {
				// Log error
			}
		}(notifier)
	}
}

// EvaluateEvent evaluates an event against alert rules
func (m *AlertManager) EvaluateEvent(event *ContractEvent) {
	m.mu.RLock()
	rules := make([]*AlertRule, 0)
	for _, rule := range m.rules {
		if rule.Enabled && rule.Type == AlertTypeEvent {
			rules = append(rules, rule)
		}
	}
	m.mu.RUnlock()

	for _, rule := range rules {
		if m.matchesEventRule(event, rule) {
			m.fireAlert(rule, map[string]interface{}{
				"event":        event,
				"event_name":   event.EventName,
				"block_number": event.BlockNumber,
				"tx_hash":      event.TxHash,
			})
		}
	}
}

// EvaluateTransaction evaluates a transaction against alert rules
func (m *AlertManager) EvaluateTransaction(tx *Transaction) {
	m.mu.RLock()
	rules := make([]*AlertRule, 0)
	for _, rule := range m.rules {
		if rule.Enabled {
			rules = append(rules, rule)
		}
	}
	m.mu.RUnlock()

	for _, rule := range rules {
		switch rule.Type {
		case AlertTypeLargeTransfer:
			if m.isLargeTransfer(tx, rule) {
				m.fireAlert(rule, map[string]interface{}{
					"tx_hash": tx.Hash,
					"value":   tx.Value.String(),
					"from":    tx.From,
					"to":      tx.To,
				})
			}

		case AlertTypeFailedTx:
			if !tx.Status && m.matchesTxRule(tx, rule) {
				m.fireAlert(rule, map[string]interface{}{
					"tx_hash": tx.Hash,
					"from":    tx.From,
					"to":      tx.To,
				})
			}

		case AlertTypeGasSpike:
			if m.isGasSpike(tx, rule) {
				m.fireAlert(rule, map[string]interface{}{
					"tx_hash":   tx.Hash,
					"gas_used":  tx.GasUsed,
					"gas_price": tx.GasPrice.String(),
				})
			}
		}
	}
}

// EvaluateBalance evaluates a balance change against alert rules
func (m *AlertManager) EvaluateBalance(address string, chainID int64, balance *big.Int) {
	m.mu.RLock()
	rules := make([]*AlertRule, 0)
	for _, rule := range m.rules {
		if rule.Enabled && (rule.Type == AlertTypeLowBalance || rule.Type == AlertTypeHighBalance) {
			if rule.Contract == address && rule.ChainID == chainID {
				rules = append(rules, rule)
			}
		}
	}
	m.mu.RUnlock()

	for _, rule := range rules {
		threshold := big.NewInt(int64(rule.Condition.Threshold * 1e18)) // Convert ETH to wei

		switch rule.Type {
		case AlertTypeLowBalance:
			if balance.Cmp(threshold) < 0 {
				m.fireAlert(rule, map[string]interface{}{
					"address":   address,
					"balance":   balance.String(),
					"threshold": threshold.String(),
				})
			}

		case AlertTypeHighBalance:
			if balance.Cmp(threshold) > 0 {
				m.fireAlert(rule, map[string]interface{}{
					"address":   address,
					"balance":   balance.String(),
					"threshold": threshold.String(),
				})
			}
		}
	}
}

func (m *AlertManager) matchesEventRule(event *ContractEvent, rule *AlertRule) bool {
	if rule.Contract != "" && event.ContractAddress != rule.Contract {
		return false
	}
	if rule.ChainID != 0 && event.ChainID != rule.ChainID {
		return false
	}
	if rule.Condition.Event != "" && event.EventName != rule.Condition.Event {
		return false
	}
	return true
}

func (m *AlertManager) matchesTxRule(tx *Transaction, rule *AlertRule) bool {
	if rule.Contract != "" && tx.To != rule.Contract {
		return false
	}
	if rule.ChainID != 0 && tx.ChainID != rule.ChainID {
		return false
	}
	return true
}

func (m *AlertManager) isLargeTransfer(tx *Transaction, rule *AlertRule) bool {
	if !m.matchesTxRule(tx, rule) {
		return false
	}
	threshold := big.NewInt(int64(rule.Condition.Threshold * 1e18))
	return tx.Value.Cmp(threshold) >= 0
}

func (m *AlertManager) isGasSpike(tx *Transaction, rule *AlertRule) bool {
	if !m.matchesTxRule(tx, rule) {
		return false
	}
	return float64(tx.GasUsed) > rule.Condition.Threshold
}

func (m *AlertManager) fireAlert(rule *AlertRule, data map[string]interface{}) {
	alert := &Alert{
		ID:       fmt.Sprintf("alert_%d", time.Now().UnixNano()),
		RuleID:   rule.ID,
		RuleName: rule.Name,
		Type:     rule.Type,
		Severity: m.getSeverity(rule),
		Title:    fmt.Sprintf("[%s] %s", rule.Type, rule.Name),
		Message:  m.formatMessage(rule, data),
		Contract: rule.Contract,
		ChainID:  rule.ChainID,
		Data:     data,
		FiredAt:  time.Now(),
		Status:   AlertStatusOpen,
	}

	if txHash, ok := data["tx_hash"].(string); ok {
		alert.TxHash = txHash
	}
	if blockNum, ok := data["block_number"].(uint64); ok {
		alert.BlockNumber = blockNum
	}

	m.mu.Lock()
	m.alerts[alert.ID] = alert
	m.mu.Unlock()

	select {
	case m.alertCh <- alert:
	default:
	}
}

func (m *AlertManager) getSeverity(rule *AlertRule) string {
	switch rule.Type {
	case AlertTypeLargeTransfer:
		return "high"
	case AlertTypeFailedTx:
		return "medium"
	case AlertTypeGasSpike:
		return "low"
	case AlertTypeLowBalance:
		return "critical"
	default:
		return "info"
	}
}

func (m *AlertManager) formatMessage(rule *AlertRule, data map[string]interface{}) string {
	switch rule.Type {
	case AlertTypeLargeTransfer:
		return fmt.Sprintf("Large transfer detected: %s from %s to %s",
			data["value"], data["from"], data["to"])
	case AlertTypeFailedTx:
		return fmt.Sprintf("Transaction failed: %s", data["tx_hash"])
	case AlertTypeGasSpike:
		return fmt.Sprintf("Gas spike detected: %d gas used", data["gas_used"])
	case AlertTypeLowBalance:
		return fmt.Sprintf("Low balance alert for %s: %s wei", data["address"], data["balance"])
	case AlertTypeEvent:
		return fmt.Sprintf("Event %s detected in block %d",
			data["event_name"], data["block_number"])
	default:
		return rule.Description
	}
}

// AddRule adds an alert rule
func (m *AlertManager) AddRule(rule *AlertRule) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if rule.ID == "" {
		rule.ID = fmt.Sprintf("rule_%d", time.Now().UnixNano())
	}
	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()
	m.rules[rule.ID] = rule
}

// GetRule returns a rule by ID
func (m *AlertManager) GetRule(id string) (*AlertRule, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	rule, ok := m.rules[id]
	return rule, ok
}

// ListRules returns all rules
func (m *AlertManager) ListRules() []*AlertRule {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rules := make([]*AlertRule, 0, len(m.rules))
	for _, r := range m.rules {
		rules = append(rules, r)
	}
	return rules
}

// UpdateRule updates a rule
func (m *AlertManager) UpdateRule(rule *AlertRule) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.rules[rule.ID]; !ok {
		return fmt.Errorf("rule not found: %s", rule.ID)
	}

	rule.UpdatedAt = time.Now()
	m.rules[rule.ID] = rule
	return nil
}

// DeleteRule deletes a rule
func (m *AlertManager) DeleteRule(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.rules, id)
}

// GetAlert returns an alert by ID
func (m *AlertManager) GetAlert(id string) (*Alert, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	alert, ok := m.alerts[id]
	return alert, ok
}

// ListAlerts returns alerts with optional filters
func (m *AlertManager) ListAlerts(filter AlertFilter) []*Alert {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var results []*Alert
	for _, alert := range m.alerts {
		if m.matchesAlertFilter(alert, filter) {
			results = append(results, alert)
		}
	}
	return results
}

// AlertFilter defines filters for querying alerts
type AlertFilter struct {
	Status   AlertStatus
	Type     AlertType
	Severity string
	Contract string
	ChainID  int64
	Limit    int
}

func (m *AlertManager) matchesAlertFilter(alert *Alert, filter AlertFilter) bool {
	if filter.Status != "" && alert.Status != filter.Status {
		return false
	}
	if filter.Type != "" && alert.Type != filter.Type {
		return false
	}
	if filter.Severity != "" && alert.Severity != filter.Severity {
		return false
	}
	if filter.Contract != "" && alert.Contract != filter.Contract {
		return false
	}
	if filter.ChainID != 0 && alert.ChainID != filter.ChainID {
		return false
	}
	return true
}

// AcknowledgeAlert acknowledges an alert
func (m *AlertManager) AcknowledgeAlert(id, user string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	alert, ok := m.alerts[id]
	if !ok {
		return fmt.Errorf("alert not found: %s", id)
	}

	now := time.Now()
	alert.Status = AlertStatusAcked
	alert.AckedAt = &now
	alert.AckedBy = user

	return nil
}

// CloseAlert closes an alert
func (m *AlertManager) CloseAlert(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	alert, ok := m.alerts[id]
	if !ok {
		return fmt.Errorf("alert not found: %s", id)
	}

	alert.Status = AlertStatusClosed
	return nil
}

// GetAlertStats returns alert statistics
func (m *AlertManager) GetAlertStats() *AlertStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := &AlertStats{
		Total:      len(m.alerts),
		ByStatus:   make(map[string]int),
		ByType:     make(map[string]int),
		BySeverity: make(map[string]int),
	}

	hourAgo := time.Now().Add(-time.Hour)

	for _, alert := range m.alerts {
		stats.ByStatus[string(alert.Status)]++
		stats.ByType[string(alert.Type)]++
		stats.BySeverity[alert.Severity]++

		if alert.FiredAt.After(hourAgo) {
			stats.LastHour++
		}
	}

	return stats
}

// AlertStats contains alert statistics
type AlertStats struct {
	Total      int            `json:"total"`
	LastHour   int            `json:"last_hour"`
	ByStatus   map[string]int `json:"by_status"`
	ByType     map[string]int `json:"by_type"`
	BySeverity map[string]int `json:"by_severity"`
}
