package sync

import (
	"context"
	"time"

	"getchainlens.com/chainlens/backend/internal/explorer"
)

// ExplorerSyncConfig configures what explorer data to sync via CDC
type ExplorerSyncConfig struct {
	ID             string `json:"id"`
	Enabled        bool   `json:"enabled"`
	TargetDatabase string `json:"target_database"`

	// Entity sync settings
	SyncBlocks       bool `json:"sync_blocks"`
	SyncTransactions bool `json:"sync_transactions"`
	SyncAddresses    bool `json:"sync_addresses"`
	SyncEventLogs    bool `json:"sync_event_logs"`

	// Table names (optional, defaults provided)
	BlocksTable       string `json:"blocks_table,omitempty"`
	TransactionsTable string `json:"transactions_table,omitempty"`
	AddressesTable    string `json:"addresses_table,omitempty"`
	EventLogsTable    string `json:"event_logs_table,omitempty"`

	// Filters
	Networks         []string `json:"networks,omitempty"`          // filter by network
	MinValue         string   `json:"min_value,omitempty"`         // min tx value to sync
	ContractAddresses []string `json:"contract_addresses,omitempty"` // filter logs by contract

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ExplorerSync syncs explorer data to external databases via CDC
type ExplorerSync struct {
	config    *ExplorerSyncConfig
	cdcClient CDCClient
}

// NewExplorerSync creates a new explorer sync instance
func NewExplorerSync(config *ExplorerSyncConfig, client CDCClient) *ExplorerSync {
	// Set default table names
	if config.BlocksTable == "" {
		config.BlocksTable = "blockchain_blocks"
	}
	if config.TransactionsTable == "" {
		config.TransactionsTable = "blockchain_transactions"
	}
	if config.AddressesTable == "" {
		config.AddressesTable = "blockchain_addresses"
	}
	if config.EventLogsTable == "" {
		config.EventLogsTable = "blockchain_event_logs"
	}

	return &ExplorerSync{
		config:    config,
		cdcClient: client,
	}
}

// SyncBlock syncs a block to the target database
func (s *ExplorerSync) SyncBlock(ctx context.Context, block *explorer.Block) error {
	if !s.config.Enabled || !s.config.SyncBlocks {
		return nil
	}

	if !s.shouldSyncNetwork(block.Network) {
		return nil
	}

	event := &CDCEvent{
		Operation: "UPSERT",
		Database:  s.config.TargetDatabase,
		Table:     s.config.BlocksTable,
		Data: map[string]interface{}{
			"network":           block.Network,
			"block_number":      block.BlockNumber,
			"block_hash":        block.BlockHash,
			"parent_hash":       block.ParentHash,
			"timestamp":         block.Timestamp,
			"miner":             block.Miner,
			"gas_used":          block.GasUsed,
			"gas_limit":         block.GasLimit,
			"base_fee_per_gas":  block.BaseFeePerGas,
			"transaction_count": block.TransactionCount,
			"size":              block.Size,
		},
		Timestamp: time.Now(),
		Source:    "chainlens-explorer",
		Metadata: map[string]interface{}{
			"entity_type": "block",
		},
	}

	return s.cdcClient.PublishEvent(ctx, event)
}

// SyncTransaction syncs a transaction to the target database
func (s *ExplorerSync) SyncTransaction(ctx context.Context, tx *explorer.Transaction) error {
	if !s.config.Enabled || !s.config.SyncTransactions {
		return nil
	}

	if !s.shouldSyncNetwork(tx.Network) {
		return nil
	}

	// Check min value filter
	if s.config.MinValue != "" {
		minValue := explorer.ParseValue(s.config.MinValue)
		txValue := explorer.ParseValue(tx.Value)
		if txValue.Cmp(minValue) < 0 {
			return nil
		}
	}

	data := map[string]interface{}{
		"network":      tx.Network,
		"tx_hash":      tx.TxHash,
		"block_number": tx.BlockNumber,
		"block_hash":   tx.BlockHash,
		"tx_index":     tx.TxIndex,
		"from_address": tx.From,
		"to_address":   tx.To,
		"value":        tx.Value,
		"gas_limit":    tx.GasLimit,
		"gas_used":     tx.GasUsed,
		"gas_price":    tx.GasPrice,
		"nonce":        tx.Nonce,
		"tx_type":      tx.TxType,
		"status":       tx.Status,
		"timestamp":    tx.Timestamp,
	}

	if tx.ContractAddress != nil {
		data["contract_address"] = *tx.ContractAddress
	}
	if tx.MaxFeePerGas != nil {
		data["max_fee_per_gas"] = *tx.MaxFeePerGas
	}
	if tx.MaxPriorityFeePerGas != nil {
		data["max_priority_fee_per_gas"] = *tx.MaxPriorityFeePerGas
	}

	event := &CDCEvent{
		Operation: "UPSERT",
		Database:  s.config.TargetDatabase,
		Table:     s.config.TransactionsTable,
		Data:      data,
		Timestamp: time.Now(),
		Source:    "chainlens-explorer",
		Metadata: map[string]interface{}{
			"entity_type": "transaction",
		},
	}

	return s.cdcClient.PublishEvent(ctx, event)
}

// SyncAddress syncs an address to the target database
func (s *ExplorerSync) SyncAddress(ctx context.Context, addr *explorer.Address) error {
	if !s.config.Enabled || !s.config.SyncAddresses {
		return nil
	}

	if !s.shouldSyncNetwork(addr.Network) {
		return nil
	}

	data := map[string]interface{}{
		"network":     addr.Network,
		"address":     addr.Address,
		"balance":     addr.Balance,
		"tx_count":    addr.TxCount,
		"is_contract": addr.IsContract,
		"updated_at":  time.Now(),
	}

	if addr.ContractCreator != nil {
		data["contract_creator"] = *addr.ContractCreator
	}
	if addr.ContractCreationTx != nil {
		data["contract_creation_tx"] = *addr.ContractCreationTx
	}
	if addr.Label != nil {
		data["label"] = *addr.Label
	}
	if len(addr.Tags) > 0 {
		data["tags"] = addr.Tags
	}
	if addr.FirstSeenAt != nil {
		data["first_seen_at"] = *addr.FirstSeenAt
	}
	if addr.LastSeenAt != nil {
		data["last_seen_at"] = *addr.LastSeenAt
	}

	event := &CDCEvent{
		Operation: "UPSERT",
		Database:  s.config.TargetDatabase,
		Table:     s.config.AddressesTable,
		Data:      data,
		Timestamp: time.Now(),
		Source:    "chainlens-explorer",
		Metadata: map[string]interface{}{
			"entity_type": "address",
		},
	}

	return s.cdcClient.PublishEvent(ctx, event)
}

// SyncEventLog syncs an event log to the target database
func (s *ExplorerSync) SyncEventLog(ctx context.Context, log *explorer.EventLog) error {
	if !s.config.Enabled || !s.config.SyncEventLogs {
		return nil
	}

	if !s.shouldSyncNetwork(log.Network) {
		return nil
	}

	// Check contract address filter
	if len(s.config.ContractAddresses) > 0 {
		found := false
		for _, addr := range s.config.ContractAddresses {
			if addr == log.ContractAddress {
				found = true
				break
			}
		}
		if !found {
			return nil
		}
	}

	data := map[string]interface{}{
		"network":          log.Network,
		"tx_hash":          log.TxHash,
		"log_index":        log.LogIndex,
		"block_number":     log.BlockNumber,
		"contract_address": log.ContractAddress,
		"topic0":           log.Topic0,
		"topic1":           log.Topic1,
		"topic2":           log.Topic2,
		"topic3":           log.Topic3,
		"data":             log.Data,
		"timestamp":        log.Timestamp,
		"removed":          log.Removed,
	}

	if log.DecodedName != nil {
		data["decoded_name"] = *log.DecodedName
	}
	if log.DecodedArgs != nil {
		data["decoded_args"] = log.DecodedArgs
	}

	event := &CDCEvent{
		Operation: "INSERT",
		Database:  s.config.TargetDatabase,
		Table:     s.config.EventLogsTable,
		Data:      data,
		Timestamp: time.Now(),
		Source:    "chainlens-explorer",
		Metadata: map[string]interface{}{
			"entity_type": "event_log",
		},
	}

	return s.cdcClient.PublishEvent(ctx, event)
}

// SyncBlockWithTransactions syncs a block and all its transactions
func (s *ExplorerSync) SyncBlockWithTransactions(ctx context.Context, block *explorer.Block, txs []*explorer.Transaction, logs []*explorer.EventLog) error {
	// Sync block
	if err := s.SyncBlock(ctx, block); err != nil {
		return err
	}

	// Batch sync transactions
	if len(txs) > 0 && s.config.SyncTransactions {
		events := make([]*CDCEvent, 0, len(txs))
		for _, tx := range txs {
			if !s.shouldSyncNetwork(tx.Network) {
				continue
			}

			data := map[string]interface{}{
				"network":      tx.Network,
				"tx_hash":      tx.TxHash,
				"block_number": tx.BlockNumber,
				"block_hash":   tx.BlockHash,
				"tx_index":     tx.TxIndex,
				"from_address": tx.From,
				"to_address":   tx.To,
				"value":        tx.Value,
				"gas_limit":    tx.GasLimit,
				"gas_used":     tx.GasUsed,
				"gas_price":    tx.GasPrice,
				"nonce":        tx.Nonce,
				"tx_type":      tx.TxType,
				"status":       tx.Status,
				"timestamp":    tx.Timestamp,
			}

			events = append(events, &CDCEvent{
				Operation: "UPSERT",
				Database:  s.config.TargetDatabase,
				Table:     s.config.TransactionsTable,
				Data:      data,
				Timestamp: time.Now(),
				Source:    "chainlens-explorer",
			})
		}

		if len(events) > 0 {
			if err := s.cdcClient.BatchPublish(ctx, events); err != nil {
				return err
			}
		}
	}

	// Batch sync event logs
	if len(logs) > 0 && s.config.SyncEventLogs {
		events := make([]*CDCEvent, 0, len(logs))
		for _, log := range logs {
			if !s.shouldSyncNetwork(log.Network) {
				continue
			}

			data := map[string]interface{}{
				"network":          log.Network,
				"tx_hash":          log.TxHash,
				"log_index":        log.LogIndex,
				"block_number":     log.BlockNumber,
				"contract_address": log.ContractAddress,
				"topic0":           log.Topic0,
				"topic1":           log.Topic1,
				"topic2":           log.Topic2,
				"topic3":           log.Topic3,
				"data":             log.Data,
				"timestamp":        log.Timestamp,
			}

			events = append(events, &CDCEvent{
				Operation: "INSERT",
				Database:  s.config.TargetDatabase,
				Table:     s.config.EventLogsTable,
				Data:      data,
				Timestamp: time.Now(),
				Source:    "chainlens-explorer",
			})
		}

		if len(events) > 0 {
			if err := s.cdcClient.BatchPublish(ctx, events); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *ExplorerSync) shouldSyncNetwork(network string) bool {
	if len(s.config.Networks) == 0 {
		return true // sync all networks
	}
	for _, n := range s.config.Networks {
		if n == network {
			return true
		}
	}
	return false
}

// GetConfig returns the current configuration
func (s *ExplorerSync) GetConfig() *ExplorerSyncConfig {
	return s.config
}

// UpdateConfig updates the configuration
func (s *ExplorerSync) UpdateConfig(config *ExplorerSyncConfig) {
	config.UpdatedAt = time.Now()
	s.config = config
}

// DefaultExplorerSyncConfig returns a default configuration
func DefaultExplorerSyncConfig(targetDatabase string) *ExplorerSyncConfig {
	return &ExplorerSyncConfig{
		ID:                "default",
		Enabled:           true,
		TargetDatabase:    targetDatabase,
		SyncBlocks:        true,
		SyncTransactions:  true,
		SyncAddresses:     true,
		SyncEventLogs:     true,
		BlocksTable:       "blockchain_blocks",
		TransactionsTable: "blockchain_transactions",
		AddressesTable:    "blockchain_addresses",
		EventLogsTable:    "blockchain_event_logs",
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
}

// CreateExplorerTables creates the target tables for explorer data
func (s *ExplorerSync) CreateExplorerTables(ctx context.Context) error {
	if s.config.SyncBlocks {
		schema := map[string]string{
			"network":           "VARCHAR(50)",
			"block_number":      "BIGINT",
			"block_hash":        "VARCHAR(66)",
			"parent_hash":       "VARCHAR(66)",
			"timestamp":         "TIMESTAMPTZ",
			"miner":             "VARCHAR(42)",
			"gas_used":          "BIGINT",
			"gas_limit":         "BIGINT",
			"base_fee_per_gas":  "BIGINT",
			"transaction_count": "INT",
			"size":              "INT",
		}
		if err := s.cdcClient.CreateTable(ctx, s.config.TargetDatabase, s.config.BlocksTable, schema); err != nil {
			return err
		}
	}

	if s.config.SyncTransactions {
		schema := map[string]string{
			"network":                  "VARCHAR(50)",
			"tx_hash":                  "VARCHAR(66)",
			"block_number":             "BIGINT",
			"block_hash":               "VARCHAR(66)",
			"tx_index":                 "INT",
			"from_address":             "VARCHAR(42)",
			"to_address":               "VARCHAR(42)",
			"value":                    "NUMERIC(78,0)",
			"gas_limit":                "BIGINT",
			"gas_used":                 "BIGINT",
			"gas_price":                "BIGINT",
			"max_fee_per_gas":          "BIGINT",
			"max_priority_fee_per_gas": "BIGINT",
			"nonce":                    "BIGINT",
			"tx_type":                  "SMALLINT",
			"status":                   "SMALLINT",
			"timestamp":                "TIMESTAMPTZ",
			"contract_address":         "VARCHAR(42)",
		}
		if err := s.cdcClient.CreateTable(ctx, s.config.TargetDatabase, s.config.TransactionsTable, schema); err != nil {
			return err
		}
	}

	if s.config.SyncAddresses {
		schema := map[string]string{
			"network":              "VARCHAR(50)",
			"address":              "VARCHAR(42)",
			"balance":              "NUMERIC(78,0)",
			"tx_count":             "BIGINT",
			"is_contract":          "BOOLEAN",
			"contract_creator":     "VARCHAR(42)",
			"contract_creation_tx": "VARCHAR(66)",
			"label":                "VARCHAR(255)",
			"tags":                 "TEXT[]",
			"first_seen_at":        "TIMESTAMPTZ",
			"last_seen_at":         "TIMESTAMPTZ",
			"updated_at":           "TIMESTAMPTZ",
		}
		if err := s.cdcClient.CreateTable(ctx, s.config.TargetDatabase, s.config.AddressesTable, schema); err != nil {
			return err
		}
	}

	if s.config.SyncEventLogs {
		schema := map[string]string{
			"network":          "VARCHAR(50)",
			"tx_hash":          "VARCHAR(66)",
			"log_index":        "INT",
			"block_number":     "BIGINT",
			"contract_address": "VARCHAR(42)",
			"topic0":           "VARCHAR(66)",
			"topic1":           "VARCHAR(66)",
			"topic2":           "VARCHAR(66)",
			"topic3":           "VARCHAR(66)",
			"data":             "TEXT",
			"timestamp":        "TIMESTAMPTZ",
			"decoded_name":     "VARCHAR(100)",
			"decoded_args":     "JSONB",
			"removed":          "BOOLEAN",
		}
		if err := s.cdcClient.CreateTable(ctx, s.config.TargetDatabase, s.config.EventLogsTable, schema); err != nil {
			return err
		}
	}

	return nil
}
