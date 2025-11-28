package monitor

import (
	"math/big"
	"time"
)

// Network represents a blockchain network
type Network struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	ChainID   int64  `json:"chain_id"`
	RPCURL    string `json:"rpc_url"`
	WSURL     string `json:"ws_url,omitempty"`
	Explorer  string `json:"explorer"`
	Symbol    string `json:"symbol"`
	Supported bool   `json:"supported"`
}

// SupportedNetworks list of supported blockchain networks
var SupportedNetworks = []Network{
	{Name: "Ethereum", ChainID: 1, Symbol: "ETH", Explorer: "https://etherscan.io", Supported: true},
	{Name: "Polygon", ChainID: 137, Symbol: "MATIC", Explorer: "https://polygonscan.com", Supported: true},
	{Name: "Arbitrum", ChainID: 42161, Symbol: "ETH", Explorer: "https://arbiscan.io", Supported: true},
	{Name: "Optimism", ChainID: 10, Symbol: "ETH", Explorer: "https://optimistic.etherscan.io", Supported: true},
	{Name: "BSC", ChainID: 56, Symbol: "BNB", Explorer: "https://bscscan.com", Supported: true},
	{Name: "Avalanche", ChainID: 43114, Symbol: "AVAX", Explorer: "https://snowtrace.io", Supported: true},
	{Name: "Base", ChainID: 8453, Symbol: "ETH", Explorer: "https://basescan.org", Supported: true},
	// Testnets
	{Name: "Goerli", ChainID: 5, Symbol: "ETH", Explorer: "https://goerli.etherscan.io", Supported: true},
	{Name: "Sepolia", ChainID: 11155111, Symbol: "ETH", Explorer: "https://sepolia.etherscan.io", Supported: true},
	{Name: "Mumbai", ChainID: 80001, Symbol: "MATIC", Explorer: "https://mumbai.polygonscan.com", Supported: true},
}

// Contract represents a monitored smart contract
type Contract struct {
	ID          string                 `json:"id"`
	Address     string                 `json:"address"`
	Name        string                 `json:"name"`
	ChainID     int64                  `json:"chain_id"`
	Network     string                 `json:"network"`
	ABI         string                 `json:"abi,omitempty"`
	Verified    bool                   `json:"verified"`
	SourceCode  string                 `json:"source_code,omitempty"`
	Balance     *big.Int               `json:"balance,omitempty"`
	TxCount     int64                  `json:"tx_count"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	LastEventAt *time.Time             `json:"last_event_at,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	Status      ContractStatus         `json:"status"`
}

// ContractStatus represents contract monitoring status
type ContractStatus string

const (
	ContractStatusActive   ContractStatus = "active"
	ContractStatusPaused   ContractStatus = "paused"
	ContractStatusError    ContractStatus = "error"
	ContractStatusArchived ContractStatus = "archived"
)

// ContractEvent represents an on-chain event
type ContractEvent struct {
	ID              string                 `json:"id"`
	ContractAddress string                 `json:"contract_address"`
	ChainID         int64                  `json:"chain_id"`
	TxHash          string                 `json:"tx_hash"`
	BlockNumber     uint64                 `json:"block_number"`
	BlockHash       string                 `json:"block_hash"`
	LogIndex        uint                   `json:"log_index"`
	EventName       string                 `json:"event_name"`
	EventSignature  string                 `json:"event_signature"`
	Topics          []string               `json:"topics"`
	Data            string                 `json:"data"`
	DecodedArgs     map[string]interface{} `json:"decoded_args,omitempty"`
	Timestamp       time.Time              `json:"timestamp"`
	Processed       bool                   `json:"processed"`
}

