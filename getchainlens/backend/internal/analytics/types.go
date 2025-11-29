// Package analytics provides network analytics and statistics
package analytics

import (
	"time"
)

// DailyStats represents daily aggregated statistics for a network
type DailyStats struct {
	ID      int64     `json:"-" db:"id"`
	Network string    `json:"network" db:"network"`
	Date    time.Time `json:"date" db:"date"`

	// Block metrics
	BlockCount   int    `json:"blockCount" db:"block_count"`
	FirstBlock   *int64 `json:"firstBlock,omitempty" db:"first_block"`
	LastBlock    *int64 `json:"lastBlock,omitempty" db:"last_block"`
	AvgBlockTime *float64 `json:"avgBlockTime,omitempty" db:"avg_block_time"`

	// Transaction metrics
	TransactionCount      int64 `json:"transactionCount" db:"transaction_count"`
	SuccessfulTxCount     int64 `json:"successfulTxCount" db:"successful_tx_count"`
	FailedTxCount         int64 `json:"failedTxCount" db:"failed_tx_count"`
	ContractCreationCount int   `json:"contractCreationCount" db:"contract_creation_count"`

	// Address metrics
	UniqueSenders   int64 `json:"uniqueSenders" db:"unique_senders"`
	UniqueReceivers int64 `json:"uniqueReceivers" db:"unique_receivers"`
	NewAddresses    int64 `json:"newAddresses" db:"new_addresses"`

	// Value metrics
	TotalValueTransferred string  `json:"totalValueTransferred" db:"total_value_transferred"`
	AvgValuePerTx         *string `json:"avgValuePerTx,omitempty" db:"avg_value_per_tx"`

	// Gas metrics
	TotalGasUsed   string  `json:"totalGasUsed" db:"total_gas_used"`
	AvgGasPerTx    *int64  `json:"avgGasPerTx,omitempty" db:"avg_gas_per_tx"`
	AvgGasPrice    *string `json:"avgGasPrice,omitempty" db:"avg_gas_price"`
	MinGasPrice    *string `json:"minGasPrice,omitempty" db:"min_gas_price"`
	MaxGasPrice    *string `json:"maxGasPrice,omitempty" db:"max_gas_price"`
	AvgBaseFee     *string `json:"avgBaseFee,omitempty" db:"avg_base_fee"`
	TotalFeesBurned string `json:"totalFeesBurned" db:"total_fees_burned"`

	// Token metrics
	TokenTransferCount      int64 `json:"tokenTransferCount" db:"token_transfer_count"`
	UniqueTokensTransferred int   `json:"uniqueTokensTransferred" db:"unique_tokens_transferred"`

	// NFT metrics
	NFTTransferCount     int64 `json:"nftTransferCount" db:"nft_transfer_count"`
	NFTMintCount         int64 `json:"nftMintCount" db:"nft_mint_count"`
	UniqueNFTCollections int   `json:"uniqueNftCollections" db:"unique_nft_collections"`

	// Contract metrics
	ContractDeployCount     int `json:"contractDeployCount" db:"contract_deploy_count"`
	VerifiedContractsCount  int `json:"verifiedContractsCount" db:"verified_contracts_count"`
	ContractCallCount       int64 `json:"contractCallCount" db:"contract_call_count"`

	CreatedAt time.Time `json:"-" db:"created_at"`
	UpdatedAt time.Time `json:"-" db:"updated_at"`
}

// HourlyStats represents hourly aggregated statistics
type HourlyStats struct {
	ID      int64     `json:"-" db:"id"`
	Network string    `json:"network" db:"network"`
	Hour    time.Time `json:"hour" db:"hour"`

	BlockCount            int    `json:"blockCount" db:"block_count"`
	TransactionCount      int64  `json:"transactionCount" db:"transaction_count"`
	UniqueAddresses       int    `json:"uniqueAddresses" db:"unique_addresses"`
	TotalGasUsed          string `json:"totalGasUsed" db:"total_gas_used"`
	AvgGasPrice           *string `json:"avgGasPrice,omitempty" db:"avg_gas_price"`
	TokenTransferCount    int64  `json:"tokenTransferCount" db:"token_transfer_count"`
	NFTTransferCount      int64  `json:"nftTransferCount" db:"nft_transfer_count"`
	TotalValueTransferred string `json:"totalValueTransferred" db:"total_value_transferred"`

	CreatedAt time.Time `json:"-" db:"created_at"`
}

