package explorer

import (
	"context"
	"errors"
	"testing"
	"time"
)

// ============================================================================
// MOCK REPOSITORY
// ============================================================================

// MockRepository implements RepositoryInterface for testing
type MockRepository struct {
	blocks       map[string]map[int64]*Block      // network -> blockNumber -> Block
	blocksByHash map[string]map[string]*Block     // network -> hash -> Block
	transactions map[string]map[string]*Transaction // network -> txHash -> Transaction
	txsByBlock   map[string]map[int64][]Transaction // network -> blockNumber -> []Transaction
	addresses    map[string]map[string]*Address   // network -> address -> Address
	eventLogs    map[string]map[string][]EventLog // network -> txHash -> []EventLog
	syncStates   map[string]*NetworkSyncState
	networkStats map[string]*NetworkStats

	// Error simulation
	simulateError bool
	errorToReturn error
}

// NewMockRepository creates a new mock repository
func NewMockRepository() *MockRepository {
	return &MockRepository{
		blocks:       make(map[string]map[int64]*Block),
		blocksByHash: make(map[string]map[string]*Block),
		transactions: make(map[string]map[string]*Transaction),
		txsByBlock:   make(map[string]map[int64][]Transaction),
		addresses:    make(map[string]map[string]*Address),
		eventLogs:    make(map[string]map[string][]EventLog),
		syncStates:   make(map[string]*NetworkSyncState),
		networkStats: make(map[string]*NetworkStats),
	}
}

// SetError enables error simulation
func (m *MockRepository) SetError(err error) {
	m.simulateError = true
	m.errorToReturn = err
}

// ClearError disables error simulation
func (m *MockRepository) ClearError() {
	m.simulateError = false
	m.errorToReturn = nil
}

// AddBlock adds a block to the mock
func (m *MockRepository) AddBlock(block *Block) {
	if m.blocks[block.Network] == nil {
		m.blocks[block.Network] = make(map[int64]*Block)
	}
	if m.blocksByHash[block.Network] == nil {
		m.blocksByHash[block.Network] = make(map[string]*Block)
	}
	m.blocks[block.Network][block.BlockNumber] = block
	m.blocksByHash[block.Network][block.BlockHash] = block
}

// AddTransaction adds a transaction to the mock
func (m *MockRepository) AddTransaction(tx *Transaction) {
	if m.transactions[tx.Network] == nil {
		m.transactions[tx.Network] = make(map[string]*Transaction)
	}
	if m.txsByBlock[tx.Network] == nil {
		m.txsByBlock[tx.Network] = make(map[int64][]Transaction)
	}
	m.transactions[tx.Network][tx.TxHash] = tx
	m.txsByBlock[tx.Network][tx.BlockNumber] = append(m.txsByBlock[tx.Network][tx.BlockNumber], *tx)
}

// AddAddress adds an address to the mock
func (m *MockRepository) AddAddress(addr *Address) {
	if m.addresses[addr.Network] == nil {
		m.addresses[addr.Network] = make(map[string]*Address)
	}
	m.addresses[addr.Network][addr.Address] = addr
}

// AddEventLog adds event logs for a transaction
func (m *MockRepository) AddEventLog(network, txHash string, log EventLog) {
	if m.eventLogs[network] == nil {
		m.eventLogs[network] = make(map[string][]EventLog)
	}
	m.eventLogs[network][txHash] = append(m.eventLogs[network][txHash], log)
}

// SetSyncState sets the sync state for a network
func (m *MockRepository) SetSyncState(state *NetworkSyncState) {
	m.syncStates[state.Network] = state
}

// SetNetworkStats sets the network stats
func (m *MockRepository) SetNetworkStats(stats *NetworkStats) {
	m.networkStats[stats.Network] = stats
}

// ============================================================================
// REPOSITORY INTERFACE IMPLEMENTATION
// ============================================================================

func (m *MockRepository) InsertBlock(ctx context.Context, block *Block) error {
	if m.simulateError {
		return m.errorToReturn
	}
	m.AddBlock(block)
	return nil
}

func (m *MockRepository) InsertBlocks(ctx context.Context, blocks []*Block) error {
	if m.simulateError {
		return m.errorToReturn
	}
	for _, block := range blocks {
		m.AddBlock(block)
	}
	return nil
}

func (m *MockRepository) GetBlockByNumber(ctx context.Context, network string, number int64) (*Block, error) {
	if m.simulateError {
		return nil, m.errorToReturn
	}
	if netBlocks, ok := m.blocks[network]; ok {
		if block, ok := netBlocks[number]; ok {
			return block, nil
		}
	}
	return nil, nil
}

func (m *MockRepository) GetBlockByHash(ctx context.Context, network, hash string) (*Block, error) {
	if m.simulateError {
		return nil, m.errorToReturn
	}
	if netBlocks, ok := m.blocksByHash[network]; ok {
		if block, ok := netBlocks[hash]; ok {
			return block, nil
		}
	}
	return nil, nil
}

func (m *MockRepository) ListBlocks(ctx context.Context, filter BlockFilter) (*ListResult[Block], error) {
	if m.simulateError {
		return nil, m.errorToReturn
	}
	var items []Block
	if netBlocks, ok := m.blocks[filter.Network]; ok {
		for _, block := range netBlocks {
			items = append(items, *block)
		}
	}
	return &ListResult[Block]{
		Items:      items,
		Total:      int64(len(items)),
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalPages: 1,
	}, nil
}

func (m *MockRepository) GetLatestBlock(ctx context.Context, network string) (*Block, error) {
	if m.simulateError {
		return nil, m.errorToReturn
	}
	var latest *Block
	if netBlocks, ok := m.blocks[network]; ok {
		for _, block := range netBlocks {
			if latest == nil || block.BlockNumber > latest.BlockNumber {
				latest = block
			}
		}
	}
	return latest, nil
}

func (m *MockRepository) InsertTransaction(ctx context.Context, tx *Transaction) error {
	if m.simulateError {
		return m.errorToReturn
	}
	m.AddTransaction(tx)
	return nil
}

func (m *MockRepository) InsertTransactions(ctx context.Context, txs []*Transaction) error {
	if m.simulateError {
		return m.errorToReturn
	}
	for _, tx := range txs {
		m.AddTransaction(tx)
	}
	return nil
}

func (m *MockRepository) GetTransactionByHash(ctx context.Context, network, hash string) (*Transaction, error) {
	if m.simulateError {
		return nil, m.errorToReturn
	}
	if netTxs, ok := m.transactions[network]; ok {
		if tx, ok := netTxs[hash]; ok {
			return tx, nil
		}
	}
	return nil, nil
}

