package indexer

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	"getchainlens.com/chainlens/backend/internal/monitor"
)

// MultiChainIndexer indexes multiple blockchain networks
type MultiChainIndexer struct {
	networks    map[int64]*NetworkIndexer
	monitor     *monitor.ContractMonitor
	alertMgr    *monitor.AlertManager
	mu          sync.RWMutex
	running     bool
	stopCh      chan struct{}
}

// NetworkIndexer handles indexing for a single network
type NetworkIndexer struct {
	config       NetworkConfig
	client       monitor.ChainClient
	lastBlock    uint64
	mu           sync.RWMutex
	running      bool
	stopCh       chan struct{}
	blockCh      chan uint64
	onNewBlock   func(int64, uint64)
	onNewEvent   func(*monitor.ContractEvent)
}

// NetworkConfig contains configuration for a network
type NetworkConfig struct {
	ChainID       int64
	Name          string
	RPCURL        string
	WSURL         string
	BlockTime     time.Duration
	MaxBatchSize  int
	StartBlock    uint64
	Confirmations int
}

// NewMultiChainIndexer creates a new multi-chain indexer
func NewMultiChainIndexer(mon *monitor.ContractMonitor, alertMgr *monitor.AlertManager) *MultiChainIndexer {
	return &MultiChainIndexer{
		networks: make(map[int64]*NetworkIndexer),
		monitor:  mon,
		alertMgr: alertMgr,
		stopCh:   make(chan struct{}),
	}
}

// AddNetwork adds a network to index
func (m *MultiChainIndexer) AddNetwork(config NetworkConfig, client monitor.ChainClient) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.networks[config.ChainID]; exists {
		return fmt.Errorf("network already registered: %d", config.ChainID)
	}

	indexer := &NetworkIndexer{
		config:  config,
		client:  client,
		stopCh:  make(chan struct{}),
		blockCh: make(chan uint64, 100),
	}

	// Set callbacks
	indexer.onNewEvent = func(event *monitor.ContractEvent) {
		m.alertMgr.EvaluateEvent(event)
	}

	m.networks[config.ChainID] = indexer

	// Register client with monitor
	m.monitor.RegisterClient(config.ChainID, client)

	return nil
}

// RemoveNetwork removes a network
func (m *MultiChainIndexer) RemoveNetwork(chainID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if indexer, ok := m.networks[chainID]; ok {
		indexer.Stop()
		delete(m.networks, chainID)
	}
}

// Start starts all network indexers
func (m *MultiChainIndexer) Start(ctx context.Context) error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return nil
	}
	m.running = true
	m.mu.Unlock()

	for _, indexer := range m.networks {
		go indexer.Start(ctx)
	}

	return nil
}

// Stop stops all indexers
func (m *MultiChainIndexer) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return
	}

	close(m.stopCh)

	for _, indexer := range m.networks {
		indexer.Stop()
	}

	m.running = false
}

// GetNetworkStatus returns status for a network
func (m *MultiChainIndexer) GetNetworkStatus(chainID int64) (*NetworkStatus, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	indexer, ok := m.networks[chainID]
	if !ok {
		return nil, false
	}

	return indexer.GetStatus(), true
}

// GetAllNetworkStatus returns status for all networks
func (m *MultiChainIndexer) GetAllNetworkStatus() []*NetworkStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	statuses := make([]*NetworkStatus, 0, len(m.networks))
	for _, indexer := range m.networks {
		statuses = append(statuses, indexer.GetStatus())
	}
	return statuses
}

// GetStats returns indexer statistics
func (m *MultiChainIndexer) GetStats() *IndexerStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := &IndexerStats{
		NetworkCount: len(m.networks),
		ByNetwork:    make(map[int64]*NetworkStats),
	}

	for chainID, indexer := range m.networks {
		stats.ByNetwork[chainID] = indexer.GetStats()
		stats.TotalBlocks += indexer.GetStats().BlocksIndexed
	}

	return stats
}

// IndexerStats contains indexer statistics
type IndexerStats struct {
	NetworkCount int                    `json:"network_count"`
	TotalBlocks  uint64                 `json:"total_blocks"`
	ByNetwork    map[int64]*NetworkStats `json:"by_network"`
}

// NetworkStatus represents the status of a network indexer
type NetworkStatus struct {
	ChainID       int64     `json:"chain_id"`
	Name          string    `json:"name"`
	Running       bool      `json:"running"`
	LastBlock     uint64    `json:"last_block"`
	LastBlockTime time.Time `json:"last_block_time"`
	SyncStatus    string    `json:"sync_status"` // syncing, synced, error
	ErrorMessage  string    `json:"error_message,omitempty"`
}

// NetworkStats contains statistics for a network
type NetworkStats struct {
	ChainID       int64     `json:"chain_id"`
	BlocksIndexed uint64    `json:"blocks_indexed"`
	EventsFound   uint64    `json:"events_found"`
	TxProcessed   uint64    `json:"tx_processed"`
	StartedAt     time.Time `json:"started_at"`
}

