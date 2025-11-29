// Package explorer provides blockchain explorer functionality
package explorer

import (
	"math/big"
	"time"
)

// Block represents an indexed blockchain block
type Block struct {
	ID               int64      `json:"-" db:"id"`
	Network          string     `json:"network" db:"network"`
	BlockNumber      int64      `json:"blockNumber" db:"block_number"`
	BlockHash        string     `json:"hash" db:"block_hash"`
	ParentHash       string     `json:"parentHash" db:"parent_hash"`
	Timestamp        time.Time  `json:"timestamp" db:"timestamp"`
	Miner            string     `json:"miner" db:"miner"`
	GasUsed          int64      `json:"gasUsed" db:"gas_used"`
	GasLimit         int64      `json:"gasLimit" db:"gas_limit"`
	BaseFeePerGas    *int64     `json:"baseFeePerGas,omitempty" db:"base_fee_per_gas"`
	TransactionCount int        `json:"transactionCount" db:"transaction_count"`
	Size             int        `json:"size" db:"size"`
	ExtraData        string     `json:"extraData,omitempty" db:"extra_data"`
	CreatedAt        time.Time  `json:"-" db:"created_at"`
}

// Transaction represents an indexed blockchain transaction
type Transaction struct {
	ID                   int64      `json:"-" db:"id"`
	Network              string     `json:"network" db:"network"`
	TxHash               string     `json:"hash" db:"tx_hash"`
	BlockNumber          int64      `json:"blockNumber" db:"block_number"`
	BlockHash            string     `json:"blockHash" db:"block_hash"`
	TxIndex              int        `json:"transactionIndex" db:"tx_index"`
	From                 string     `json:"from" db:"from_address"`
	To                   *string    `json:"to" db:"to_address"`
	Value                string     `json:"value" db:"value"` // stored as string for precision
	GasPrice             *int64     `json:"gasPrice,omitempty" db:"gas_price"`
	GasLimit             int64      `json:"gas" db:"gas_limit"`
	GasUsed              *int64     `json:"gasUsed,omitempty" db:"gas_used"`
	MaxFeePerGas         *int64     `json:"maxFeePerGas,omitempty" db:"max_fee_per_gas"`
	MaxPriorityFeePerGas *int64     `json:"maxPriorityFeePerGas,omitempty" db:"max_priority_fee_per_gas"`
	InputData            string     `json:"input" db:"input_data"`
	Nonce                int64      `json:"nonce" db:"nonce"`
	TxType               int        `json:"type" db:"tx_type"`
	Status               *int       `json:"status" db:"status"` // 0=failed, 1=success, nil=pending
	Timestamp            time.Time  `json:"timestamp" db:"timestamp"`
	ContractAddress      *string    `json:"contractAddress,omitempty" db:"contract_address"`
	ErrorMessage         *string    `json:"error,omitempty" db:"error_message"`
	CreatedAt            time.Time  `json:"-" db:"created_at"`
}

// Address represents an indexed blockchain address/account
type Address struct {
	ID                 int64      `json:"-" db:"id"`
	Network            string     `json:"network" db:"network"`
	Address            string     `json:"address" db:"address"`
	Balance            string     `json:"balance" db:"balance"` // stored as string for precision
	TxCount            int64      `json:"transactionCount" db:"tx_count"`
	IsContract         bool       `json:"isContract" db:"is_contract"`
	ContractCreator    *string    `json:"contractCreator,omitempty" db:"contract_creator"`
	ContractCreationTx *string    `json:"creationTxHash,omitempty" db:"contract_creation_tx"`
	CodeHash           *string    `json:"codeHash,omitempty" db:"code_hash"`
	FirstSeenAt        *time.Time `json:"firstSeen,omitempty" db:"first_seen_at"`
	LastSeenAt         *time.Time `json:"lastSeen,omitempty" db:"last_seen_at"`
	Label              *string    `json:"label,omitempty" db:"label"`
	Tags               []string   `json:"tags,omitempty" db:"tags"`
	UpdatedAt          time.Time  `json:"-" db:"updated_at"`
}

// EventLog represents an indexed contract event log
type EventLog struct {
	ID              int64           `json:"-" db:"id"`
	Network         string          `json:"network" db:"network"`
	TxHash          string          `json:"transactionHash" db:"tx_hash"`
	LogIndex        int             `json:"logIndex" db:"log_index"`
	BlockNumber     int64           `json:"blockNumber" db:"block_number"`
	ContractAddress string          `json:"address" db:"contract_address"`
	Topic0          *string         `json:"topic0,omitempty" db:"topic0"`
	Topic1          *string         `json:"topic1,omitempty" db:"topic1"`
	Topic2          *string         `json:"topic2,omitempty" db:"topic2"`
	Topic3          *string         `json:"topic3,omitempty" db:"topic3"`
	Data            string          `json:"data" db:"data"`
	Timestamp       time.Time       `json:"timestamp" db:"timestamp"`
	DecodedName     *string         `json:"eventName,omitempty" db:"decoded_name"`
	DecodedArgs     map[string]any  `json:"decodedArgs,omitempty" db:"decoded_args"`
	Removed         bool            `json:"removed" db:"removed"`
	CreatedAt       time.Time       `json:"-" db:"created_at"`
}

