package analytics

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RepositoryInterface defines the interface for analytics repository operations
type RepositoryInterface interface {
	// Daily Stats
	GetDailyStats(ctx context.Context, network string, startDate, endDate time.Time) ([]*DailyStats, error)
	GetDailyStatsForDate(ctx context.Context, network string, date time.Time) (*DailyStats, error)
	UpsertDailyStats(ctx context.Context, s *DailyStats) error

	// Hourly Stats
	GetHourlyStats(ctx context.Context, network string, startTime, endTime time.Time) ([]*HourlyStats, error)
	UpsertHourlyStats(ctx context.Context, s *HourlyStats) error

	// Gas Prices
	GetLatestGasPrice(ctx context.Context, network string) (*GasPrice, error)
	GetGasPriceHistory(ctx context.Context, network string, startTime, endTime time.Time, limit int) ([]*GasPrice, error)
	InsertGasPrice(ctx context.Context, g *GasPrice) error

	// Network Overview
	GetNetworkOverview(ctx context.Context, network string) (*NetworkOverview, error)
	UpdateNetworkOverview(ctx context.Context, o *NetworkOverview) error

	// Top Tokens
	GetTopTokens(ctx context.Context, network string, date time.Time, limit int) ([]*TopToken, error)
	UpsertTopToken(ctx context.Context, t *TopToken) error

	// Top Contracts
	GetTopContracts(ctx context.Context, network string, date time.Time, limit int) ([]*TopContract, error)
	UpsertTopContract(ctx context.Context, c *TopContract) error

	// Aggregation
	AggregateDailyStats(ctx context.Context, network string, date time.Time) error
	RefreshNetworkOverview(ctx context.Context, network string) error

	// Charts
	GetTransactionCountChart(ctx context.Context, network string, days int) ([]ChartDataPoint, error)
	GetGasPriceChart(ctx context.Context, network string, hours int) ([]ChartDataPoint, error)
	GetActiveAddressesChart(ctx context.Context, network string, days int) ([]ChartDataPoint, error)
}

// Repository provides database operations for analytics
type Repository struct {
	db *pgxpool.Pool
}

// Ensure Repository implements RepositoryInterface
var _ RepositoryInterface = (*Repository)(nil)

// NewRepository creates a new analytics repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// ============================================================================
// DAILY STATS
// ============================================================================

// GetDailyStats retrieves daily stats for a date range
func (r *Repository) GetDailyStats(ctx context.Context, network string, startDate, endDate time.Time) ([]*DailyStats, error) {
	query := `
		SELECT id, network, date, block_count, first_block, last_block, avg_block_time,
			transaction_count, successful_tx_count, failed_tx_count, contract_creation_count,
			unique_senders, unique_receivers, new_addresses,
			total_value_transferred, avg_value_per_tx,
			total_gas_used, avg_gas_per_tx, avg_gas_price, min_gas_price, max_gas_price,
			avg_base_fee, total_fees_burned,
			token_transfer_count, unique_tokens_transferred,
			nft_transfer_count, nft_mint_count, unique_nft_collections,
			contract_deploy_count, verified_contracts_count, contract_call_count,
			created_at, updated_at
		FROM daily_stats
		WHERE network = $1 AND date >= $2 AND date <= $3
		ORDER BY date DESC`

	rows, err := r.db.Query(ctx, query, network, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("get daily stats: %w", err)
	}
	defer rows.Close()

	var stats []*DailyStats
	for rows.Next() {
		var s DailyStats
		if err := rows.Scan(
			&s.ID, &s.Network, &s.Date, &s.BlockCount, &s.FirstBlock, &s.LastBlock, &s.AvgBlockTime,
			&s.TransactionCount, &s.SuccessfulTxCount, &s.FailedTxCount, &s.ContractCreationCount,
			&s.UniqueSenders, &s.UniqueReceivers, &s.NewAddresses,
			&s.TotalValueTransferred, &s.AvgValuePerTx,
			&s.TotalGasUsed, &s.AvgGasPerTx, &s.AvgGasPrice, &s.MinGasPrice, &s.MaxGasPrice,
			&s.AvgBaseFee, &s.TotalFeesBurned,
			&s.TokenTransferCount, &s.UniqueTokensTransferred,
			&s.NFTTransferCount, &s.NFTMintCount, &s.UniqueNFTCollections,
			&s.ContractDeployCount, &s.VerifiedContractsCount, &s.ContractCallCount,
			&s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan daily stats: %w", err)
		}
		stats = append(stats, &s)
	}

	return stats, nil
}

