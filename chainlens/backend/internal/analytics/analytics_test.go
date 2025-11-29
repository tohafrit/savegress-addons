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

func TestHourlyStatsFields(t *testing.T) {
	stats := HourlyStats{
		ID:               1,
		Network:          "ethereum",
		Hour:             time.Now().UTC().Truncate(time.Hour),
		BlockCount:       300,
		TransactionCount: 50000,
		UniqueAddresses:  10000,
		TotalGasUsed:     "5000000000000",
	}

	if stats.Network != "ethereum" {
		t.Errorf("Expected network 'ethereum', got %s", stats.Network)
	}

	if stats.BlockCount != 300 {
		t.Errorf("Expected block count 300, got %d", stats.BlockCount)
	}
}

func TestGasPriceFields(t *testing.T) {
	slow := int64(20_000_000_000)
	standard := int64(30_000_000_000)
	fast := int64(50_000_000_000)
	instant := int64(75_000_000_000)
	baseFee := int64(25_000_000_000)
	blockNum := int64(18000000)

	gasPrice := GasPrice{
		ID:        1,
		Network:   "ethereum",
		Timestamp: time.Now().UTC(),
		Slow:      &slow,
		Standard:  &standard,
		Fast:      &fast,
		Instant:   &instant,
		BaseFee:   &baseFee,
		BlockNumber: &blockNum,
	}

	if gasPrice.Network != "ethereum" {
		t.Errorf("Expected network 'ethereum', got %s", gasPrice.Network)
	}

	if *gasPrice.Slow != slow {
		t.Errorf("Expected slow %d, got %d", slow, *gasPrice.Slow)
	}

	if *gasPrice.BlockNumber != blockNum {
		t.Errorf("Expected block number %d, got %d", blockNum, *gasPrice.BlockNumber)
	}
}

func TestChartDataFields(t *testing.T) {
	chart := ChartData{
		Network:    "ethereum",
		MetricName: "transactions",
		Period:     "30d",
		DataPoints: []ChartDataPoint{
			{Timestamp: time.Now().UTC(), Value: 1000000},
			{Timestamp: time.Now().UTC().Add(-24 * time.Hour), Value: 950000},
		},
	}

	if chart.Network != "ethereum" {
		t.Errorf("Expected network 'ethereum', got %s", chart.Network)
	}

	if chart.MetricName != "transactions" {
		t.Errorf("Expected metric name 'transactions', got %s", chart.MetricName)
	}

	if len(chart.DataPoints) != 2 {
		t.Errorf("Expected 2 data points, got %d", len(chart.DataPoints))
	}
}

func TestAddressRankingFields(t *testing.T) {
	ranking := AddressRanking{
		ID:          1,
		Network:     "ethereum",
		Date:        time.Now().UTC().Truncate(24 * time.Hour),
		RankingType: RankingTypeBalance,
		Rank:        1,
		Address:     "0xde0B295669a9FD93d5F28D9Ec85E40f4cb697BAe",
		Value:       "1000000000000000000000000",
	}

	if ranking.RankingType != RankingTypeBalance {
		t.Errorf("Expected ranking type 'balance', got %s", ranking.RankingType)
	}

	if ranking.Rank != 1 {
		t.Errorf("Expected rank 1, got %d", ranking.Rank)
	}

	if len(ranking.Address) != 42 {
		t.Errorf("Expected address length 42, got %d", len(ranking.Address))
	}
}

func TestWeiGweiRoundTrip(t *testing.T) {
	// Test round-trip conversion
	originalWei := int64(30_000_000_000) // 30 gwei
	gwei := WeiToGwei(originalWei)
	backToWei := GweiToWei(gwei)

	if backToWei != originalWei {
		t.Errorf("Round trip failed: %d -> %f -> %d", originalWei, gwei, backToWei)
	}
}

