package monitor

import (
	"context"
	"math/big"
	"testing"
	"time"
)

type mockChainClient struct {
	chainID       int64
	balance       *big.Int
	blockNumber   uint64
	logs          []*ContractEvent
	balanceErr    error
	blockErr      error
	logsErr       error
}

func (c *mockChainClient) ChainID() int64 {
	return c.chainID
}

func (c *mockChainClient) GetBalance(ctx context.Context, address string) (*big.Int, error) {
	return c.balance, c.balanceErr
}

func (c *mockChainClient) GetBlockNumber(ctx context.Context) (uint64, error) {
	return c.blockNumber, c.blockErr
}

func (c *mockChainClient) GetLogs(ctx context.Context, filter LogFilter) ([]*ContractEvent, error) {
	return c.logs, c.logsErr
}

func (c *mockChainClient) GetTransaction(ctx context.Context, hash string) (*Transaction, error) {
	return nil, nil
}

func (c *mockChainClient) GetTransactionReceipt(ctx context.Context, hash string) (*TransactionReceipt, error) {
	return nil, nil
}

func (c *mockChainClient) TraceTransaction(ctx context.Context, hash string) (*TransactionTrace, error) {
	return nil, nil
}

func (c *mockChainClient) Call(ctx context.Context, msg CallMsg) ([]byte, error) {
	return nil, nil
}

func TestNewContractMonitor(t *testing.T) {
	m := NewContractMonitor()
	if m == nil {
		t.Fatal("expected monitor to be created")
	}
	if m.contracts == nil {
		t.Error("contracts map not initialized")
	}
	if m.events == nil {
		t.Error("events map not initialized")
	}
	if m.clients == nil {
		t.Error("clients map not initialized")
	}
}

func TestContractMonitor_RegisterClient(t *testing.T) {
	m := NewContractMonitor()
	client := &mockChainClient{chainID: 1}

	m.RegisterClient(1, client)

	m.mu.RLock()
	c, ok := m.clients[1]
	m.mu.RUnlock()

	if !ok {
		t.Error("client not registered")
	}
	if c.ChainID() != 1 {
		t.Errorf("expected chainID 1, got %d", c.ChainID())
	}
}

func TestContractMonitor_SetEventCallback(t *testing.T) {
	m := NewContractMonitor()
	called := false

	m.SetEventCallback(func(event *ContractEvent) {
		called = true
	})

	// Trigger callback
	m.handleEvent(&ContractEvent{
		ContractAddress: "0x123",
		ChainID:         1,
	})

	if !called {
		t.Error("callback not called")
	}
}

func TestContractMonitor_StartStop(t *testing.T) {
	m := NewContractMonitor()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := m.Start(ctx)
	if err != nil {
		t.Errorf("Start() error = %v", err)
	}

	if !m.running {
		t.Error("expected monitor to be running")
	}

	// Start again should be no-op
	err = m.Start(ctx)
	if err != nil {
		t.Errorf("Start() second call error = %v", err)
	}

	m.Stop()

	if m.running {
		t.Error("expected monitor to be stopped")
	}
}

func TestContractMonitor_AddContract(t *testing.T) {
	m := NewContractMonitor()

	contract := &Contract{
		Address: "0x123",
		ChainID: 1,
		Name:    "Test Contract",
	}

	err := m.AddContract(contract)
	if err != nil {
		t.Errorf("AddContract() error = %v", err)
	}

	if contract.ID == "" {
		t.Error("expected contract ID to be set")
	}
	if contract.Status != ContractStatusActive {
		t.Errorf("expected status %s, got %s", ContractStatusActive, contract.Status)
	}
	if contract.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}

	// Add duplicate
	err = m.AddContract(&Contract{Address: "0x123", ChainID: 1})
	if err == nil {
		t.Error("expected error for duplicate contract")
	}
}

func TestContractMonitor_RemoveContract(t *testing.T) {
	m := NewContractMonitor()

	m.AddContract(&Contract{Address: "0x123", ChainID: 1})

	m.RemoveContract("0x123", 1)

	_, ok := m.GetContract("0x123", 1)
	if ok {
		t.Error("expected contract to be removed")
	}
}

func TestContractMonitor_GetContract(t *testing.T) {
	m := NewContractMonitor()

	m.AddContract(&Contract{Address: "0x123", ChainID: 1, Name: "Test"})

	c, ok := m.GetContract("0x123", 1)
	if !ok {
		t.Error("expected contract to be found")
	}
	if c.Name != "Test" {
		t.Errorf("expected name 'Test', got %s", c.Name)
	}

	_, ok = m.GetContract("0x456", 1)
	if ok {
		t.Error("expected contract not to be found")
	}
}

