package monitor

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"
)

// ContractMonitor monitors smart contracts
type ContractMonitor struct {
	contracts   map[string]*Contract // address:chainID -> contract
	events      map[string][]*ContractEvent
	clients     map[int64]ChainClient
	mu          sync.RWMutex
	running     bool
	stopCh      chan struct{}
	eventCh     chan *ContractEvent
	onEvent     func(*ContractEvent)
}

// ChainClient interface for blockchain RPC calls
type ChainClient interface {
	ChainID() int64
	GetBalance(ctx context.Context, address string) (*big.Int, error)
	GetBlockNumber(ctx context.Context) (uint64, error)
	GetLogs(ctx context.Context, filter LogFilter) ([]*ContractEvent, error)
	GetTransaction(ctx context.Context, hash string) (*Transaction, error)
	GetTransactionReceipt(ctx context.Context, hash string) (*TransactionReceipt, error)
	TraceTransaction(ctx context.Context, hash string) (*TransactionTrace, error)
	Call(ctx context.Context, msg CallMsg) ([]byte, error)
}

// LogFilter defines filter parameters for logs
type LogFilter struct {
	FromBlock uint64
	ToBlock   uint64
	Addresses []string
	Topics    [][]string
}

// TransactionReceipt represents a transaction receipt
type TransactionReceipt struct {
	TxHash      string
	BlockNumber uint64
	GasUsed     uint64
	Status      bool
	Logs        []*ContractEvent
}

// CallMsg defines parameters for eth_call
type CallMsg struct {
	To    string
	Data  []byte
	Value *big.Int
}

// NewContractMonitor creates a new contract monitor
func NewContractMonitor() *ContractMonitor {
	return &ContractMonitor{
		contracts: make(map[string]*Contract),
		events:    make(map[string][]*ContractEvent),
		clients:   make(map[int64]ChainClient),
		stopCh:    make(chan struct{}),
		eventCh:   make(chan *ContractEvent, 1000),
	}
}

// RegisterClient registers a blockchain client for a chain
func (m *ContractMonitor) RegisterClient(chainID int64, client ChainClient) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clients[chainID] = client
}

// SetEventCallback sets callback for new events
func (m *ContractMonitor) SetEventCallback(fn func(*ContractEvent)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onEvent = fn
}

// Start starts monitoring
func (m *ContractMonitor) Start(ctx context.Context) error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return nil
	}
	m.running = true
	m.mu.Unlock()

	go m.pollLoop(ctx)
	go m.processEvents(ctx)

	return nil
}

// Stop stops monitoring
func (m *ContractMonitor) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.running {
		close(m.stopCh)
		m.running = false
	}
}

func (m *ContractMonitor) pollLoop(ctx context.Context) {
	ticker := time.NewTicker(12 * time.Second) // Block time
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.pollAllContracts(ctx)
		}
	}
}

func (m *ContractMonitor) pollAllContracts(ctx context.Context) {
	m.mu.RLock()
	contracts := make([]*Contract, 0, len(m.contracts))
	for _, c := range m.contracts {
		if c.Status == ContractStatusActive {
			contracts = append(contracts, c)
		}
	}
	m.mu.RUnlock()

	for _, contract := range contracts {
		m.pollContract(ctx, contract)
	}
}

func (m *ContractMonitor) pollContract(ctx context.Context, contract *Contract) {
	m.mu.RLock()
	client, ok := m.clients[contract.ChainID]
	m.mu.RUnlock()

	if !ok {
		return
	}

	// Get latest block
	blockNumber, err := client.GetBlockNumber(ctx)
	if err != nil {
		return
	}

	// Calculate from block (last 100 blocks or since last event)
	fromBlock := blockNumber - 100
	if contract.LastEventAt != nil {
		// Would need to track last processed block
	}

	// Get logs
	filter := LogFilter{
		FromBlock: fromBlock,
		ToBlock:   blockNumber,
		Addresses: []string{contract.Address},
	}

	events, err := client.GetLogs(ctx, filter)
	if err != nil {
		return
	}

	for _, event := range events {
		event.ContractAddress = contract.Address
		event.ChainID = contract.ChainID
		m.eventCh <- event
	}

	// Update balance
	balance, err := client.GetBalance(ctx, contract.Address)
	if err == nil {
		m.mu.Lock()
		contract.Balance = balance
		contract.UpdatedAt = time.Now()
		m.mu.Unlock()
	}
}

func (m *ContractMonitor) processEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		case event := <-m.eventCh:
			m.handleEvent(event)
		}
	}
}