// GetDailyStatsForDate retrieves stats for a specific date
func (r *Repository) GetDailyStatsForDate(ctx context.Context, network string, date time.Time) (*DailyStats, error) {
	query := `
		SELECT id, network, date, block_count, first_block, last_block, avg_block_time,
			transaction_count, successful_tx_count, failed_tx_count, contract_creation_count,
			unique_senders, unique_receivers, new_addresses,
			total_value_transferred, avg_value_per_tx,
			total_gas_used, avg_gas_per_tx, avg_gas_price, min_gas_price, max_gas_price,
			avg_base_fee, total_fees_burned,
			token_transfer_count, unique_tokens_transferred,
			nft_transfer_count, nft_mint_count, unique_nft_collections,
			contract_deploy_count, verified_contracts_count, contract_call_count,
			created_at, updated_at
		FROM daily_stats
		WHERE network = $1 AND date = $2`

	var s DailyStats
	err := r.db.QueryRow(ctx, query, network, date).Scan(
		&s.ID, &s.Network, &s.Date, &s.BlockCount, &s.FirstBlock, &s.LastBlock, &s.AvgBlockTime,
		&s.TransactionCount, &s.SuccessfulTxCount, &s.FailedTxCount, &s.ContractCreationCount,
		&s.UniqueSenders, &s.UniqueReceivers, &s.NewAddresses,
		&s.TotalValueTransferred, &s.AvgValuePerTx,
		&s.TotalGasUsed, &s.AvgGasPerTx, &s.AvgGasPrice, &s.MinGasPrice, &s.MaxGasPrice,
		&s.AvgBaseFee, &s.TotalFeesBurned,
		&s.TokenTransferCount, &s.UniqueTokensTransferred,
		&s.NFTTransferCount, &s.NFTMintCount, &s.UniqueNFTCollections,
		&s.ContractDeployCount, &s.VerifiedContractsCount, &s.ContractCallCount,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get daily stats for date: %w", err)
	}

	return &s, nil
}

// UpsertDailyStats creates or updates daily stats
func (r *Repository) UpsertDailyStats(ctx context.Context, s *DailyStats) error {
	query := `
		INSERT INTO daily_stats (
			network, date, block_count, first_block, last_block, avg_block_time,
			transaction_count, successful_tx_count, failed_tx_count, contract_creation_count,
			unique_senders, unique_receivers, new_addresses,
			total_value_transferred, avg_value_per_tx,
			total_gas_used, avg_gas_per_tx, avg_gas_price, min_gas_price, max_gas_price,
			avg_base_fee, total_fees_burned,
			token_transfer_count, unique_tokens_transferred,
			nft_transfer_count, nft_mint_count, unique_nft_collections,
			contract_deploy_count, verified_contracts_count, contract_call_count
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30)
		ON CONFLICT (network, date) DO UPDATE SET
			block_count = EXCLUDED.block_count,
			first_block = EXCLUDED.first_block,
			last_block = EXCLUDED.last_block,
			avg_block_time = EXCLUDED.avg_block_time,
			transaction_count = EXCLUDED.transaction_count,
			successful_tx_count = EXCLUDED.successful_tx_count,
			failed_tx_count = EXCLUDED.failed_tx_count,
			contract_creation_count = EXCLUDED.contract_creation_count,
			unique_senders = EXCLUDED.unique_senders,
			unique_receivers = EXCLUDED.unique_receivers,
			new_addresses = EXCLUDED.new_addresses,
			total_value_transferred = EXCLUDED.total_value_transferred,
			avg_value_per_tx = EXCLUDED.avg_value_per_tx,
			total_gas_used = EXCLUDED.total_gas_used,
			avg_gas_per_tx = EXCLUDED.avg_gas_per_tx,
			avg_gas_price = EXCLUDED.avg_gas_price,
			min_gas_price = EXCLUDED.min_gas_price,
			max_gas_price = EXCLUDED.max_gas_price,
			avg_base_fee = EXCLUDED.avg_base_fee,
			total_fees_burned = EXCLUDED.total_fees_burned,
			token_transfer_count = EXCLUDED.token_transfer_count,
			unique_tokens_transferred = EXCLUDED.unique_tokens_transferred,
			nft_transfer_count = EXCLUDED.nft_transfer_count,
			nft_mint_count = EXCLUDED.nft_mint_count,
			unique_nft_collections = EXCLUDED.unique_nft_collections,
			contract_deploy_count = EXCLUDED.contract_deploy_count,
			verified_contracts_count = EXCLUDED.verified_contracts_count,
			contract_call_count = EXCLUDED.contract_call_count,
			updated_at = NOW()
		RETURNING id`

	return r.db.QueryRow(ctx, query,
		s.Network, s.Date, s.BlockCount, s.FirstBlock, s.LastBlock, s.AvgBlockTime,
		s.TransactionCount, s.SuccessfulTxCount, s.FailedTxCount, s.ContractCreationCount,
		s.UniqueSenders, s.UniqueReceivers, s.NewAddresses,
		s.TotalValueTransferred, s.AvgValuePerTx,
		s.TotalGasUsed, s.AvgGasPerTx, s.AvgGasPrice, s.MinGasPrice, s.MaxGasPrice,
		s.AvgBaseFee, s.TotalFeesBurned,
		s.TokenTransferCount, s.UniqueTokensTransferred,
		s.NFTTransferCount, s.NFTMintCount, s.UniqueNFTCollections,
		s.ContractDeployCount, s.VerifiedContractsCount, s.ContractCallCount,
	).Scan(&s.ID)
}

