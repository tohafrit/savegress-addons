package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

// Service provides analytics business logic
type Service struct {
	repo       RepositoryInterface
	rpcClients map[string]RPCClient

	// Background aggregation
	stopCh chan struct{}
	wg     sync.WaitGroup
}

// RPCClient interface for blockchain RPC calls
type RPCClient interface {
	Call(ctx context.Context, method string, params ...interface{}) (json.RawMessage, error)
}

// NewService creates a new analytics service
func NewService(repo RepositoryInterface) *Service {
	return &Service{
		repo:       repo,
		rpcClients: make(map[string]RPCClient),
		stopCh:     make(chan struct{}),
	}
}

// SetRPCClient sets the RPC client for a network
func (s *Service) SetRPCClient(network string, client RPCClient) {
	s.rpcClients[network] = client
}

// Start starts background analytics jobs
func (s *Service) Start() {
	// Start daily aggregation job
	s.wg.Add(1)
	go s.dailyAggregationJob()

	// Start gas price tracking job
	s.wg.Add(1)
	go s.gasPriceTrackingJob()

	// Start overview refresh job
	s.wg.Add(1)
	go s.overviewRefreshJob()
}

// Stop stops background jobs
func (s *Service) Stop() {
	close(s.stopCh)
	s.wg.Wait()
}

// dailyAggregationJob runs daily stats aggregation
func (s *Service) dailyAggregationJob() {
	defer s.wg.Done()

	// Run initial aggregation for yesterday
	s.aggregateYesterdayStats()

	// Then run daily at midnight UTC
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case t := <-ticker.C:
			// Check if it's midnight UTC (0-1 hour)
			if t.UTC().Hour() == 0 {
				s.aggregateYesterdayStats()
			}
		}
	}
}

// aggregateYesterdayStats aggregates stats for yesterday
func (s *Service) aggregateYesterdayStats() {
	ctx := context.Background()
	yesterday := time.Now().UTC().AddDate(0, 0, -1).Truncate(24 * time.Hour)

	for _, network := range SupportedNetworks {
		if err := s.repo.AggregateDailyStats(ctx, network, yesterday); err != nil {
			log.Printf("Error aggregating daily stats for %s: %v", network, err)
		}
	}
}

// gasPriceTrackingJob tracks gas prices periodically
func (s *Service) gasPriceTrackingJob() {
	defer s.wg.Done()

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.updateGasPrices()
		}
	}
}

// updateGasPrices fetches and stores current gas prices
func (s *Service) updateGasPrices() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for network, client := range s.rpcClients {
		gasPrice, err := s.fetchGasPrice(ctx, client, network)
		if err != nil {
			log.Printf("Error fetching gas price for %s: %v", network, err)
			continue
		}

		if err := s.repo.InsertGasPrice(ctx, gasPrice); err != nil {
			log.Printf("Error storing gas price for %s: %v", network, err)
		}
	}
}

// fetchGasPrice fetches current gas price from RPC
func (s *Service) fetchGasPrice(ctx context.Context, client RPCClient, network string) (*GasPrice, error) {
	now := time.Now().UTC().Truncate(time.Second)
	gasPrice := &GasPrice{
		Network:   network,
		Timestamp: now,
	}

	// Get gas price
	result, err := client.Call(ctx, "eth_gasPrice")
	if err != nil {
		return nil, fmt.Errorf("eth_gasPrice: %w", err)
	}

	var gasPriceHex string
	if err := json.Unmarshal(result, &gasPriceHex); err != nil {
		return nil, fmt.Errorf("parse gas price: %w", err)
	}

	price := hexToInt64(gasPriceHex)
	gasPrice.Standard = &price
	gasPrice.Slow = int64Ptr(price * 80 / 100)      // 80% of standard
	gasPrice.Fast = int64Ptr(price * 120 / 100)     // 120% of standard
	gasPrice.Instant = int64Ptr(price * 150 / 100)  // 150% of standard

	// Try to get EIP-1559 data
	baseFee, priorityFee := s.getEIP1559Data(ctx, client)
	if baseFee > 0 {
		gasPrice.BaseFee = &baseFee
		gasPrice.PriorityFeeSlow = int64Ptr(priorityFee * 80 / 100)
		gasPrice.PriorityFeeStandard = &priorityFee
		gasPrice.PriorityFeeFast = int64Ptr(priorityFee * 150 / 100)
	}

	// Get block number
	blockResult, err := client.Call(ctx, "eth_blockNumber")
	if err == nil {
		var blockHex string
		if json.Unmarshal(blockResult, &blockHex) == nil {
			blockNum := hexToInt64(blockHex)
			gasPrice.BlockNumber = &blockNum
		}
	}

	return gasPrice, nil
}