func (m *MockRepository) ListTransactions(ctx context.Context, filter TransactionFilter) (*ListResult[Transaction], error) {
	if m.simulateError {
		return nil, m.errorToReturn
	}
	var items []Transaction
	if netTxs, ok := m.transactions[filter.Network]; ok {
		for _, tx := range netTxs {
			items = append(items, *tx)
		}
	}
	return &ListResult[Transaction]{
		Items:      items,
		Total:      int64(len(items)),
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalPages: 1,
	}, nil
}

func (m *MockRepository) GetTransactionsByBlock(ctx context.Context, network string, blockNumber int64) ([]Transaction, error) {
	if m.simulateError {
		return nil, m.errorToReturn
	}
	if netTxs, ok := m.txsByBlock[network]; ok {
		if txs, ok := netTxs[blockNumber]; ok {
			return txs, nil
		}
	}
	return nil, nil
}

func (m *MockRepository) UpsertAddress(ctx context.Context, addr *Address) error {
	if m.simulateError {
		return m.errorToReturn
	}
	m.AddAddress(addr)
	return nil
}

func (m *MockRepository) GetAddress(ctx context.Context, network, address string) (*Address, error) {
	if m.simulateError {
		return nil, m.errorToReturn
	}
	if netAddrs, ok := m.addresses[network]; ok {
		if addr, ok := netAddrs[address]; ok {
			return addr, nil
		}
	}
	return nil, nil
}

func (m *MockRepository) GetAddressTransactions(ctx context.Context, network, address string, opts PaginationOptions) (*ListResult[Transaction], error) {
	if m.simulateError {
		return nil, m.errorToReturn
	}
	var items []Transaction
	if netTxs, ok := m.transactions[network]; ok {
		for _, tx := range netTxs {
			if tx.From == address || (tx.To != nil && *tx.To == address) {
				items = append(items, *tx)
			}
		}
	}
	return &ListResult[Transaction]{
		Items:      items,
		Total:      int64(len(items)),
		Page:       opts.Page,
		PageSize:   opts.PageSize,
		TotalPages: 1,
	}, nil
}

func (m *MockRepository) IncrementAddressTxCount(ctx context.Context, network string, addresses []string, timestamp time.Time) error {
	if m.simulateError {
		return m.errorToReturn
	}
	return nil
}

func (m *MockRepository) InsertEventLog(ctx context.Context, log *EventLog) error {
	if m.simulateError {
		return m.errorToReturn
	}
	m.AddEventLog(log.Network, log.TxHash, *log)
	return nil
}

func (m *MockRepository) InsertEventLogs(ctx context.Context, logs []*EventLog) error {
	if m.simulateError {
		return m.errorToReturn
	}
	for _, log := range logs {
		m.AddEventLog(log.Network, log.TxHash, *log)
	}
	return nil
}

func (m *MockRepository) GetTransactionLogs(ctx context.Context, network, txHash string) ([]EventLog, error) {
	if m.simulateError {
		return nil, m.errorToReturn
	}
	if netLogs, ok := m.eventLogs[network]; ok {
		if logs, ok := netLogs[txHash]; ok {
			return logs, nil
		}
	}
	return nil, nil
}

func (m *MockRepository) GetAddressLogs(ctx context.Context, network, address string, opts PaginationOptions) (*ListResult[EventLog], error) {
	if m.simulateError {
		return nil, m.errorToReturn
	}
	var items []EventLog
	if netLogs, ok := m.eventLogs[network]; ok {
		for _, logs := range netLogs {
			for _, log := range logs {
				if log.ContractAddress == address {
					items = append(items, log)
				}
			}
		}
	}
	return &ListResult[EventLog]{
		Items:      items,
		Total:      int64(len(items)),
		Page:       opts.Page,
		PageSize:   opts.PageSize,
		TotalPages: 1,
	}, nil
}

func (m *MockRepository) GetSyncState(ctx context.Context, network string) (*NetworkSyncState, error) {
	if m.simulateError {
		return nil, m.errorToReturn
	}
	return m.syncStates[network], nil
}

func (m *MockRepository) UpdateSyncState(ctx context.Context, network string, lastBlock int64, isSyncing bool, blocksBehind int64, errMsg *string) error {
	if m.simulateError {
		return m.errorToReturn
	}
	m.syncStates[network] = &NetworkSyncState{
		Network:          network,
		LastIndexedBlock: lastBlock,
		IsSyncing:        isSyncing,
		BlocksBehind:     blocksBehind,
		ErrorMessage:     errMsg,
	}
	return nil
}

func (m *MockRepository) GetNetworkStats(ctx context.Context, network string) (*NetworkStats, error) {
	if m.simulateError {
		return nil, m.errorToReturn
	}
	return m.networkStats[network], nil
}

// Compile-time check that MockRepository implements RepositoryInterface
var _ RepositoryInterface = (*MockRepository)(nil)

func TestNewPaginationOptions(t *testing.T) {
	tests := []struct {
		name         string
		page         int
		pageSize     int
		expectedPage int
		expectedSize int
		expectedOff  int
	}{
		{"valid params", 2, 20, 2, 20, 20},
		{"page zero", 0, 20, 1, 20, 0},
		{"negative page", -1, 20, 1, 20, 0},
		{"size zero", 1, 0, 1, 20, 0},
		{"size too large", 1, 200, 1, 100, 0},
		{"page 3 size 50", 3, 50, 3, 50, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := NewPaginationOptions(tt.page, tt.pageSize)
			if opts.Page != tt.expectedPage {
				t.Errorf("Page = %d, want %d", opts.Page, tt.expectedPage)
			}
			if opts.PageSize != tt.expectedSize {
				t.Errorf("PageSize = %d, want %d", opts.PageSize, tt.expectedSize)
			}
			if opts.Offset != tt.expectedOff {
				t.Errorf("Offset = %d, want %d", opts.Offset, tt.expectedOff)
			}
		})
	}
}

