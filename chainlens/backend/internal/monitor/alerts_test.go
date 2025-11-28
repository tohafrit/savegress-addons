package monitor

import (
	"context"
	"math/big"
	"testing"
	"time"
)

type mockNotifier struct {
	sentAlerts []*Alert
	returnErr  error
}

func (n *mockNotifier) Send(ctx context.Context, alert *Alert) error {
	n.sentAlerts = append(n.sentAlerts, alert)
	return n.returnErr
}

func (n *mockNotifier) Type() string {
	return "mock"
}

func TestNewAlertManager(t *testing.T) {
	m := NewAlertManager()
	if m == nil {
		t.Fatal("expected manager to be created")
	}
	if m.rules == nil {
		t.Error("rules map not initialized")
	}
	if m.alerts == nil {
		t.Error("alerts map not initialized")
	}
	if m.notifiers == nil {
		t.Error("notifiers map not initialized")
	}
}

func TestAlertManager_RegisterNotifier(t *testing.T) {
	m := NewAlertManager()
	notifier := &mockNotifier{}

	m.RegisterNotifier("test", notifier)

	m.mu.RLock()
	n, ok := m.notifiers["test"]
	m.mu.RUnlock()

	if !ok {
		t.Error("notifier not registered")
	}
	if n != notifier {
		t.Error("wrong notifier registered")
	}
}

func TestAlertManager_StartStop(t *testing.T) {
	m := NewAlertManager()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := m.Start(ctx)
	if err != nil {
		t.Errorf("Start() error = %v", err)
	}

	if !m.running {
		t.Error("expected manager to be running")
	}

	// Start again should be no-op
	err = m.Start(ctx)
	if err != nil {
		t.Errorf("Start() second call error = %v", err)
	}

	m.Stop()

	if m.running {
		t.Error("expected manager to be stopped")
	}

	// Stop again should be no-op
	m.Stop() // Should not panic
}

func TestAlertManager_AddRule(t *testing.T) {
	m := NewAlertManager()

	rule := &AlertRule{
		Name:    "Test Rule",
		Type:    AlertTypeEvent,
		Enabled: true,
	}

	m.AddRule(rule)

	if rule.ID == "" {
		t.Error("expected rule ID to be generated")
	}
	if rule.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
	if rule.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}

	// Add rule with ID
	rule2 := &AlertRule{
		ID:      "custom-id",
		Name:    "Test Rule 2",
		Type:    AlertTypeFailedTx,
		Enabled: true,
	}
	m.AddRule(rule2)

	if rule2.ID != "custom-id" {
		t.Errorf("expected rule ID 'custom-id', got %s", rule2.ID)
	}
}

func TestAlertManager_GetRule(t *testing.T) {
	m := NewAlertManager()

	rule := &AlertRule{
		ID:   "test-rule",
		Name: "Test Rule",
	}
	m.AddRule(rule)

	found, ok := m.GetRule("test-rule")
	if !ok {
		t.Error("expected rule to be found")
	}
	if found.Name != "Test Rule" {
		t.Errorf("expected name 'Test Rule', got %s", found.Name)
	}

	_, ok = m.GetRule("nonexistent")
	if ok {
		t.Error("expected rule not to be found")
	}
}

func TestAlertManager_ListRules(t *testing.T) {
	m := NewAlertManager()

	m.AddRule(&AlertRule{ID: "rule-1", Name: "Rule 1"})
	m.AddRule(&AlertRule{ID: "rule-2", Name: "Rule 2"})
	m.AddRule(&AlertRule{ID: "rule-3", Name: "Rule 3"})

	rules := m.ListRules()
	if len(rules) != 3 {
		t.Errorf("expected 3 rules, got %d", len(rules))
	}
}

func TestAlertManager_UpdateRule(t *testing.T) {
	m := NewAlertManager()

	rule := &AlertRule{
		ID:   "test-rule",
		Name: "Test Rule",
	}
	m.AddRule(rule)

	updated := &AlertRule{
		ID:   "test-rule",
		Name: "Updated Rule",
	}

	err := m.UpdateRule(updated)
	if err != nil {
		t.Errorf("UpdateRule() error = %v", err)
	}

	found, _ := m.GetRule("test-rule")
	if found.Name != "Updated Rule" {
		t.Errorf("expected name 'Updated Rule', got %s", found.Name)
	}

	// Update nonexistent rule
	err = m.UpdateRule(&AlertRule{ID: "nonexistent"})
	if err == nil {
		t.Error("expected error for nonexistent rule")
	}
}

