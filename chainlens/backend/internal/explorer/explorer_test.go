package explorer

import (
	"testing"
	"time"
)

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