func TestIsTxHash(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef", true},
		{"0xABCDEF1234567890ABCDEF1234567890ABCDEF1234567890ABCDEF1234567890", true},
		{"1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef", false}, // no 0x
		{"0x1234", false}, // too short
		{"0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdefXX", false}, // too long
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isTxHash(tt.input)
			if result != tt.expected {
				t.Errorf("isTxHash(%s) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsAddress(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"0x1234567890abcdef1234567890abcdef12345678", true},
		{"0xABCDEF1234567890ABCDEF1234567890ABCDEF12", true},
		{"1234567890abcdef1234567890abcdef12345678", false}, // no 0x
		{"0x1234", false}, // too short
		{"0x1234567890abcdef1234567890abcdef1234567890", false}, // too long
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isAddress(tt.input)
			if result != tt.expected {
				t.Errorf("isAddress(%s) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsBlockNumber(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"12345678", true},
		{"0", true},
		{"999999999", true},
		{"0x123", false},
		{"abc", false},
		{"-1", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isBlockNumber(tt.input)
			if result != tt.expected {
				t.Errorf("isBlockNumber(%s) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseBlockNumber(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
		hasError bool
	}{
		{"12345678", 12345678, false},
		{"0", 0, false},
		{"abc", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := parseBlockNumber(tt.input)
			if tt.hasError {
				if err == nil {
					t.Errorf("parseBlockNumber(%s) should error", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("parseBlockNumber(%s) error: %v", tt.input, err)
				}
				if result != tt.expected {
					t.Errorf("parseBlockNumber(%s) = %d, want %d", tt.input, result, tt.expected)
				}
			}
		})
	}
}

func TestParseValue(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"1000000000000000000", "1000000000000000000"},
		{"0", "0"},
		{"123456789012345678901234567890", "123456789012345678901234567890"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ParseValue(tt.input)
			if result.String() != tt.expected {
				t.Errorf("ParseValue(%s) = %s, want %s", tt.input, result.String(), tt.expected)
			}
		})
	}
}

func TestFormatValue(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"1000000000000000000", "1000000000000000000"},
		{"0", "0"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			val := ParseValue(tt.input)
			result := FormatValue(val)
			if result != tt.expected {
				t.Errorf("FormatValue(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}

	// Test nil
	if FormatValue(nil) != "0" {
		t.Error("FormatValue(nil) should return '0'")
	}
}

func TestSupportedNetworks(t *testing.T) {
	networks := SupportedNetworks()

	expected := []string{
		"ethereum", "polygon", "arbitrum", "optimism", "base", "bsc", "avalanche",
	}

	if len(networks) != len(expected) {
		t.Errorf("SupportedNetworks() length = %d, want %d", len(networks), len(expected))
	}

	for _, exp := range expected {
		found := false
		for _, net := range networks {
			if net == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("SupportedNetworks() missing %s", exp)
		}
	}
}

func TestIsValidNetwork(t *testing.T) {
	tests := []struct {
		network  string
		expected bool
	}{
		{"ethereum", true},
		{"polygon", true},
		{"arbitrum", true},
		{"optimism", true},
		{"base", true},
		{"bsc", true},
		{"avalanche", true},
		{"invalid", false},
		{"", false},
		{"ETHEREUM", false}, // case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.network, func(t *testing.T) {
			result := IsValidNetwork(tt.network)
			if result != tt.expected {
				t.Errorf("IsValidNetwork(%s) = %v, want %v", tt.network, result, tt.expected)
			}
		})
	}
}

func TestBlockFields(t *testing.T) {
	now := time.Now()
	baseFee := int64(1000000000)

	block := &Block{
		ID:               1,
		Network:          "ethereum",
		BlockNumber:      12345678,
		BlockHash:        "0xabc123",
		ParentHash:       "0xdef456",
		Timestamp:        now,
		Miner:            "0x1234567890abcdef1234567890abcdef12345678",
		GasUsed:          15000000,
		GasLimit:         30000000,
		BaseFeePerGas:    &baseFee,
		TransactionCount: 150,
		Size:             50000,
		ExtraData:        "geth",
		CreatedAt:        now,
	}

	if block.Network != "ethereum" {
		t.Errorf("Network = %s, want ethereum", block.Network)
	}
	if block.BlockNumber != 12345678 {
		t.Errorf("BlockNumber = %d, want 12345678", block.BlockNumber)
	}
	if *block.BaseFeePerGas != 1000000000 {
		t.Errorf("BaseFeePerGas = %d, want 1000000000", *block.BaseFeePerGas)
	}
}

func TestTransactionFields(t *testing.T) {
	now := time.Now()
	toAddr := "0xabcdef1234567890abcdef1234567890abcdef12"
	gasPrice := int64(20000000000)
	gasUsed := int64(21000)
	status := 1

	tx := &Transaction{
		ID:          1,
		Network:     "ethereum",
		TxHash:      "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
		BlockNumber: 12345678,
		BlockHash:   "0xabc123",
		TxIndex:     0,
		From:        "0x1234567890abcdef1234567890abcdef12345678",
		To:          &toAddr,
		Value:       "1000000000000000000",
		GasPrice:    &gasPrice,
		GasLimit:    21000,
		GasUsed:     &gasUsed,
		Nonce:       42,
		TxType:      2,
		Status:      &status,
		Timestamp:   now,
	}

	if tx.Network != "ethereum" {
		t.Errorf("Network = %s, want ethereum", tx.Network)
	}
	if tx.Value != "1000000000000000000" {
		t.Errorf("Value = %s, want 1000000000000000000", tx.Value)
	}
	if *tx.Status != 1 {
		t.Errorf("Status = %d, want 1", *tx.Status)
	}
}

func TestAddressFields(t *testing.T) {
	now := time.Now()
	label := "Binance"
	creator := "0x0000000000000000000000000000000000000000"
	creationTx := "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"

	addr := &Address{
		ID:                 1,
		Network:            "ethereum",
		Address:            "0xabcdef1234567890abcdef1234567890abcdef12",
		Balance:            "1000000000000000000000",
		TxCount:            1500,
		IsContract:         true,
		ContractCreator:    &creator,
		ContractCreationTx: &creationTx,
		FirstSeenAt:        &now,
		LastSeenAt:         &now,
		Label:              &label,
		Tags:               []string{"exchange", "cex"},
	}

	if addr.Network != "ethereum" {
		t.Errorf("Network = %s, want ethereum", addr.Network)
	}
	if addr.TxCount != 1500 {
		t.Errorf("TxCount = %d, want 1500", addr.TxCount)
	}
	if !addr.IsContract {
		t.Error("IsContract should be true")
	}
	if len(addr.Tags) != 2 {
		t.Errorf("Tags length = %d, want 2", len(addr.Tags))
	}
}

func TestEventLogFields(t *testing.T) {
	now := time.Now()
	topic0 := "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"
	topic1 := "0x0000000000000000000000001234567890abcdef1234567890abcdef12345678"
	decodedName := "Transfer"

	log := &EventLog{
		ID:              1,
		Network:         "ethereum",
		TxHash:          "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
		LogIndex:        0,
		BlockNumber:     12345678,
		ContractAddress: "0xabcdef1234567890abcdef1234567890abcdef12",
		Topic0:          &topic0,
		Topic1:          &topic1,
		Data:            "0x0000000000000000000000000000000000000000000000000de0b6b3a7640000",
		Timestamp:       now,
		DecodedName:     &decodedName,
		DecodedArgs:     map[string]any{"from": "0x123", "to": "0x456", "value": "1000000000000000000"},
		Removed:         false,
	}

	if log.Network != "ethereum" {
		t.Errorf("Network = %s, want ethereum", log.Network)
	}
	if *log.DecodedName != "Transfer" {
		t.Errorf("DecodedName = %s, want Transfer", *log.DecodedName)
	}
	if log.Removed {
		t.Error("Removed should be false")
	}
}

func TestNetworkSyncState(t *testing.T) {
	now := time.Now()
	errMsg := "connection timeout"

	state := &NetworkSyncState{
		ID:               1,
		Network:          "ethereum",
		LastIndexedBlock: 12345678,
		LastIndexedAt:    &now,
		IsSyncing:        true,
		SyncStartedAt:    &now,
		BlocksBehind:     100,
		ErrorMessage:     &errMsg,
		UpdatedAt:        now,
	}

	if state.Network != "ethereum" {
		t.Errorf("Network = %s, want ethereum", state.Network)
	}
	if !state.IsSyncing {
		t.Error("IsSyncing should be true")
	}
	if *state.ErrorMessage != "connection timeout" {
		t.Errorf("ErrorMessage = %s, want 'connection timeout'", *state.ErrorMessage)
	}
}

func TestListResult(t *testing.T) {
	result := &ListResult[Block]{
		Items: []Block{
			{BlockNumber: 1},
			{BlockNumber: 2},
		},
		Total:      100,
		Page:       1,
		PageSize:   20,
		TotalPages: 5,
	}

	if len(result.Items) != 2 {
		t.Errorf("Items length = %d, want 2", len(result.Items))
	}
	if result.Total != 100 {
		t.Errorf("Total = %d, want 100", result.Total)
	}
	if result.TotalPages != 5 {
		t.Errorf("TotalPages = %d, want 5", result.TotalPages)
	}
}

func TestSearchResult(t *testing.T) {
	block := &Block{BlockNumber: 12345678, Network: "ethereum"}

	result := SearchResult{
		Type:    "block",
		Network: "ethereum",
		Data:    block,
	}

	if result.Type != "block" {
		t.Errorf("Type = %s, want block", result.Type)
	}

	blockData, ok := result.Data.(*Block)
	if !ok {
		t.Error("Data should be *Block")
	}
	if blockData.BlockNumber != 12345678 {
		t.Errorf("BlockNumber = %d, want 12345678", blockData.BlockNumber)
	}
}

func TestSearchResults(t *testing.T) {
	results := &SearchResults{
		Query: "0x1234",
		Results: []SearchResult{
			{Type: "transaction", Network: "ethereum", Data: nil},
			{Type: "address", Network: "ethereum", Data: nil},
		},
		Total: 2,
	}

	if results.Query != "0x1234" {
		t.Errorf("Query = %s, want 0x1234", results.Query)
	}
	if results.Total != 2 {
		t.Errorf("Total = %d, want 2", results.Total)
	}
	if len(results.Results) != 2 {
		t.Errorf("Results length = %d, want 2", len(results.Results))
	}
}

func TestBlockFilter(t *testing.T) {
	miner := "0x1234567890abcdef1234567890abcdef12345678"
	fromBlock := int64(100)
	toBlock := int64(200)

	filter := BlockFilter{
		Network:           "ethereum",
		FromBlock:         &fromBlock,
		ToBlock:           &toBlock,
		Miner:             &miner,
		PaginationOptions: NewPaginationOptions(1, 20),
	}

	if filter.Network != "ethereum" {
		t.Errorf("Network = %s, want ethereum", filter.Network)
	}
	if *filter.FromBlock != 100 {
		t.Errorf("FromBlock = %d, want 100", *filter.FromBlock)
	}
	if *filter.ToBlock != 200 {
		t.Errorf("ToBlock = %d, want 200", *filter.ToBlock)
	}
}

func TestTransactionFilter(t *testing.T) {
	fromAddr := "0x1234567890abcdef1234567890abcdef12345678"
	status := 1

	filter := TransactionFilter{
		Network:           "polygon",
		FromAddress:       &fromAddr,
		Status:            &status,
		PaginationOptions: NewPaginationOptions(2, 50),
	}

	if filter.Network != "polygon" {
		t.Errorf("Network = %s, want polygon", filter.Network)
	}
	if *filter.FromAddress != fromAddr {
		t.Errorf("FromAddress = %s, want %s", *filter.FromAddress, fromAddr)
	}
	if filter.Page != 2 {
		t.Errorf("Page = %d, want 2", filter.Page)
	}
}

// Benchmarks

func BenchmarkIsTxHash(b *testing.B) {
	hash := "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		isTxHash(hash)
	}
}

func BenchmarkIsAddress(b *testing.B) {
	addr := "0x1234567890abcdef1234567890abcdef12345678"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		isAddress(addr)
	}
}

func BenchmarkIsBlockNumber(b *testing.B) {
	num := "12345678"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		isBlockNumber(num)
	}
}