// ============================================================================
// HOURLY STATS
// ============================================================================

// GetHourlyStats retrieves hourly stats for a time range
func (r *Repository) GetHourlyStats(ctx context.Context, network string, startTime, endTime time.Time) ([]*HourlyStats, error) {
	query := `
		SELECT id, network, hour, block_count, transaction_count, unique_addresses,
			total_gas_used, avg_gas_price, token_transfer_count, nft_transfer_count,
			total_value_transferred, created_at
		FROM hourly_stats
		WHERE network = $1 AND hour >= $2 AND hour <= $3
		ORDER BY hour DESC`

	rows, err := r.db.Query(ctx, query, network, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("get hourly stats: %w", err)
	}
	defer rows.Close()

	var stats []*HourlyStats
	for rows.Next() {
		var s HourlyStats
		if err := rows.Scan(
			&s.ID, &s.Network, &s.Hour, &s.BlockCount, &s.TransactionCount, &s.UniqueAddresses,
			&s.TotalGasUsed, &s.AvgGasPrice, &s.TokenTransferCount, &s.NFTTransferCount,
			&s.TotalValueTransferred, &s.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan hourly stats: %w", err)
		}
		stats = append(stats, &s)
	}

	return stats, nil
}

// UpsertHourlyStats creates or updates hourly stats
func (r *Repository) UpsertHourlyStats(ctx context.Context, s *HourlyStats) error {
	query := `
		INSERT INTO hourly_stats (
			network, hour, block_count, transaction_count, unique_addresses,
			total_gas_used, avg_gas_price, token_transfer_count, nft_transfer_count,
			total_value_transferred
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (network, hour) DO UPDATE SET
			block_count = EXCLUDED.block_count,
			transaction_count = EXCLUDED.transaction_count,
			unique_addresses = EXCLUDED.unique_addresses,
			total_gas_used = EXCLUDED.total_gas_used,
			avg_gas_price = EXCLUDED.avg_gas_price,
			token_transfer_count = EXCLUDED.token_transfer_count,
			nft_transfer_count = EXCLUDED.nft_transfer_count,
			total_value_transferred = EXCLUDED.total_value_transferred
		RETURNING id`

	return r.db.QueryRow(ctx, query,
		s.Network, s.Hour, s.BlockCount, s.TransactionCount, s.UniqueAddresses,
		s.TotalGasUsed, s.AvgGasPrice, s.TokenTransferCount, s.NFTTransferCount,
		s.TotalValueTransferred,
	).Scan(&s.ID)
}

// ============================================================================
// GAS PRICES
// ============================================================================

// GetLatestGasPrice retrieves the latest gas price for a network
func (r *Repository) GetLatestGasPrice(ctx context.Context, network string) (*GasPrice, error) {
	query := `
		SELECT id, network, timestamp, slow, standard, fast, instant,
			base_fee, priority_fee_slow, priority_fee_standard, priority_fee_fast,
			block_number, pending_tx_count, created_at
		FROM gas_prices
		WHERE network = $1
		ORDER BY timestamp DESC
		LIMIT 1`

	var g GasPrice
	err := r.db.QueryRow(ctx, query, network).Scan(
		&g.ID, &g.Network, &g.Timestamp, &g.Slow, &g.Standard, &g.Fast, &g.Instant,
		&g.BaseFee, &g.PriorityFeeSlow, &g.PriorityFeeStandard, &g.PriorityFeeFast,
		&g.BlockNumber, &g.PendingTxCount, &g.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get latest gas price: %w", err)
	}

	return &g, nil
}