// GasPrice represents gas price data at a point in time
type GasPrice struct {
	ID        int64     `json:"-" db:"id"`
	Network   string    `json:"network" db:"network"`
	Timestamp time.Time `json:"timestamp" db:"timestamp"`

	// Gas prices in wei
	Slow     *int64 `json:"slow,omitempty" db:"slow"`
	Standard *int64 `json:"standard,omitempty" db:"standard"`
	Fast     *int64 `json:"fast,omitempty" db:"fast"`
	Instant  *int64 `json:"instant,omitempty" db:"instant"`

	// EIP-1559 data
	BaseFee             *int64 `json:"baseFee,omitempty" db:"base_fee"`
	PriorityFeeSlow     *int64 `json:"priorityFeeSlow,omitempty" db:"priority_fee_slow"`
	PriorityFeeStandard *int64 `json:"priorityFeeStandard,omitempty" db:"priority_fee_standard"`
	PriorityFeeFast     *int64 `json:"priorityFeeFast,omitempty" db:"priority_fee_fast"`

	// Block info
	BlockNumber    *int64 `json:"blockNumber,omitempty" db:"block_number"`
	PendingTxCount *int   `json:"pendingTxCount,omitempty" db:"pending_tx_count"`

	CreatedAt time.Time `json:"-" db:"created_at"`
}

// GasPriceEstimate represents current gas price estimates
type GasPriceEstimate struct {
	Network   string    `json:"network"`
	Timestamp time.Time `json:"timestamp"`

	// Prices in Gwei for easier consumption
	SlowGwei     float64 `json:"slowGwei"`
	StandardGwei float64 `json:"standardGwei"`
	FastGwei     float64 `json:"fastGwei"`
	InstantGwei  float64 `json:"instantGwei"`

	// EIP-1559 data in Gwei
	BaseFeeGwei         float64 `json:"baseFeeGwei"`
	PriorityFeeSlowGwei float64 `json:"priorityFeeSlowGwei"`
	PriorityFeeStdGwei  float64 `json:"priorityFeeStdGwei"`
	PriorityFeeFastGwei float64 `json:"priorityFeeFastGwei"`

	// Estimated times
	SlowTime     string `json:"slowTime"`     // e.g., "~10 min"
	StandardTime string `json:"standardTime"` // e.g., "~3 min"
	FastTime     string `json:"fastTime"`     // e.g., "~1 min"
	InstantTime  string `json:"instantTime"`  // e.g., "~15 sec"
}

// NetworkOverview represents current state of a network
type NetworkOverview struct {
	ID      int    `json:"-" db:"id"`
	Network string `json:"network" db:"network"`

	// Current state
	LatestBlock     *int64     `json:"latestBlock,omitempty" db:"latest_block"`
	LatestBlockTime *time.Time `json:"latestBlockTime,omitempty" db:"latest_block_time"`
	PendingTxCount  int        `json:"pendingTxCount" db:"pending_tx_count"`

	// Cumulative stats
	TotalBlocks         int64 `json:"totalBlocks" db:"total_blocks"`
	TotalTransactions   int64 `json:"totalTransactions" db:"total_transactions"`
	TotalAddresses      int64 `json:"totalAddresses" db:"total_addresses"`
	TotalContracts      int64 `json:"totalContracts" db:"total_contracts"`
	TotalTokens         int64 `json:"totalTokens" db:"total_tokens"`
	TotalNFTCollections int64 `json:"totalNftCollections" db:"total_nft_collections"`

	// 24h metrics
	TxCount24h          int64   `json:"txCount24h" db:"tx_count_24h"`
	ActiveAddresses24h  int64   `json:"activeAddresses24h" db:"active_addresses_24h"`
	GasUsed24h          string  `json:"gasUsed24h" db:"gas_used_24h"`
	AvgGasPrice24h      *string `json:"avgGasPrice24h,omitempty" db:"avg_gas_price_24h"`
	TokenTransfers24h   int64   `json:"tokenTransfers24h" db:"token_transfers_24h"`
	NFTTransfers24h     int64   `json:"nftTransfers24h" db:"nft_transfers_24h"`

	// Current gas prices
	GasPriceSlow     *int64 `json:"gasPriceSlow,omitempty" db:"gas_price_slow"`
	GasPriceStandard *int64 `json:"gasPriceStandard,omitempty" db:"gas_price_standard"`
	GasPriceFast     *int64 `json:"gasPriceFast,omitempty" db:"gas_price_fast"`
	BaseFee          *int64 `json:"baseFee,omitempty" db:"base_fee"`

	// Network info
	ChainID                 *int   `json:"chainId,omitempty" db:"chain_id"`
	NativeCurrency          string `json:"nativeCurrency" db:"native_currency"`
	NativeCurrencyDecimals  int    `json:"nativeCurrencyDecimals" db:"native_currency_decimals"`

	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}