func TestAlertManager_DeleteRule(t *testing.T) {
	m := NewAlertManager()

	m.AddRule(&AlertRule{ID: "test-rule"})

	m.DeleteRule("test-rule")

	_, ok := m.GetRule("test-rule")
	if ok {
		t.Error("expected rule to be deleted")
	}
}

func TestAlertManager_EvaluateEvent(t *testing.T) {
	m := NewAlertManager()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m.Start(ctx)
	defer m.Stop()

	rule := &AlertRule{
		ID:       "event-rule",
		Name:     "Event Rule",
		Type:     AlertTypeEvent,
		Contract: "0x123",
		ChainID:  1,
		Enabled:  true,
		Condition: AlertCondition{
			Event: "Transfer",
		},
		Channels: []string{"test"},
	}
	m.AddRule(rule)

	notifier := &mockNotifier{}
	m.RegisterNotifier("test", notifier)

	event := &ContractEvent{
		ContractAddress: "0x123",
		ChainID:         1,
		EventName:       "Transfer",
		TxHash:          "0xabc",
		BlockNumber:     12345,
	}

	m.EvaluateEvent(event)
	time.Sleep(50 * time.Millisecond)

	alerts := m.ListAlerts(AlertFilter{})
	if len(alerts) == 0 {
		t.Error("expected alert to be created")
	}
}

func TestAlertManager_EvaluateEvent_NoMatch(t *testing.T) {
	m := NewAlertManager()

	rule := &AlertRule{
		ID:       "event-rule",
		Type:     AlertTypeEvent,
		Contract: "0x123",
		ChainID:  1,
		Enabled:  true,
		Condition: AlertCondition{
			Event: "Transfer",
		},
	}
	m.AddRule(rule)

	event := &ContractEvent{
		ContractAddress: "0x456", // Different contract
		ChainID:         1,
		EventName:       "Transfer",
	}

	m.EvaluateEvent(event)

	alerts := m.ListAlerts(AlertFilter{})
	if len(alerts) != 0 {
		t.Error("expected no alert for mismatched contract")
	}
}

func TestAlertManager_EvaluateTransaction_LargeTransfer(t *testing.T) {
	m := NewAlertManager()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m.Start(ctx)
	defer m.Stop()

	rule := &AlertRule{
		ID:      "large-transfer-rule",
		Name:    "Large Transfer Rule",
		Type:    AlertTypeLargeTransfer,
		ChainID: 1,
		Enabled: true,
		Condition: AlertCondition{
			Threshold: 1.0, // 1 ETH
		},
	}
	m.AddRule(rule)

	tx := &Transaction{
		Hash:    "0xabc",
		ChainID: 1,
		Value:   big.NewInt(2e18), // 2 ETH
		From:    "0x111",
		To:      "0x222",
	}

	m.EvaluateTransaction(tx)
	time.Sleep(50 * time.Millisecond)

	alerts := m.ListAlerts(AlertFilter{Type: AlertTypeLargeTransfer})
	if len(alerts) == 0 {
		t.Error("expected large transfer alert")
	}
}

func TestAlertManager_EvaluateTransaction_FailedTx(t *testing.T) {
	m := NewAlertManager()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m.Start(ctx)
	defer m.Stop()

	rule := &AlertRule{
		ID:      "failed-tx-rule",
		Name:    "Failed TX Rule",
		Type:    AlertTypeFailedTx,
		ChainID: 1,
		Enabled: true,
	}
	m.AddRule(rule)

	tx := &Transaction{
		Hash:    "0xabc",
		ChainID: 1,
		Status:  false, // Failed
		Value:   big.NewInt(0),
		From:    "0x111",
		To:      "0x222",
	}

	m.EvaluateTransaction(tx)
	time.Sleep(50 * time.Millisecond)

	alerts := m.ListAlerts(AlertFilter{Type: AlertTypeFailedTx})
	if len(alerts) == 0 {
		t.Error("expected failed tx alert")
	}
}

