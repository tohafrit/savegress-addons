package analytics

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestWeiToGwei(t *testing.T) {
	tests := []struct {
		name     string
		wei      int64
		expected float64
	}{
		{
			name:     "1 gwei",
			wei:      1_000_000_000,
			expected: 1.0,
		},
		{
			name:     "0.5 gwei",
			wei:      500_000_000,
			expected: 0.5,
		},
		{
			name:     "100 gwei",
			wei:      100_000_000_000,
			expected: 100.0,
		},
		{
			name:     "zero",
			wei:      0,
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WeiToGwei(tt.wei)
			if result != tt.expected {
				t.Errorf("WeiToGwei(%d) = %f, expected %f", tt.wei, result, tt.expected)
			}
		})
	}
}

func TestGweiToWei(t *testing.T) {
	tests := []struct {
		name     string
		gwei     float64
		expected int64
	}{
		{
			name:     "1 gwei",
			gwei:     1.0,
			expected: 1_000_000_000,
		},
		{
			name:     "0.5 gwei",
			gwei:     0.5,
			expected: 500_000_000,
		},
		{
			name:     "100 gwei",
			gwei:     100.0,
			expected: 100_000_000_000,
		},
		{
			name:     "zero",
			gwei:     0.0,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GweiToWei(tt.gwei)
			if result != tt.expected {
				t.Errorf("GweiToWei(%f) = %d, expected %d", tt.gwei, result, tt.expected)
			}
		})
	}
}

func TestHexToInt64(t *testing.T) {
	tests := []struct {
		name     string
		hex      string
		expected int64
	}{
		{
			name:     "simple hex",
			hex:      "0x10",
			expected: 16,
		},
		{
			name:     "larger hex",
			hex:      "0xff",
			expected: 255,
		},
		{
			name:     "uppercase hex",
			hex:      "0xFF",
			expected: 255,
		},
		{
			name:     "gas price example",
			hex:      "0x3b9aca00",
			expected: 1_000_000_000, // 1 gwei
		},
		{
			name:     "block number",
			hex:      "0x1234567",
			expected: 19088743,
		},
		{
			name:     "empty after prefix",
			hex:      "0x",
			expected: 0,
		},
		{
			name:     "no prefix",
			hex:      "ff",
			expected: 0,
		},
		{
			name:     "zero",
			hex:      "0x0",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hexToInt64(tt.hex)
			if result != tt.expected {
				t.Errorf("hexToInt64(%s) = %d, expected %d", tt.hex, result, tt.expected)
			}
		})
	}
}

func TestSupportedNetworks(t *testing.T) {
	expected := []string{
		"ethereum",
		"polygon",
		"arbitrum",
		"optimism",
		"base",
		"bsc",
		"avalanche",
	}

	if len(SupportedNetworks) != len(expected) {
		t.Errorf("Expected %d networks, got %d", len(expected), len(SupportedNetworks))
	}

	for i, network := range expected {
		if SupportedNetworks[i] != network {
			t.Errorf("Expected network[%d] = %s, got %s", i, network, SupportedNetworks[i])
		}
	}
}

func TestNetworkChainIDs(t *testing.T) {
	tests := []struct {
		network  string
		expected int
	}{
		{"ethereum", 1},
		{"polygon", 137},
		{"arbitrum", 42161},
		{"optimism", 10},
		{"base", 8453},
		{"bsc", 56},
		{"avalanche", 43114},
	}

	for _, tt := range tests {
		t.Run(tt.network, func(t *testing.T) {
			chainID, ok := NetworkChainIDs[tt.network]
			if !ok {
				t.Errorf("Network %s not found in NetworkChainIDs", tt.network)
				return
			}
			if chainID != tt.expected {
				t.Errorf("ChainID for %s = %d, expected %d", tt.network, chainID, tt.expected)
			}
		})
	}
}