// getEIP1559Data gets base fee and priority fee
func (s *Service) getEIP1559Data(ctx context.Context, client RPCClient) (baseFee, priorityFee int64) {
	// Get latest block for base fee
	result, err := client.Call(ctx, "eth_getBlockByNumber", "latest", false)
	if err != nil {
		return 0, 0
	}

	var block struct {
		BaseFeePerGas string `json:"baseFeePerGas"`
	}
	if err := json.Unmarshal(result, &block); err != nil {
		return 0, 0
	}

	if block.BaseFeePerGas != "" {
		baseFee = hexToInt64(block.BaseFeePerGas)
	}

	// Get max priority fee
	result, err = client.Call(ctx, "eth_maxPriorityFeePerGas")
	if err == nil {
		var feeHex string
		if json.Unmarshal(result, &feeHex) == nil {
			priorityFee = hexToInt64(feeHex)
		}
	}

	return baseFee, priorityFee
}

// overviewRefreshJob refreshes network overviews periodically
func (s *Service) overviewRefreshJob() {
	defer s.wg.Done()

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	// Initial refresh
	s.refreshOverviews()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.refreshOverviews()
		}
	}
}

// refreshOverviews updates network overviews
func (s *Service) refreshOverviews() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, network := range SupportedNetworks {
		if err := s.repo.RefreshNetworkOverview(ctx, network); err != nil {
			log.Printf("Error refreshing overview for %s: %v", network, err)
		}

		// Update gas prices in overview
		if gasPrice, err := s.repo.GetLatestGasPrice(ctx, network); err == nil && gasPrice != nil {
			overview, err := s.repo.GetNetworkOverview(ctx, network)
			if err == nil && overview != nil {
				overview.GasPriceSlow = gasPrice.Slow
				overview.GasPriceStandard = gasPrice.Standard
				overview.GasPriceFast = gasPrice.Fast
				overview.BaseFee = gasPrice.BaseFee
				s.repo.UpdateNetworkOverview(ctx, overview)
			}
		}
	}
}

// GetNetworkOverview retrieves network overview
func (s *Service) GetNetworkOverview(ctx context.Context, network string) (*NetworkOverview, error) {
	return s.repo.GetNetworkOverview(ctx, network)
}

// GetDailyStats retrieves daily stats for a date range
func (s *Service) GetDailyStats(ctx context.Context, network string, startDate, endDate time.Time) ([]*DailyStats, error) {
	return s.repo.GetDailyStats(ctx, network, startDate, endDate)
}

// GetHourlyStats retrieves hourly stats for a time range
func (s *Service) GetHourlyStats(ctx context.Context, network string, startTime, endTime time.Time) ([]*HourlyStats, error) {
	return s.repo.GetHourlyStats(ctx, network, startTime, endTime)
}