func TestAlertManager_EvaluateTransaction_GasSpike(t *testing.T) {
	m := NewAlertManager()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m.Start(ctx)
	defer m.Stop()

	rule := &AlertRule{
		ID:      "gas-spike-rule",
		Name:    "Gas Spike Rule",
		Type:    AlertTypeGasSpike,
		ChainID: 1,
		Enabled: true,
		Condition: AlertCondition{
			Threshold: 100000,
		},
	}
	m.AddRule(rule)

	tx := &Transaction{
		Hash:     "0xabc",
		ChainID:  1,
		Status:   true,
		Value:    big.NewInt(0),
		GasUsed:  500000, // High gas
		GasPrice: big.NewInt(100e9),
		From:     "0x111",
		To:       "0x222",
	}

	m.EvaluateTransaction(tx)
	time.Sleep(50 * time.Millisecond)

	alerts := m.ListAlerts(AlertFilter{Type: AlertTypeGasSpike})
	if len(alerts) == 0 {
		t.Error("expected gas spike alert")
	}
}

func TestAlertManager_EvaluateBalance_Low(t *testing.T) {
	m := NewAlertManager()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m.Start(ctx)
	defer m.Stop()

	rule := &AlertRule{
		ID:       "low-balance-rule",
		Name:     "Low Balance Rule",
		Type:     AlertTypeLowBalance,
		Contract: "0x123",
		ChainID:  1,
		Enabled:  true,
		Condition: AlertCondition{
			Threshold: 1.0, // 1 ETH
		},
	}
	m.AddRule(rule)

	balance := big.NewInt(5e17) // 0.5 ETH - below threshold

	m.EvaluateBalance("0x123", 1, balance)
	time.Sleep(50 * time.Millisecond)

	alerts := m.ListAlerts(AlertFilter{Type: AlertTypeLowBalance})
	if len(alerts) == 0 {
		t.Error("expected low balance alert")
	}
}

func TestAlertManager_EvaluateBalance_High(t *testing.T) {
	m := NewAlertManager()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m.Start(ctx)
	defer m.Stop()

	rule := &AlertRule{
		ID:       "high-balance-rule",
		Name:     "High Balance Rule",
		Type:     AlertTypeHighBalance,
		Contract: "0x123",
		ChainID:  1,
		Enabled:  true,
		Condition: AlertCondition{
			Threshold: 100.0, // 100 ETH
		},
	}
	m.AddRule(rule)

	balance := new(big.Int).Mul(big.NewInt(200), big.NewInt(1e18)) // 200 ETH - above threshold

	m.EvaluateBalance("0x123", 1, balance)
	time.Sleep(50 * time.Millisecond)

	alerts := m.ListAlerts(AlertFilter{Type: AlertTypeHighBalance})
	if len(alerts) == 0 {
		t.Error("expected high balance alert")
	}
}

func TestAlertManager_GetAlert(t *testing.T) {
	m := NewAlertManager()

	m.mu.Lock()
	m.alerts["test-alert"] = &Alert{
		ID:    "test-alert",
		Title: "Test Alert",
	}
	m.mu.Unlock()

	alert, ok := m.GetAlert("test-alert")
	if !ok {
		t.Error("expected alert to be found")
	}
	if alert.Title != "Test Alert" {
		t.Errorf("expected title 'Test Alert', got %s", alert.Title)
	}

	_, ok = m.GetAlert("nonexistent")
	if ok {
		t.Error("expected alert not to be found")
	}
}

