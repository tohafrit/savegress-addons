package sync

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"getchainlens.com/chainlens/backend/internal/monitor"
)

func TestNewCDCBridge(t *testing.T) {
	mon := monitor.NewContractMonitor()
	client := NewMockCDCClient()

	bridge := NewCDCBridge(mon, client)
	if bridge == nil {
		t.Fatal("NewCDCBridge returned nil")
	}
	if bridge.configs == nil {
		t.Error("configs should not be nil")
	}
	if bridge.status == nil {
		t.Error("status should not be nil")
	}
}

func TestNewCDCBridge_NilClient(t *testing.T) {
	mon := monitor.NewContractMonitor()
	bridge := NewCDCBridge(mon, nil)
	if bridge == nil {
		t.Fatal("NewCDCBridge returned nil")
	}
}

func TestCDCBridge_StartStop(t *testing.T) {
	mon := monitor.NewContractMonitor()
	client := NewMockCDCClient()
	bridge := NewCDCBridge(mon, client)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err := bridge.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Should be idempotent
	err = bridge.Start(ctx)
	if err != nil {
		t.Fatalf("Second Start should not fail: %v", err)
	}

	// Let it run briefly
	time.Sleep(100 * time.Millisecond)

	bridge.Stop()

	// Stop should be idempotent
	bridge.Stop()
}

func TestCDCBridge_AddConfig(t *testing.T) {
	mon := monitor.NewContractMonitor()
	bridge := NewCDCBridge(mon, nil)

	config := &monitor.CDCSyncConfig{
		Contract:       "0x1234567890abcdef1234567890abcdef12345678",
		ChainID:        1,
		TargetDatabase: "analytics",
		Enabled:        true,
	}

	err := bridge.AddConfig(config)
	if err != nil {
		t.Fatalf("AddConfig failed: %v", err)
	}

	if config.ID == "" {
		t.Error("Config ID should be auto-generated")
	}
	if config.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}

	// Verify it was added
	retrieved, ok := bridge.GetConfig(config.ID)
	if !ok {
		t.Error("Config should be retrievable")
	}
	if retrieved.Contract != config.Contract {
		t.Errorf("Contract = %s, want %s", retrieved.Contract, config.Contract)
	}
}

func TestCDCBridge_AddConfig_WithID(t *testing.T) {
	mon := monitor.NewContractMonitor()
	bridge := NewCDCBridge(mon, nil)

	config := &monitor.CDCSyncConfig{
		ID:       "my-custom-id",
		Contract: "0x123",
		ChainID:  1,
	}

	bridge.AddConfig(config)

	retrieved, ok := bridge.GetConfig("my-custom-id")
	if !ok {
		t.Error("Config should be retrievable by custom ID")
	}
	if retrieved.ID != "my-custom-id" {
		t.Errorf("ID = %s, want my-custom-id", retrieved.ID)
	}
}

func TestCDCBridge_GetConfig_NotFound(t *testing.T) {
	mon := monitor.NewContractMonitor()
	bridge := NewCDCBridge(mon, nil)

	_, ok := bridge.GetConfig("non-existent")
	if ok {
		t.Error("Should not find non-existent config")
	}
}

func TestCDCBridge_ListConfigs(t *testing.T) {
	mon := monitor.NewContractMonitor()
	bridge := NewCDCBridge(mon, nil)

	bridge.AddConfig(&monitor.CDCSyncConfig{ID: "c1", Contract: "0x111", ChainID: 1})
	bridge.AddConfig(&monitor.CDCSyncConfig{ID: "c2", Contract: "0x222", ChainID: 1})
	bridge.AddConfig(&monitor.CDCSyncConfig{ID: "c3", Contract: "0x333", ChainID: 137})

	configs := bridge.ListConfigs()
	if len(configs) != 3 {
		t.Errorf("Expected 3 configs, got %d", len(configs))
	}
}

func TestCDCBridge_UpdateConfig(t *testing.T) {
	mon := monitor.NewContractMonitor()
	bridge := NewCDCBridge(mon, nil)

	config := &monitor.CDCSyncConfig{
		ID:       "test-config",
		Contract: "0x111",
		ChainID:  1,
		Enabled:  false,
	}
	bridge.AddConfig(config)

	// Update
	updatedConfig := &monitor.CDCSyncConfig{
		ID:       "test-config",
		Contract: "0x111",
		ChainID:  1,
		Enabled:  true,
	}
	err := bridge.UpdateConfig(updatedConfig)
	if err != nil {
		t.Fatalf("UpdateConfig failed: %v", err)
	}

	retrieved, _ := bridge.GetConfig("test-config")
	if !retrieved.Enabled {
		t.Error("Config should be enabled after update")
	}
}