func TestNetworkNativeCurrencies(t *testing.T) {
	tests := []struct {
		network  string
		expected string
	}{
		{"ethereum", "ETH"},
		{"polygon", "MATIC"},
		{"arbitrum", "ETH"},
		{"optimism", "ETH"},
		{"base", "ETH"},
		{"bsc", "BNB"},
		{"avalanche", "AVAX"},
	}

	for _, tt := range tests {
		t.Run(tt.network, func(t *testing.T) {
			currency, ok := NetworkNativeCurrencies[tt.network]
			if !ok {
				t.Errorf("Network %s not found in NetworkNativeCurrencies", tt.network)
				return
			}
			if currency != tt.expected {
				t.Errorf("Currency for %s = %s, expected %s", tt.network, currency, tt.expected)
			}
		})
	}
}

func TestDailyStatsFields(t *testing.T) {
	stats := DailyStats{
		ID:               1,
		Network:          "ethereum",
		Date:             time.Now().UTC().Truncate(24 * time.Hour),
		BlockCount:       7200,
		TransactionCount: 1000000,
		UniqueSenders:    50000,
		UniqueReceivers:  45000,
	}

	if stats.Network != "ethereum" {
		t.Errorf("Expected network 'ethereum', got %s", stats.Network)
	}

	if stats.BlockCount != 7200 {
		t.Errorf("Expected block count 7200, got %d", stats.BlockCount)
	}

	if stats.TransactionCount != 1000000 {
		t.Errorf("Expected transaction count 1000000, got %d", stats.TransactionCount)
	}
}

func TestGasPriceEstimateFields(t *testing.T) {
	estimate := GasPriceEstimate{
		Network:      "ethereum",
		Timestamp:    time.Now().UTC(),
		SlowGwei:     20.0,
		StandardGwei: 30.0,
		FastGwei:     50.0,
		InstantGwei:  75.0,
		BaseFeeGwei:  25.0,
		SlowTime:     "~10 min",
		StandardTime: "~3 min",
		FastTime:     "~1 min",
		InstantTime:  "~15 sec",
	}

	if estimate.SlowGwei >= estimate.StandardGwei {
		t.Error("Slow gas price should be less than standard")
	}

	if estimate.StandardGwei >= estimate.FastGwei {
		t.Error("Standard gas price should be less than fast")
	}

	if estimate.FastGwei >= estimate.InstantGwei {
		t.Error("Fast gas price should be less than instant")
	}
}

func TestChartDataPoint(t *testing.T) {
	now := time.Now().UTC()
	point := ChartDataPoint{
		Timestamp: now,
		Value:     12345.67,
		Label:     "Transaction count",
	}

	if !point.Timestamp.Equal(now) {
		t.Error("Timestamp mismatch")
	}

	if point.Value != 12345.67 {
		t.Errorf("Expected value 12345.67, got %f", point.Value)
	}

	if point.Label != "Transaction count" {
		t.Errorf("Expected label 'Transaction count', got %s", point.Label)
	}
}

func TestNetworkOverviewFields(t *testing.T) {
	latestBlock := int64(18000000)
	latestTime := time.Now().UTC()
	slow := int64(20_000_000_000)
	standard := int64(30_000_000_000)
	fast := int64(50_000_000_000)

	overview := NetworkOverview{
		Network:           "ethereum",
		LatestBlock:       &latestBlock,
		LatestBlockTime:   &latestTime,
		TotalBlocks:       18000000,
		TotalTransactions: 2000000000,
		TotalAddresses:    300000000,
		TxCount24h:        1200000,
		GasPriceSlow:      &slow,
		GasPriceStandard:  &standard,
		GasPriceFast:      &fast,
		NativeCurrency:    "ETH",
	}

	if overview.Network != "ethereum" {
		t.Errorf("Expected network 'ethereum', got %s", overview.Network)
	}

	if *overview.LatestBlock != latestBlock {
		t.Errorf("Expected latest block %d, got %d", latestBlock, *overview.LatestBlock)
	}

	if *overview.GasPriceSlow != slow {
		t.Errorf("Expected slow gas price %d, got %d", slow, *overview.GasPriceSlow)
	}
}

