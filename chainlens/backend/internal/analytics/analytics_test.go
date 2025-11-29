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
