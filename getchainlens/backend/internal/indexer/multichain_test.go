package indexer

import (
	"context"
	"sync"
	"testing"
	"time"

	"getchainlens.com/chainlens/backend/internal/monitor"
)

func TestNewMultiChainIndexer(t *testing.T) {
	mon := monitor.NewContractMonitor()
	alertMgr := monitor.NewAlertManager()

	indexer := NewMultiChainIndexer(mon, alertMgr)
	if indexer == nil {
		t.Fatal("NewMultiChainIndexer returned nil")
	}
	if indexer.networks == nil {
		t.Error("networks map should not be nil")
	}
}

func TestMultiChainIndexer_AddNetwork(t *testing.T) {
	mon := monitor.NewContractMonitor()
	alertMgr := monitor.NewAlertManager()
	indexer := NewMultiChainIndexer(mon, alertMgr)

	config := NetworkConfig{
		ChainID:       1,
		Name:          "Ethereum",
		RPCURL:        "http://localhost:8545",
		BlockTime:     12 * time.Second,
		Confirmations: 12,
	}

	client := NewMockChainClient(1)
	err := indexer.AddNetwork(config, client)
	if err != nil {
		t.Fatalf("AddNetwork failed: %v", err)
	}

	// Try adding same network again
	err = indexer.AddNetwork(config, client)
	if err == nil {
		t.Error("Expected error when adding duplicate network")
	}
}

func TestMultiChainIndexer_RemoveNetwork(t *testing.T) {
	mon := monitor.NewContractMonitor()
	alertMgr := monitor.NewAlertManager()
	indexer := NewMultiChainIndexer(mon, alertMgr)

	config := NetworkConfig{
		ChainID: 1,
		Name:    "Ethereum",
	}
	client := NewMockChainClient(1)
	indexer.AddNetwork(config, client)

	indexer.RemoveNetwork(1)

	// Verify network is removed
	_, ok := indexer.GetNetworkStatus(1)
	if ok {
		t.Error("Network should be removed")
	}
}

func TestMultiChainIndexer_RemoveNetwork_NotExists(t *testing.T) {
	mon := monitor.NewContractMonitor()
	alertMgr := monitor.NewAlertManager()
	indexer := NewMultiChainIndexer(mon, alertMgr)

	// Should not panic
	indexer.RemoveNetwork(999)
}

func TestMultiChainIndexer_StartStop(t *testing.T) {
	mon := monitor.NewContractMonitor()
	alertMgr := monitor.NewAlertManager()
	indexer := NewMultiChainIndexer(mon, alertMgr)

	config := NetworkConfig{
		ChainID:   1,
		Name:      "Ethereum",
		BlockTime: 100 * time.Millisecond,
	}
	client := NewMockChainClient(1)
	indexer.AddNetwork(config, client)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err := indexer.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Should be idempotent
	err = indexer.Start(ctx)
	if err != nil {
		t.Fatalf("Second Start should not fail: %v", err)
	}

	// Let it run briefly
	time.Sleep(100 * time.Millisecond)

	indexer.Stop()

	// Stop should be idempotent
	indexer.Stop()
}

func TestMultiChainIndexer_GetNetworkStatus(t *testing.T) {
	mon := monitor.NewContractMonitor()
	alertMgr := monitor.NewAlertManager()
	indexer := NewMultiChainIndexer(mon, alertMgr)

	config := NetworkConfig{
		ChainID: 1,
		Name:    "Ethereum",
	}
	client := NewMockChainClient(1)
	indexer.AddNetwork(config, client)

	status, ok := indexer.GetNetworkStatus(1)
	if !ok {
		t.Fatal("Expected to find network status")
	}
	if status.ChainID != 1 {
		t.Errorf("ChainID = %d, want 1", status.ChainID)
	}
	if status.Name != "Ethereum" {
		t.Errorf("Name = %s, want Ethereum", status.Name)
	}

	// Non-existent network
	_, ok = indexer.GetNetworkStatus(999)
	if ok {
		t.Error("Should not find non-existent network")
	}
}

