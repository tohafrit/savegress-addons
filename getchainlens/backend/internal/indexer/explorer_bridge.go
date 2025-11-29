package indexer

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"getchainlens.com/chainlens/backend/internal/explorer"
	"getchainlens.com/chainlens/backend/internal/monitor"
)

// ExplorerBridge connects the indexer to the explorer for storing blockchain data
type ExplorerBridge struct {
	explorer     *explorer.Explorer
	networkNames map[int64]string
}

// NewExplorerBridge creates a new explorer bridge
func NewExplorerBridge(exp *explorer.Explorer) *ExplorerBridge {
	return &ExplorerBridge{
		explorer: exp,
		networkNames: map[int64]string{
			1:     "ethereum",
			137:   "polygon",
			42161: "arbitrum",
			10:    "optimism",
			8453:  "base",
			56:    "bsc",
			43114: "avalanche",
			11155111: "sepolia",
			80001:    "mumbai",
		},
	}
}

// GetNetworkName returns the network name for a chain ID
func (b *ExplorerBridge) GetNetworkName(chainID int64) string {
	if name, ok := b.networkNames[chainID]; ok {
		return name
	}
	return fmt.Sprintf("chain-%d", chainID)
}

// IndexBlockData indexes a full block with transactions and logs
func (b *ExplorerBridge) IndexBlockData(ctx context.Context, chainID int64, blockData *BlockData) error {
	network := b.GetNetworkName(chainID)

	// Convert to explorer types
	block := b.convertBlock(network, blockData)
	txs := b.convertTransactions(network, blockData)
	logs := b.convertLogs(network, blockData)

	return b.explorer.IndexBlock(ctx, block, txs, logs)
}

// BlockData contains full block information
type BlockData struct {
	Number           uint64
	Hash             string
	ParentHash       string
	Timestamp        time.Time
	Miner            string
	GasUsed          uint64
	GasLimit         uint64
	BaseFeePerGas    *big.Int
	Size             int
	ExtraData        string
	Transactions     []*TransactionData
	Logs             []*LogData
}

// TransactionData contains transaction information
type TransactionData struct {
	Hash                 string
	BlockNumber          uint64
	BlockHash            string
	Index                int
	From                 string
	To                   *string
	Value                *big.Int
	GasPrice             *big.Int
	GasLimit             uint64
	GasUsed              *uint64
	MaxFeePerGas         *big.Int
	MaxPriorityFeePerGas *big.Int
	Input                string
	Nonce                uint64
	Type                 int
	Status               *int
	ContractAddress      *string
	Error                *string
}

// LogData contains event log information
type LogData struct {
	TxHash          string
	LogIndex        int
	BlockNumber     uint64
	ContractAddress string
	Topics          []string
	Data            string
	Removed         bool
}

func (b *ExplorerBridge) convertBlock(network string, data *BlockData) *explorer.Block {
	block := &explorer.Block{
		Network:          network,
		BlockNumber:      int64(data.Number),
		BlockHash:        data.Hash,
		ParentHash:       data.ParentHash,
		Timestamp:        data.Timestamp,
		Miner:            data.Miner,
		GasUsed:          int64(data.GasUsed),
		GasLimit:         int64(data.GasLimit),
		TransactionCount: len(data.Transactions),
		Size:             data.Size,
		ExtraData:        data.ExtraData,
	}

	if data.BaseFeePerGas != nil {
		baseFee := data.BaseFeePerGas.Int64()
		block.BaseFeePerGas = &baseFee
	}

	return block
}

func (b *ExplorerBridge) convertTransactions(network string, data *BlockData) []*explorer.Transaction {
	txs := make([]*explorer.Transaction, len(data.Transactions))

	for i, txData := range data.Transactions {
		tx := &explorer.Transaction{
			Network:     network,
			TxHash:      txData.Hash,
			BlockNumber: int64(txData.BlockNumber),
			BlockHash:   txData.BlockHash,
			TxIndex:     txData.Index,
			From:        txData.From,
			To:          txData.To,
			Value:       "0",
			GasLimit:    int64(txData.GasLimit),
			InputData:   txData.Input,
			Nonce:       int64(txData.Nonce),
			TxType:      txData.Type,
			Status:      txData.Status,
			Timestamp:   data.Timestamp,
			ContractAddress: txData.ContractAddress,
			ErrorMessage:    txData.Error,
		}

		if txData.Value != nil {
			tx.Value = txData.Value.String()
		}
		if txData.GasPrice != nil {
			gasPrice := txData.GasPrice.Int64()
			tx.GasPrice = &gasPrice
		}
		if txData.GasUsed != nil {
			gasUsed := int64(*txData.GasUsed)
			tx.GasUsed = &gasUsed
		}
		if txData.MaxFeePerGas != nil {
			maxFee := txData.MaxFeePerGas.Int64()
			tx.MaxFeePerGas = &maxFee
		}
		if txData.MaxPriorityFeePerGas != nil {
			maxPriority := txData.MaxPriorityFeePerGas.Int64()
			tx.MaxPriorityFeePerGas = &maxPriority
		}

		txs[i] = tx
	}

	return txs
}