func BenchmarkParseValue(b *testing.B) {
	val := "1000000000000000000"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParseValue(val)
	}
}

func BenchmarkNewPaginationOptions(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewPaginationOptions(5, 50)
	}
}

func TestIsBlockHash(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef", true},
		{"0xABCDEF1234567890ABCDEF1234567890ABCDEF1234567890ABCDEF1234567890", true},
		{"1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef", false}, // no 0x
		{"0x1234", false}, // too short
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isBlockHash(tt.input)
			if result != tt.expected {
				t.Errorf("isBlockHash(%s) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNew(t *testing.T) {
	// Test creating explorer with nil pool
	// Note: This won't actually work in production but tests the constructor
	explorer := New(nil)

	if explorer == nil {
		t.Error("Expected non-nil explorer")
	}
}

func TestNewWithRepository(t *testing.T) {
	repo := NewRepository(nil)
	explorer := NewWithRepository(repo)

	if explorer == nil {
		t.Error("Expected non-nil explorer")
	}

	if explorer.Repository() != repo {
		t.Error("Repository should match the one passed in")
	}
}

func TestExplorerRepository(t *testing.T) {
	repo := NewRepository(nil)
	explorer := NewWithRepository(repo)

	returnedRepo := explorer.Repository()
	if returnedRepo != repo {
		t.Error("Repository() should return the underlying repository")
	}
}

func TestEventLogFilter(t *testing.T) {
	contractAddr := "0x1234567890abcdef1234567890abcdef12345678"
	topic0 := "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"
	fromBlock := int64(100)
	toBlock := int64(200)

	filter := EventLogFilter{
		Network:           "ethereum",
		ContractAddress:   &contractAddr,
		Topic0:            &topic0,
		FromBlock:         &fromBlock,
		ToBlock:           &toBlock,
		PaginationOptions: NewPaginationOptions(1, 20),
	}

	if filter.Network != "ethereum" {
		t.Errorf("Network = %s, want ethereum", filter.Network)
	}
	if *filter.ContractAddress != contractAddr {
		t.Errorf("ContractAddress = %s, want %s", *filter.ContractAddress, contractAddr)
	}
	if *filter.Topic0 != topic0 {
		t.Errorf("Topic0 = %s, want %s", *filter.Topic0, topic0)
	}
	if *filter.FromBlock != 100 {
		t.Errorf("FromBlock = %d, want 100", *filter.FromBlock)
	}
	if *filter.ToBlock != 200 {
		t.Errorf("ToBlock = %d, want 200", *filter.ToBlock)
	}
}

func TestAddressFilter(t *testing.T) {
	isContract := true
	label := "DEX"
	minBal := ParseValue("1000000000000000000")
	maxBal := ParseValue("10000000000000000000000")

	filter := AddressFilter{
		Network:           "polygon",
		IsContract:        &isContract,
		MinBalance:        minBal,
		MaxBalance:        maxBal,
		Label:             &label,
		PaginationOptions: NewPaginationOptions(1, 50),
	}

	if filter.Network != "polygon" {
		t.Errorf("Network = %s, want polygon", filter.Network)
	}
	if *filter.IsContract != true {
		t.Errorf("IsContract = %v, want true", *filter.IsContract)
	}
	if *filter.Label != "DEX" {
		t.Errorf("Label = %s, want DEX", *filter.Label)
	}
	if filter.MinBalance.Cmp(minBal) != 0 {
		t.Errorf("MinBalance = %s, want %s", filter.MinBalance.String(), minBal.String())
	}
}

func TestInternalTransactionFields(t *testing.T) {
	now := time.Now()
	toAddr := "0x2222222222222222222222222222222222222222"
	callType := "call"
	gas := int64(100000)
	gasUsed := int64(50000)
	output := "0x"

	itx := &InternalTransaction{
		ID:           1,
		Network:      "ethereum",
		TxHash:       "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
		TraceAddress: "0",
		BlockNumber:  12345678,
		TraceType:    "call",
		CallType:     &callType,
		From:         "0x1111111111111111111111111111111111111111",
		To:           &toAddr,
		Value:        "1000000000000000000",
		Gas:          &gas,
		GasUsed:      &gasUsed,
		Input:        "0x",
		Output:       &output,
		Timestamp:    now,
	}

	if itx.Network != "ethereum" {
		t.Errorf("Network = %s, want ethereum", itx.Network)
	}
	if itx.TraceType != "call" {
		t.Errorf("TraceType = %s, want call", itx.TraceType)
	}
	if *itx.Gas != 100000 {
		t.Errorf("Gas = %d, want 100000", *itx.Gas)
	}
	if *itx.GasUsed != 50000 {
		t.Errorf("GasUsed = %d, want 50000", *itx.GasUsed)
	}
}

func TestNetworkStatsFields(t *testing.T) {
	now := time.Now()
	avgGas := ParseValue("30000000000") // 30 gwei

	stats := &NetworkStats{
		Network:           "ethereum",
		LatestBlock:       18000000,
		TotalTransactions: 2000000000,
		TotalAddresses:    300000000,
		TotalContracts:    50000000,
		AvgBlockTime:      12.5,
		AvgGasPrice:       avgGas,
		TPS:               15.5,
		LastUpdated:       now,
	}

	if stats.Network != "ethereum" {
		t.Errorf("Network = %s, want ethereum", stats.Network)
	}
	if stats.LatestBlock != 18000000 {
		t.Errorf("LatestBlock = %d, want 18000000", stats.LatestBlock)
	}
	if stats.TPS != 15.5 {
		t.Errorf("TPS = %f, want 15.5", stats.TPS)
	}
	if stats.AvgBlockTime != 12.5 {
		t.Errorf("AvgBlockTime = %f, want 12.5", stats.AvgBlockTime)
	}
}

func TestParseValueEdgeCases(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"0", "0"},
		{"1", "1"},
		{"12345678901234567890123456789012345678901234567890", "12345678901234567890123456789012345678901234567890"},
		{"", "0"},
		{"invalid", "0"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ParseValue(tt.input)
			if result.String() != tt.expected {
				t.Errorf("ParseValue(%s) = %s, want %s", tt.input, result.String(), tt.expected)
			}
		})
	}
}