func TestMultiChainIndexer_GetAllNetworkStatus(t *testing.T) {
	mon := monitor.NewContractMonitor()
	alertMgr := monitor.NewAlertManager()
	indexer := NewMultiChainIndexer(mon, alertMgr)

	indexer.AddNetwork(NetworkConfig{ChainID: 1, Name: "Ethereum"}, NewMockChainClient(1))
	indexer.AddNetwork(NetworkConfig{ChainID: 137, Name: "Polygon"}, NewMockChainClient(137))

	statuses := indexer.GetAllNetworkStatus()
	if len(statuses) != 2 {
		t.Errorf("Expected 2 statuses, got %d", len(statuses))
	}
}

func TestMultiChainIndexer_GetStats(t *testing.T) {
	mon := monitor.NewContractMonitor()
	alertMgr := monitor.NewAlertManager()
	indexer := NewMultiChainIndexer(mon, alertMgr)

	indexer.AddNetwork(NetworkConfig{ChainID: 1, Name: "Ethereum"}, NewMockChainClient(1))
	indexer.AddNetwork(NetworkConfig{ChainID: 137, Name: "Polygon"}, NewMockChainClient(137))

	stats := indexer.GetStats()
	if stats.NetworkCount != 2 {
		t.Errorf("NetworkCount = %d, want 2", stats.NetworkCount)
	}
	if len(stats.ByNetwork) != 2 {
		t.Errorf("ByNetwork len = %d, want 2", len(stats.ByNetwork))
	}
}

// NetworkIndexer tests

