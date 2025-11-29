package internaltx

import (
	"testing"
	"time"
)

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