func TestCDCBridge_UpdateConfig_NotFound(t *testing.T) {
	mon := monitor.NewContractMonitor()
	bridge := NewCDCBridge(mon, nil)

	err := bridge.UpdateConfig(&monitor.CDCSyncConfig{ID: "non-existent"})
	if err == nil {
		t.Error("Expected error for non-existent config")
	}
}

func TestCDCBridge_DeleteConfig(t *testing.T) {
	mon := monitor.NewContractMonitor()
	bridge := NewCDCBridge(mon, nil)

	config := &monitor.CDCSyncConfig{ID: "to-delete", Contract: "0x111"}
	bridge.AddConfig(config)

	bridge.DeleteConfig("to-delete")

	_, ok := bridge.GetConfig("to-delete")
	if ok {
		t.Error("Config should be deleted")
	}
}

func TestCDCBridge_DeleteConfig_NotFound(t *testing.T) {
	mon := monitor.NewContractMonitor()
	bridge := NewCDCBridge(mon, nil)

	// Should not panic
	bridge.DeleteConfig("non-existent")
}

func TestCDCBridge_GetStatus(t *testing.T) {
	mon := monitor.NewContractMonitor()
	bridge := NewCDCBridge(mon, nil)

	config := &monitor.CDCSyncConfig{ID: "test", Contract: "0x111"}
	bridge.AddConfig(config)

	status, ok := bridge.GetStatus("test")
	if !ok {
		t.Fatal("Expected to find status")
	}
	if status.ConfigID != "test" {
		t.Errorf("ConfigID = %s, want test", status.ConfigID)
	}
	if status.Status != "running" {
		t.Errorf("Status = %s, want running", status.Status)
	}
}

func TestCDCBridge_GetStatus_NotFound(t *testing.T) {
	mon := monitor.NewContractMonitor()
	bridge := NewCDCBridge(mon, nil)

	_, ok := bridge.GetStatus("non-existent")
	if ok {
		t.Error("Should not find non-existent status")
	}
}

func TestCDCBridge_GetAllStatus(t *testing.T) {
	mon := monitor.NewContractMonitor()
	bridge := NewCDCBridge(mon, nil)

	bridge.AddConfig(&monitor.CDCSyncConfig{Contract: "0x111"})
	bridge.AddConfig(&monitor.CDCSyncConfig{Contract: "0x222"})

	statuses := bridge.GetAllStatus()
	if len(statuses) != 2 {
		t.Errorf("Expected 2 statuses, got %d", len(statuses))
	}
}

func TestCDCBridge_PauseSync(t *testing.T) {
	mon := monitor.NewContractMonitor()
	bridge := NewCDCBridge(mon, nil)

	config := &monitor.CDCSyncConfig{ID: "test", Contract: "0x111", Enabled: true}
	bridge.AddConfig(config)

	err := bridge.PauseSync("test")
	if err != nil {
		t.Fatalf("PauseSync failed: %v", err)
	}

	retrieved, _ := bridge.GetConfig("test")
	if retrieved.Enabled {
		t.Error("Config should be disabled after pause")
	}

	status, _ := bridge.GetStatus("test")
	if status.Status != "stopped" {
		t.Errorf("Status = %s, want stopped", status.Status)
	}
}

func TestCDCBridge_PauseSync_NotFound(t *testing.T) {
	mon := monitor.NewContractMonitor()
	bridge := NewCDCBridge(mon, nil)

	err := bridge.PauseSync("non-existent")
	if err == nil {
		t.Error("Expected error for non-existent config")
	}
}

func TestCDCBridge_ResumeSync(t *testing.T) {
	mon := monitor.NewContractMonitor()
	bridge := NewCDCBridge(mon, nil)

	config := &monitor.CDCSyncConfig{ID: "test", Contract: "0x111", Enabled: false}
	bridge.AddConfig(config)
	bridge.status["test"].Status = "stopped"

	err := bridge.ResumeSync("test")
	if err != nil {
		t.Fatalf("ResumeSync failed: %v", err)
	}

	retrieved, _ := bridge.GetConfig("test")
	if !retrieved.Enabled {
		t.Error("Config should be enabled after resume")
	}

	status, _ := bridge.GetStatus("test")
	if status.Status != "running" {
		t.Errorf("Status = %s, want running", status.Status)
	}
}