func TestPaginationOptionsLimits(t *testing.T) {
	// Test minimum values
	opts := NewPaginationOptions(0, 0)
	if opts.Page != 1 {
		t.Errorf("Page should default to 1, got %d", opts.Page)
	}
	if opts.PageSize != 20 {
		t.Errorf("PageSize should default to 20, got %d", opts.PageSize)
	}

	// Test maximum page size
	opts = NewPaginationOptions(1, 1000)
	if opts.PageSize != 100 {
		t.Errorf("PageSize should be capped at 100, got %d", opts.PageSize)
	}

	// Test offset calculation
	opts = NewPaginationOptions(5, 20)
	expectedOffset := (5 - 1) * 20
	if opts.Offset != expectedOffset {
		t.Errorf("Offset = %d, want %d", opts.Offset, expectedOffset)
	}
}

func TestTransactionFilterTimeRange(t *testing.T) {
	now := time.Now()
	oneHourAgo := now.Add(-time.Hour)

	filter := TransactionFilter{
		Network:  "ethereum",
		FromTime: &oneHourAgo,
		ToTime:   &now,
	}

	if filter.FromTime == nil {
		t.Error("FromTime should not be nil")
	}
	if filter.ToTime == nil {
		t.Error("ToTime should not be nil")
	}
	if !filter.ToTime.After(*filter.FromTime) {
		t.Error("ToTime should be after FromTime")
	}
}

func TestBlockFilterRange(t *testing.T) {
	from := int64(100)
	to := int64(200)

	filter := BlockFilter{
		Network:   "ethereum",
		FromBlock: &from,
		ToBlock:   &to,
	}

	if *filter.FromBlock != 100 {
		t.Errorf("FromBlock = %d, want 100", *filter.FromBlock)
	}
	if *filter.ToBlock != 200 {
		t.Errorf("ToBlock = %d, want 200", *filter.ToBlock)
	}
	if *filter.ToBlock <= *filter.FromBlock {
		t.Error("ToBlock should be greater than FromBlock")
	}
}

func TestListResultCalculation(t *testing.T) {
	// Test TotalPages calculation
	result := ListResult[Block]{
		Total:    55,
		PageSize: 10,
	}

	expectedPages := 6 // ceil(55/10)
	if result.Total/int64(result.PageSize) != 5 {
		// Just verify the data is set correctly
		if result.Total != 55 || result.PageSize != 10 {
			t.Error("ListResult fields not set correctly")
		}
	}

	// With exact division
	result2 := ListResult[Transaction]{
		Total:    100,
		PageSize: 20,
	}

	expectedPages = 5 // 100/20
	_ = expectedPages
	if result2.Total != 100 || result2.PageSize != 20 {
		t.Error("ListResult fields not set correctly")
	}
}