// Transaction represents a blockchain transaction
type Transaction struct {
	Hash         string                 `json:"hash"`
	ChainID      int64                  `json:"chain_id"`
	BlockNumber  uint64                 `json:"block_number"`
	From         string                 `json:"from"`
	To           string                 `json:"to"`
	Value        *big.Int               `json:"value"`
	GasUsed      uint64                 `json:"gas_used"`
	GasPrice     *big.Int               `json:"gas_price"`
	Input        string                 `json:"input"`
	Status       bool                   `json:"status"`
	Timestamp    time.Time              `json:"timestamp"`
	MethodID     string                 `json:"method_id,omitempty"`
	MethodName   string                 `json:"method_name,omitempty"`
	DecodedInput map[string]interface{} `json:"decoded_input,omitempty"`
	Events       []*ContractEvent       `json:"events,omitempty"`
	InternalTxs  []*InternalTransaction `json:"internal_txs,omitempty"`
	Trace        *TransactionTrace      `json:"trace,omitempty"`
}

// InternalTransaction represents an internal call
type InternalTransaction struct {
	Type      string   `json:"type"`
	From      string   `json:"from"`
	To        string   `json:"to"`
	Value     *big.Int `json:"value"`
	GasUsed   uint64   `json:"gas_used"`
	Input     string   `json:"input"`
	Output    string   `json:"output"`
	Error     string   `json:"error,omitempty"`
	CallDepth int      `json:"call_depth"`
}

// TransactionTrace represents detailed transaction trace
type TransactionTrace struct {
	TxHash    string           `json:"tx_hash"`
	Status    string           `json:"status"`
	GasUsed   uint64           `json:"gas_used"`
	Output    string           `json:"output"`
	Error     string           `json:"error,omitempty"`
	Calls     []*TraceCall     `json:"calls,omitempty"`
	StateRead []*StorageRead   `json:"state_read,omitempty"`
	StateWrite []*StorageWrite `json:"state_write,omitempty"`
}

// TraceCall represents a call in transaction trace
type TraceCall struct {
	Type      string       `json:"type"` // CALL, DELEGATECALL, STATICCALL, CREATE, etc.
	From      string       `json:"from"`
	To        string       `json:"to"`
	Value     string       `json:"value"`
	Gas       uint64       `json:"gas"`
	GasUsed   uint64       `json:"gas_used"`
	Input     string       `json:"input"`
	Output    string       `json:"output"`
	Error     string       `json:"error,omitempty"`
	Depth     int          `json:"depth"`
	Calls     []*TraceCall `json:"calls,omitempty"`
}

// StorageRead represents a storage read operation
type StorageRead struct {
	Address string `json:"address"`
	Slot    string `json:"slot"`
	Value   string `json:"value"`
}

// StorageWrite represents a storage write operation
type StorageWrite struct {
	Address  string `json:"address"`
	Slot     string `json:"slot"`
	OldValue string `json:"old_value"`
	NewValue string `json:"new_value"`
}

// GasProfile represents gas usage statistics for a contract
type GasProfile struct {
	ContractAddress string              `json:"contract_address"`
	ChainID         int64               `json:"chain_id"`
	Period          string              `json:"period"`
	TotalGas        uint64              `json:"total_gas"`
	TotalCost       *big.Int            `json:"total_cost"`
	TxCount         int                 `json:"tx_count"`
	ByFunction      map[string]*GasInfo `json:"by_function"`
	Suggestions     []GasSuggestion     `json:"suggestions"`
	GeneratedAt     time.Time           `json:"generated_at"`
}

// GasInfo contains gas statistics for a function
type GasInfo struct {
	Function  string   `json:"function"`
	AvgGas    uint64   `json:"avg_gas"`
	MinGas    uint64   `json:"min_gas"`
	MaxGas    uint64   `json:"max_gas"`
	TotalGas  uint64   `json:"total_gas"`
	CallCount int      `json:"call_count"`
	TotalCost *big.Int `json:"total_cost"`
}

// GasSuggestion represents a gas optimization suggestion
type GasSuggestion struct {
	Function      string `json:"function"`
	Issue         string `json:"issue"`
	Suggestion    string `json:"suggestion"`
	PotentialSave uint64 `json:"potential_save"`
	Priority      string `json:"priority"` // low, medium, high
}