func TestCDCBridge_ResumeSync_NotFound(t *testing.T) {
	mon := monitor.NewContractMonitor()
	bridge := NewCDCBridge(mon, nil)

	err := bridge.ResumeSync("non-existent")
	if err == nil {
		t.Error("Expected error for non-existent config")
	}
}

func TestCDCBridge_CreateConfigFromJSON(t *testing.T) {
	mon := monitor.NewContractMonitor()
	bridge := NewCDCBridge(mon, nil)

	jsonData := []byte(`{
		"id": "json-config",
		"contract": "0x1234567890abcdef1234567890abcdef12345678",
		"chain_id": 1,
		"target_database": "analytics",
		"enabled": true
	}`)

	config, err := bridge.CreateConfigFromJSON(jsonData)
	if err != nil {
		t.Fatalf("CreateConfigFromJSON failed: %v", err)
	}

	if config.Contract != "0x1234567890abcdef1234567890abcdef12345678" {
		t.Errorf("Contract = %s", config.Contract)
	}
}

func TestCDCBridge_CreateConfigFromJSON_Invalid(t *testing.T) {
	mon := monitor.NewContractMonitor()
	bridge := NewCDCBridge(mon, nil)

	_, err := bridge.CreateConfigFromJSON([]byte("invalid json"))
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestCDCBridge_GetBridgeStats(t *testing.T) {
	mon := monitor.NewContractMonitor()
	bridge := NewCDCBridge(mon, nil)

	bridge.AddConfig(&monitor.CDCSyncConfig{ID: "c1", Contract: "0x111", Enabled: true})
	bridge.AddConfig(&monitor.CDCSyncConfig{ID: "c2", Contract: "0x222", Enabled: true})
	bridge.AddConfig(&monitor.CDCSyncConfig{ID: "c3", Contract: "0x333", Enabled: false})

	stats := bridge.GetBridgeStats()
	if stats.TotalConfigs != 3 {
		t.Errorf("TotalConfigs = %d, want 3", stats.TotalConfigs)
	}
	if stats.ActiveConfigs != 2 {
		t.Errorf("ActiveConfigs = %d, want 2", stats.ActiveConfigs)
	}
}

func TestCDCBridge_handleEvent(t *testing.T) {
	mon := monitor.NewContractMonitor()
	bridge := NewCDCBridge(mon, nil)

	event := &monitor.ContractEvent{
		ContractAddress: "0x123",
		EventName:       "Transfer",
		ChainID:         1,
	}

	// Should not block when channel has space
	bridge.handleEvent(event)

	// Read the event
	select {
	case received := <-bridge.eventCh:
		if received.EventName != "Transfer" {
			t.Errorf("EventName = %s, want Transfer", received.EventName)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Event should be received")
	}
}

func TestCDCBridge_handleEvent_ChannelFull(t *testing.T) {
	mon := monitor.NewContractMonitor()
	bridge := &CDCBridge{
		configs:   make(map[string]*monitor.CDCSyncConfig),
		status:    make(map[string]*monitor.SyncStatus),
		monitor:   mon,
		eventCh:   make(chan *monitor.ContractEvent, 1), // Small buffer
		stopCh:    make(chan struct{}),
	}

	// Fill the channel
	bridge.eventCh <- &monitor.ContractEvent{}

	// This should not block even when channel is full
	done := make(chan struct{})
	go func() {
		bridge.handleEvent(&monitor.ContractEvent{})
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(100 * time.Millisecond):
		t.Error("handleEvent should not block when channel is full")
	}
}

func TestCDCBridge_transformEvent(t *testing.T) {
	mon := monitor.NewContractMonitor()
	bridge := NewCDCBridge(mon, nil)

	// Add a config with event mapping
	config := &monitor.CDCSyncConfig{
		ID:             "test",
		Contract:       "0x1234567890abcdef1234567890abcdef12345678",
		ChainID:        1,
		TargetDatabase: "analytics",
		Enabled:        true,
		Events: []monitor.EventMapping{
			{
				EventName: "Transfer",
				TableName: "transfers",
				FieldMap: map[string]string{
					"from":   "sender",
					"to":     "receiver",
					"amount": "value",
				},
			},
		},
	}
	bridge.AddConfig(config)

	event := &monitor.ContractEvent{
		ContractAddress: "0x1234567890abcdef1234567890abcdef12345678",
		ChainID:         1,
		EventName:       "Transfer",
		TxHash:          "0xabc",
		BlockNumber:     12345678,
		LogIndex:        0,
		Timestamp:       time.Now(),
		DecodedArgs: map[string]interface{}{
			"from":   "0x111",
			"to":     "0x222",
			"amount": "1000000000000000000",
		},
	}

	cdcEvents := bridge.transformEvent(event)
	if len(cdcEvents) != 1 {
		t.Fatalf("Expected 1 CDC event, got %d", len(cdcEvents))
	}

	cdcEvent := cdcEvents[0]
	if cdcEvent.Operation != "INSERT" {
		t.Errorf("Operation = %s, want INSERT", cdcEvent.Operation)
	}
	if cdcEvent.Database != "analytics" {
		t.Errorf("Database = %s, want analytics", cdcEvent.Database)
	}
	if cdcEvent.Table != "transfers" {
		t.Errorf("Table = %s, want transfers", cdcEvent.Table)
	}
	if cdcEvent.Data["sender"] != "0x111" {
		t.Errorf("sender = %v, want 0x111", cdcEvent.Data["sender"])
	}
	if cdcEvent.Data["receiver"] != "0x222" {
		t.Errorf("receiver = %v, want 0x222", cdcEvent.Data["receiver"])
	}
}

func TestCDCBridge_transformEvent_NoMatch(t *testing.T) {
	mon := monitor.NewContractMonitor()
	bridge := NewCDCBridge(mon, nil)

	config := &monitor.CDCSyncConfig{
		ID:       "test",
		Contract: "0x111",
		ChainID:  1,
		Enabled:  true,
	}
	bridge.AddConfig(config)

	// Event from different contract
	event := &monitor.ContractEvent{
		ContractAddress: "0x999",
		ChainID:         1,
		EventName:       "Transfer",
	}

	cdcEvents := bridge.transformEvent(event)
	if len(cdcEvents) != 0 {
		t.Errorf("Expected 0 CDC events, got %d", len(cdcEvents))
	}
}

func TestCDCBridge_transformEvent_Disabled(t *testing.T) {
	mon := monitor.NewContractMonitor()
	bridge := NewCDCBridge(mon, nil)

	config := &monitor.CDCSyncConfig{
		ID:       "test",
		Contract: "0x111",
		ChainID:  1,
		Enabled:  false, // Disabled
		Events: []monitor.EventMapping{
			{EventName: "Transfer", TableName: "transfers"},
		},
	}
	bridge.AddConfig(config)

	event := &monitor.ContractEvent{
		ContractAddress: "0x111",
		ChainID:         1,
		EventName:       "Transfer",
	}

	cdcEvents := bridge.transformEvent(event)
	if len(cdcEvents) != 0 {
		t.Errorf("Expected 0 CDC events for disabled config, got %d", len(cdcEvents))
	}
}

func TestCDCBridge_flushBatch(t *testing.T) {
	mon := monitor.NewContractMonitor()
	client := NewMockCDCClient()
	bridge := NewCDCBridge(mon, client)

	config := &monitor.CDCSyncConfig{ID: "test"}
	bridge.AddConfig(config)

	batch := []*CDCEvent{
		{Operation: "INSERT", Table: "test1"},
		{Operation: "INSERT", Table: "test2"},
	}

	ctx := context.Background()
	bridge.flushBatch(ctx, batch)

	events := client.GetEvents()
	if len(events) != 2 {
		t.Errorf("Expected 2 events, got %d", len(events))
	}
}

func TestCDCBridge_flushBatch_NilClient(t *testing.T) {
	mon := monitor.NewContractMonitor()
	bridge := NewCDCBridge(mon, nil)

	batch := []*CDCEvent{
		{Operation: "INSERT", Table: "test1"},
	}

	// Should not panic with nil client
	ctx := context.Background()
	bridge.flushBatch(ctx, batch)
}

func TestCDCBridge_flushBatch_EmptyBatch(t *testing.T) {
	mon := monitor.NewContractMonitor()
	client := NewMockCDCClient()
	bridge := NewCDCBridge(mon, client)

	// Should not call client
	ctx := context.Background()
	bridge.flushBatch(ctx, []*CDCEvent{})

	if len(client.GetEvents()) != 0 {
		t.Error("Should not publish empty batch")
	}
}

// MockCDCClient tests

func TestMockCDCClient(t *testing.T) {
	client := NewMockCDCClient()
	if client == nil {
		t.Fatal("NewMockCDCClient returned nil")
	}

	ctx := context.Background()

	event := &CDCEvent{
		Operation: "INSERT",
		Database:  "test",
		Table:     "events",
		Data:      map[string]interface{}{"key": "value"},
	}

	err := client.PublishEvent(ctx, event)
	if err != nil {
		t.Fatalf("PublishEvent failed: %v", err)
	}

	events := client.GetEvents()
	if len(events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(events))
	}
	if events[0].Table != "events" {
		t.Errorf("Table = %s, want events", events[0].Table)
	}
}

func TestMockCDCClient_BatchPublish(t *testing.T) {
	client := NewMockCDCClient()
	ctx := context.Background()

	batch := []*CDCEvent{
		{Operation: "INSERT", Table: "t1"},
		{Operation: "UPDATE", Table: "t2"},
		{Operation: "DELETE", Table: "t3"},
	}

	err := client.BatchPublish(ctx, batch)
	if err != nil {
		t.Fatalf("BatchPublish failed: %v", err)
	}

	events := client.GetEvents()
	if len(events) != 3 {
		t.Errorf("Expected 3 events, got %d", len(events))
	}
}

func TestMockCDCClient_CreateTable(t *testing.T) {
	client := NewMockCDCClient()
	ctx := context.Background()

	schema := map[string]string{
		"id":        "BIGINT",
		"name":      "VARCHAR(255)",
		"timestamp": "TIMESTAMP",
	}

	err := client.CreateTable(ctx, "analytics", "events", schema)
	if err != nil {
		t.Fatalf("CreateTable failed: %v", err)
	}
}

func TestMockCDCClient_Concurrent(t *testing.T) {
	client := NewMockCDCClient()
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			event := &CDCEvent{
				Operation: "INSERT",
				Table:     "test",
			}
			client.PublishEvent(ctx, event)
		}(i)
	}

	wg.Wait()

	events := client.GetEvents()
	if len(events) != 100 {
		t.Errorf("Expected 100 events, got %d", len(events))
	}
}