func TestContractMonitor_ListContracts(t *testing.T) {
	m := NewContractMonitor()

	m.AddContract(&Contract{Address: "0x111", ChainID: 1})
	m.AddContract(&Contract{Address: "0x222", ChainID: 1})
	m.AddContract(&Contract{Address: "0x333", ChainID: 137})

	contracts := m.ListContracts()
	if len(contracts) != 3 {
		t.Errorf("expected 3 contracts, got %d", len(contracts))
	}
}

func TestContractMonitor_GetContractsByChain(t *testing.T) {
	m := NewContractMonitor()

	m.AddContract(&Contract{Address: "0x111", ChainID: 1})
	m.AddContract(&Contract{Address: "0x222", ChainID: 1})
	m.AddContract(&Contract{Address: "0x333", ChainID: 137})

	contracts := m.GetContractsByChain(1)
	if len(contracts) != 2 {
		t.Errorf("expected 2 contracts for chain 1, got %d", len(contracts))
	}

	contracts = m.GetContractsByChain(137)
	if len(contracts) != 1 {
		t.Errorf("expected 1 contract for chain 137, got %d", len(contracts))
	}
}

func TestContractMonitor_GetEvents(t *testing.T) {
	m := NewContractMonitor()

	m.AddContract(&Contract{Address: "0x123", ChainID: 1})

	// Add events manually
	m.mu.Lock()
	m.events["0x123:1"] = []*ContractEvent{
		{ID: "1", EventName: "Event1"},
		{ID: "2", EventName: "Event2"},
		{ID: "3", EventName: "Event3"},
	}
	m.mu.Unlock()

	events := m.GetEvents("0x123", 1, 0)
	if len(events) != 3 {
		t.Errorf("expected 3 events, got %d", len(events))
	}

	events = m.GetEvents("0x123", 1, 2)
	if len(events) != 2 {
		t.Errorf("expected 2 events with limit, got %d", len(events))
	}
}

func TestContractMonitor_GetRecentEvents(t *testing.T) {
	m := NewContractMonitor()

	m.AddContract(&Contract{Address: "0x111", ChainID: 1})
	m.AddContract(&Contract{Address: "0x222", ChainID: 1})

	m.mu.Lock()
	m.events["0x111:1"] = []*ContractEvent{{ID: "1"}, {ID: "2"}}
	m.events["0x222:1"] = []*ContractEvent{{ID: "3"}, {ID: "4"}}
	m.mu.Unlock()

	events := m.GetRecentEvents(0)
	if len(events) != 4 {
		t.Errorf("expected 4 total events, got %d", len(events))
	}

	events = m.GetRecentEvents(2)
	if len(events) != 2 {
		t.Errorf("expected 2 events with limit, got %d", len(events))
	}
}

func TestContractMonitor_PauseContract(t *testing.T) {
	m := NewContractMonitor()

	m.AddContract(&Contract{Address: "0x123", ChainID: 1})

	err := m.PauseContract("0x123", 1)
	if err != nil {
		t.Errorf("PauseContract() error = %v", err)
	}

	c, _ := m.GetContract("0x123", 1)
	if c.Status != ContractStatusPaused {
		t.Errorf("expected status %s, got %s", ContractStatusPaused, c.Status)
	}

	err = m.PauseContract("0x456", 1)
	if err == nil {
		t.Error("expected error for nonexistent contract")
	}
}

func TestContractMonitor_ResumeContract(t *testing.T) {
	m := NewContractMonitor()

	m.AddContract(&Contract{Address: "0x123", ChainID: 1})
	m.PauseContract("0x123", 1)

	err := m.ResumeContract("0x123", 1)
	if err != nil {
		t.Errorf("ResumeContract() error = %v", err)
	}

	c, _ := m.GetContract("0x123", 1)
	if c.Status != ContractStatusActive {
		t.Errorf("expected status %s, got %s", ContractStatusActive, c.Status)
	}

	err = m.ResumeContract("0x456", 1)
	if err == nil {
		t.Error("expected error for nonexistent contract")
	}
}

func TestContractMonitor_UpdateContract(t *testing.T) {
	m := NewContractMonitor()

	m.AddContract(&Contract{Address: "0x123", ChainID: 1})

	updates := map[string]interface{}{
		"name": "Updated Name",
		"abi":  "[{}]",
		"tags": []string{"defi", "token"},
	}

	err := m.UpdateContract("0x123", 1, updates)
	if err != nil {
		t.Errorf("UpdateContract() error = %v", err)
	}

	c, _ := m.GetContract("0x123", 1)
	if c.Name != "Updated Name" {
		t.Errorf("expected name 'Updated Name', got %s", c.Name)
	}
	if c.ABI != "[{}]" {
		t.Errorf("expected ABI '[{}]', got %s", c.ABI)
	}
	if len(c.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(c.Tags))
	}

	err = m.UpdateContract("0x456", 1, updates)
	if err == nil {
		t.Error("expected error for nonexistent contract")
	}
}