// GetGasPriceHistory retrieves gas price history for a time range
func (r *Repository) GetGasPriceHistory(ctx context.Context, network string, startTime, endTime time.Time, limit int) ([]*GasPrice, error) {
	query := `
		SELECT id, network, timestamp, slow, standard, fast, instant,
			base_fee, priority_fee_slow, priority_fee_standard, priority_fee_fast,
			block_number, pending_tx_count, created_at
		FROM gas_prices
		WHERE network = $1 AND timestamp >= $2 AND timestamp <= $3
		ORDER BY timestamp DESC
		LIMIT $4`

	rows, err := r.db.Query(ctx, query, network, startTime, endTime, limit)
	if err != nil {
		return nil, fmt.Errorf("get gas price history: %w", err)
	}
	defer rows.Close()

	var prices []*GasPrice
	for rows.Next() {
		var g GasPrice
		if err := rows.Scan(
			&g.ID, &g.Network, &g.Timestamp, &g.Slow, &g.Standard, &g.Fast, &g.Instant,
			&g.BaseFee, &g.PriorityFeeSlow, &g.PriorityFeeStandard, &g.PriorityFeeFast,
			&g.BlockNumber, &g.PendingTxCount, &g.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan gas price: %w", err)
		}
		prices = append(prices, &g)
	}

	return prices, nil
}

// InsertGasPrice records a new gas price entry
func (r *Repository) InsertGasPrice(ctx context.Context, g *GasPrice) error {
	query := `
		INSERT INTO gas_prices (
			network, timestamp, slow, standard, fast, instant,
			base_fee, priority_fee_slow, priority_fee_standard, priority_fee_fast,
			block_number, pending_tx_count
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (network, timestamp) DO UPDATE SET
			slow = EXCLUDED.slow,
			standard = EXCLUDED.standard,
			fast = EXCLUDED.fast,
			instant = EXCLUDED.instant,
			base_fee = EXCLUDED.base_fee,
			priority_fee_slow = EXCLUDED.priority_fee_slow,
			priority_fee_standard = EXCLUDED.priority_fee_standard,
			priority_fee_fast = EXCLUDED.priority_fee_fast,
			block_number = EXCLUDED.block_number,
			pending_tx_count = EXCLUDED.pending_tx_count
		RETURNING id`

	return r.db.QueryRow(ctx, query,
		g.Network, g.Timestamp, g.Slow, g.Standard, g.Fast, g.Instant,
		g.BaseFee, g.PriorityFeeSlow, g.PriorityFeeStandard, g.PriorityFeeFast,
		g.BlockNumber, g.PendingTxCount,
	).Scan(&g.ID)
}

// ============================================================================
// NETWORK OVERVIEW
// ============================================================================

// GetNetworkOverview retrieves the overview for a network
func (r *Repository) GetNetworkOverview(ctx context.Context, network string) (*NetworkOverview, error) {
	query := `
		SELECT id, network, latest_block, latest_block_time, pending_tx_count,
			total_blocks, total_transactions, total_addresses, total_contracts,
			total_tokens, total_nft_collections,
			tx_count_24h, active_addresses_24h, gas_used_24h, avg_gas_price_24h,
			token_transfers_24h, nft_transfers_24h,
			gas_price_slow, gas_price_standard, gas_price_fast, base_fee,
			chain_id, native_currency, native_currency_decimals, updated_at
		FROM network_overview
		WHERE network = $1`

	var o NetworkOverview
	err := r.db.QueryRow(ctx, query, network).Scan(
		&o.ID, &o.Network, &o.LatestBlock, &o.LatestBlockTime, &o.PendingTxCount,
		&o.TotalBlocks, &o.TotalTransactions, &o.TotalAddresses, &o.TotalContracts,
		&o.TotalTokens, &o.TotalNFTCollections,
		&o.TxCount24h, &o.ActiveAddresses24h, &o.GasUsed24h, &o.AvgGasPrice24h,
		&o.TokenTransfers24h, &o.NFTTransfers24h,
		&o.GasPriceSlow, &o.GasPriceStandard, &o.GasPriceFast, &o.BaseFee,
		&o.ChainID, &o.NativeCurrency, &o.NativeCurrencyDecimals, &o.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get network overview: %w", err)
	}

	return &o, nil
}