// GetCurrentGasPrice retrieves current gas price estimate
func (s *Service) GetCurrentGasPrice(ctx context.Context, network string) (*GasPriceEstimate, error) {
	gasPrice, err := s.repo.GetLatestGasPrice(ctx, network)
	if err != nil {
		return nil, err
	}
	if gasPrice == nil {
		return nil, fmt.Errorf("no gas price data for %s", network)
	}

	estimate := &GasPriceEstimate{
		Network:   network,
		Timestamp: gasPrice.Timestamp,
	}

	if gasPrice.Slow != nil {
		estimate.SlowGwei = WeiToGwei(*gasPrice.Slow)
	}
	if gasPrice.Standard != nil {
		estimate.StandardGwei = WeiToGwei(*gasPrice.Standard)
	}
	if gasPrice.Fast != nil {
		estimate.FastGwei = WeiToGwei(*gasPrice.Fast)
	}
	if gasPrice.Instant != nil {
		estimate.InstantGwei = WeiToGwei(*gasPrice.Instant)
	}
	if gasPrice.BaseFee != nil {
		estimate.BaseFeeGwei = WeiToGwei(*gasPrice.BaseFee)
	}
	if gasPrice.PriorityFeeSlow != nil {
		estimate.PriorityFeeSlowGwei = WeiToGwei(*gasPrice.PriorityFeeSlow)
	}
	if gasPrice.PriorityFeeStandard != nil {
		estimate.PriorityFeeStdGwei = WeiToGwei(*gasPrice.PriorityFeeStandard)
	}
	if gasPrice.PriorityFeeFast != nil {
		estimate.PriorityFeeFastGwei = WeiToGwei(*gasPrice.PriorityFeeFast)
	}

	estimate.SlowTime = "~10 min"
	estimate.StandardTime = "~3 min"
	estimate.FastTime = "~1 min"
	estimate.InstantTime = "~15 sec"

	return estimate, nil
}

// GetGasPriceHistory retrieves gas price history
func (s *Service) GetGasPriceHistory(ctx context.Context, network string, hours int) ([]*GasPrice, error) {
	endTime := time.Now().UTC()
	startTime := endTime.Add(-time.Duration(hours) * time.Hour)
	return s.repo.GetGasPriceHistory(ctx, network, startTime, endTime, 1000)
}

// GetTransactionChart retrieves transaction count chart data
func (s *Service) GetTransactionChart(ctx context.Context, network string, days int) (*ChartData, error) {
	points, err := s.repo.GetTransactionCountChart(ctx, network, days)
	if err != nil {
		return nil, err
	}

	period := "30d"
	if days <= 1 {
		period = "24h"
	} else if days <= 7 {
		period = "7d"
	}

	return &ChartData{
		Network:    network,
		MetricName: "transactions",
		Period:     period,
		DataPoints: points,
	}, nil
}

// GetGasChart retrieves gas price chart data
func (s *Service) GetGasChart(ctx context.Context, network string, hours int) (*ChartData, error) {
	points, err := s.repo.GetGasPriceChart(ctx, network, hours)
	if err != nil {
		return nil, err
	}

	period := "24h"
	if hours <= 6 {
		period = "6h"
	} else if hours <= 12 {
		period = "12h"
	}

	return &ChartData{
		Network:    network,
		MetricName: "gas_price",
		Period:     period,
		DataPoints: points,
	}, nil
}

// GetActiveAddressesChart retrieves active addresses chart data
func (s *Service) GetActiveAddressesChart(ctx context.Context, network string, days int) (*ChartData, error) {
	points, err := s.repo.GetActiveAddressesChart(ctx, network, days)
	if err != nil {
		return nil, err
	}

	period := "30d"
	if days <= 7 {
		period = "7d"
	}

	return &ChartData{
		Network:    network,
		MetricName: "active_addresses",
		Period:     period,
		DataPoints: points,
	}, nil
}

// GetTopTokens retrieves top tokens
func (s *Service) GetTopTokens(ctx context.Context, network string, limit int) ([]*TopToken, error) {
	today := time.Now().UTC().Truncate(24 * time.Hour)
	return s.repo.GetTopTokens(ctx, network, today, limit)
}

// GetTopContracts retrieves top contracts
func (s *Service) GetTopContracts(ctx context.Context, network string, limit int) ([]*TopContract, error) {
	today := time.Now().UTC().Truncate(24 * time.Hour)
	return s.repo.GetTopContracts(ctx, network, today, limit)
}

// Helper functions

func hexToInt64(hex string) int64 {
	if len(hex) < 2 || hex[:2] != "0x" {
		return 0
	}
	hex = hex[2:]
	var result int64
	for _, c := range hex {
		result *= 16
		switch {
		case c >= '0' && c <= '9':
			result += int64(c - '0')
		case c >= 'a' && c <= 'f':
			result += int64(c - 'a' + 10)
		case c >= 'A' && c <= 'F':
			result += int64(c - 'A' + 10)
		}
	}
	return result
}

func int64Ptr(v int64) *int64 {
	return &v
}