func TestAlertManager_ListAlerts_WithFilter(t *testing.T) {
	m := NewAlertManager()

	m.mu.Lock()
	m.alerts["alert-1"] = &Alert{ID: "alert-1", Status: AlertStatusOpen, Type: AlertTypeEvent, Severity: "high", Contract: "0x123", ChainID: 1}
	m.alerts["alert-2"] = &Alert{ID: "alert-2", Status: AlertStatusAcked, Type: AlertTypeFailedTx, Severity: "low", Contract: "0x456", ChainID: 137}
	m.alerts["alert-3"] = &Alert{ID: "alert-3", Status: AlertStatusOpen, Type: AlertTypeEvent, Severity: "high", Contract: "0x123", ChainID: 1}
	m.mu.Unlock()

	// Filter by status
	alerts := m.ListAlerts(AlertFilter{Status: AlertStatusOpen})
	if len(alerts) != 2 {
		t.Errorf("expected 2 open alerts, got %d", len(alerts))
	}

	// Filter by type
	alerts = m.ListAlerts(AlertFilter{Type: AlertTypeEvent})
	if len(alerts) != 2 {
		t.Errorf("expected 2 event alerts, got %d", len(alerts))
	}

	// Filter by severity
	alerts = m.ListAlerts(AlertFilter{Severity: "high"})
	if len(alerts) != 2 {
		t.Errorf("expected 2 high severity alerts, got %d", len(alerts))
	}

	// Filter by contract
	alerts = m.ListAlerts(AlertFilter{Contract: "0x123"})
	if len(alerts) != 2 {
		t.Errorf("expected 2 alerts for contract 0x123, got %d", len(alerts))
	}

	// Filter by chain
	alerts = m.ListAlerts(AlertFilter{ChainID: 137})
	if len(alerts) != 1 {
		t.Errorf("expected 1 alert for chain 137, got %d", len(alerts))
	}
}

func TestAlertManager_AcknowledgeAlert(t *testing.T) {
	m := NewAlertManager()

	m.mu.Lock()
	m.alerts["test-alert"] = &Alert{
		ID:     "test-alert",
		Status: AlertStatusOpen,
	}
	m.mu.Unlock()

	err := m.AcknowledgeAlert("test-alert", "user@test.com")
	if err != nil {
		t.Errorf("AcknowledgeAlert() error = %v", err)
	}

	alert, _ := m.GetAlert("test-alert")
	if alert.Status != AlertStatusAcked {
		t.Errorf("expected status %s, got %s", AlertStatusAcked, alert.Status)
	}
	if alert.AckedBy != "user@test.com" {
		t.Errorf("expected AckedBy 'user@test.com', got %s", alert.AckedBy)
	}
	if alert.AckedAt == nil {
		t.Error("expected AckedAt to be set")
	}

	// Ack nonexistent alert
	err = m.AcknowledgeAlert("nonexistent", "user")
	if err == nil {
		t.Error("expected error for nonexistent alert")
	}
}

func TestAlertManager_CloseAlert(t *testing.T) {
	m := NewAlertManager()

	m.mu.Lock()
	m.alerts["test-alert"] = &Alert{
		ID:     "test-alert",
		Status: AlertStatusAcked,
	}
	m.mu.Unlock()

	err := m.CloseAlert("test-alert")
	if err != nil {
		t.Errorf("CloseAlert() error = %v", err)
	}

	alert, _ := m.GetAlert("test-alert")
	if alert.Status != AlertStatusClosed {
		t.Errorf("expected status %s, got %s", AlertStatusClosed, alert.Status)
	}

	// Close nonexistent alert
	err = m.CloseAlert("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent alert")
	}
}

func TestAlertManager_GetAlertStats(t *testing.T) {
	m := NewAlertManager()

	now := time.Now()
	twoHoursAgo := now.Add(-2 * time.Hour)

	m.mu.Lock()
	m.alerts["alert-1"] = &Alert{ID: "alert-1", Status: AlertStatusOpen, Type: AlertTypeEvent, Severity: "high", FiredAt: now}
	m.alerts["alert-2"] = &Alert{ID: "alert-2", Status: AlertStatusAcked, Type: AlertTypeFailedTx, Severity: "low", FiredAt: now}
	m.alerts["alert-3"] = &Alert{ID: "alert-3", Status: AlertStatusOpen, Type: AlertTypeEvent, Severity: "high", FiredAt: twoHoursAgo}
	m.mu.Unlock()

	stats := m.GetAlertStats()

	if stats.Total != 3 {
		t.Errorf("expected total 3, got %d", stats.Total)
	}
	if stats.LastHour != 2 {
		t.Errorf("expected LastHour 2, got %d", stats.LastHour)
	}
	if stats.ByStatus["open"] != 2 {
		t.Errorf("expected 2 open alerts, got %d", stats.ByStatus["open"])
	}
	if stats.ByType["event"] != 2 {
		t.Errorf("expected 2 event alerts, got %d", stats.ByType["event"])
	}
	if stats.BySeverity["high"] != 2 {
		t.Errorf("expected 2 high severity alerts, got %d", stats.BySeverity["high"])
	}
}