// UpdateNetworkOverview updates the network overview
func (r *Repository) UpdateNetworkOverview(ctx context.Context, o *NetworkOverview) error {
	query := `
		UPDATE network_overview SET
			latest_block = $2,
			latest_block_time = $3,
			pending_tx_count = $4,
			total_blocks = $5,
			total_transactions = $6,
			total_addresses = $7,
			total_contracts = $8,
			total_tokens = $9,
			total_nft_collections = $10,
			tx_count_24h = $11,
			active_addresses_24h = $12,
			gas_used_24h = $13,
			avg_gas_price_24h = $14,
			token_transfers_24h = $15,
			nft_transfers_24h = $16,
			gas_price_slow = $17,
			gas_price_standard = $18,
			gas_price_fast = $19,
			base_fee = $20,
			updated_at = NOW()
		WHERE network = $1`

	_, err := r.db.Exec(ctx, query,
		o.Network, o.LatestBlock, o.LatestBlockTime, o.PendingTxCount,
		o.TotalBlocks, o.TotalTransactions, o.TotalAddresses, o.TotalContracts,
		o.TotalTokens, o.TotalNFTCollections,
		o.TxCount24h, o.ActiveAddresses24h, o.GasUsed24h, o.AvgGasPrice24h,
		o.TokenTransfers24h, o.NFTTransfers24h,
		o.GasPriceSlow, o.GasPriceStandard, o.GasPriceFast, o.BaseFee,
	)
	return err
}

// ============================================================================
// TOP TOKENS
// ============================================================================

// GetTopTokens retrieves top tokens for a date
func (r *Repository) GetTopTokens(ctx context.Context, network string, date time.Time, limit int) ([]*TopToken, error) {
	query := `
		SELECT id, network, date, rank, token_address, token_name, token_symbol,
			transfer_count, unique_holders, volume, created_at
		FROM top_tokens
		WHERE network = $1 AND date = $2
		ORDER BY rank
		LIMIT $3`

	rows, err := r.db.Query(ctx, query, network, date, limit)
	if err != nil {
		return nil, fmt.Errorf("get top tokens: %w", err)
	}
	defer rows.Close()

	var tokens []*TopToken
	for rows.Next() {
		var t TopToken
		if err := rows.Scan(
			&t.ID, &t.Network, &t.Date, &t.Rank, &t.TokenAddress, &t.TokenName, &t.TokenSymbol,
			&t.TransferCount, &t.UniqueHolders, &t.Volume, &t.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan top token: %w", err)
		}
		tokens = append(tokens, &t)
	}

	return tokens, nil
}

// UpsertTopToken creates or updates a top token entry
func (r *Repository) UpsertTopToken(ctx context.Context, t *TopToken) error {
	query := `
		INSERT INTO top_tokens (
			network, date, rank, token_address, token_name, token_symbol,
			transfer_count, unique_holders, volume
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (network, date, rank) DO UPDATE SET
			token_address = EXCLUDED.token_address,
			token_name = EXCLUDED.token_name,
			token_symbol = EXCLUDED.token_symbol,
			transfer_count = EXCLUDED.transfer_count,
			unique_holders = EXCLUDED.unique_holders,
			volume = EXCLUDED.volume
		RETURNING id`

	return r.db.QueryRow(ctx, query,
		t.Network, t.Date, t.Rank, t.TokenAddress, t.TokenName, t.TokenSymbol,
		t.TransferCount, t.UniqueHolders, t.Volume,
	).Scan(&t.ID)
}

// ============================================================================
// TOP CONTRACTS
// ============================================================================

// GetTopContracts retrieves top contracts for a date
func (r *Repository) GetTopContracts(ctx context.Context, network string, date time.Time, limit int) ([]*TopContract, error) {
	query := `
		SELECT id, network, date, rank, contract_address, contract_name,
			call_count, unique_callers, gas_used, created_at
		FROM top_contracts
		WHERE network = $1 AND date = $2
		ORDER BY rank
		LIMIT $3`

	rows, err := r.db.Query(ctx, query, network, date, limit)
	if err != nil {
		return nil, fmt.Errorf("get top contracts: %w", err)
	}
	defer rows.Close()

	var contracts []*TopContract
	for rows.Next() {
		var c TopContract
		if err := rows.Scan(
			&c.ID, &c.Network, &c.Date, &c.Rank, &c.ContractAddress, &c.ContractName,
			&c.CallCount, &c.UniqueCallers, &c.GasUsed, &c.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan top contract: %w", err)
		}
		contracts = append(contracts, &c)
	}

	return contracts, nil
}