func TestNetworkIndexer_Start(t *testing.T) {
	config := NetworkConfig{
		ChainID:       1,
		Name:          "Ethereum",
		BlockTime:     50 * time.Millisecond,
		Confirmations: 1,
	}
	client := NewMockChainClient(1)

	indexer := &NetworkIndexer{
		config:  config,
		client:  client,
		stopCh:  make(chan struct{}),
		blockCh: make(chan uint64, 100),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	indexer.Start(ctx)
	time.Sleep(100 * time.Millisecond)
	indexer.Stop()

	// Should be idempotent
	indexer.Start(ctx) // Should return immediately since already started then stopped
}

func TestNetworkIndexer_Stop(t *testing.T) {
	config := NetworkConfig{
		ChainID:       1,
		BlockTime:     50 * time.Millisecond,
		Confirmations: 1,
	}

	client := NewMockChainClient(1)

	indexer := &NetworkIndexer{
		config:  config,
		client:  client,
		stopCh:  make(chan struct{}),
		blockCh: make(chan uint64, 100),
	}

	// Stop without starting
	indexer.Stop()

	// Start and stop
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	indexer.stopCh = make(chan struct{}) // Reset channel
	indexer.Start(ctx)
	time.Sleep(50 * time.Millisecond)
	indexer.Stop()
	indexer.Stop() // Double stop should be safe
}

func TestNetworkIndexer_GetStatus(t *testing.T) {
	config := NetworkConfig{
		ChainID: 1,
		Name:    "Ethereum",
	}

	indexer := &NetworkIndexer{
		config:    config,
		lastBlock: 12345678,
		running:   true,
	}

	status := indexer.GetStatus()
	if status.ChainID != 1 {
		t.Errorf("ChainID = %d, want 1", status.ChainID)
	}
	if status.Name != "Ethereum" {
		t.Errorf("Name = %s, want Ethereum", status.Name)
	}
	if status.LastBlock != 12345678 {
		t.Errorf("LastBlock = %d, want 12345678", status.LastBlock)
	}
	if !status.Running {
		t.Error("Expected Running = true")
	}
	if status.SyncStatus != "syncing" {
		t.Errorf("SyncStatus = %s, want syncing", status.SyncStatus)
	}

	// When not running
	indexer.running = false
	status = indexer.GetStatus()
	if status.SyncStatus != "stopped" {
		t.Errorf("SyncStatus = %s, want stopped", status.SyncStatus)
	}
}

func TestNetworkIndexer_GetStats(t *testing.T) {
	indexer := &NetworkIndexer{
		config: NetworkConfig{
			ChainID: 1,
		},
		lastBlock: 1000000,
	}

	stats := indexer.GetStats()
	if stats.ChainID != 1 {
		t.Errorf("ChainID = %d, want 1", stats.ChainID)
	}
	if stats.BlocksIndexed != 1000000 {
		t.Errorf("BlocksIndexed = %d, want 1000000", stats.BlocksIndexed)
	}
}

func TestNetworkIndexer_SetBlockCallback(t *testing.T) {
	indexer := &NetworkIndexer{}

	var called bool
	var mu sync.Mutex
	indexer.SetBlockCallback(func(chainID int64, blockNumber uint64) {
		mu.Lock()
		called = true
		mu.Unlock()
	})

	if indexer.onNewBlock == nil {
		t.Error("onNewBlock should be set")
	}

	indexer.onNewBlock(1, 100)

	mu.Lock()
	if !called {
		t.Error("Callback should have been called")
	}
	mu.Unlock()
}

func TestNetworkIndexer_SetEventCallback(t *testing.T) {
	indexer := &NetworkIndexer{}

	var called bool
	var mu sync.Mutex
	indexer.SetEventCallback(func(event *monitor.ContractEvent) {
		mu.Lock()
		called = true
		mu.Unlock()
	})

	if indexer.onNewEvent == nil {
		t.Error("onNewEvent should be set")
	}

	indexer.onNewEvent(&monitor.ContractEvent{})

	mu.Lock()
	if !called {
		t.Error("Callback should have been called")
	}
	mu.Unlock()
}

func TestNetworkIndexer_indexBlock(t *testing.T) {
	client := NewMockChainClient(1)

	indexer := &NetworkIndexer{
		config: NetworkConfig{
			ChainID: 1,
		},
		client: client,
		onNewEvent: func(event *monitor.ContractEvent) {
			// Event callback
		},
	}

	ctx := context.Background()
	indexer.indexBlock(ctx, 12345678)

	if indexer.lastBlock != 12345678 {
		t.Errorf("lastBlock = %d, want 12345678", indexer.lastBlock)
	}
}

func TestNetworkIndexer_pollNewBlocks(t *testing.T) {
	client := NewMockChainClient(1)

	indexer := &NetworkIndexer{
		config: NetworkConfig{
			ChainID:       1,
			Confirmations: 1,
		},
		client:    client,
		lastBlock: 1000000,
		blockCh:   make(chan uint64, 100),
	}

	ctx := context.Background()
	indexer.pollNewBlocks(ctx)

	// Should have queued some blocks
	select {
	case block := <-indexer.blockCh:
		if block <= 1000000 {
			t.Errorf("Expected block > 1000000, got %d", block)
		}
	case <-time.After(100 * time.Millisecond):
		// May not have blocks if confirmations not met
	}
}

// MockChainClient tests

func TestMockChainClient(t *testing.T) {
	client := NewMockChainClient(1)

	if client.ChainID() != 1 {
		t.Errorf("ChainID() = %d, want 1", client.ChainID())
	}

	ctx := context.Background()

	balance, err := client.GetBalance(ctx, "0x123")
	if err != nil {
		t.Errorf("GetBalance failed: %v", err)
	}
	if balance == nil {
		t.Error("Balance should not be nil")
	}

	blockNum, err := client.GetBlockNumber(ctx)
	if err != nil {
		t.Errorf("GetBlockNumber failed: %v", err)
	}
	if blockNum == 0 {
		t.Error("BlockNumber should not be 0")
	}

	// Block number should increment
	blockNum2, _ := client.GetBlockNumber(ctx)
	if blockNum2 <= blockNum {
		t.Error("BlockNumber should increment")
	}

	logs, err := client.GetLogs(ctx, monitor.LogFilter{})
	if err != nil {
		t.Errorf("GetLogs failed: %v", err)
	}
	if logs == nil {
		t.Error("Logs should not be nil (even if empty)")
	}

	tx, err := client.GetTransaction(ctx, "0x123")
	if err != nil {
		t.Errorf("GetTransaction failed: %v", err)
	}
	if tx.Hash != "0x123" {
		t.Errorf("Transaction hash = %s, want 0x123", tx.Hash)
	}

	receipt, err := client.GetTransactionReceipt(ctx, "0x456")
	if err != nil {
		t.Errorf("GetTransactionReceipt failed: %v", err)
	}
	if receipt.TxHash != "0x456" {
		t.Errorf("Receipt TxHash = %s, want 0x456", receipt.TxHash)
	}
	if !receipt.Status {
		t.Error("Receipt Status should be true")
	}

	trace, err := client.TraceTransaction(ctx, "0x789")
	if err != nil {
		t.Errorf("TraceTransaction failed: %v", err)
	}
	if trace.TxHash != "0x789" {
		t.Errorf("Trace TxHash = %s, want 0x789", trace.TxHash)
	}

	callResult, err := client.Call(ctx, monitor.CallMsg{})
	if err != nil {
		t.Errorf("Call failed: %v", err)
	}
	if callResult == nil {
		t.Error("Call result should not be nil")
	}
}

// Type tests

func TestNetworkConfig_Fields(t *testing.T) {
	config := NetworkConfig{
		ChainID:       1,
		Name:          "Ethereum",
		RPCURL:        "http://localhost:8545",
		WSURL:         "ws://localhost:8546",
		BlockTime:     12 * time.Second,
		MaxBatchSize:  100,
		StartBlock:    15000000,
		Confirmations: 12,
	}

	if config.ChainID != 1 {
		t.Error("ChainID mismatch")
	}
	if config.Name != "Ethereum" {
		t.Error("Name mismatch")
	}
	if config.BlockTime != 12*time.Second {
		t.Error("BlockTime mismatch")
	}
}

func TestNetworkStatus_Fields(t *testing.T) {
	now := time.Now()
	status := NetworkStatus{
		ChainID:       1,
		Name:          "Ethereum",
		Running:       true,
		LastBlock:     15000000,
		LastBlockTime: now,
		SyncStatus:    "syncing",
		ErrorMessage:  "",
	}

	if status.ChainID != 1 {
		t.Error("ChainID mismatch")
	}
	if !status.Running {
		t.Error("Running mismatch")
	}
}

func TestNetworkStats_Fields(t *testing.T) {
	now := time.Now()
	stats := NetworkStats{
		ChainID:       1,
		BlocksIndexed: 15000000,
		EventsFound:   1000000,
		TxProcessed:   5000000,
		StartedAt:     now,
	}

	if stats.BlocksIndexed != 15000000 {
		t.Error("BlocksIndexed mismatch")
	}
}

func TestIndexerStats_Fields(t *testing.T) {
	stats := IndexerStats{
		NetworkCount: 3,
		TotalBlocks:  50000000,
		ByNetwork: map[int64]*NetworkStats{
			1:   {ChainID: 1, BlocksIndexed: 20000000},
			137: {ChainID: 137, BlocksIndexed: 30000000},
		},
	}

	if stats.NetworkCount != 3 {
		t.Error("NetworkCount mismatch")
	}
	if len(stats.ByNetwork) != 2 {
		t.Error("ByNetwork len mismatch")
	}
}

// Benchmarks

func BenchmarkNetworkIndexer_GetStatus(b *testing.B) {
	indexer := &NetworkIndexer{
		config: NetworkConfig{
			ChainID: 1,
			Name:    "Ethereum",
		},
		lastBlock: 15000000,
		running:   true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		indexer.GetStatus()
	}
}

func BenchmarkMockChainClient_GetBlockNumber(b *testing.B) {
	client := NewMockChainClient(1)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.GetBlockNumber(ctx)
	}
}