func TestGetSeverity(t *testing.T) {
	m := NewAlertManager()

	tests := []struct {
		alertType AlertType
		expected  string
	}{
		{AlertTypeLargeTransfer, "high"},
		{AlertTypeFailedTx, "medium"},
		{AlertTypeGasSpike, "low"},
		{AlertTypeLowBalance, "critical"},
		{AlertTypeEvent, "info"},
		{AlertTypeHighBalance, "info"},
	}

	for _, test := range tests {
		rule := &AlertRule{Type: test.alertType}
		severity := m.getSeverity(rule)
		if severity != test.expected {
			t.Errorf("expected severity %s for type %s, got %s", test.expected, test.alertType, severity)
		}
	}
}

func TestFormatMessage(t *testing.T) {
	m := NewAlertManager()

	tests := []struct {
		alertType AlertType
		data      map[string]interface{}
		contains  string
	}{
		{AlertTypeLargeTransfer, map[string]interface{}{"value": "1000", "from": "0x123", "to": "0x456"}, "Large transfer"},
		{AlertTypeFailedTx, map[string]interface{}{"tx_hash": "0xabc"}, "Transaction failed"},
		{AlertTypeGasSpike, map[string]interface{}{"gas_used": 100000}, "Gas spike"},
		{AlertTypeLowBalance, map[string]interface{}{"address": "0x123", "balance": "1000"}, "Low balance"},
		{AlertTypeEvent, map[string]interface{}{"event_name": "Transfer", "block_number": 12345}, "Event Transfer"},
	}

	for _, test := range tests {
		rule := &AlertRule{Type: test.alertType}
		msg := m.formatMessage(rule, test.data)
		if msg == "" {
			t.Errorf("expected non-empty message for type %s", test.alertType)
		}
	}
}

func TestMatchesEventRule(t *testing.T) {
	m := NewAlertManager()

	rule := &AlertRule{
		Contract: "0x123",
		ChainID:  1,
		Condition: AlertCondition{
			Event: "Transfer",
		},
	}

	// Matching event
	event := &ContractEvent{
		ContractAddress: "0x123",
		ChainID:         1,
		EventName:       "Transfer",
	}
	if !m.matchesEventRule(event, rule) {
		t.Error("expected event to match rule")
	}

	// Wrong contract
	event.ContractAddress = "0x456"
	if m.matchesEventRule(event, rule) {
		t.Error("expected event not to match with wrong contract")
	}

	// Wrong chain
	event.ContractAddress = "0x123"
	event.ChainID = 137
	if m.matchesEventRule(event, rule) {
		t.Error("expected event not to match with wrong chain")
	}

	// Wrong event name
	event.ChainID = 1
	event.EventName = "Approval"
	if m.matchesEventRule(event, rule) {
		t.Error("expected event not to match with wrong event name")
	}
}

func TestMatchesTxRule(t *testing.T) {
	m := NewAlertManager()

	rule := &AlertRule{
		Contract: "0x123",
		ChainID:  1,
	}

	tx := &Transaction{
		To:      "0x123",
		ChainID: 1,
	}

	if !m.matchesTxRule(tx, rule) {
		t.Error("expected tx to match rule")
	}

	tx.To = "0x456"
	if m.matchesTxRule(tx, rule) {
		t.Error("expected tx not to match with wrong contract")
	}
}

func TestProcessAlerts_ContextCancellation(t *testing.T) {
	m := NewAlertManager()

	ctx, cancel := context.WithCancel(context.Background())
	m.Start(ctx)

	// Cancel context
	cancel()
	time.Sleep(50 * time.Millisecond)

	// Should have stopped
}

func TestSendNotifications_MissingRule(t *testing.T) {
	m := NewAlertManager()

	ctx := context.Background()

	alert := &Alert{
		ID:     "test-alert",
		RuleID: "nonexistent",
	}

	// Should not panic
	m.sendNotifications(ctx, alert)
}