// Start starts the network indexer
func (n *NetworkIndexer) Start(ctx context.Context) {
	n.mu.Lock()
	if n.running {
		n.mu.Unlock()
		return
	}
	n.running = true
	n.mu.Unlock()

	go n.blockPollLoop(ctx)
	go n.processBlocks(ctx)
}

// Stop stops the network indexer
func (n *NetworkIndexer) Stop() {
	n.mu.Lock()
	defer n.mu.Unlock()

	if !n.running {
		return
	}

	close(n.stopCh)
	n.running = false
}

func (n *NetworkIndexer) blockPollLoop(ctx context.Context) {
	ticker := time.NewTicker(n.config.BlockTime)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-n.stopCh:
			return
		case <-ticker.C:
			n.pollNewBlocks(ctx)
		}
	}
}

func (n *NetworkIndexer) pollNewBlocks(ctx context.Context) {
	blockNumber, err := n.client.GetBlockNumber(ctx)
	if err != nil {
		return
	}

	n.mu.Lock()
	lastBlock := n.lastBlock
	n.mu.Unlock()

	// Account for confirmations
	confirmedBlock := blockNumber - uint64(n.config.Confirmations)
	if confirmedBlock <= lastBlock {
		return
	}

	// Queue blocks to process
	for block := lastBlock + 1; block <= confirmedBlock; block++ {
		select {
		case n.blockCh <- block:
		default:
			// Channel full, skip
			return
		}
	}
}

func (n *NetworkIndexer) processBlocks(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-n.stopCh:
			return
		case blockNumber := <-n.blockCh:
			n.indexBlock(ctx, blockNumber)
		}
	}
}

func (n *NetworkIndexer) indexBlock(ctx context.Context, blockNumber uint64) {
	// Get logs for this block
	filter := monitor.LogFilter{
		FromBlock: blockNumber,
		ToBlock:   blockNumber,
	}

	events, err := n.client.GetLogs(ctx, filter)
	if err != nil {
		return
	}

	for _, event := range events {
		event.ChainID = n.config.ChainID
		if n.onNewEvent != nil {
			n.onNewEvent(event)
		}
	}

	// Update last block
	n.mu.Lock()
	if blockNumber > n.lastBlock {
		n.lastBlock = blockNumber
	}
	n.mu.Unlock()

	if n.onNewBlock != nil {
		n.onNewBlock(n.config.ChainID, blockNumber)
	}
}

// GetStatus returns the indexer status
func (n *NetworkIndexer) GetStatus() *NetworkStatus {
	n.mu.RLock()
	defer n.mu.RUnlock()

	status := &NetworkStatus{
		ChainID:   n.config.ChainID,
		Name:      n.config.Name,
		Running:   n.running,
		LastBlock: n.lastBlock,
	}

	if n.running {
		status.SyncStatus = "syncing"
	} else {
		status.SyncStatus = "stopped"
	}

	return status
}

// GetStats returns statistics
func (n *NetworkIndexer) GetStats() *NetworkStats {
	n.mu.RLock()
	defer n.mu.RUnlock()

	return &NetworkStats{
		ChainID:       n.config.ChainID,
		BlocksIndexed: n.lastBlock,
	}
}

// SetBlockCallback sets callback for new blocks
func (n *NetworkIndexer) SetBlockCallback(fn func(int64, uint64)) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.onNewBlock = fn
}

// SetEventCallback sets callback for new events
func (n *NetworkIndexer) SetEventCallback(fn func(*monitor.ContractEvent)) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.onNewEvent = fn
}

// MockChainClient is a mock implementation for testing
type MockChainClient struct {
	chainID     int64
	blockNumber uint64
	mu          sync.Mutex
}

// NewMockChainClient creates a mock chain client
func NewMockChainClient(chainID int64) *MockChainClient {
	return &MockChainClient{
		chainID:     chainID,
		blockNumber: 1000000,
	}
}

func (c *MockChainClient) ChainID() int64 {
	return c.chainID
}

func (c *MockChainClient) GetBalance(ctx context.Context, address string) (*big.Int, error) {
	return big.NewInt(1000000000000000000), nil // 1 ETH
}

func (c *MockChainClient) GetBlockNumber(ctx context.Context) (uint64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.blockNumber++
	return c.blockNumber, nil
}

func (c *MockChainClient) GetLogs(ctx context.Context, filter monitor.LogFilter) ([]*monitor.ContractEvent, error) {
	return []*monitor.ContractEvent{}, nil
}

func (c *MockChainClient) GetTransaction(ctx context.Context, hash string) (*monitor.Transaction, error) {
	return &monitor.Transaction{Hash: hash}, nil
}

func (c *MockChainClient) GetTransactionReceipt(ctx context.Context, hash string) (*monitor.TransactionReceipt, error) {
	return &monitor.TransactionReceipt{TxHash: hash, Status: true}, nil
}

func (c *MockChainClient) TraceTransaction(ctx context.Context, hash string) (*monitor.TransactionTrace, error) {
	return &monitor.TransactionTrace{TxHash: hash, Status: "success"}, nil
}

func (c *MockChainClient) Call(ctx context.Context, msg monitor.CallMsg) ([]byte, error) {
	return []byte{}, nil
}
