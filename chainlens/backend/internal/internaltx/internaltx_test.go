package internaltx

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
	internalTxs      map[string]map[string][]*InternalTransaction // network -> txHash -> []InternalTransaction
	processingStatus map[string]map[string]*TraceProcessingStatus // network -> txHash -> status
	createdContracts map[string][]*InternalTransaction           // network -> []InternalTransaction

	// Error simulation
	simulateError bool
	errorToReturn error
}

// NewMockRepository creates a new mock repository
func NewMockRepository() *MockRepository {
	return &MockRepository{
		internalTxs:      make(map[string]map[string][]*InternalTransaction),
		processingStatus: make(map[string]map[string]*TraceProcessingStatus),
		createdContracts: make(map[string][]*InternalTransaction),
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

// AddInternalTransaction adds an internal transaction to the mock
func (m *MockRepository) AddInternalTransaction(tx *InternalTransaction) {
	if m.internalTxs[tx.Network] == nil {
		m.internalTxs[tx.Network] = make(map[string][]*InternalTransaction)
	}
	m.internalTxs[tx.Network][tx.TxHash] = append(m.internalTxs[tx.Network][tx.TxHash], tx)

	// Track created contracts
	if tx.CreatedContract != nil {
		m.createdContracts[tx.Network] = append(m.createdContracts[tx.Network], tx)
	}
}

// SetProcessingStatus sets the processing status for a transaction
func (m *MockRepository) SetProcessingStatus(status *TraceProcessingStatus) {
	if m.processingStatus[status.Network] == nil {
		m.processingStatus[status.Network] = make(map[string]*TraceProcessingStatus)
	}
	m.processingStatus[status.Network][status.TxHash] = status
}

// ============================================================================
// REPOSITORY INTERFACE IMPLEMENTATION
// ============================================================================

func (m *MockRepository) InsertInternalTransaction(ctx context.Context, tx *InternalTransaction) error {
	if m.simulateError {
		return m.errorToReturn
	}
	m.AddInternalTransaction(tx)
	return nil
}

func (m *MockRepository) InsertInternalTransactionsBatch(ctx context.Context, txs []*InternalTransaction) error {
	if m.simulateError {
		return m.errorToReturn
	}
	for _, tx := range txs {
		m.AddInternalTransaction(tx)
	}
	return nil
}

func (m *MockRepository) GetByTxHash(ctx context.Context, network, txHash string) ([]*InternalTransaction, error) {
	if m.simulateError {
		return nil, m.errorToReturn
	}
	if netTxs, ok := m.internalTxs[network]; ok {
		if txs, ok := netTxs[txHash]; ok {
			return txs, nil
		}
	}
	return nil, nil
}

func (m *MockRepository) GetByAddress(ctx context.Context, filter *InternalTxFilter) ([]*InternalTransaction, error) {
	if m.simulateError {
		return nil, m.errorToReturn
	}
	var result []*InternalTransaction
	if netTxs, ok := m.internalTxs[filter.Network]; ok {
		for _, txs := range netTxs {
			for _, tx := range txs {
				// Check address filters
				matchFrom := filter.FromAddress == "" || tx.FromAddress == filter.FromAddress
				matchTo := filter.ToAddress == "" || (tx.ToAddress != nil && *tx.ToAddress == filter.ToAddress)

				// If both addresses are set and equal, match either from or to
				if filter.FromAddress != "" && filter.ToAddress != "" && filter.FromAddress == filter.ToAddress {
					if tx.FromAddress == filter.FromAddress || (tx.ToAddress != nil && *tx.ToAddress == filter.ToAddress) {
						result = append(result, tx)
					}
				} else if matchFrom || matchTo {
					result = append(result, tx)
				}
			}
		}
	}

	// Apply pagination
	pageSize := filter.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	page := filter.Page
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * pageSize

	if offset >= len(result) {
		return nil, nil
	}
	end := offset + pageSize
	if end > len(result) {
		end = len(result)
	}

	return result[offset:end], nil
}

func (m *MockRepository) GetCreatedContracts(ctx context.Context, network string, limit, offset int) ([]*InternalTransaction, error) {
	if m.simulateError {
		return nil, m.errorToReturn
	}
	contracts := m.createdContracts[network]
	if offset >= len(contracts) {
		return nil, nil
	}
	end := offset + limit
	if end > len(contracts) {
		end = len(contracts)
	}
	return contracts[offset:end], nil
}

func (m *MockRepository) GetCallStats(ctx context.Context, network, txHash string) (*CallStats, error) {
	if m.simulateError {
		return nil, m.errorToReturn
	}

	if netTxs, ok := m.internalTxs[network]; ok {
		if txs, ok := netTxs[txHash]; ok {
			stats := &CallStats{
				TotalCalls:      len(txs),
				CallsByType:     make(map[string]int),
				TotalValueMoved: "0",
			}

			addressSet := make(map[string]bool)
			for _, tx := range txs {
				if tx.Depth > stats.MaxDepth {
					stats.MaxDepth = tx.Depth
				}
				stats.CallsByType[tx.TraceType]++
				addressSet[tx.FromAddress] = true
				if tx.ToAddress != nil {
					addressSet[*tx.ToAddress] = true
				}
				if tx.CreatedContract != nil {
					stats.ContractsCreated++
				}
				if tx.Reverted || tx.Error != nil {
					stats.FailedCalls++
				}
				if tx.GasUsed != nil {
					stats.TotalGasUsed += *tx.GasUsed
				}
			}
			stats.UniqueAddresses = len(addressSet)
			return stats, nil
		}
	}
	return nil, nil
}

func (m *MockRepository) DeleteByTxHash(ctx context.Context, network, txHash string) error {
	if m.simulateError {
		return m.errorToReturn
	}
	if netTxs, ok := m.internalTxs[network]; ok {
		delete(netTxs, txHash)
	}
	return nil
}

func (m *MockRepository) GetOrCreateProcessingStatus(ctx context.Context, network, txHash string, blockNumber int64) (*TraceProcessingStatus, error) {
	if m.simulateError {
		return nil, m.errorToReturn
	}
	if m.processingStatus[network] == nil {
		m.processingStatus[network] = make(map[string]*TraceProcessingStatus)
	}
	if status, ok := m.processingStatus[network][txHash]; ok {
		return status, nil
	}
	status := &TraceProcessingStatus{
		Network:     network,
		TxHash:      txHash,
		BlockNumber: blockNumber,
		Status:      StatusPending,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	m.processingStatus[network][txHash] = status
	return status, nil
}

func (m *MockRepository) UpdateProcessingStatus(ctx context.Context, network, txHash, status string, errorMsg *string) error {
	if m.simulateError {
		return m.errorToReturn
	}
	if m.processingStatus[network] != nil {
		if s, ok := m.processingStatus[network][txHash]; ok {
			s.Status = status
			s.ErrorMessage = errorMsg
			now := time.Now().UTC()
			s.LastAttemptAt = &now
			s.UpdatedAt = now
			if status == StatusFailed {
				s.RetryCount++
			}
		}
	}
	return nil
}

func (m *MockRepository) GetPendingTraces(ctx context.Context, network string, limit int, maxRetries int) ([]*PendingTraceJob, error) {
	if m.simulateError {
		return nil, m.errorToReturn
	}
	var jobs []*PendingTraceJob
	if netStatus, ok := m.processingStatus[network]; ok {
		for txHash, status := range netStatus {
			if (status.Status == StatusPending || status.Status == StatusFailed) && status.RetryCount < maxRetries {
				jobs = append(jobs, &PendingTraceJob{
					Network:     network,
					TxHash:      txHash,
					BlockNumber: status.BlockNumber,
					RetryCount:  status.RetryCount,
				})
				if len(jobs) >= limit {
					break
				}
			}
		}
	}
	return jobs, nil
}

func (m *MockRepository) MarkAsProcessing(ctx context.Context, network string, txHashes []string) error {
	if m.simulateError {
		return m.errorToReturn
	}
	for _, txHash := range txHashes {
		if m.processingStatus[network] != nil {
			if status, ok := m.processingStatus[network][txHash]; ok {
				status.Status = StatusProcessing
				now := time.Now().UTC()
				status.LastAttemptAt = &now
			}
		}
	}
	return nil
}

func (m *MockRepository) GetProcessingStatus(ctx context.Context, network, txHash string) (*TraceProcessingStatus, error) {
	if m.simulateError {
		return nil, m.errorToReturn
	}
	if netStatus, ok := m.processingStatus[network]; ok {
		if status, ok := netStatus[txHash]; ok {
			return status, nil
		}
	}
	return nil, nil
}

func (m *MockRepository) CountByStatus(ctx context.Context, network string) (map[string]int64, error) {
	if m.simulateError {
		return nil, m.errorToReturn
	}
	counts := make(map[string]int64)
	if netStatus, ok := m.processingStatus[network]; ok {
		for _, status := range netStatus {
			counts[status.Status]++
		}
	}
	return counts, nil
}

// Compile-time check that MockRepository implements RepositoryInterface
var _ RepositoryInterface = (*MockRepository)(nil)

func TestTraceTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"CALL", TraceTypeCall, "CALL"},
		{"STATICCALL", TraceTypeStaticCall, "STATICCALL"},
		{"DELEGATECALL", TraceTypeDelegateCall, "DELEGATECALL"},
		{"CREATE", TraceTypeCreate, "CREATE"},
		{"CREATE2", TraceTypeCreate2, "CREATE2"},
		{"SUICIDE", TraceTypeSuicide, "SUICIDE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestProcessingStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"pending", StatusPending, "pending"},
		{"processing", StatusProcessing, "processing"},
		{"completed", StatusCompleted, "completed"},
		{"failed", StatusFailed, "failed"},
		{"skipped", StatusSkipped, "skipped"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestValidTraceTypes(t *testing.T) {
	types := ValidTraceTypes()
	expected := []string{"CALL", "STATICCALL", "DELEGATECALL", "CREATE", "CREATE2", "SUICIDE"}

	if len(types) != len(expected) {
		t.Errorf("Expected %d trace types, got %d", len(expected), len(types))
	}

	for i, typ := range expected {
		if types[i] != typ {
			t.Errorf("Expected type[%d] = %s, got %s", i, typ, types[i])
		}
	}
}

func TestIsValidTraceType(t *testing.T) {
	tests := []struct {
		traceType string
		valid     bool
	}{
		{"CALL", true},
		{"STATICCALL", true},
		{"DELEGATECALL", true},
		{"CREATE", true},
		{"CREATE2", true},
		{"SUICIDE", true},
		{"INVALID", false},
		{"call", false}, // case sensitive
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.traceType, func(t *testing.T) {
			result := IsValidTraceType(tt.traceType)
			if result != tt.valid {
				t.Errorf("IsValidTraceType(%s) = %v, expected %v", tt.traceType, result, tt.valid)
			}
		})
	}
}

func TestIsContractCreation(t *testing.T) {
	tests := []struct {
		traceType  string
		isCreation bool
	}{
		{"CREATE", true},
		{"CREATE2", true},
		{"CALL", false},
		{"STATICCALL", false},
		{"DELEGATECALL", false},
		{"SUICIDE", false},
	}

	for _, tt := range tests {
		t.Run(tt.traceType, func(t *testing.T) {
			result := IsContractCreation(tt.traceType)
			if result != tt.isCreation {
				t.Errorf("IsContractCreation(%s) = %v, expected %v", tt.traceType, result, tt.isCreation)
			}
		})
	}
}

func TestNormalizeTraceType(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"call", "CALL"},
		{"CALL", "CALL"},
		{"staticcall", "STATICCALL"},
		{"STATICCALL", "STATICCALL"},
		{"delegatecall", "DELEGATECALL"},
		{"DELEGATECALL", "DELEGATECALL"},
		{"create", "CREATE"},
		{"CREATE", "CREATE"},
		{"create2", "CREATE2"},
		{"CREATE2", "CREATE2"},
		{"suicide", "SUICIDE"},
		{"SUICIDE", "SUICIDE"},
		{"selfdestruct", "SUICIDE"},
		{"SELFDESTRUCT", "SUICIDE"},
		{"unknown", "unknown"}, // unknown types are passed through
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := NormalizeTraceType(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeTraceType(%s) = %s, expected %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestInternalTransactionFields(t *testing.T) {
	parentIdx := 0
	toAddr := "0x1234567890123456789012345678901234567890"
	gas := int64(21000)
	gasUsed := int64(21000)
	input := "0x"
	output := "0x"
	errMsg := "out of gas"
	created := "0xabcdef1234567890abcdef1234567890abcdef12"

	tx := InternalTransaction{
		ID:               1,
		Network:          "ethereum",
		TxHash:           "0xabc123",
		TraceIndex:       0,
		BlockNumber:      18000000,
		ParentTraceIndex: &parentIdx,
		Depth:            1,
		TraceType:        TraceTypeCall,
		FromAddress:      "0x0000000000000000000000000000000000000001",
		ToAddress:        &toAddr,
		Value:            "1000000000000000000",
		Gas:              &gas,
		GasUsed:          &gasUsed,
		InputData:        &input,
		OutputData:       &output,
		Error:            &errMsg,
		Reverted:         true,
		CreatedContract:  &created,
		Timestamp:        time.Now().UTC(),
	}

	if tx.Network != "ethereum" {
		t.Errorf("Expected network 'ethereum', got %s", tx.Network)
	}

	if tx.TraceType != TraceTypeCall {
		t.Errorf("Expected trace type 'CALL', got %s", tx.TraceType)
	}

	if *tx.ParentTraceIndex != 0 {
		t.Errorf("Expected parent trace index 0, got %d", *tx.ParentTraceIndex)
	}

	if !tx.Reverted {
		t.Error("Expected reverted to be true")
	}
}

func TestInternalTxFilter(t *testing.T) {
	filter := InternalTxFilter{
		Network:     "ethereum",
		TxHash:      "0xabc123",
		FromAddress: "0x1111111111111111111111111111111111111111",
		ToAddress:   "0x2222222222222222222222222222222222222222",
		TraceType:   TraceTypeCall,
		MinValue:    "1000000000000000000",
		BlockFrom:   17000000,
		BlockTo:     18000000,
		Page:        1,
		PageSize:    20,
	}

	if filter.Network != "ethereum" {
		t.Errorf("Expected network 'ethereum', got %s", filter.Network)
	}

	if filter.Page != 1 {
		t.Errorf("Expected page 1, got %d", filter.Page)
	}

	if filter.PageSize != 20 {
		t.Errorf("Expected page size 20, got %d", filter.PageSize)
	}
}

func TestCallStats(t *testing.T) {
	stats := CallStats{
		TotalCalls:      100,
		MaxDepth:        5,
		TotalValueMoved: "10000000000000000000",
		TotalGasUsed:    5000000,
		CallsByType: map[string]int{
			"CALL":         80,
			"STATICCALL":   15,
			"DELEGATECALL": 5,
		},
		UniqueAddresses:  50,
		ContractsCreated: 2,
		FailedCalls:      3,
	}

	if stats.TotalCalls != 100 {
		t.Errorf("Expected 100 total calls, got %d", stats.TotalCalls)
	}

	if stats.MaxDepth != 5 {
		t.Errorf("Expected max depth 5, got %d", stats.MaxDepth)
	}

	if stats.CallsByType["CALL"] != 80 {
		t.Errorf("Expected 80 CALL operations, got %d", stats.CallsByType["CALL"])
	}

	if stats.FailedCalls != 3 {
		t.Errorf("Expected 3 failed calls, got %d", stats.FailedCalls)
	}
}

func TestTraceTree(t *testing.T) {
	toAddr := "0x1234567890123456789012345678901234567890"

	root := &TraceTree{
		Call: &InternalTransaction{
			TraceIndex:  0,
			TraceType:   TraceTypeCall,
			FromAddress: "0x0000000000000000000000000000000000000001",
			ToAddress:   &toAddr,
			Depth:       0,
		},
		Children: []*TraceTree{
			{
				Call: &InternalTransaction{
					TraceIndex:  1,
					TraceType:   TraceTypeStaticCall,
					FromAddress: toAddr,
					ToAddress:   &toAddr,
					Depth:       1,
				},
			},
		},
	}

	if root.Call == nil {
		t.Fatal("Root call should not be nil")
	}

	if len(root.Children) != 1 {
		t.Errorf("Expected 1 child, got %d", len(root.Children))
	}

	if root.Call.Depth != 0 {
		t.Errorf("Expected root depth 0, got %d", root.Call.Depth)
	}

	if root.Children[0].Call.Depth != 1 {
		t.Errorf("Expected child depth 1, got %d", root.Children[0].Call.Depth)
	}
}

func TestTraceProcessingStatus(t *testing.T) {
	now := time.Now().UTC()
	errMsg := "RPC timeout"

	status := TraceProcessingStatus{
		ID:            1,
		Network:       "ethereum",
		BlockNumber:   18000000,
		TxHash:        "0xabc123",
		Status:        StatusFailed,
		ErrorMessage:  &errMsg,
		RetryCount:    2,
		LastAttemptAt: &now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if status.Status != StatusFailed {
		t.Errorf("Expected status 'failed', got %s", status.Status)
	}

	if status.RetryCount != 2 {
		t.Errorf("Expected retry count 2, got %d", status.RetryCount)
	}

	if *status.ErrorMessage != errMsg {
		t.Errorf("Expected error message '%s', got %s", errMsg, *status.ErrorMessage)
	}
}

func TestPendingTraceJob(t *testing.T) {
	job := PendingTraceJob{
		Network:     "ethereum",
		TxHash:      "0xabc123",
		BlockNumber: 18000000,
		RetryCount:  0,
	}

	if job.Network != "ethereum" {
		t.Errorf("Expected network 'ethereum', got %s", job.Network)
	}

	if job.RetryCount != 0 {
		t.Errorf("Expected retry count 0, got %d", job.RetryCount)
	}
}

func TestInternalTxSummary(t *testing.T) {
	summary := InternalTxSummary{
		Network:            "ethereum",
		Address:            "0x1234567890123456789012345678901234567890",
		TotalInternalTxs:   1000,
		TotalValueReceived: "50000000000000000000",
		TotalValueSent:     "30000000000000000000",
		LastInternalTx:     time.Now().UTC(),
	}

	if summary.TotalInternalTxs != 1000 {
		t.Errorf("Expected 1000 total internal txs, got %d", summary.TotalInternalTxs)
	}

	if len(summary.Address) != 42 {
		t.Errorf("Expected address length 42, got %d", len(summary.Address))
	}
}

func TestBuildTraceTree(t *testing.T) {
	parentIdx := 0
	toAddr := "0x1234567890123456789012345678901234567890"

	txs := []*InternalTransaction{
		{
			TraceIndex:       0,
			ParentTraceIndex: nil,
			Depth:            0,
			TraceType:        TraceTypeCall,
			FromAddress:      "0x0000000000000000000000000000000000000001",
			ToAddress:        &toAddr,
		},
		{
			TraceIndex:       1,
			ParentTraceIndex: &parentIdx,
			Depth:            1,
			TraceType:        TraceTypeStaticCall,
			FromAddress:      toAddr,
			ToAddress:        &toAddr,
		},
		{
			TraceIndex:       2,
			ParentTraceIndex: &parentIdx,
			Depth:            1,
			TraceType:        TraceTypeDelegateCall,
			FromAddress:      toAddr,
			ToAddress:        &toAddr,
		},
	}

	tree := buildTraceTree(txs)

	if tree == nil {
		t.Fatal("Expected non-nil tree")
	}

	if tree.Call.TraceIndex != 0 {
		t.Errorf("Expected root trace index 0, got %d", tree.Call.TraceIndex)
	}

	if len(tree.Children) != 2 {
		t.Errorf("Expected 2 children, got %d", len(tree.Children))
	}

	if tree.Children[0].Call.TraceIndex != 1 {
		t.Errorf("Expected first child trace index 1, got %d", tree.Children[0].Call.TraceIndex)
	}

	if tree.Children[1].Call.TraceIndex != 2 {
		t.Errorf("Expected second child trace index 2, got %d", tree.Children[1].Call.TraceIndex)
	}
}

func TestBuildTraceTreeEmpty(t *testing.T) {
	tree := buildTraceTree([]*InternalTransaction{})

	if tree != nil {
		t.Error("Expected nil tree for empty input")
	}
}

func TestBuildTraceTreeNested(t *testing.T) {
	parentIdx0 := 0
	parentIdx1 := 1
	toAddr := "0x1234567890123456789012345678901234567890"

	txs := []*InternalTransaction{
		{
			TraceIndex:       0,
			ParentTraceIndex: nil,
			Depth:            0,
			TraceType:        TraceTypeCall,
			FromAddress:      "0x0000000000000000000000000000000000000001",
			ToAddress:        &toAddr,
		},
		{
			TraceIndex:       1,
			ParentTraceIndex: &parentIdx0,
			Depth:            1,
			TraceType:        TraceTypeCall,
			FromAddress:      toAddr,
			ToAddress:        &toAddr,
		},
		{
			TraceIndex:       2,
			ParentTraceIndex: &parentIdx1,
			Depth:            2,
			TraceType:        TraceTypeStaticCall,
			FromAddress:      toAddr,
			ToAddress:        &toAddr,
		},
	}

	tree := buildTraceTree(txs)

	if tree == nil {
		t.Fatal("Expected non-nil tree")
	}

	// Check nesting: root -> child -> grandchild
	if len(tree.Children) != 1 {
		t.Fatalf("Expected 1 child at root, got %d", len(tree.Children))
	}

	if len(tree.Children[0].Children) != 1 {
		t.Fatalf("Expected 1 grandchild, got %d", len(tree.Children[0].Children))
	}

	grandchild := tree.Children[0].Children[0]
	if grandchild.Call.TraceIndex != 2 {
		t.Errorf("Expected grandchild trace index 2, got %d", grandchild.Call.TraceIndex)
	}

	if grandchild.Call.Depth != 2 {
		t.Errorf("Expected grandchild depth 2, got %d", grandchild.Call.Depth)
	}
}

func TestServiceCreation(t *testing.T) {
	service := NewService(nil, nil)

	if service == nil {
		t.Fatal("Expected non-nil service")
	}

	if service.rpcURLs == nil {
		t.Error("Expected initialized rpcURLs map")
	}

	if service.stopCh == nil {
		t.Error("Expected initialized stopCh channel")
	}

	if service.maxRetries != 3 {
		t.Errorf("Expected maxRetries 3, got %d", service.maxRetries)
	}

	if service.batchSize != 10 {
		t.Errorf("Expected batchSize 10, got %d", service.batchSize)
	}
}

func TestServiceSetRPCURL(t *testing.T) {
	service := NewService(nil, nil)

	service.SetRPCURL("ethereum", "https://eth.example.com")
	service.SetRPCURL("polygon", "https://polygon.example.com")

	if len(service.rpcURLs) != 2 {
		t.Errorf("Expected 2 RPC URLs, got %d", len(service.rpcURLs))
	}

	if service.rpcURLs["ethereum"] != "https://eth.example.com" {
		t.Errorf("Unexpected ethereum RPC URL: %s", service.rpcURLs["ethereum"])
	}

	if service.rpcURLs["polygon"] != "https://polygon.example.com" {
		t.Errorf("Unexpected polygon RPC URL: %s", service.rpcURLs["polygon"])
	}
}

func TestServiceStopWithoutStart(t *testing.T) {
	service := NewService(nil, nil)
	// Stopping without starting should not panic
	service.Stop()
}

func TestInt64Ptr(t *testing.T) {
	val := int64(12345)
	ptr := int64Ptr(val)

	if ptr == nil {
		t.Fatal("Expected non-nil pointer")
	}

	if *ptr != val {
		t.Errorf("Expected %d, got %d", val, *ptr)
	}

	// Ensure it's a new pointer
	*ptr = 0
	if *ptr != 0 {
		t.Error("Pointer value should be modified")
	}
}

func TestBuildTraceTreeSingleNode(t *testing.T) {
	toAddr := "0x1234567890123456789012345678901234567890"

	txs := []*InternalTransaction{
		{
			TraceIndex:       0,
			ParentTraceIndex: nil,
			Depth:            0,
			TraceType:        TraceTypeCall,
			FromAddress:      "0x0000000000000000000000000000000000000001",
			ToAddress:        &toAddr,
		},
	}

	tree := buildTraceTree(txs)

	if tree == nil {
		t.Fatal("Expected non-nil tree")
	}

	if tree.Call.TraceIndex != 0 {
		t.Errorf("Expected trace index 0, got %d", tree.Call.TraceIndex)
	}

	if len(tree.Children) != 0 {
		t.Errorf("Expected 0 children, got %d", len(tree.Children))
	}
}

func TestBuildTraceTreeDeepNesting(t *testing.T) {
	toAddr := "0x1234567890123456789012345678901234567890"
	parentIdx0 := 0
	parentIdx1 := 1
	parentIdx2 := 2

	txs := []*InternalTransaction{
		{TraceIndex: 0, ParentTraceIndex: nil, Depth: 0, TraceType: TraceTypeCall, FromAddress: "0x1", ToAddress: &toAddr},
		{TraceIndex: 1, ParentTraceIndex: &parentIdx0, Depth: 1, TraceType: TraceTypeCall, FromAddress: "0x1", ToAddress: &toAddr},
		{TraceIndex: 2, ParentTraceIndex: &parentIdx1, Depth: 2, TraceType: TraceTypeCall, FromAddress: "0x1", ToAddress: &toAddr},
		{TraceIndex: 3, ParentTraceIndex: &parentIdx2, Depth: 3, TraceType: TraceTypeCall, FromAddress: "0x1", ToAddress: &toAddr},
	}

	tree := buildTraceTree(txs)

	if tree == nil {
		t.Fatal("Expected non-nil tree")
	}

	// Traverse down
	current := tree
	for i := 0; i < 4; i++ {
		if current == nil {
			t.Fatalf("Expected node at depth %d", i)
		}
		if current.Call.Depth != i {
			t.Errorf("Expected depth %d, got %d", i, current.Call.Depth)
		}
		if i < 3 {
			if len(current.Children) != 1 {
				t.Fatalf("Expected 1 child at depth %d, got %d", i, len(current.Children))
			}
			current = current.Children[0]
		}
	}
}

func TestInternalTxFilterDefaults(t *testing.T) {
	filter := InternalTxFilter{
		Network: "ethereum",
	}

	if filter.Network != "ethereum" {
		t.Errorf("Expected network 'ethereum', got %s", filter.Network)
	}

	// Default values should be zero
	if filter.Page != 0 {
		t.Errorf("Expected page 0, got %d", filter.Page)
	}

	if filter.PageSize != 0 {
		t.Errorf("Expected page size 0, got %d", filter.PageSize)
	}
}

func TestCallStatsZeroValues(t *testing.T) {
	stats := CallStats{}

	if stats.TotalCalls != 0 {
		t.Errorf("Expected 0 total calls, got %d", stats.TotalCalls)
	}

	if stats.MaxDepth != 0 {
		t.Errorf("Expected 0 max depth, got %d", stats.MaxDepth)
	}

	if stats.CallsByType != nil {
		t.Error("Expected nil CallsByType")
	}
}

func TestTraceProcessingStatusPending(t *testing.T) {
	now := time.Now().UTC()

	status := TraceProcessingStatus{
		Network:     "ethereum",
		TxHash:      "0xabc123",
		Status:      StatusPending,
		RetryCount:  0,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if status.Status != StatusPending {
		t.Errorf("Expected status 'pending', got %s", status.Status)
	}

	if status.RetryCount != 0 {
		t.Errorf("Expected 0 retries, got %d", status.RetryCount)
	}

	if status.ErrorMessage != nil {
		t.Error("Expected nil error message")
	}

	if status.LastAttemptAt != nil {
		t.Error("Expected nil last attempt")
	}
}

func TestNormalizeTraceTypeUnknown(t *testing.T) {
	// Unknown types should be passed through
	result := NormalizeTraceType("UNKNOWN_TYPE")
	if result != "UNKNOWN_TYPE" {
		t.Errorf("Expected 'UNKNOWN_TYPE', got %s", result)
	}
}

func TestIsContractCreationCases(t *testing.T) {
	// Test lowercase (after normalization)
	if IsContractCreation("create") {
		t.Error("IsContractCreation should be case sensitive (lowercase)")
	}

	// Test normalized
	if !IsContractCreation(NormalizeTraceType("create")) {
		t.Error("Expected CREATE to be contract creation after normalization")
	}
}

func TestInternalTransactionOptionalFields(t *testing.T) {
	// Test with all optional fields nil
	tx := InternalTransaction{
		Network:     "ethereum",
		TxHash:      "0xabc123",
		TraceIndex:  0,
		BlockNumber: 18000000,
		Depth:       0,
		TraceType:   TraceTypeCall,
		FromAddress: "0x0000000000000000000000000000000000000001",
		Value:       "0",
		Timestamp:   time.Now().UTC(),
	}

	if tx.ParentTraceIndex != nil {
		t.Error("Expected nil ParentTraceIndex")
	}
	if tx.ToAddress != nil {
		t.Error("Expected nil ToAddress")
	}
	if tx.Gas != nil {
		t.Error("Expected nil Gas")
	}
	if tx.GasUsed != nil {
		t.Error("Expected nil GasUsed")
	}
	if tx.InputData != nil {
		t.Error("Expected nil InputData")
	}
	if tx.OutputData != nil {
		t.Error("Expected nil OutputData")
	}
	if tx.Error != nil {
		t.Error("Expected nil Error")
	}
	if tx.CreatedContract != nil {
		t.Error("Expected nil CreatedContract")
	}
}

func TestTraceTreeEmptyChildren(t *testing.T) {
	toAddr := "0x1234567890123456789012345678901234567890"

	tree := &TraceTree{
		Call: &InternalTransaction{
			TraceIndex:  0,
			TraceType:   TraceTypeCall,
			FromAddress: "0x1",
			ToAddress:   &toAddr,
		},
		Children: []*TraceTree{},
	}

	if len(tree.Children) != 0 {
		t.Errorf("Expected 0 children, got %d", len(tree.Children))
	}
}

func TestValidTraceTypesCount(t *testing.T) {
	types := ValidTraceTypes()

	// Ensure we have all expected types
	expectedCount := 6
	if len(types) != expectedCount {
		t.Errorf("Expected %d trace types, got %d", expectedCount, len(types))
	}
}

func TestInternalTxSummaryZeroValues(t *testing.T) {
	summary := InternalTxSummary{
		Network: "ethereum",
		Address: "0x1234567890123456789012345678901234567890",
	}

	if summary.TotalInternalTxs != 0 {
		t.Errorf("Expected 0 total internal txs, got %d", summary.TotalInternalTxs)
	}

	if summary.TotalValueReceived != "" {
		t.Errorf("Expected empty TotalValueReceived, got %s", summary.TotalValueReceived)
	}

	if summary.TotalValueSent != "" {
		t.Errorf("Expected empty TotalValueSent, got %s", summary.TotalValueSent)
	}
}

func TestPendingTraceJobFields(t *testing.T) {
	job := PendingTraceJob{
		Network:     "polygon",
		TxHash:      "0xdef456",
		BlockNumber: 50000000,
		RetryCount:  5,
	}

	if job.Network != "polygon" {
		t.Errorf("Expected network 'polygon', got %s", job.Network)
	}

	if job.TxHash != "0xdef456" {
		t.Errorf("Expected txHash '0xdef456', got %s", job.TxHash)
	}

	if job.BlockNumber != 50000000 {
		t.Errorf("Expected block number 50000000, got %d", job.BlockNumber)
	}

	if job.RetryCount != 5 {
		t.Errorf("Expected retry count 5, got %d", job.RetryCount)
	}
}

func TestBuildTraceTreeMultipleRoots(t *testing.T) {
	// This tests a malformed case where there might be multiple root nodes
	// The function should handle this by returning the first root it finds
	toAddr := "0x1234567890123456789012345678901234567890"

	txs := []*InternalTransaction{
		{TraceIndex: 0, ParentTraceIndex: nil, Depth: 0, TraceType: TraceTypeCall, FromAddress: "0x1", ToAddress: &toAddr},
		{TraceIndex: 1, ParentTraceIndex: nil, Depth: 0, TraceType: TraceTypeCall, FromAddress: "0x2", ToAddress: &toAddr}, // Another root
	}

	tree := buildTraceTree(txs)

	// Should return the last root in iteration (implementation dependent)
	if tree == nil {
		t.Fatal("Expected non-nil tree even with multiple roots")
	}
}

// ============================================================================
// SERVICE TESTS WITH MOCK REPOSITORY
// ============================================================================

func TestServiceWithMock_GetInternalTransactions(t *testing.T) {
	mock := NewMockRepository()
	service := NewService(mock, nil)
	ctx := context.Background()

	txHash := "0xabc123"
	toAddr := "0x2222222222222222222222222222222222222222"

	// Add internal transactions
	mock.AddInternalTransaction(&InternalTransaction{
		Network:     "ethereum",
		TxHash:      txHash,
		TraceIndex:  0,
		BlockNumber: 18000000,
		TraceType:   TraceTypeCall,
		FromAddress: "0x1111111111111111111111111111111111111111",
		ToAddress:   &toAddr,
		Depth:       0,
	})
	mock.AddInternalTransaction(&InternalTransaction{
		Network:     "ethereum",
		TxHash:      txHash,
		TraceIndex:  1,
		BlockNumber: 18000000,
		TraceType:   TraceTypeStaticCall,
		FromAddress: toAddr,
		ToAddress:   &toAddr,
		Depth:       1,
	})

	result, err := service.GetInternalTransactions(ctx, "ethereum", txHash)
	if err != nil {
		t.Fatalf("GetInternalTransactions failed: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("Expected 2 transactions, got %d", len(result))
	}
	if result[0].TraceType != TraceTypeCall {
		t.Errorf("Expected CALL, got %s", result[0].TraceType)
	}
}

func TestServiceWithMock_GetInternalTransactionTree(t *testing.T) {
	mock := NewMockRepository()
	service := NewService(mock, nil)
	ctx := context.Background()

	txHash := "0xabc123"
	toAddr := "0x2222222222222222222222222222222222222222"
	parentIdx := 0

	mock.AddInternalTransaction(&InternalTransaction{
		Network:          "ethereum",
		TxHash:           txHash,
		TraceIndex:       0,
		ParentTraceIndex: nil,
		Depth:            0,
		TraceType:        TraceTypeCall,
		FromAddress:      "0x1111111111111111111111111111111111111111",
		ToAddress:        &toAddr,
	})
	mock.AddInternalTransaction(&InternalTransaction{
		Network:          "ethereum",
		TxHash:           txHash,
		TraceIndex:       1,
		ParentTraceIndex: &parentIdx,
		Depth:            1,
		TraceType:        TraceTypeStaticCall,
		FromAddress:      toAddr,
		ToAddress:        &toAddr,
	})

	tree, err := service.GetInternalTransactionTree(ctx, "ethereum", txHash)
	if err != nil {
		t.Fatalf("GetInternalTransactionTree failed: %v", err)
	}
	if tree == nil {
		t.Fatal("Expected non-nil tree")
	}
	if tree.Call.TraceIndex != 0 {
		t.Errorf("Expected root trace index 0, got %d", tree.Call.TraceIndex)
	}
	if len(tree.Children) != 1 {
		t.Errorf("Expected 1 child, got %d", len(tree.Children))
	}
}

func TestServiceWithMock_GetInternalTransactionTree_Empty(t *testing.T) {
	mock := NewMockRepository()
	service := NewService(mock, nil)
	ctx := context.Background()

	tree, err := service.GetInternalTransactionTree(ctx, "ethereum", "0xnonexistent")
	if err != nil {
		t.Fatalf("GetInternalTransactionTree failed: %v", err)
	}
	if tree != nil {
		t.Error("Expected nil tree for non-existent transaction")
	}
}

func TestServiceWithMock_GetAddressInternalTransactions(t *testing.T) {
	mock := NewMockRepository()
	service := NewService(mock, nil)
	ctx := context.Background()

	address := "0x1111111111111111111111111111111111111111"
	toAddr := "0x2222222222222222222222222222222222222222"

	mock.AddInternalTransaction(&InternalTransaction{
		Network:     "ethereum",
		TxHash:      "0xtx1",
		TraceIndex:  0,
		TraceType:   TraceTypeCall,
		FromAddress: address,
		ToAddress:   &toAddr,
	})
	mock.AddInternalTransaction(&InternalTransaction{
		Network:     "ethereum",
		TxHash:      "0xtx2",
		TraceIndex:  0,
		TraceType:   TraceTypeCall,
		FromAddress: "0xother",
		ToAddress:   &address,
	})
	mock.AddInternalTransaction(&InternalTransaction{
		Network:     "ethereum",
		TxHash:      "0xtx3",
		TraceIndex:  0,
		TraceType:   TraceTypeCall,
		FromAddress: "0xother",
		ToAddress:   &toAddr,
	})

	result, err := service.GetAddressInternalTransactions(ctx, "ethereum", address, 1, 20)
	if err != nil {
		t.Fatalf("GetAddressInternalTransactions failed: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("Expected 2 transactions, got %d", len(result))
	}
}

func TestServiceWithMock_GetCallStats(t *testing.T) {
	mock := NewMockRepository()
	service := NewService(mock, nil)
	ctx := context.Background()

	txHash := "0xabc123"
	toAddr := "0x2222222222222222222222222222222222222222"
	gas := int64(21000)
	created := "0xnewcontract"
	errMsg := "revert"

	mock.AddInternalTransaction(&InternalTransaction{
		Network:     "ethereum",
		TxHash:      txHash,
		TraceIndex:  0,
		TraceType:   TraceTypeCall,
		FromAddress: "0x1111111111111111111111111111111111111111",
		ToAddress:   &toAddr,
		Depth:       0,
		GasUsed:     &gas,
	})
	mock.AddInternalTransaction(&InternalTransaction{
		Network:     "ethereum",
		TxHash:      txHash,
		TraceIndex:  1,
		TraceType:   TraceTypeCreate,
		FromAddress: toAddr,
		Depth:       1,
		GasUsed:     &gas,
		CreatedContract: &created,
	})
	mock.AddInternalTransaction(&InternalTransaction{
		Network:     "ethereum",
		TxHash:      txHash,
		TraceIndex:  2,
		TraceType:   TraceTypeCall,
		FromAddress: toAddr,
		ToAddress:   &toAddr,
		Depth:       1,
		GasUsed:     &gas,
		Error:       &errMsg,
		Reverted:    true,
	})

	stats, err := service.GetCallStats(ctx, "ethereum", txHash)
	if err != nil {
		t.Fatalf("GetCallStats failed: %v", err)
	}
	if stats == nil {
		t.Fatal("Expected non-nil stats")
	}
	if stats.TotalCalls != 3 {
		t.Errorf("Expected 3 total calls, got %d", stats.TotalCalls)
	}
	if stats.MaxDepth != 1 {
		t.Errorf("Expected max depth 1, got %d", stats.MaxDepth)
	}
	if stats.ContractsCreated != 1 {
		t.Errorf("Expected 1 contract created, got %d", stats.ContractsCreated)
	}
	if stats.FailedCalls != 1 {
		t.Errorf("Expected 1 failed call, got %d", stats.FailedCalls)
	}
	if stats.TotalGasUsed != 63000 {
		t.Errorf("Expected total gas used 63000, got %d", stats.TotalGasUsed)
	}
}

func TestServiceWithMock_GetCreatedContracts(t *testing.T) {
	mock := NewMockRepository()
	service := NewService(mock, nil)
	ctx := context.Background()

	created1 := "0xcontract1"
	created2 := "0xcontract2"

	mock.AddInternalTransaction(&InternalTransaction{
		Network:         "ethereum",
		TxHash:          "0xtx1",
		TraceIndex:      0,
		TraceType:       TraceTypeCreate,
		FromAddress:     "0x1111111111111111111111111111111111111111",
		CreatedContract: &created1,
	})
	mock.AddInternalTransaction(&InternalTransaction{
		Network:         "ethereum",
		TxHash:          "0xtx2",
		TraceIndex:      0,
		TraceType:       TraceTypeCreate2,
		FromAddress:     "0x1111111111111111111111111111111111111111",
		CreatedContract: &created2,
	})
	mock.AddInternalTransaction(&InternalTransaction{
		Network:     "ethereum",
		TxHash:      "0xtx3",
		TraceIndex:  0,
		TraceType:   TraceTypeCall,
		FromAddress: "0x1111111111111111111111111111111111111111",
	})

	result, err := service.GetCreatedContracts(ctx, "ethereum", 10, 0)
	if err != nil {
		t.Fatalf("GetCreatedContracts failed: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("Expected 2 created contracts, got %d", len(result))
	}
}

func TestServiceWithMock_QueueForTracing(t *testing.T) {
	mock := NewMockRepository()
	service := NewService(mock, nil)
	ctx := context.Background()

	err := service.QueueForTracing(ctx, "ethereum", "0xabc123", 18000000)
	if err != nil {
		t.Fatalf("QueueForTracing failed: %v", err)
	}

	status, err := service.GetProcessingStatus(ctx, "ethereum", "0xabc123")
	if err != nil {
		t.Fatalf("GetProcessingStatus failed: %v", err)
	}
	if status == nil {
		t.Fatal("Expected non-nil status")
	}
	if status.Status != StatusPending {
		t.Errorf("Expected status 'pending', got %s", status.Status)
	}
}

func TestServiceWithMock_GetProcessingStats(t *testing.T) {
	mock := NewMockRepository()
	service := NewService(mock, nil)
	ctx := context.Background()

	mock.SetProcessingStatus(&TraceProcessingStatus{Network: "ethereum", TxHash: "0x1", Status: StatusPending})
	mock.SetProcessingStatus(&TraceProcessingStatus{Network: "ethereum", TxHash: "0x2", Status: StatusPending})
	mock.SetProcessingStatus(&TraceProcessingStatus{Network: "ethereum", TxHash: "0x3", Status: StatusCompleted})
	mock.SetProcessingStatus(&TraceProcessingStatus{Network: "ethereum", TxHash: "0x4", Status: StatusFailed})

	stats, err := service.GetProcessingStats(ctx, "ethereum")
	if err != nil {
		t.Fatalf("GetProcessingStats failed: %v", err)
	}
	if stats[StatusPending] != 2 {
		t.Errorf("Expected 2 pending, got %d", stats[StatusPending])
	}
	if stats[StatusCompleted] != 1 {
		t.Errorf("Expected 1 completed, got %d", stats[StatusCompleted])
	}
	if stats[StatusFailed] != 1 {
		t.Errorf("Expected 1 failed, got %d", stats[StatusFailed])
	}
}

func TestServiceWithMock_ErrorHandling(t *testing.T) {
	mock := NewMockRepository()
	service := NewService(mock, nil)
	ctx := context.Background()

	testErr := errors.New("database error")
	mock.SetError(testErr)

	// Test GetInternalTransactions error
	_, err := service.GetInternalTransactions(ctx, "ethereum", "0x123")
	if err == nil {
		t.Error("Expected error from GetInternalTransactions")
	}

	// Test GetInternalTransactionTree error
	_, err = service.GetInternalTransactionTree(ctx, "ethereum", "0x123")
	if err == nil {
		t.Error("Expected error from GetInternalTransactionTree")
	}

	// Test GetCallStats error
	_, err = service.GetCallStats(ctx, "ethereum", "0x123")
	if err == nil {
		t.Error("Expected error from GetCallStats")
	}

	// Test GetProcessingStatus error
	_, err = service.GetProcessingStatus(ctx, "ethereum", "0x123")
	if err == nil {
		t.Error("Expected error from GetProcessingStatus")
	}

	// Test QueueForTracing error
	err = service.QueueForTracing(ctx, "ethereum", "0x123", 100)
	if err == nil {
		t.Error("Expected error from QueueForTracing")
	}
}

func TestServiceWithMock_TraceTransactionOnDemand_NoRPCURL(t *testing.T) {
	mock := NewMockRepository()
	service := NewService(mock, nil)
	ctx := context.Background()

	_, err := service.TraceTransactionOnDemand(ctx, "ethereum", "0x123")
	if err == nil {
		t.Error("Expected error when no RPC URL configured")
	}
}

func TestServiceWithMock_TraceTransactionOnDemand_ExistingTrace(t *testing.T) {
	mock := NewMockRepository()
	service := NewService(mock, nil)
	service.SetRPCURL("ethereum", "https://eth.example.com")
	ctx := context.Background()

	txHash := "0xabc123"
	toAddr := "0x2222222222222222222222222222222222222222"

	// Add existing trace data
	mock.AddInternalTransaction(&InternalTransaction{
		Network:     "ethereum",
		TxHash:      txHash,
		TraceIndex:  0,
		TraceType:   TraceTypeCall,
		FromAddress: "0x1111111111111111111111111111111111111111",
		ToAddress:   &toAddr,
	})

	result, err := service.TraceTransactionOnDemand(ctx, "ethereum", txHash)
	if err != nil {
		t.Fatalf("TraceTransactionOnDemand failed: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("Expected 1 transaction, got %d", len(result))
	}
}

func TestMockRepository_InsertAndGetByAddress(t *testing.T) {
	mock := NewMockRepository()
	ctx := context.Background()

	address := "0x1111111111111111111111111111111111111111"
	toAddr := "0x2222222222222222222222222222222222222222"

	// Insert transactions
	err := mock.InsertInternalTransaction(ctx, &InternalTransaction{
		Network:     "ethereum",
		TxHash:      "0xtx1",
		TraceIndex:  0,
		TraceType:   TraceTypeCall,
		FromAddress: address,
		ToAddress:   &toAddr,
	})
	if err != nil {
		t.Fatalf("InsertInternalTransaction failed: %v", err)
	}

	// Get by address with same from and to
	filter := &InternalTxFilter{
		Network:     "ethereum",
		FromAddress: address,
		ToAddress:   address,
		Page:        1,
		PageSize:    20,
	}
	result, err := mock.GetByAddress(ctx, filter)
	if err != nil {
		t.Fatalf("GetByAddress failed: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("Expected 1 transaction, got %d", len(result))
	}
}

func TestMockRepository_ProcessingStatusWorkflow(t *testing.T) {
	mock := NewMockRepository()
	ctx := context.Background()

	// Create status
	status, err := mock.GetOrCreateProcessingStatus(ctx, "ethereum", "0xabc", 100)
	if err != nil {
		t.Fatalf("GetOrCreateProcessingStatus failed: %v", err)
	}
	if status.Status != StatusPending {
		t.Errorf("Expected pending status, got %s", status.Status)
	}

	// Get pending traces
	jobs, err := mock.GetPendingTraces(ctx, "ethereum", 10, 3)
	if err != nil {
		t.Fatalf("GetPendingTraces failed: %v", err)
	}
	if len(jobs) != 1 {
		t.Errorf("Expected 1 job, got %d", len(jobs))
	}

	// Mark as processing
	err = mock.MarkAsProcessing(ctx, "ethereum", []string{"0xabc"})
	if err != nil {
		t.Fatalf("MarkAsProcessing failed: %v", err)
	}

	status, _ = mock.GetProcessingStatus(ctx, "ethereum", "0xabc")
	if status.Status != StatusProcessing {
		t.Errorf("Expected processing status, got %s", status.Status)
	}

	// Update to completed
	err = mock.UpdateProcessingStatus(ctx, "ethereum", "0xabc", StatusCompleted, nil)
	if err != nil {
		t.Fatalf("UpdateProcessingStatus failed: %v", err)
	}

	status, _ = mock.GetProcessingStatus(ctx, "ethereum", "0xabc")
	if status.Status != StatusCompleted {
		t.Errorf("Expected completed status, got %s", status.Status)
	}
}

func TestMockRepository_DeleteByTxHash(t *testing.T) {
	mock := NewMockRepository()
	ctx := context.Background()

	txHash := "0xabc123"
	toAddr := "0x2222222222222222222222222222222222222222"

	mock.AddInternalTransaction(&InternalTransaction{
		Network:     "ethereum",
		TxHash:      txHash,
		TraceIndex:  0,
		TraceType:   TraceTypeCall,
		FromAddress: "0x1111111111111111111111111111111111111111",
		ToAddress:   &toAddr,
	})

	// Verify it exists
	txs, _ := mock.GetByTxHash(ctx, "ethereum", txHash)
	if len(txs) != 1 {
		t.Fatal("Expected transaction to exist before delete")
	}

	// Delete
	err := mock.DeleteByTxHash(ctx, "ethereum", txHash)
	if err != nil {
		t.Fatalf("DeleteByTxHash failed: %v", err)
	}

	// Verify it's gone
	txs, _ = mock.GetByTxHash(ctx, "ethereum", txHash)
	if len(txs) != 0 {
		t.Error("Expected transaction to be deleted")
	}
}

func TestMockRepository_InsertBatch(t *testing.T) {
	mock := NewMockRepository()
	ctx := context.Background()

	txHash := "0xabc123"
	toAddr := "0x2222222222222222222222222222222222222222"

	txs := []*InternalTransaction{
		{Network: "ethereum", TxHash: txHash, TraceIndex: 0, TraceType: TraceTypeCall, FromAddress: "0x1", ToAddress: &toAddr},
		{Network: "ethereum", TxHash: txHash, TraceIndex: 1, TraceType: TraceTypeStaticCall, FromAddress: "0x2", ToAddress: &toAddr},
		{Network: "ethereum", TxHash: txHash, TraceIndex: 2, TraceType: TraceTypeDelegateCall, FromAddress: "0x3", ToAddress: &toAddr},
	}

	err := mock.InsertInternalTransactionsBatch(ctx, txs)
	if err != nil {
		t.Fatalf("InsertInternalTransactionsBatch failed: %v", err)
	}

	result, _ := mock.GetByTxHash(ctx, "ethereum", txHash)
	if len(result) != 3 {
		t.Errorf("Expected 3 transactions, got %d", len(result))
	}
}