func TestContractMonitor_GetStats(t *testing.T) {
	m := NewContractMonitor()

	m.AddContract(&Contract{Address: "0x111", ChainID: 1})
	m.AddContract(&Contract{Address: "0x222", ChainID: 1})
	m.AddContract(&Contract{Address: "0x333", ChainID: 137})
	m.PauseContract("0x333", 137)

	m.mu.Lock()
	m.events["0x111:1"] = []*ContractEvent{{ID: "1"}, {ID: "2"}}
	m.events["0x222:1"] = []*ContractEvent{{ID: "3"}}
	m.mu.Unlock()

	stats := m.GetStats()

	if stats.TotalContracts != 3 {
		t.Errorf("expected 3 contracts, got %d", stats.TotalContracts)
	}
	if stats.TotalEvents != 3 {
		t.Errorf("expected 3 events, got %d", stats.TotalEvents)
	}
	if stats.ByChain[1] != 2 {
		t.Errorf("expected 2 contracts on chain 1, got %d", stats.ByChain[1])
	}
	if stats.ByChain[137] != 1 {
		t.Errorf("expected 1 contract on chain 137, got %d", stats.ByChain[137])
	}
	if stats.ByStatus["active"] != 2 {
		t.Errorf("expected 2 active contracts, got %d", stats.ByStatus["active"])
	}
	if stats.ByStatus["paused"] != 1 {
		t.Errorf("expected 1 paused contract, got %d", stats.ByStatus["paused"])
	}
}

func TestContractMonitor_HandleEvent(t *testing.T) {
	m := NewContractMonitor()

	m.AddContract(&Contract{Address: "0x123", ChainID: 1})

	event := &ContractEvent{
		ContractAddress: "0x123",
		ChainID:         1,
		EventName:       "Transfer",
	}

	m.handleEvent(event)

	events := m.GetEvents("0x123", 1, 0)
	if len(events) != 1 {
		t.Errorf("expected 1 event, got %d", len(events))
	}

	c, _ := m.GetContract("0x123", 1)
	if c.LastEventAt == nil {
		t.Error("expected LastEventAt to be set")
	}
}

func TestContractMonitor_PollContract(t *testing.T) {
	m := NewContractMonitor()

	client := &mockChainClient{
		chainID:     1,
		blockNumber: 1000,
		balance:     big.NewInt(1e18),
		logs: []*ContractEvent{
			{ID: "event-1", EventName: "Transfer"},
		},
	}
	m.RegisterClient(1, client)

	contract := &Contract{Address: "0x123", ChainID: 1}
	m.AddContract(contract)

	ctx := context.Background()

	// Start event processing
	go func() {
		for event := range m.eventCh {
			m.handleEvent(event)
		}
	}()

	m.pollContract(ctx, contract)
	time.Sleep(50 * time.Millisecond)

	// Check balance was updated
	c, _ := m.GetContract("0x123", 1)
	if c.Balance == nil || c.Balance.Cmp(big.NewInt(1e18)) != 0 {
		t.Error("expected balance to be updated")
	}
}

func TestContractMonitor_PollContract_NoClient(t *testing.T) {
	m := NewContractMonitor()

	contract := &Contract{Address: "0x123", ChainID: 999} // No client for chain 999
	m.AddContract(contract)

	ctx := context.Background()

	// Should not panic
	m.pollContract(ctx, contract)
}

func TestContractMonitor_PollAllContracts(t *testing.T) {
	m := NewContractMonitor()

	client := &mockChainClient{
		chainID:     1,
		blockNumber: 1000,
		balance:     big.NewInt(1e18),
		logs:        []*ContractEvent{},
	}
	m.RegisterClient(1, client)

	m.AddContract(&Contract{Address: "0x111", ChainID: 1, Status: ContractStatusActive})
	m.AddContract(&Contract{Address: "0x222", ChainID: 1, Status: ContractStatusPaused}) // Should be skipped

	ctx := context.Background()
	m.pollAllContracts(ctx)
	// Should complete without error
}

func TestContractMonitor_ProcessEvents_ContextCancellation(t *testing.T) {
	m := NewContractMonitor()

	ctx, cancel := context.WithCancel(context.Background())

	go m.processEvents(ctx)

	cancel()
	time.Sleep(50 * time.Millisecond)
	// Should have exited
}

func TestContractMonitor_PollLoop_ContextCancellation(t *testing.T) {
	m := NewContractMonitor()

	ctx, cancel := context.WithCancel(context.Background())

	go m.pollLoop(ctx)

	cancel()
	time.Sleep(50 * time.Millisecond)
	// Should have exited
}