// AlertRule defines an alert trigger condition
type AlertRule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Contract    string                 `json:"contract,omitempty"`
	ChainID     int64                  `json:"chain_id"`
	Type        AlertType              `json:"type"`
	Condition   AlertCondition         `json:"condition"`
	Channels    []string               `json:"channels"`
	Enabled     bool                   `json:"enabled"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// AlertType defines the type of alert
type AlertType string

const (
	AlertTypeEvent         AlertType = "event"
	AlertTypeLargeTransfer AlertType = "large_transfer"
	AlertTypeFailedTx      AlertType = "failed_tx"
	AlertTypeGasSpike      AlertType = "gas_spike"
	AlertTypeLowBalance    AlertType = "low_balance"
	AlertTypeHighBalance   AlertType = "high_balance"
	AlertTypeWhale         AlertType = "whale"
	AlertTypeCustom        AlertType = "custom"
)

// AlertCondition defines the condition for triggering an alert
type AlertCondition struct {
	Event       string  `json:"event,omitempty"`
	Threshold   float64 `json:"threshold,omitempty"`
	Operator    string  `json:"operator,omitempty"` // >, <, ==, >=, <=
	Parameter   string  `json:"parameter,omitempty"`
	Expression  string  `json:"expression,omitempty"` // For custom conditions
}

// Alert represents a triggered alert
type Alert struct {
	ID          string                 `json:"id"`
	RuleID      string                 `json:"rule_id"`
	RuleName    string                 `json:"rule_name"`
	Type        AlertType              `json:"type"`
	Severity    string                 `json:"severity"`
	Title       string                 `json:"title"`
	Message     string                 `json:"message"`
	Contract    string                 `json:"contract,omitempty"`
	ChainID     int64                  `json:"chain_id"`
	TxHash      string                 `json:"tx_hash,omitempty"`
	BlockNumber uint64                 `json:"block_number,omitempty"`
	Data        map[string]interface{} `json:"data,omitempty"`
	FiredAt     time.Time              `json:"fired_at"`
	Status      AlertStatus            `json:"status"`
	AckedAt     *time.Time             `json:"acknowledged_at,omitempty"`
	AckedBy     string                 `json:"acknowledged_by,omitempty"`
}

// AlertStatus represents alert status
type AlertStatus string

const (
	AlertStatusOpen   AlertStatus = "open"
	AlertStatusAcked  AlertStatus = "acknowledged"
	AlertStatusClosed AlertStatus = "closed"
)

// CDCSyncConfig contains CDC synchronization configuration
type CDCSyncConfig struct {
	ID              string              `json:"id"`
	Enabled         bool                `json:"enabled"`
	Contract        string              `json:"contract"`
	ChainID         int64               `json:"chain_id"`
	Events          []EventMapping      `json:"events"`
	BalanceSync     bool                `json:"balance_sync"`
	SyncInterval    time.Duration       `json:"sync_interval"`
	TargetDatabase  string              `json:"target_database"`
	TargetTable     string              `json:"target_table"`
	WalletMapping   *WalletMapping      `json:"wallet_mapping,omitempty"`
	CreatedAt       time.Time           `json:"created_at"`
	LastSyncAt      *time.Time          `json:"last_sync_at,omitempty"`
}

// EventMapping maps on-chain events to database columns
type EventMapping struct {
	EventName  string            `json:"event_name"`
	TableName  string            `json:"table_name"`
	FieldMap   map[string]string `json:"field_map"` // event_arg -> db_column
	OnConflict string            `json:"on_conflict,omitempty"` // update, ignore, fail
}

// WalletMapping maps wallet addresses to user IDs
type WalletMapping struct {
	UserTable   string `json:"user_table"`
	WalletField string `json:"wallet_field"`
	UserIDField string `json:"user_id_field"`
}

// SyncStatus represents CDC sync status
type SyncStatus struct {
	ConfigID        string    `json:"config_id"`
	Status          string    `json:"status"` // running, stopped, error
	LastSyncAt      time.Time `json:"last_sync_at"`
	LastBlockNumber uint64    `json:"last_block_number"`
	EventsProcessed int64     `json:"events_processed"`
	ErrorCount      int       `json:"error_count"`
	LastError       string    `json:"last_error,omitempty"`
}