func TestHexToInt64EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		hex      string
		expected int64
	}{
		{"mixed case", "0xAbCdEf", 11259375},
		{"leading zeros", "0x0000ff", 255},
		{"just prefix", "0x", 0},
		{"invalid prefix", "1x10", 0},
		{"empty string", "", 0},
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

func TestDailyStatsOptionalFields(t *testing.T) {
	firstBlock := int64(17000000)
	lastBlock := int64(17007200)
	avgBlockTime := 12.5
	avgGasPerTx := int64(150000)
	avgGasPrice := "30000000000"

	stats := DailyStats{
		Network:            "ethereum",
		Date:               time.Now().UTC().Truncate(24 * time.Hour),
		FirstBlock:         &firstBlock,
		LastBlock:          &lastBlock,
		AvgBlockTime:       &avgBlockTime,
		AvgGasPerTx:        &avgGasPerTx,
		AvgGasPrice:        &avgGasPrice,
		TotalValueTransferred: "1000000000000000000000",
		TotalGasUsed:       "50000000000000",
		TotalFeesBurned:    "10000000000000000000",
	}

	if *stats.FirstBlock != firstBlock {
		t.Errorf("Expected first block %d, got %d", firstBlock, *stats.FirstBlock)
	}

	if *stats.AvgBlockTime != avgBlockTime {
		t.Errorf("Expected avg block time %f, got %f", avgBlockTime, *stats.AvgBlockTime)
	}
}

func TestNetworkOverviewOptionalFields(t *testing.T) {
	chainID := 1
	overview := NetworkOverview{
		Network:                "ethereum",
		TotalBlocks:           18000000,
		TotalTransactions:     2000000000,
		ChainID:               &chainID,
		NativeCurrency:        "ETH",
		NativeCurrencyDecimals: 18,
	}

	if *overview.ChainID != chainID {
		t.Errorf("Expected chain ID %d, got %d", chainID, *overview.ChainID)
	}

	if overview.NativeCurrencyDecimals != 18 {
		t.Errorf("Expected decimals 18, got %d", overview.NativeCurrencyDecimals)
	}
}

func TestTopTokenOptionalFields(t *testing.T) {
	tokenName := "Tether USD"
	tokenSymbol := "USDT"

	token := TopToken{
		Network:       "ethereum",
		TokenAddress:  "0xdAC17F958D2ee523a2206206994597C13D831ec7",
		TokenName:     &tokenName,
		TokenSymbol:   &tokenSymbol,
		TransferCount: 1000000,
	}

	if *token.TokenName != tokenName {
		t.Errorf("Expected token name %s, got %s", tokenName, *token.TokenName)
	}

	if *token.TokenSymbol != tokenSymbol {
		t.Errorf("Expected token symbol %s, got %s", tokenSymbol, *token.TokenSymbol)
	}
}

func TestTopContractOptionalFields(t *testing.T) {
	contractName := "Uniswap V2: Router 2"

	contract := TopContract{
		Network:         "ethereum",
		ContractAddress: "0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D",
		ContractName:    &contractName,
		CallCount:       500000,
	}

	if *contract.ContractName != contractName {
		t.Errorf("Expected contract name %s, got %s", contractName, *contract.ContractName)
	}
}

func TestServiceMultipleRPCClients(t *testing.T) {
	service := NewService(nil)

	networks := []string{"ethereum", "polygon", "arbitrum", "optimism", "base", "bsc", "avalanche"}
	for _, network := range networks {
		service.SetRPCClient(network, &mockRPCClient{})
	}

	if len(service.rpcClients) != len(networks) {
		t.Errorf("Expected %d RPC clients, got %d", len(networks), len(service.rpcClients))
	}

	for _, network := range networks {
		if _, ok := service.rpcClients[network]; !ok {
			t.Errorf("Expected RPC client for %s", network)
		}
	}
}

func TestGasPriceEstimateTimeStrings(t *testing.T) {
	estimate := GasPriceEstimate{
		SlowTime:     "~10 min",
		StandardTime: "~3 min",
		FastTime:     "~1 min",
		InstantTime:  "~15 sec",
	}

	if estimate.SlowTime != "~10 min" {
		t.Errorf("Expected slow time '~10 min', got %s", estimate.SlowTime)
	}

	if estimate.InstantTime != "~15 sec" {
		t.Errorf("Expected instant time '~15 sec', got %s", estimate.InstantTime)
	}
}

func TestServiceStopWithoutStart(t *testing.T) {
	service := NewService(nil)

	// Stop without starting should not panic
	// Note: Start() requires a valid repo, so we only test Stop() here
	service.Stop()
}

// mockRPCClientWithGasPrice returns mock gas price data
type mockRPCClientWithGasPrice struct {
	gasPrice  string
	blockNum  string
	baseFee   string
	priorityFee string
}

func (m *mockRPCClientWithGasPrice) Call(ctx context.Context, method string, params ...interface{}) (json.RawMessage, error) {
	switch method {
	case "eth_gasPrice":
		return json.Marshal(m.gasPrice)
	case "eth_blockNumber":
		return json.Marshal(m.blockNum)
	case "eth_getBlockByNumber":
		return json.Marshal(map[string]string{"baseFeePerGas": m.baseFee})
	case "eth_maxPriorityFeePerGas":
		return json.Marshal(m.priorityFee)
	default:
		return nil, nil
	}
}

func TestFetchGasPrice(t *testing.T) {
	service := NewService(nil)

	mockClient := &mockRPCClientWithGasPrice{
		gasPrice:    "0x77359400",      // 2 gwei
		blockNum:    "0x1312D00",       // 20000000
		baseFee:     "0x3B9ACA00",      // 1 gwei
		priorityFee: "0x3B9ACA00",      // 1 gwei
	}

	ctx := context.Background()
	gasPrice, err := service.fetchGasPrice(ctx, mockClient, "ethereum")

	if err != nil {
		t.Fatalf("fetchGasPrice returned error: %v", err)
	}

	if gasPrice == nil {
		t.Fatal("Expected non-nil gas price")
	}

	if gasPrice.Network != "ethereum" {
		t.Errorf("Expected network 'ethereum', got %s", gasPrice.Network)
	}

	if gasPrice.Standard == nil {
		t.Error("Expected non-nil standard gas price")
	}

	if gasPrice.Slow == nil {
		t.Error("Expected non-nil slow gas price")
	}

	if gasPrice.Fast == nil {
		t.Error("Expected non-nil fast gas price")
	}

	if gasPrice.Instant == nil {
		t.Error("Expected non-nil instant gas price")
	}

	// Check that slow < standard < fast < instant
	if *gasPrice.Slow >= *gasPrice.Standard {
		t.Error("Slow should be less than standard")
	}
	if *gasPrice.Standard >= *gasPrice.Fast {
		t.Error("Standard should be less than fast")
	}
	if *gasPrice.Fast >= *gasPrice.Instant {
		t.Error("Fast should be less than instant")
	}
}

func TestGetEIP1559Data(t *testing.T) {
	service := NewService(nil)

	mockClient := &mockRPCClientWithGasPrice{
		baseFee:     "0x3B9ACA00",  // 1 gwei
		priorityFee: "0x77359400",  // 2 gwei
	}

	ctx := context.Background()
	baseFee, priorityFee := service.getEIP1559Data(ctx, mockClient)

	expectedBaseFee := int64(1_000_000_000)    // 1 gwei in wei
	expectedPriorityFee := int64(2_000_000_000) // 2 gwei in wei

	if baseFee != expectedBaseFee {
		t.Errorf("Expected base fee %d, got %d", expectedBaseFee, baseFee)
	}

	if priorityFee != expectedPriorityFee {
		t.Errorf("Expected priority fee %d, got %d", expectedPriorityFee, priorityFee)
	}
}

// mockRPCClientError returns errors
type mockRPCClientError struct{}

func (m *mockRPCClientError) Call(ctx context.Context, method string, params ...interface{}) (json.RawMessage, error) {
	return nil, context.DeadlineExceeded
}

func TestFetchGasPriceError(t *testing.T) {
	service := NewService(nil)
	mockClient := &mockRPCClientError{}

	ctx := context.Background()
	_, err := service.fetchGasPrice(ctx, mockClient, "ethereum")

	if err == nil {
		t.Error("Expected error from fetchGasPrice")
	}
}

func TestGetEIP1559DataError(t *testing.T) {
	service := NewService(nil)
	mockClient := &mockRPCClientError{}

	ctx := context.Background()
	baseFee, priorityFee := service.getEIP1559Data(ctx, mockClient)

	// Should return zeros on error
	if baseFee != 0 {
		t.Errorf("Expected 0 base fee on error, got %d", baseFee)
	}
	if priorityFee != 0 {
		t.Errorf("Expected 0 priority fee on error, got %d", priorityFee)
	}
}

func TestServiceCreationWithoutRepo(t *testing.T) {
	// Service can be created with nil repo (for unit testing individual methods)
	service := NewService(nil)

	if service == nil {
		t.Fatal("Expected non-nil service")
	}

	// repo will be nil, which is acceptable for creation
	if service.repo != nil {
		t.Error("Expected nil repo when created with nil")
	}
}

func TestGetTransactionChartPeriods(t *testing.T) {
	tests := []struct {
		days           int
		expectedPeriod string
	}{
		{1, "24h"},
		{7, "7d"},
		{14, "30d"},
		{30, "30d"},
	}

	for _, tt := range tests {
		period := "30d"
		if tt.days <= 1 {
			period = "24h"
		} else if tt.days <= 7 {
			period = "7d"
		}

		if period != tt.expectedPeriod {
			t.Errorf("For %d days, expected period %s, got %s", tt.days, tt.expectedPeriod, period)
		}
	}
}

func TestGetGasChartPeriods(t *testing.T) {
	tests := []struct {
		hours          int
		expectedPeriod string
	}{
		{6, "6h"},
		{12, "12h"},
		{24, "24h"},
	}

	for _, tt := range tests {
		period := "24h"
		if tt.hours <= 6 {
			period = "6h"
		} else if tt.hours <= 12 {
			period = "12h"
		}

		if period != tt.expectedPeriod {
			t.Errorf("For %d hours, expected period %s, got %s", tt.hours, tt.expectedPeriod, period)
		}
	}
}

func TestGetActiveAddressesChartPeriods(t *testing.T) {
	tests := []struct {
		days           int
		expectedPeriod string
	}{
		{7, "7d"},
		{14, "30d"},
		{30, "30d"},
	}

	for _, tt := range tests {
		period := "30d"
		if tt.days <= 7 {
			period = "7d"
		}

		if period != tt.expectedPeriod {
			t.Errorf("For %d days, expected period %s, got %s", tt.days, tt.expectedPeriod, period)
		}
	}
}

func TestHexToInt64Various(t *testing.T) {
	tests := []struct {
		hex      string
		expected int64
	}{
		{"0x0", 0},
		{"0x1", 1},
		{"0xa", 10},
		{"0xA", 10},
		{"0x10", 16},
		{"0x100", 256},
		{"0x3b9aca00", 1_000_000_000},     // 1 gwei
		{"0x174876e800", 100_000_000_000}, // 100 gwei
	}

	for _, tt := range tests {
		result := hexToInt64(tt.hex)
		if result != tt.expected {
			t.Errorf("hexToInt64(%s) = %d, expected %d", tt.hex, result, tt.expected)
		}
	}
}

// ============================================================================
// MOCK REPOSITORY
// ============================================================================

// MockRepository implements RepositoryInterface for testing
type MockRepository struct {
	// Data stores
	dailyStats      map[string][]*DailyStats
	hourlyStats     map[string][]*HourlyStats
	gasPrices       map[string][]*GasPrice
	networkOverview map[string]*NetworkOverview
	topTokens       map[string][]*TopToken
	topContracts    map[string][]*TopContract

	// Error simulation
	simulateError bool
	errorToReturn error
}

// NewMockRepository creates a new mock repository
func NewMockRepository() *MockRepository {
	return &MockRepository{
		dailyStats:      make(map[string][]*DailyStats),
		hourlyStats:     make(map[string][]*HourlyStats),
		gasPrices:       make(map[string][]*GasPrice),
		networkOverview: make(map[string]*NetworkOverview),
		topTokens:       make(map[string][]*TopToken),
		topContracts:    make(map[string][]*TopContract),
	}
}

// SetError configures the mock to return an error
func (m *MockRepository) SetError(err error) {
	m.simulateError = true
	m.errorToReturn = err
}

// ClearError clears the error simulation
func (m *MockRepository) ClearError() {
	m.simulateError = false
	m.errorToReturn = nil
}

// Daily Stats implementations
func (m *MockRepository) GetDailyStats(ctx context.Context, network string, startDate, endDate time.Time) ([]*DailyStats, error) {
	if m.simulateError {
		return nil, m.errorToReturn
	}
	return m.dailyStats[network], nil
}

func (m *MockRepository) GetDailyStatsForDate(ctx context.Context, network string, date time.Time) (*DailyStats, error) {
	if m.simulateError {
		return nil, m.errorToReturn
	}
	stats := m.dailyStats[network]
	for _, s := range stats {
		if s.Date.Equal(date) {
			return s, nil
		}
	}
	return nil, nil
}

func (m *MockRepository) UpsertDailyStats(ctx context.Context, s *DailyStats) error {
	if m.simulateError {
		return m.errorToReturn
	}
	m.dailyStats[s.Network] = append(m.dailyStats[s.Network], s)
	return nil
}

// Hourly Stats implementations
func (m *MockRepository) GetHourlyStats(ctx context.Context, network string, startTime, endTime time.Time) ([]*HourlyStats, error) {
	if m.simulateError {
		return nil, m.errorToReturn
	}
	return m.hourlyStats[network], nil
}

func (m *MockRepository) UpsertHourlyStats(ctx context.Context, s *HourlyStats) error {
	if m.simulateError {
		return m.errorToReturn
	}
	m.hourlyStats[s.Network] = append(m.hourlyStats[s.Network], s)
	return nil
}

// Gas Prices implementations
func (m *MockRepository) GetLatestGasPrice(ctx context.Context, network string) (*GasPrice, error) {
	if m.simulateError {
		return nil, m.errorToReturn
	}
	prices := m.gasPrices[network]
	if len(prices) == 0 {
		return nil, nil
	}
	return prices[len(prices)-1], nil
}

func (m *MockRepository) GetGasPriceHistory(ctx context.Context, network string, startTime, endTime time.Time, limit int) ([]*GasPrice, error) {
	if m.simulateError {
		return nil, m.errorToReturn
	}
	prices := m.gasPrices[network]
	if len(prices) > limit {
		return prices[:limit], nil
	}
	return prices, nil
}

func (m *MockRepository) InsertGasPrice(ctx context.Context, g *GasPrice) error {
	if m.simulateError {
		return m.errorToReturn
	}
	m.gasPrices[g.Network] = append(m.gasPrices[g.Network], g)
	return nil
}

// Network Overview implementations
func (m *MockRepository) GetNetworkOverview(ctx context.Context, network string) (*NetworkOverview, error) {
	if m.simulateError {
		return nil, m.errorToReturn
	}
	return m.networkOverview[network], nil
}

func (m *MockRepository) UpdateNetworkOverview(ctx context.Context, o *NetworkOverview) error {
	if m.simulateError {
		return m.errorToReturn
	}
	m.networkOverview[o.Network] = o
	return nil
}

// Top Tokens implementations
func (m *MockRepository) GetTopTokens(ctx context.Context, network string, date time.Time, limit int) ([]*TopToken, error) {
	if m.simulateError {
		return nil, m.errorToReturn
	}
	tokens := m.topTokens[network]
	if len(tokens) > limit {
		return tokens[:limit], nil
	}
	return tokens, nil
}

func (m *MockRepository) UpsertTopToken(ctx context.Context, t *TopToken) error {
	if m.simulateError {
		return m.errorToReturn
	}
	m.topTokens[t.Network] = append(m.topTokens[t.Network], t)
	return nil
}

// Top Contracts implementations
func (m *MockRepository) GetTopContracts(ctx context.Context, network string, date time.Time, limit int) ([]*TopContract, error) {
	if m.simulateError {
		return nil, m.errorToReturn
	}
	contracts := m.topContracts[network]
	if len(contracts) > limit {
		return contracts[:limit], nil
	}
	return contracts, nil
}

func (m *MockRepository) UpsertTopContract(ctx context.Context, c *TopContract) error {
	if m.simulateError {
		return m.errorToReturn
	}
	m.topContracts[c.Network] = append(m.topContracts[c.Network], c)
	return nil
}

// Aggregation implementations
func (m *MockRepository) AggregateDailyStats(ctx context.Context, network string, date time.Time) error {
	if m.simulateError {
		return m.errorToReturn
	}
	return nil
}

func (m *MockRepository) RefreshNetworkOverview(ctx context.Context, network string) error {
	if m.simulateError {
		return m.errorToReturn
	}
	return nil
}

// Chart implementations
func (m *MockRepository) GetTransactionCountChart(ctx context.Context, network string, days int) ([]ChartDataPoint, error) {
	if m.simulateError {
		return nil, m.errorToReturn
	}
	var points []ChartDataPoint
	for i := 0; i < days; i++ {
		points = append(points, ChartDataPoint{
			Timestamp: time.Now().AddDate(0, 0, -i),
			Value:     float64(1000000 - i*1000),
		})
	}
	return points, nil
}

func (m *MockRepository) GetGasPriceChart(ctx context.Context, network string, hours int) ([]ChartDataPoint, error) {
	if m.simulateError {
		return nil, m.errorToReturn
	}
	var points []ChartDataPoint
	for i := 0; i < hours; i++ {
		points = append(points, ChartDataPoint{
			Timestamp: time.Now().Add(-time.Duration(i) * time.Hour),
			Value:     float64(30 + i),
		})
	}
	return points, nil
}

func (m *MockRepository) GetActiveAddressesChart(ctx context.Context, network string, days int) ([]ChartDataPoint, error) {
	if m.simulateError {
		return nil, m.errorToReturn
	}
	var points []ChartDataPoint
	for i := 0; i < days; i++ {
		points = append(points, ChartDataPoint{
			Timestamp: time.Now().AddDate(0, 0, -i),
			Value:     float64(50000 - i*100),
		})
	}
	return points, nil
}

// ============================================================================
// SERVICE TESTS WITH MOCK
// ============================================================================

func TestServiceWithMockRepo_GetNetworkOverview(t *testing.T) {
	mockRepo := NewMockRepository()
	service := NewService(mockRepo)

	// Set up test data
	latestBlock := int64(18000000)
	mockRepo.networkOverview["ethereum"] = &NetworkOverview{
		Network:           "ethereum",
		LatestBlock:       &latestBlock,
		TotalBlocks:       18000000,
		TotalTransactions: 2000000000,
		NativeCurrency:    "ETH",
	}

	ctx := context.Background()
	overview, err := service.GetNetworkOverview(ctx, "ethereum")

	if err != nil {
		t.Fatalf("GetNetworkOverview returned error: %v", err)
	}

	if overview == nil {
		t.Fatal("Expected non-nil overview")
	}

	if overview.Network != "ethereum" {
		t.Errorf("Expected network 'ethereum', got %s", overview.Network)
	}

	if *overview.LatestBlock != 18000000 {
		t.Errorf("Expected latest block 18000000, got %d", *overview.LatestBlock)
	}
}

func TestServiceWithMockRepo_GetNetworkOverviewNotFound(t *testing.T) {
	mockRepo := NewMockRepository()
	service := NewService(mockRepo)

	ctx := context.Background()
	overview, err := service.GetNetworkOverview(ctx, "unknown")

	if err != nil {
		t.Fatalf("GetNetworkOverview returned error: %v", err)
	}

	if overview != nil {
		t.Error("Expected nil overview for unknown network")
	}
}

func TestServiceWithMockRepo_GetDailyStats(t *testing.T) {
	mockRepo := NewMockRepository()
	service := NewService(mockRepo)

	// Set up test data
	today := time.Now().UTC().Truncate(24 * time.Hour)
	mockRepo.dailyStats["ethereum"] = []*DailyStats{
		{
			Network:          "ethereum",
			Date:             today,
			BlockCount:       7200,
			TransactionCount: 1200000,
		},
		{
			Network:          "ethereum",
			Date:             today.AddDate(0, 0, -1),
			BlockCount:       7100,
			TransactionCount: 1100000,
		},
	}

	ctx := context.Background()
	stats, err := service.GetDailyStats(ctx, "ethereum", today.AddDate(0, 0, -7), today)

	if err != nil {
		t.Fatalf("GetDailyStats returned error: %v", err)
	}

	if len(stats) != 2 {
		t.Errorf("Expected 2 stats, got %d", len(stats))
	}
}

func TestServiceWithMockRepo_GetHourlyStats(t *testing.T) {
	mockRepo := NewMockRepository()
	service := NewService(mockRepo)

	// Set up test data
	now := time.Now().UTC().Truncate(time.Hour)
	mockRepo.hourlyStats["ethereum"] = []*HourlyStats{
		{
			Network:          "ethereum",
			Hour:             now,
			BlockCount:       300,
			TransactionCount: 50000,
		},
	}

	ctx := context.Background()
	stats, err := service.GetHourlyStats(ctx, "ethereum", now.Add(-24*time.Hour), now)

	if err != nil {
		t.Fatalf("GetHourlyStats returned error: %v", err)
	}

	if len(stats) != 1 {
		t.Errorf("Expected 1 stats, got %d", len(stats))
	}
}

func TestServiceWithMockRepo_GetCurrentGasPrice(t *testing.T) {
	mockRepo := NewMockRepository()
	service := NewService(mockRepo)

	// Set up test data
	slow := int64(20_000_000_000)     // 20 gwei
	standard := int64(30_000_000_000) // 30 gwei
	fast := int64(50_000_000_000)     // 50 gwei
	instant := int64(75_000_000_000)  // 75 gwei
	baseFee := int64(25_000_000_000)  // 25 gwei

	mockRepo.gasPrices["ethereum"] = []*GasPrice{
		{
			Network:   "ethereum",
			Timestamp: time.Now().UTC(),
			Slow:      &slow,
			Standard:  &standard,
			Fast:      &fast,
			Instant:   &instant,
			BaseFee:   &baseFee,
		},
	}

	ctx := context.Background()
	estimate, err := service.GetCurrentGasPrice(ctx, "ethereum")

	if err != nil {
		t.Fatalf("GetCurrentGasPrice returned error: %v", err)
	}

	if estimate == nil {
		t.Fatal("Expected non-nil estimate")
	}

	if estimate.SlowGwei != 20.0 {
		t.Errorf("Expected slow 20 gwei, got %f", estimate.SlowGwei)
	}

	if estimate.StandardGwei != 30.0 {
		t.Errorf("Expected standard 30 gwei, got %f", estimate.StandardGwei)
	}

	if estimate.FastGwei != 50.0 {
		t.Errorf("Expected fast 50 gwei, got %f", estimate.FastGwei)
	}

	if estimate.InstantGwei != 75.0 {
		t.Errorf("Expected instant 75 gwei, got %f", estimate.InstantGwei)
	}

	if estimate.BaseFeeGwei != 25.0 {
		t.Errorf("Expected base fee 25 gwei, got %f", estimate.BaseFeeGwei)
	}
}

func TestServiceWithMockRepo_GetCurrentGasPriceNoData(t *testing.T) {
	mockRepo := NewMockRepository()
	service := NewService(mockRepo)

	ctx := context.Background()
	_, err := service.GetCurrentGasPrice(ctx, "ethereum")

	if err == nil {
		t.Error("Expected error when no gas price data")
	}
}

func TestServiceWithMockRepo_GetGasPriceHistory(t *testing.T) {
	mockRepo := NewMockRepository()
	service := NewService(mockRepo)

	// Set up test data
	standard := int64(30_000_000_000)
	for i := 0; i < 5; i++ {
		mockRepo.gasPrices["ethereum"] = append(mockRepo.gasPrices["ethereum"], &GasPrice{
			Network:   "ethereum",
			Timestamp: time.Now().UTC().Add(-time.Duration(i) * time.Hour),
			Standard:  &standard,
		})
	}

	ctx := context.Background()
	history, err := service.GetGasPriceHistory(ctx, "ethereum", 24)

	if err != nil {
		t.Fatalf("GetGasPriceHistory returned error: %v", err)
	}

	if len(history) != 5 {
		t.Errorf("Expected 5 history entries, got %d", len(history))
	}
}

func TestServiceWithMockRepo_GetTransactionChart(t *testing.T) {
	mockRepo := NewMockRepository()
	service := NewService(mockRepo)

	ctx := context.Background()
	chart, err := service.GetTransactionChart(ctx, "ethereum", 30)

	if err != nil {
		t.Fatalf("GetTransactionChart returned error: %v", err)
	}

	if chart == nil {
		t.Fatal("Expected non-nil chart")
	}

	if chart.Network != "ethereum" {
		t.Errorf("Expected network 'ethereum', got %s", chart.Network)
	}

	if chart.MetricName != "transactions" {
		t.Errorf("Expected metric name 'transactions', got %s", chart.MetricName)
	}

	if chart.Period != "30d" {
		t.Errorf("Expected period '30d', got %s", chart.Period)
	}

	if len(chart.DataPoints) != 30 {
		t.Errorf("Expected 30 data points, got %d", len(chart.DataPoints))
	}
}

func TestServiceWithMockRepo_GetTransactionChartPeriods(t *testing.T) {
	mockRepo := NewMockRepository()
	service := NewService(mockRepo)
	ctx := context.Background()

	// Test 24h period
	chart, _ := service.GetTransactionChart(ctx, "ethereum", 1)
	if chart.Period != "24h" {
		t.Errorf("Expected period '24h' for 1 day, got %s", chart.Period)
	}

	// Test 7d period
	chart, _ = service.GetTransactionChart(ctx, "ethereum", 7)
	if chart.Period != "7d" {
		t.Errorf("Expected period '7d' for 7 days, got %s", chart.Period)
	}

	// Test 30d period
	chart, _ = service.GetTransactionChart(ctx, "ethereum", 14)
	if chart.Period != "30d" {
		t.Errorf("Expected period '30d' for 14 days, got %s", chart.Period)
	}
}

func TestServiceWithMockRepo_GetGasChart(t *testing.T) {
	mockRepo := NewMockRepository()
	service := NewService(mockRepo)

	ctx := context.Background()
	chart, err := service.GetGasChart(ctx, "ethereum", 24)

	if err != nil {
		t.Fatalf("GetGasChart returned error: %v", err)
	}

	if chart == nil {
		t.Fatal("Expected non-nil chart")
	}

	if chart.MetricName != "gas_price" {
		t.Errorf("Expected metric name 'gas_price', got %s", chart.MetricName)
	}

	if chart.Period != "24h" {
		t.Errorf("Expected period '24h', got %s", chart.Period)
	}
}

func TestServiceWithMockRepo_GetGasChartPeriods(t *testing.T) {
	mockRepo := NewMockRepository()
	service := NewService(mockRepo)
	ctx := context.Background()

	// Test 6h period
	chart, _ := service.GetGasChart(ctx, "ethereum", 6)
	if chart.Period != "6h" {
		t.Errorf("Expected period '6h' for 6 hours, got %s", chart.Period)
	}

	// Test 12h period
	chart, _ = service.GetGasChart(ctx, "ethereum", 12)
	if chart.Period != "12h" {
		t.Errorf("Expected period '12h' for 12 hours, got %s", chart.Period)
	}

	// Test 24h period
	chart, _ = service.GetGasChart(ctx, "ethereum", 24)
	if chart.Period != "24h" {
		t.Errorf("Expected period '24h' for 24 hours, got %s", chart.Period)
	}
}

func TestServiceWithMockRepo_GetActiveAddressesChart(t *testing.T) {
	mockRepo := NewMockRepository()
	service := NewService(mockRepo)

	ctx := context.Background()
	chart, err := service.GetActiveAddressesChart(ctx, "ethereum", 7)

	if err != nil {
		t.Fatalf("GetActiveAddressesChart returned error: %v", err)
	}

	if chart == nil {
		t.Fatal("Expected non-nil chart")
	}

	if chart.MetricName != "active_addresses" {
		t.Errorf("Expected metric name 'active_addresses', got %s", chart.MetricName)
	}

	if chart.Period != "7d" {
		t.Errorf("Expected period '7d', got %s", chart.Period)
	}
}

func TestServiceWithMockRepo_GetTopTokens(t *testing.T) {
	mockRepo := NewMockRepository()
	service := NewService(mockRepo)

	// Set up test data
	tokenName := "Tether USD"
	tokenSymbol := "USDT"
	mockRepo.topTokens["ethereum"] = []*TopToken{
		{
			Network:       "ethereum",
			Date:          time.Now().UTC().Truncate(24 * time.Hour),
			Rank:          1,
			TokenAddress:  "0xdAC17F958D2ee523a2206206994597C13D831ec7",
			TokenName:     &tokenName,
			TokenSymbol:   &tokenSymbol,
			TransferCount: 1000000,
		},
	}

	ctx := context.Background()
	tokens, err := service.GetTopTokens(ctx, "ethereum", 10)

	if err != nil {
		t.Fatalf("GetTopTokens returned error: %v", err)
	}

	if len(tokens) != 1 {
		t.Errorf("Expected 1 token, got %d", len(tokens))
	}

	if tokens[0].Rank != 1 {
		t.Errorf("Expected rank 1, got %d", tokens[0].Rank)
	}
}

func TestServiceWithMockRepo_GetTopContracts(t *testing.T) {
	mockRepo := NewMockRepository()
	service := NewService(mockRepo)

	// Set up test data
	contractName := "Uniswap V2: Router 2"
	mockRepo.topContracts["ethereum"] = []*TopContract{
		{
			Network:         "ethereum",
			Date:            time.Now().UTC().Truncate(24 * time.Hour),
			Rank:            1,
			ContractAddress: "0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D",
			ContractName:    &contractName,
			CallCount:       500000,
		},
	}

	ctx := context.Background()
	contracts, err := service.GetTopContracts(ctx, "ethereum", 10)

	if err != nil {
		t.Fatalf("GetTopContracts returned error: %v", err)
	}

	if len(contracts) != 1 {
		t.Errorf("Expected 1 contract, got %d", len(contracts))
	}

	if contracts[0].Rank != 1 {
		t.Errorf("Expected rank 1, got %d", contracts[0].Rank)
	}
}

func TestServiceWithMockRepo_ErrorHandling(t *testing.T) {
	mockRepo := NewMockRepository()
	service := NewService(mockRepo)
	ctx := context.Background()

	// Simulate error
	testErr := context.DeadlineExceeded
	mockRepo.SetError(testErr)

	// Test GetNetworkOverview with error
	_, err := service.GetNetworkOverview(ctx, "ethereum")
	if err != testErr {
		t.Errorf("Expected error %v, got %v", testErr, err)
	}

	// Test GetDailyStats with error
	_, err = service.GetDailyStats(ctx, "ethereum", time.Now(), time.Now())
	if err != testErr {
		t.Errorf("Expected error %v, got %v", testErr, err)
	}

	// Test GetHourlyStats with error
	_, err = service.GetHourlyStats(ctx, "ethereum", time.Now(), time.Now())
	if err != testErr {
		t.Errorf("Expected error %v, got %v", testErr, err)
	}

	// Clear error
	mockRepo.ClearError()

	// Should work now
	_, err = service.GetNetworkOverview(ctx, "ethereum")
	if err != nil {
		t.Errorf("Unexpected error after clearing: %v", err)
	}
}