// Type tests

func TestCDCEvent_JSON(t *testing.T) {
	event := CDCEvent{
		Operation: "INSERT",
		Database:  "analytics",
		Table:     "transfers",
		Data:      map[string]interface{}{"from": "0x111", "to": "0x222"},
		Timestamp: time.Now(),
		Source:    "chainlens",
		Metadata:  map[string]interface{}{"chain_id": 1},
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded CDCEvent
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.Operation != "INSERT" {
		t.Errorf("Operation = %s, want INSERT", decoded.Operation)
	}
	if decoded.Table != "transfers" {
		t.Errorf("Table = %s, want transfers", decoded.Table)
	}
}

func TestBridgeStats_Fields(t *testing.T) {
	stats := BridgeStats{
		TotalConfigs:         10,
		ActiveConfigs:        7,
		TotalEventsProcessed: 1000000,
		TotalErrors:          5,
	}

	if stats.TotalConfigs != 10 {
		t.Error("TotalConfigs mismatch")
	}
	if stats.ActiveConfigs != 7 {
		t.Error("ActiveConfigs mismatch")
	}
	if stats.TotalEventsProcessed != 1000000 {
		t.Error("TotalEventsProcessed mismatch")
	}
}

func TestBridgeStats_JSON(t *testing.T) {
	stats := BridgeStats{
		TotalConfigs:  5,
		ActiveConfigs: 3,
	}

	data, err := json.Marshal(stats)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded BridgeStats
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.TotalConfigs != 5 {
		t.Errorf("TotalConfigs = %d, want 5", decoded.TotalConfigs)
	}
}

// Benchmarks

func BenchmarkCDCBridge_AddConfig(b *testing.B) {
	mon := monitor.NewContractMonitor()
	bridge := NewCDCBridge(mon, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		config := &monitor.CDCSyncConfig{
			Contract: "0x111",
			ChainID:  1,
		}
		bridge.AddConfig(config)
	}
}

func BenchmarkCDCBridge_transformEvent(b *testing.B) {
	mon := monitor.NewContractMonitor()
	bridge := NewCDCBridge(mon, nil)

	config := &monitor.CDCSyncConfig{
		ID:       "test",
		Contract: "0x111",
		ChainID:  1,
		Enabled:  true,
		Events: []monitor.EventMapping{
			{
				EventName: "Transfer",
				TableName: "transfers",
				FieldMap:  map[string]string{"from": "sender", "to": "receiver"},
			},
		},
	}
	bridge.AddConfig(config)

	event := &monitor.ContractEvent{
		ContractAddress: "0x111",
		ChainID:         1,
		EventName:       "Transfer",
		DecodedArgs:     map[string]interface{}{"from": "0x1", "to": "0x2"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bridge.transformEvent(event)
	}
}

func BenchmarkMockCDCClient_PublishEvent(b *testing.B) {
	client := NewMockCDCClient()
	ctx := context.Background()
	event := &CDCEvent{
		Operation: "INSERT",
		Table:     "test",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.PublishEvent(ctx, event)
	}
}