// InternalTransaction represents a trace/internal transaction
type InternalTransaction struct {
	ID           int64     `json:"-" db:"id"`
	Network      string    `json:"network" db:"network"`
	TxHash       string    `json:"transactionHash" db:"tx_hash"`
	TraceAddress string    `json:"traceAddress" db:"trace_address"`
	BlockNumber  int64     `json:"blockNumber" db:"block_number"`
	TraceType    string    `json:"type" db:"trace_type"`
	CallType     *string   `json:"callType,omitempty" db:"call_type"`
	From         string    `json:"from" db:"from_address"`
	To           *string   `json:"to,omitempty" db:"to_address"`
	Value        string    `json:"value" db:"value"`
	Gas          *int64    `json:"gas,omitempty" db:"gas"`
	GasUsed      *int64    `json:"gasUsed,omitempty" db:"gas_used"`
	Input        string    `json:"input,omitempty" db:"input_data"`
	Output       *string   `json:"output,omitempty" db:"output_data"`
	Error        *string   `json:"error,omitempty" db:"error"`
	Timestamp    time.Time `json:"timestamp" db:"timestamp"`
	CreatedAt    time.Time `json:"-" db:"created_at"`
}

// NetworkSyncState represents the indexer state for a network
type NetworkSyncState struct {
	ID               int       `json:"-" db:"id"`
	Network          string    `json:"network" db:"network"`
	LastIndexedBlock int64     `json:"lastIndexedBlock" db:"last_indexed_block"`
	LastIndexedAt    *time.Time `json:"lastIndexedAt,omitempty" db:"last_indexed_at"`
	IsSyncing        bool      `json:"isSyncing" db:"is_syncing"`
	SyncStartedAt    *time.Time `json:"syncStartedAt,omitempty" db:"sync_started_at"`
	BlocksBehind     int64     `json:"blocksBehind" db:"blocks_behind"`
	ErrorMessage     *string   `json:"error,omitempty" db:"error_message"`
	UpdatedAt        time.Time `json:"-" db:"updated_at"`
}

// NetworkStats represents aggregated network statistics
type NetworkStats struct {
	Network            string    `json:"network"`
	LatestBlock        int64     `json:"latestBlock"`
	TotalTransactions  int64     `json:"totalTransactions"`
	TotalAddresses     int64     `json:"totalAddresses"`
	TotalContracts     int64     `json:"totalContracts"`
	AvgBlockTime       float64   `json:"avgBlockTime"`
	AvgGasPrice        *big.Int  `json:"avgGasPrice"`
	TPS                float64   `json:"tps"` // transactions per second
	LastUpdated        time.Time `json:"lastUpdated"`
}

// Pagination options
type PaginationOptions struct {
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
	Offset   int `json:"-"`
}

// BlockFilter for querying blocks
type BlockFilter struct {
	Network   string
	FromBlock *int64
	ToBlock   *int64
	Miner     *string
	PaginationOptions
}

// TransactionFilter for querying transactions
type TransactionFilter struct {
	Network     string
	BlockNumber *int64
	FromAddress *string
	ToAddress   *string
	Status      *int // 0=failed, 1=success
	FromTime    *time.Time
	ToTime      *time.Time
	PaginationOptions
}

// EventLogFilter for querying event logs
type EventLogFilter struct {
	Network         string
	ContractAddress *string
	Topic0          *string
	FromBlock       *int64
	ToBlock         *int64
	PaginationOptions
}

// AddressFilter for querying addresses
type AddressFilter struct {
	Network    string
	IsContract *bool
	MinBalance *big.Int
	MaxBalance *big.Int
	Label      *string
	PaginationOptions
}

// SearchResult represents a universal search result
type SearchResult struct {
	Type    string      `json:"type"` // block, transaction, address, token, contract
	Network string      `json:"network"`
	Data    interface{} `json:"data"`
}

// SearchResults contains multiple search results
type SearchResults struct {
	Query   string         `json:"query"`
	Results []SearchResult `json:"results"`
	Total   int            `json:"total"`
}

// ListResult is a generic paginated list response
type ListResult[T any] struct {
	Items      []T   `json:"items"`
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	PageSize   int   `json:"pageSize"`
	TotalPages int   `json:"totalPages"`
}

// NewPaginationOptions creates pagination options with defaults
func NewPaginationOptions(page, pageSize int) PaginationOptions {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return PaginationOptions{
		Page:     page,
		PageSize: pageSize,
		Offset:   (page - 1) * pageSize,
	}
}