func (b *ExplorerBridge) convertLogs(network string, data *BlockData) []*explorer.EventLog {
	logs := make([]*explorer.EventLog, len(data.Logs))

	for i, logData := range data.Logs {
		log := &explorer.EventLog{
			Network:         network,
			TxHash:          logData.TxHash,
			LogIndex:        logData.LogIndex,
			BlockNumber:     int64(logData.BlockNumber),
			ContractAddress: logData.ContractAddress,
			Data:            logData.Data,
			Timestamp:       data.Timestamp,
			Removed:         logData.Removed,
		}

		if len(logData.Topics) > 0 {
			log.Topic0 = &logData.Topics[0]
		}
		if len(logData.Topics) > 1 {
			log.Topic1 = &logData.Topics[1]
		}
		if len(logData.Topics) > 2 {
			log.Topic2 = &logData.Topics[2]
		}
		if len(logData.Topics) > 3 {
			log.Topic3 = &logData.Topics[3]
		}

		logs[i] = log
	}

	return logs
}

// ConvertContractEvent converts a monitor.ContractEvent to explorer.EventLog
func (b *ExplorerBridge) ConvertContractEvent(event *monitor.ContractEvent) *explorer.EventLog {
	network := b.GetNetworkName(event.ChainID)

	log := &explorer.EventLog{
		Network:         network,
		TxHash:          event.TxHash,
		LogIndex:        int(event.LogIndex),
		BlockNumber:     int64(event.BlockNumber),
		ContractAddress: event.ContractAddress,
		Data:            event.Data,
		Timestamp:       event.Timestamp,
	}

	if len(event.Topics) > 0 {
		log.Topic0 = &event.Topics[0]
	}
	if len(event.Topics) > 1 {
		log.Topic1 = &event.Topics[1]
	}
	if len(event.Topics) > 2 {
		log.Topic2 = &event.Topics[2]
	}
	if len(event.Topics) > 3 {
		log.Topic3 = &event.Topics[3]
	}

	return log
}

// RegisterWithIndexer registers the bridge with a multi-chain indexer
func (b *ExplorerBridge) RegisterWithIndexer(indexer *MultiChainIndexer) {
	// This would be called to set up callbacks
	// For now, the indexer needs to be modified to support full block fetching
}

// ChainClient interface for fetching full block data
type FullBlockClient interface {
	monitor.ChainClient
	GetBlock(ctx context.Context, number uint64) (*BlockData, error)
}

// EnhancedNetworkIndexer extends NetworkIndexer with full block indexing
type EnhancedNetworkIndexer struct {
	*NetworkIndexer
	bridge     *ExplorerBridge
	fullClient FullBlockClient
}

// NewEnhancedNetworkIndexer creates an enhanced indexer with explorer support
func NewEnhancedNetworkIndexer(config NetworkConfig, client FullBlockClient, bridge *ExplorerBridge) *EnhancedNetworkIndexer {
	base := &NetworkIndexer{
		config:  config,
		client:  client,
		stopCh:  make(chan struct{}),
		blockCh: make(chan uint64, 100),
	}

	enhanced := &EnhancedNetworkIndexer{
		NetworkIndexer: base,
		bridge:         bridge,
		fullClient:     client,
	}

	return enhanced
}

// IndexBlockWithExplorer indexes a block and stores in explorer database
func (e *EnhancedNetworkIndexer) IndexBlockWithExplorer(ctx context.Context, blockNumber uint64) error {
	blockData, err := e.fullClient.GetBlock(ctx, blockNumber)
	if err != nil {
		return fmt.Errorf("get block %d: %w", blockNumber, err)
	}

	if err := e.bridge.IndexBlockData(ctx, e.config.ChainID, blockData); err != nil {
		return fmt.Errorf("index block data: %w", err)
	}

	return nil
}