func (m *ContractMonitor) handleEvent(event *ContractEvent) {
	key := fmt.Sprintf("%s:%d", event.ContractAddress, event.ChainID)

	m.mu.Lock()
	m.events[key] = append(m.events[key], event)

	// Update contract last event time
	if contract, ok := m.contracts[key]; ok {
		now := time.Now()
		contract.LastEventAt = &now
	}

	cb := m.onEvent
	m.mu.Unlock()

	if cb != nil {
		cb(event)
	}
}

// AddContract adds a contract to monitoring
func (m *ContractMonitor) AddContract(contract *Contract) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%d", contract.Address, contract.ChainID)

	if _, exists := m.contracts[key]; exists {
		return fmt.Errorf("contract already monitored: %s", key)
	}

	contract.CreatedAt = time.Now()
	contract.UpdatedAt = time.Now()
	contract.Status = ContractStatusActive
	contract.ID = key

	m.contracts[key] = contract
	m.events[key] = make([]*ContractEvent, 0)

	return nil
}

// RemoveContract removes a contract from monitoring
func (m *ContractMonitor) RemoveContract(address string, chainID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%d", address, chainID)
	delete(m.contracts, key)
	delete(m.events, key)
}

// GetContract returns a monitored contract
func (m *ContractMonitor) GetContract(address string, chainID int64) (*Contract, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := fmt.Sprintf("%s:%d", address, chainID)
	c, ok := m.contracts[key]
	return c, ok
}

// ListContracts returns all monitored contracts
func (m *ContractMonitor) ListContracts() []*Contract {
	m.mu.RLock()
	defer m.mu.RUnlock()

	contracts := make([]*Contract, 0, len(m.contracts))
	for _, c := range m.contracts {
		contracts = append(contracts, c)
	}
	return contracts
}

// GetContractsByChain returns contracts for a specific chain
func (m *ContractMonitor) GetContractsByChain(chainID int64) []*Contract {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var contracts []*Contract
	for _, c := range m.contracts {
		if c.ChainID == chainID {
			contracts = append(contracts, c)
		}
	}
	return contracts
}

// GetEvents returns events for a contract
func (m *ContractMonitor) GetEvents(address string, chainID int64, limit int) []*ContractEvent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := fmt.Sprintf("%s:%d", address, chainID)
	events := m.events[key]

	if limit > 0 && len(events) > limit {
		return events[len(events)-limit:]
	}
	return events
}

// GetRecentEvents returns recent events across all contracts
func (m *ContractMonitor) GetRecentEvents(limit int) []*ContractEvent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var allEvents []*ContractEvent
	for _, events := range m.events {
		allEvents = append(allEvents, events...)
	}

	// Sort by timestamp (simplified - in production would use proper sorting)
	if limit > 0 && len(allEvents) > limit {
		return allEvents[len(allEvents)-limit:]
	}
	return allEvents
}

// PauseContract pauses monitoring for a contract
func (m *ContractMonitor) PauseContract(address string, chainID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%d", address, chainID)
	contract, ok := m.contracts[key]
	if !ok {
		return fmt.Errorf("contract not found: %s", key)
	}

	contract.Status = ContractStatusPaused
	contract.UpdatedAt = time.Now()
	return nil
}

// ResumeContract resumes monitoring for a contract
func (m *ContractMonitor) ResumeContract(address string, chainID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%d", address, chainID)
	contract, ok := m.contracts[key]
	if !ok {
		return fmt.Errorf("contract not found: %s", key)
	}

	contract.Status = ContractStatusActive
	contract.UpdatedAt = time.Now()
	return nil
}

// UpdateContract updates contract metadata
func (m *ContractMonitor) UpdateContract(address string, chainID int64, updates map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%d", address, chainID)
	contract, ok := m.contracts[key]
	if !ok {
		return fmt.Errorf("contract not found: %s", key)
	}

	if name, ok := updates["name"].(string); ok {
		contract.Name = name
	}
	if abi, ok := updates["abi"].(string); ok {
		contract.ABI = abi
	}
	if tags, ok := updates["tags"].([]string); ok {
		contract.Tags = tags
	}

	contract.UpdatedAt = time.Now()
	return nil
}

// GetStats returns monitoring statistics
func (m *ContractMonitor) GetStats() *MonitorStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := &MonitorStats{
		TotalContracts: len(m.contracts),
		ByChain:        make(map[int64]int),
		ByStatus:       make(map[string]int),
	}

	for _, c := range m.contracts {
		stats.ByChain[c.ChainID]++
		stats.ByStatus[string(c.Status)]++
	}

	for _, events := range m.events {
		stats.TotalEvents += len(events)
	}

	return stats
}

// MonitorStats contains monitoring statistics
type MonitorStats struct {
	TotalContracts int            `json:"total_contracts"`
	TotalEvents    int            `json:"total_events"`
	ByChain        map[int64]int  `json:"by_chain"`
	ByStatus       map[string]int `json:"by_status"`
}