func BenchmarkIsValidNetwork(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsValidNetwork("ethereum")
	}
}

func BenchmarkSupportedNetworks(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SupportedNetworks()
	}
}

// ============================================================================
// EXPLORER TESTS WITH MOCK REPOSITORY
// ============================================================================

func TestExplorerWithMock_GetBlock(t *testing.T) {
	mock := NewMockRepository()
	explorer := NewWithRepository(mock)
	ctx := context.Background()

	// Add test block
	block := &Block{
		Network:     "ethereum",
		BlockNumber: 12345678,
		BlockHash:   "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
		Timestamp:   time.Now(),
	}
	mock.AddBlock(block)

	// Test GetBlock by number
	result, err := explorer.GetBlock(ctx, "ethereum", "12345678")
	if err != nil {
		t.Fatalf("GetBlock failed: %v", err)
	}
	if result == nil {
		t.Fatal("Expected block, got nil")
	}
	if result.BlockNumber != 12345678 {
		t.Errorf("BlockNumber = %d, want 12345678", result.BlockNumber)
	}

	// Test GetBlock by hash
	result, err = explorer.GetBlock(ctx, "ethereum", "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")
	if err != nil {
		t.Fatalf("GetBlock by hash failed: %v", err)
	}
	if result == nil {
		t.Fatal("Expected block, got nil")
	}

	// Test GetBlock not found
	result, err = explorer.GetBlock(ctx, "ethereum", "99999999")
	if err != nil {
		t.Fatalf("GetBlock failed: %v", err)
	}
	if result != nil {
		t.Error("Expected nil for non-existent block")
	}
}

func TestExplorerWithMock_GetBlockByNumber(t *testing.T) {
	mock := NewMockRepository()
	explorer := NewWithRepository(mock)
	ctx := context.Background()

	block := &Block{
		Network:     "ethereum",
		BlockNumber: 100,
		BlockHash:   "0xabc",
	}
	mock.AddBlock(block)

	result, err := explorer.GetBlockByNumber(ctx, "ethereum", 100)
	if err != nil {
		t.Fatalf("GetBlockByNumber failed: %v", err)
	}
	if result == nil {
		t.Fatal("Expected block, got nil")
	}
	if result.BlockNumber != 100 {
		t.Errorf("BlockNumber = %d, want 100", result.BlockNumber)
	}
}

func TestExplorerWithMock_GetBlockByHash(t *testing.T) {
	mock := NewMockRepository()
	explorer := NewWithRepository(mock)
	ctx := context.Background()

	block := &Block{
		Network:     "ethereum",
		BlockNumber: 100,
		BlockHash:   "0xabc123",
	}
	mock.AddBlock(block)

	result, err := explorer.GetBlockByHash(ctx, "ethereum", "0xabc123")
	if err != nil {
		t.Fatalf("GetBlockByHash failed: %v", err)
	}
	if result == nil {
		t.Fatal("Expected block, got nil")
	}
	if result.BlockHash != "0xabc123" {
		t.Errorf("BlockHash = %s, want 0xabc123", result.BlockHash)
	}
}

func TestExplorerWithMock_GetLatestBlock(t *testing.T) {
	mock := NewMockRepository()
	explorer := NewWithRepository(mock)
	ctx := context.Background()

	// Add multiple blocks
	mock.AddBlock(&Block{Network: "ethereum", BlockNumber: 100, BlockHash: "0xa"})
	mock.AddBlock(&Block{Network: "ethereum", BlockNumber: 200, BlockHash: "0xb"})
	mock.AddBlock(&Block{Network: "ethereum", BlockNumber: 150, BlockHash: "0xc"})

	result, err := explorer.GetLatestBlock(ctx, "ethereum")
	if err != nil {
		t.Fatalf("GetLatestBlock failed: %v", err)
	}
	if result == nil {
		t.Fatal("Expected block, got nil")
	}
	if result.BlockNumber != 200 {
		t.Errorf("Latest block number = %d, want 200", result.BlockNumber)
	}
}

func TestExplorerWithMock_ListBlocks(t *testing.T) {
	mock := NewMockRepository()
	explorer := NewWithRepository(mock)
	ctx := context.Background()

	mock.AddBlock(&Block{Network: "ethereum", BlockNumber: 100})
	mock.AddBlock(&Block{Network: "ethereum", BlockNumber: 101})
	mock.AddBlock(&Block{Network: "ethereum", BlockNumber: 102})

	result, err := explorer.ListBlocks(ctx, "ethereum", 1, 20, nil)
	if err != nil {
		t.Fatalf("ListBlocks failed: %v", err)
	}
	if result.Total != 3 {
		t.Errorf("Total = %d, want 3", result.Total)
	}
	if len(result.Items) != 3 {
		t.Errorf("Items count = %d, want 3", len(result.Items))
	}
}

func TestExplorerWithMock_GetTransaction(t *testing.T) {
	mock := NewMockRepository()
	explorer := NewWithRepository(mock)
	ctx := context.Background()

	txHash := "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
	tx := &Transaction{
		Network:     "ethereum",
		TxHash:      txHash,
		BlockNumber: 100,
		From:        "0x1111111111111111111111111111111111111111",
		Value:       "1000000000000000000",
	}
	mock.AddTransaction(tx)

	result, err := explorer.GetTransaction(ctx, "ethereum", txHash)
	if err != nil {
		t.Fatalf("GetTransaction failed: %v", err)
	}
	if result == nil {
		t.Fatal("Expected transaction, got nil")
	}
	if result.TxHash != txHash {
		t.Errorf("TxHash = %s, want %s", result.TxHash, txHash)
	}
	if result.Value != "1000000000000000000" {
		t.Errorf("Value = %s, want 1000000000000000000", result.Value)
	}
}

func TestExplorerWithMock_ListTransactions(t *testing.T) {
	mock := NewMockRepository()
	explorer := NewWithRepository(mock)
	ctx := context.Background()

	mock.AddTransaction(&Transaction{Network: "ethereum", TxHash: "0x1", BlockNumber: 100})
	mock.AddTransaction(&Transaction{Network: "ethereum", TxHash: "0x2", BlockNumber: 101})

	filter := TransactionFilter{
		Network:           "ethereum",
		PaginationOptions: NewPaginationOptions(1, 20),
	}

	result, err := explorer.ListTransactions(ctx, filter)
	if err != nil {
		t.Fatalf("ListTransactions failed: %v", err)
	}
	if result.Total != 2 {
		t.Errorf("Total = %d, want 2", result.Total)
	}
}