func TestRankingTypeConstants(t *testing.T) {
	if RankingTypeBalance != "balance" {
		t.Errorf("Expected 'balance', got %s", RankingTypeBalance)
	}

	if RankingTypeTxCount != "tx_count" {
		t.Errorf("Expected 'tx_count', got %s", RankingTypeTxCount)
	}

	if RankingTypeGasSpent != "gas_spent" {
		t.Errorf("Expected 'gas_spent', got %s", RankingTypeGasSpent)
	}
}

func TestStatsFilter(t *testing.T) {
	filter := StatsFilter{
		Network:   "ethereum",
		StartDate: time.Now().AddDate(0, 0, -30),
		EndDate:   time.Now(),
		Interval:  "day",
		Limit:     100,
	}

	if filter.Network != "ethereum" {
		t.Errorf("Expected network 'ethereum', got %s", filter.Network)
	}

	if filter.Interval != "day" {
		t.Errorf("Expected interval 'day', got %s", filter.Interval)
	}

	if filter.Limit != 100 {
		t.Errorf("Expected limit 100, got %d", filter.Limit)
	}

	if filter.EndDate.Before(filter.StartDate) {
		t.Error("End date should be after start date")
	}
}

func TestTopTokenFields(t *testing.T) {
	token := TopToken{
		Network:       "ethereum",
		Date:          time.Now().UTC().Truncate(24 * time.Hour),
		Rank:          1,
		TokenAddress:  "0xdAC17F958D2ee523a2206206994597C13D831ec7",
		TransferCount: 1000000,
		UniqueHolders: 50000,
		Volume:        "1000000000000",
	}

	if len(token.TokenAddress) != 42 {
		t.Errorf("Expected address length 42, got %d", len(token.TokenAddress))
	}

	if token.Rank != 1 {
		t.Errorf("Expected rank 1, got %d", token.Rank)
	}
}

func TestTopContractFields(t *testing.T) {
	contract := TopContract{
		Network:         "ethereum",
		Date:            time.Now().UTC().Truncate(24 * time.Hour),
		Rank:            1,
		ContractAddress: "0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D",
		CallCount:       500000,
		UniqueCallers:   100000,
		GasUsed:         "50000000000000",
	}

	if len(contract.ContractAddress) != 42 {
		t.Errorf("Expected address length 42, got %d", len(contract.ContractAddress))
	}

	if contract.Rank != 1 {
		t.Errorf("Expected rank 1, got %d", contract.Rank)
	}
}

func TestInt64Ptr(t *testing.T) {
	val := int64(12345)
	ptr := int64Ptr(val)

	if *ptr != val {
		t.Errorf("Expected %d, got %d", val, *ptr)
	}

	// Ensure it's a new pointer
	*ptr = 0
	if *ptr != 0 {
		t.Error("Pointer value should be modified")
	}
}

func TestServiceCreation(t *testing.T) {
	service := NewService(nil)

	if service == nil {
		t.Fatal("Expected non-nil service")
	}

	if service.rpcClients == nil {
		t.Error("Expected initialized rpcClients map")
	}

	if service.stopCh == nil {
		t.Error("Expected initialized stopCh channel")
	}
}

func TestServiceSetRPCClient(t *testing.T) {
	service := NewService(nil)

	// Create a mock RPC client
	mockClient := &mockRPCClient{}
	service.SetRPCClient("ethereum", mockClient)

	if len(service.rpcClients) != 1 {
		t.Errorf("Expected 1 RPC client, got %d", len(service.rpcClients))
	}

	client, ok := service.rpcClients["ethereum"]
	if !ok {
		t.Error("RPC client not set correctly")
	}
	if client == nil {
		t.Error("RPC client is nil")
	}
}

// mockRPCClient is a mock implementation of RPCClient for testing
type mockRPCClient struct{}

func (m *mockRPCClient) Call(ctx context.Context, method string, params ...interface{}) (json.RawMessage, error) {
	return nil, nil
}