// UpsertTopContract creates or updates a top contract entry
func (r *Repository) UpsertTopContract(ctx context.Context, c *TopContract) error {
	query := `
		INSERT INTO top_contracts (
			network, date, rank, contract_address, contract_name,
			call_count, unique_callers, gas_used
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (network, date, rank) DO UPDATE SET
			contract_address = EXCLUDED.contract_address,
			contract_name = EXCLUDED.contract_name,
			call_count = EXCLUDED.call_count,
			unique_callers = EXCLUDED.unique_callers,
			gas_used = EXCLUDED.gas_used
		RETURNING id`

	return r.db.QueryRow(ctx, query,
		c.Network, c.Date, c.Rank, c.ContractAddress, c.ContractName,
		c.CallCount, c.UniqueCallers, c.GasUsed,
	).Scan(&c.ID)
}

// ============================================================================
// AGGREGATION QUERIES
// ============================================================================

// AggregateDailyStats triggers the daily stats aggregation function
func (r *Repository) AggregateDailyStats(ctx context.Context, network string, date time.Time) error {
	_, err := r.db.Exec(ctx, "SELECT aggregate_daily_stats($1, $2)", network, date)
	return err
}

// RefreshNetworkOverview triggers the network overview update function
func (r *Repository) RefreshNetworkOverview(ctx context.Context, network string) error {
	_, err := r.db.Exec(ctx, "SELECT update_network_overview($1)", network)
	return err
}

// GetTransactionCountChart retrieves transaction count data for charts
func (r *Repository) GetTransactionCountChart(ctx context.Context, network string, days int) ([]ChartDataPoint, error) {
	query := `
		SELECT date, transaction_count
		FROM daily_stats
		WHERE network = $1 AND date >= CURRENT_DATE - $2::INT
		ORDER BY date`

	rows, err := r.db.Query(ctx, query, network, days)
	if err != nil {
		return nil, fmt.Errorf("get transaction count chart: %w", err)
	}
	defer rows.Close()

	var points []ChartDataPoint
	for rows.Next() {
		var date time.Time
		var count int64
		if err := rows.Scan(&date, &count); err != nil {
			return nil, fmt.Errorf("scan chart point: %w", err)
		}
		points = append(points, ChartDataPoint{
			Timestamp: date,
			Value:     float64(count),
		})
	}

	return points, nil
}

// GetGasPriceChart retrieves gas price data for charts
func (r *Repository) GetGasPriceChart(ctx context.Context, network string, hours int) ([]ChartDataPoint, error) {
	query := `
		SELECT timestamp, standard
		FROM gas_prices
		WHERE network = $1 AND timestamp >= NOW() - ($2 || ' hours')::INTERVAL
		ORDER BY timestamp`

	rows, err := r.db.Query(ctx, query, network, hours)
	if err != nil {
		return nil, fmt.Errorf("get gas price chart: %w", err)
	}
	defer rows.Close()

	var points []ChartDataPoint
	for rows.Next() {
		var timestamp time.Time
		var price *int64
		if err := rows.Scan(&timestamp, &price); err != nil {
			return nil, fmt.Errorf("scan chart point: %w", err)
		}
		value := float64(0)
		if price != nil {
			value = WeiToGwei(*price)
		}
		points = append(points, ChartDataPoint{
			Timestamp: timestamp,
			Value:     value,
		})
	}

	return points, nil
}

// GetActiveAddressesChart retrieves active addresses data for charts
func (r *Repository) GetActiveAddressesChart(ctx context.Context, network string, days int) ([]ChartDataPoint, error) {
	query := `
		SELECT date, unique_senders + unique_receivers as active
		FROM daily_stats
		WHERE network = $1 AND date >= CURRENT_DATE - $2::INT
		ORDER BY date`

	rows, err := r.db.Query(ctx, query, network, days)
	if err != nil {
		return nil, fmt.Errorf("get active addresses chart: %w", err)
	}
	defer rows.Close()

	var points []ChartDataPoint
	for rows.Next() {
		var date time.Time
		var count int64
		if err := rows.Scan(&date, &count); err != nil {
			return nil, fmt.Errorf("scan chart point: %w", err)
		}
		points = append(points, ChartDataPoint{
			Timestamp: date,
			Value:     float64(count),
		})
	}

	return points, nil
}