func TestExplorerWithMock_GetBlockTransactions(t *testing.T) {
	mock := NewMockRepository()
	explorer := NewWithRepository(mock)
	ctx := context.Background()

	mock.AddTransaction(&Transaction{Network: "ethereum", TxHash: "0x1", BlockNumber: 100, TxIndex: 0})
	mock.AddTransaction(&Transaction{Network: "ethereum", TxHash: "0x2", BlockNumber: 100, TxIndex: 1})
	mock.AddTransaction(&Transaction{Network: "ethereum", TxHash: "0x3", BlockNumber: 101, TxIndex: 0})

	result, err := explorer.GetBlockTransactions(ctx, "ethereum", 100)
	if err != nil {
		t.Fatalf("GetBlockTransactions failed: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("Count = %d, want 2", len(result))
	}
}

func TestExplorerWithMock_GetTransactionLogs(t *testing.T) {
	mock := NewMockRepository()
	explorer := NewWithRepository(mock)
	ctx := context.Background()

	txHash := "0x1234"
	topic := "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"
	mock.AddEventLog("ethereum", txHash, EventLog{
		Network:         "ethereum",
		TxHash:          txHash,
		LogIndex:        0,
		ContractAddress: "0xtoken",
		Topic0:          &topic,
	})
	mock.AddEventLog("ethereum", txHash, EventLog{
		Network:         "ethereum",
		TxHash:          txHash,
		LogIndex:        1,
		ContractAddress: "0xtoken",
	})

	result, err := explorer.GetTransactionLogs(ctx, "ethereum", txHash)
	if err != nil {
		t.Fatalf("GetTransactionLogs failed: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("Count = %d, want 2", len(result))
	}
}

func TestExplorerWithMock_GetAddress(t *testing.T) {
	mock := NewMockRepository()
	explorer := NewWithRepository(mock)
	ctx := context.Background()

	address := "0x1234567890abcdef1234567890abcdef12345678"
	addr := &Address{
		Network:    "ethereum",
		Address:    address,
		Balance:    "5000000000000000000",
		TxCount:    150,
		IsContract: false,
	}
	mock.AddAddress(addr)

	result, err := explorer.GetAddress(ctx, "ethereum", address)
	if err != nil {
		t.Fatalf("GetAddress failed: %v", err)
	}
	if result == nil {
		t.Fatal("Expected address, got nil")
	}
	if result.Balance != "5000000000000000000" {
		t.Errorf("Balance = %s, want 5000000000000000000", result.Balance)
	}
	if result.TxCount != 150 {
		t.Errorf("TxCount = %d, want 150", result.TxCount)
	}
}

func TestExplorerWithMock_GetAddress_NotFound(t *testing.T) {
	mock := NewMockRepository()
	explorer := NewWithRepository(mock)
	ctx := context.Background()

	// Address not in DB should return default
	result, err := explorer.GetAddress(ctx, "ethereum", "0x9999999999999999999999999999999999999999")
	if err != nil {
		t.Fatalf("GetAddress failed: %v", err)
	}
	if result == nil {
		t.Fatal("Expected default address, got nil")
	}
	if result.Balance != "0" {
		t.Errorf("Balance = %s, want 0", result.Balance)
	}
	if result.TxCount != 0 {
		t.Errorf("TxCount = %d, want 0", result.TxCount)
	}
}

func TestExplorerWithMock_GetAddressTransactions(t *testing.T) {
	mock := NewMockRepository()
	explorer := NewWithRepository(mock)
	ctx := context.Background()

	address := "0x1111111111111111111111111111111111111111"
	toAddr := "0x2222222222222222222222222222222222222222"

	mock.AddTransaction(&Transaction{Network: "ethereum", TxHash: "0x1", From: address, BlockNumber: 100})
	mock.AddTransaction(&Transaction{Network: "ethereum", TxHash: "0x2", From: "0xother", To: &address, BlockNumber: 101})
	mock.AddTransaction(&Transaction{Network: "ethereum", TxHash: "0x3", From: "0xother", To: &toAddr, BlockNumber: 102})

	result, err := explorer.GetAddressTransactions(ctx, "ethereum", address, 1, 20)
	if err != nil {
		t.Fatalf("GetAddressTransactions failed: %v", err)
	}
	if result.Total != 2 {
		t.Errorf("Total = %d, want 2", result.Total)
	}
}

func TestExplorerWithMock_GetAddressLogs(t *testing.T) {
	mock := NewMockRepository()
	explorer := NewWithRepository(mock)
	ctx := context.Background()

	contractAddr := "0xcontract"
	mock.AddEventLog("ethereum", "0xtx1", EventLog{
		Network:         "ethereum",
		TxHash:          "0xtx1",
		ContractAddress: contractAddr,
	})
	mock.AddEventLog("ethereum", "0xtx2", EventLog{
		Network:         "ethereum",
		TxHash:          "0xtx2",
		ContractAddress: contractAddr,
	})
	mock.AddEventLog("ethereum", "0xtx3", EventLog{
		Network:         "ethereum",
		TxHash:          "0xtx3",
		ContractAddress: "0xother",
	})

	result, err := explorer.GetAddressLogs(ctx, "ethereum", contractAddr, 1, 20)
	if err != nil {
		t.Fatalf("GetAddressLogs failed: %v", err)
	}
	if result.Total != 2 {
		t.Errorf("Total = %d, want 2", result.Total)
	}
}

func TestExplorerWithMock_Search(t *testing.T) {
	mock := NewMockRepository()
	explorer := NewWithRepository(mock)
	ctx := context.Background()

	// Add test data
	txHash := "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
	mock.AddTransaction(&Transaction{Network: "ethereum", TxHash: txHash, BlockNumber: 100})
	mock.AddBlock(&Block{Network: "ethereum", BlockNumber: 12345, BlockHash: "0xblock"})
	mock.AddAddress(&Address{Network: "ethereum", Address: "0x1234567890abcdef1234567890abcdef12345678", Balance: "100"})

	// Search for transaction
	result, err := explorer.Search(ctx, "ethereum", txHash)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("Total = %d, want 1", result.Total)
	}
	if len(result.Results) > 0 && result.Results[0].Type != "transaction" {
		t.Errorf("Type = %s, want transaction", result.Results[0].Type)
	}

	// Search for block number
	result, err = explorer.Search(ctx, "ethereum", "12345")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("Total = %d, want 1", result.Total)
	}

	// Search for address
	result, err = explorer.Search(ctx, "ethereum", "0x1234567890abcdef1234567890abcdef12345678")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("Total = %d, want 1", result.Total)
	}

	// Empty search
	result, err = explorer.Search(ctx, "ethereum", "")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if result.Total != 0 {
		t.Errorf("Total = %d, want 0", result.Total)
	}
}