// TopToken represents a top token entry
type TopToken struct {
	ID      int64     `json:"-" db:"id"`
	Network string    `json:"network" db:"network"`
	Date    time.Time `json:"date" db:"date"`
	Rank    int       `json:"rank" db:"rank"`

	TokenAddress  string  `json:"tokenAddress" db:"token_address"`
	TokenName     *string `json:"tokenName,omitempty" db:"token_name"`
	TokenSymbol   *string `json:"tokenSymbol,omitempty" db:"token_symbol"`

	TransferCount int64  `json:"transferCount" db:"transfer_count"`
	UniqueHolders int    `json:"uniqueHolders" db:"unique_holders"`
	Volume        string `json:"volume" db:"volume"`

	CreatedAt time.Time `json:"-" db:"created_at"`
}

// TopContract represents a top contract entry
type TopContract struct {
	ID      int64     `json:"-" db:"id"`
	Network string    `json:"network" db:"network"`
	Date    time.Time `json:"date" db:"date"`
	Rank    int       `json:"rank" db:"rank"`

	ContractAddress string  `json:"contractAddress" db:"contract_address"`
	ContractName    *string `json:"contractName,omitempty" db:"contract_name"`

	CallCount     int64  `json:"callCount" db:"call_count"`
	UniqueCallers int    `json:"uniqueCallers" db:"unique_callers"`
	GasUsed       string `json:"gasUsed" db:"gas_used"`

	CreatedAt time.Time `json:"-" db:"created_at"`
}

// AddressRanking represents an address ranking entry
type AddressRanking struct {
	ID          int64     `json:"-" db:"id"`
	Network     string    `json:"network" db:"network"`
	Date        time.Time `json:"date" db:"date"`
	RankingType string    `json:"rankingType" db:"ranking_type"` // balance, tx_count, gas_spent
	Rank        int       `json:"rank" db:"rank"`

	Address string `json:"address" db:"address"`
	Value   string `json:"value" db:"value"`

	CreatedAt time.Time `json:"-" db:"created_at"`
}

// ChartDataPoint represents a single point in a time series chart
type ChartDataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
	Label     string    `json:"label,omitempty"`
}

// ChartData represents chart data with metadata
type ChartData struct {
	Network    string           `json:"network"`
	MetricName string           `json:"metricName"`
	Period     string           `json:"period"` // "24h", "7d", "30d", "1y"
	DataPoints []ChartDataPoint `json:"dataPoints"`
}

// StatsFilter contains filter options for stats queries
type StatsFilter struct {
	Network   string
	StartDate time.Time
	EndDate   time.Time
	Interval  string // "hour", "day", "week", "month"
	Limit     int
}

// RankingType constants
const (
	RankingTypeBalance  = "balance"
	RankingTypeTxCount  = "tx_count"
	RankingTypeGasSpent = "gas_spent"
)

// Supported networks
var SupportedNetworks = []string{
	"ethereum",
	"polygon",
	"arbitrum",
	"optimism",
	"base",
	"bsc",
	"avalanche",
}

// Network chain IDs
var NetworkChainIDs = map[string]int{
	"ethereum":  1,
	"polygon":   137,
	"arbitrum":  42161,
	"optimism":  10,
	"base":      8453,
	"bsc":       56,
	"avalanche": 43114,
}

// Network native currencies
var NetworkNativeCurrencies = map[string]string{
	"ethereum":  "ETH",
	"polygon":   "MATIC",
	"arbitrum":  "ETH",
	"optimism":  "ETH",
	"base":      "ETH",
	"bsc":       "BNB",
	"avalanche": "AVAX",
}

// WeiToGwei converts wei to gwei
func WeiToGwei(wei int64) float64 {
	return float64(wei) / 1e9
}

// GweiToWei converts gwei to wei
func GweiToWei(gwei float64) int64 {
	return int64(gwei * 1e9)
}