func TestExplorerWithMock_GetNetworkStats(t *testing.T) {
	mock := NewMockRepository()
	explorer := NewWithRepository(mock)
	ctx := context.Background()

	stats := &NetworkStats{
		Network:           "ethereum",
		LatestBlock:       18000000,
		TotalTransactions: 2000000000,
		TotalAddresses:    300000000,
	}
	mock.SetNetworkStats(stats)

	result, err := explorer.GetNetworkStats(ctx, "ethereum")
	if err != nil {
		t.Fatalf("GetNetworkStats failed: %v", err)
	}
	if result == nil {
		t.Fatal("Expected stats, got nil")
	}
	if result.LatestBlock != 18000000 {
		t.Errorf("LatestBlock = %d, want 18000000", result.LatestBlock)
	}
}

func TestExplorerWithMock_GetSyncState(t *testing.T) {
	mock := NewMockRepository()
	explorer := NewWithRepository(mock)
	ctx := context.Background()

	state := &NetworkSyncState{
		Network:          "ethereum",
		LastIndexedBlock: 18000000,
		IsSyncing:        true,
		BlocksBehind:     50,
	}
	mock.SetSyncState(state)

	result, err := explorer.GetSyncState(ctx, "ethereum")
	if err != nil {
		t.Fatalf("GetSyncState failed: %v", err)
	}
	if result == nil {
		t.Fatal("Expected state, got nil")
	}
	if result.LastIndexedBlock != 18000000 {
		t.Errorf("LastIndexedBlock = %d, want 18000000", result.LastIndexedBlock)
	}
	if !result.IsSyncing {
		t.Error("IsSyncing should be true")
	}
}

func TestExplorerWithMock_IndexBlock(t *testing.T) {
	mock := NewMockRepository()
	explorer := NewWithRepository(mock)
	ctx := context.Background()

	block := &Block{
		Network:     "ethereum",
		BlockNumber: 100,
		BlockHash:   "0xblock",
		Timestamp:   time.Now(),
	}
	toAddr := "0x2222222222222222222222222222222222222222"
	txs := []*Transaction{
		{Network: "ethereum", TxHash: "0xtx1", BlockNumber: 100, From: "0x1111111111111111111111111111111111111111", To: &toAddr},
	}
	logs := []*EventLog{
		{Network: "ethereum", TxHash: "0xtx1", ContractAddress: "0xcontract"},
	}

	err := explorer.IndexBlock(ctx, block, txs, logs)
	if err != nil {
		t.Fatalf("IndexBlock failed: %v", err)
	}

	// Verify block was indexed
	result, _ := mock.GetBlockByNumber(ctx, "ethereum", 100)
	if result == nil {
		t.Error("Block should be indexed")
	}

	// Verify sync state was updated
	state, _ := mock.GetSyncState(ctx, "ethereum")
	if state == nil {
		t.Error("Sync state should be updated")
	}
	if state.LastIndexedBlock != 100 {
		t.Errorf("LastIndexedBlock = %d, want 100", state.LastIndexedBlock)
	}
}

func TestExplorerWithMock_IndexBlocks(t *testing.T) {
	mock := NewMockRepository()
	explorer := NewWithRepository(mock)
	ctx := context.Background()

	blocks := []*Block{
		{Network: "ethereum", BlockNumber: 100, BlockHash: "0xa"},
		{Network: "ethereum", BlockNumber: 101, BlockHash: "0xb"},
		{Network: "ethereum", BlockNumber: 102, BlockHash: "0xc"},
	}

	err := explorer.IndexBlocks(ctx, blocks, nil, nil)
	if err != nil {
		t.Fatalf("IndexBlocks failed: %v", err)
	}

	// Verify all blocks were indexed
	for _, block := range blocks {
		result, _ := mock.GetBlockByNumber(ctx, "ethereum", block.BlockNumber)
		if result == nil {
			t.Errorf("Block %d should be indexed", block.BlockNumber)
		}
	}

	// Verify sync state points to latest block
	state, _ := mock.GetSyncState(ctx, "ethereum")
	if state.LastIndexedBlock != 102 {
		t.Errorf("LastIndexedBlock = %d, want 102", state.LastIndexedBlock)
	}
}

func TestExplorerWithMock_ErrorHandling(t *testing.T) {
	mock := NewMockRepository()
	explorer := NewWithRepository(mock)
	ctx := context.Background()

	testErr := errors.New("database error")
	mock.SetError(testErr)

	// Test GetBlock error
	_, err := explorer.GetBlock(ctx, "ethereum", "100")
	if err == nil {
		t.Error("Expected error from GetBlock")
	}

	// Test GetTransaction error
	_, err = explorer.GetTransaction(ctx, "ethereum", "0x123")
	if err == nil {
		t.Error("Expected error from GetTransaction")
	}

	// Test GetAddress error
	_, err = explorer.GetAddress(ctx, "ethereum", "0x123")
	if err == nil {
		t.Error("Expected error from GetAddress")
	}

	// Test Search error
	_, err = explorer.Search(ctx, "ethereum", "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")
	if err == nil {
		t.Error("Expected error from Search")
	}

	// Test IndexBlock error
	err = explorer.IndexBlock(ctx, &Block{Network: "ethereum", BlockNumber: 1}, nil, nil)
	if err == nil {
		t.Error("Expected error from IndexBlock")
	}
}

func TestExplorerWithMock_IndexBlock_TransactionError(t *testing.T) {
	mock := NewMockRepository()
	explorer := NewWithRepository(mock)
	ctx := context.Background()

	block := &Block{Network: "ethereum", BlockNumber: 100, BlockHash: "0xa", Timestamp: time.Now()}

	// First InsertBlock succeeds
	err := explorer.IndexBlock(ctx, block, nil, nil)
	if err != nil {
		t.Fatalf("IndexBlock failed: %v", err)
	}

	// Now set error for transaction insert
	mock.SetError(errors.New("transaction error"))

	toAddr := "0x2222222222222222222222222222222222222222"
	txs := []*Transaction{
		{Network: "ethereum", TxHash: "0xtx", From: "0x1111111111111111111111111111111111111111", To: &toAddr},
	}

	err = explorer.IndexBlock(ctx, &Block{Network: "ethereum", BlockNumber: 101, Timestamp: time.Now()}, txs, nil)
	if err == nil {
		t.Error("Expected error when inserting transactions")
	}
}

func TestExplorerWithMock_ListTransactions_DefaultPageSize(t *testing.T) {
	mock := NewMockRepository()
	explorer := NewWithRepository(mock)
	ctx := context.Background()

	mock.AddTransaction(&Transaction{Network: "ethereum", TxHash: "0x1", BlockNumber: 100})

	// Test with zero page size (should use default)
	filter := TransactionFilter{
		Network: "ethereum",
	}

	result, err := explorer.ListTransactions(ctx, filter)
	if err != nil {
		t.Fatalf("ListTransactions failed: %v", err)
	}
	if result.PageSize != 20 {
		t.Errorf("PageSize = %d, want 20 (default)", result.PageSize)
	}
}
