package explorer

import (
	"context"
	"fmt"
	"math/big"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Explorer provides blockchain explorer functionality
type Explorer struct {
	repo RepositoryInterface
}

// New creates a new Explorer instance
func New(pool *pgxpool.Pool) *Explorer {
	return &Explorer{
		repo: NewRepository(pool),
	}
}

// NewWithRepository creates an Explorer with a custom repository (for testing)
func NewWithRepository(repo RepositoryInterface) *Explorer {
	return &Explorer{repo: repo}
}

// Repository returns the underlying repository (for backwards compatibility)
func (e *Explorer) Repository() RepositoryInterface {
	return e.repo
}

// ============================================================================
// BLOCKS
// ============================================================================

// GetBlock retrieves a block by number or hash
func (e *Explorer) GetBlock(ctx context.Context, network, identifier string) (*Block, error) {
	// Check if identifier is a block number
	if isBlockNumber(identifier) {
		number, _ := parseBlockNumber(identifier)
		return e.repo.GetBlockByNumber(ctx, network, number)
	}

	// Otherwise treat as hash
	return e.repo.GetBlockByHash(ctx, network, identifier)
}

// GetBlockByNumber retrieves a block by number
func (e *Explorer) GetBlockByNumber(ctx context.Context, network string, number int64) (*Block, error) {
	return e.repo.GetBlockByNumber(ctx, network, number)
}

// GetBlockByHash retrieves a block by hash
func (e *Explorer) GetBlockByHash(ctx context.Context, network, hash string) (*Block, error) {
	return e.repo.GetBlockByHash(ctx, network, hash)
}

// GetLatestBlock returns the latest indexed block
func (e *Explorer) GetLatestBlock(ctx context.Context, network string) (*Block, error) {
	return e.repo.GetLatestBlock(ctx, network)
}

// ListBlocks retrieves blocks with filtering
func (e *Explorer) ListBlocks(ctx context.Context, network string, page, pageSize int, miner *string) (*ListResult[Block], error) {
	filter := BlockFilter{
		Network:           network,
		Miner:             miner,
		PaginationOptions: NewPaginationOptions(page, pageSize),
	}
	return e.repo.ListBlocks(ctx, filter)
}

// GetBlockTransactions retrieves all transactions for a block
func (e *Explorer) GetBlockTransactions(ctx context.Context, network string, blockNumber int64) ([]Transaction, error) {
	return e.repo.GetTransactionsByBlock(ctx, network, blockNumber)
}

// ============================================================================
// TRANSACTIONS
// ============================================================================

// GetTransaction retrieves a transaction by hash
func (e *Explorer) GetTransaction(ctx context.Context, network, hash string) (*Transaction, error) {
	return e.repo.GetTransactionByHash(ctx, network, hash)
}

// ListTransactions retrieves transactions with filtering
func (e *Explorer) ListTransactions(ctx context.Context, filter TransactionFilter) (*ListResult[Transaction], error) {
	if filter.PageSize == 0 {
		filter.PaginationOptions = NewPaginationOptions(filter.Page, 20)
	}
	return e.repo.ListTransactions(ctx, filter)
}

// GetTransactionLogs retrieves event logs for a transaction
func (e *Explorer) GetTransactionLogs(ctx context.Context, network, txHash string) ([]EventLog, error) {
	return e.repo.GetTransactionLogs(ctx, network, txHash)
}

// ============================================================================
// ADDRESSES
// ============================================================================

// GetAddress retrieves address information
func (e *Explorer) GetAddress(ctx context.Context, network, address string) (*Address, error) {
	addr, err := e.repo.GetAddress(ctx, network, address)
	if err != nil {
		return nil, err
	}

	// If address doesn't exist in DB, return a default one
	if addr == nil {
		addr = &Address{
			Network:    network,
			Address:    address,
			Balance:    "0",
			TxCount:    0,
			IsContract: false,
		}
	}

	return addr, nil
}

// GetAddressTransactions retrieves transactions for an address
func (e *Explorer) GetAddressTransactions(ctx context.Context, network, address string, page, pageSize int) (*ListResult[Transaction], error) {
	opts := NewPaginationOptions(page, pageSize)
	return e.repo.GetAddressTransactions(ctx, network, address, opts)
}

// GetAddressLogs retrieves event logs for a contract address
func (e *Explorer) GetAddressLogs(ctx context.Context, network, address string, page, pageSize int) (*ListResult[EventLog], error) {
	opts := NewPaginationOptions(page, pageSize)
	return e.repo.GetAddressLogs(ctx, network, address, opts)
}

// ============================================================================
// SEARCH
// ============================================================================

// Search performs a universal search across blocks, transactions, and addresses
func (e *Explorer) Search(ctx context.Context, network, query string) (*SearchResults, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return &SearchResults{Query: query, Results: []SearchResult{}, Total: 0}, nil
	}

	results := &SearchResults{
		Query:   query,
		Results: []SearchResult{},
	}

	// Check if it's a transaction hash (66 chars, starts with 0x)
	if isTxHash(query) {
		tx, err := e.repo.GetTransactionByHash(ctx, network, query)
		if err != nil {
			return nil, err
		}
		if tx != nil {
			results.Results = append(results.Results, SearchResult{
				Type:    "transaction",
				Network: network,
				Data:    tx,
			})
		}
	}

	// Check if it's a block hash (66 chars, starts with 0x) or block number
	if isBlockHash(query) {
		block, err := e.repo.GetBlockByHash(ctx, network, query)
		if err != nil {
			return nil, err
		}
		if block != nil {
			results.Results = append(results.Results, SearchResult{
				Type:    "block",
				Network: network,
				Data:    block,
			})
		}
	} else if isBlockNumber(query) {
		number, _ := parseBlockNumber(query)
		block, err := e.repo.GetBlockByNumber(ctx, network, number)
		if err != nil {
			return nil, err
		}
		if block != nil {
			results.Results = append(results.Results, SearchResult{
				Type:    "block",
				Network: network,
				Data:    block,
			})
		}
	}

	// Check if it's an address (42 chars, starts with 0x)
	if isAddress(query) {
		addr, err := e.repo.GetAddress(ctx, network, query)
		if err != nil {
			return nil, err
		}
		if addr != nil {
			results.Results = append(results.Results, SearchResult{
				Type:    "address",
				Network: network,
				Data:    addr,
			})
		} else {
			// Return a default address entry even if not in DB
			results.Results = append(results.Results, SearchResult{
				Type:    "address",
				Network: network,
				Data: &Address{
					Network: network,
					Address: query,
					Balance: "0",
				},
			})
		}
	}

	results.Total = len(results.Results)
	return results, nil
}

// ============================================================================
// STATS
// ============================================================================

// GetNetworkStats retrieves aggregated network statistics
func (e *Explorer) GetNetworkStats(ctx context.Context, network string) (*NetworkStats, error) {
	return e.repo.GetNetworkStats(ctx, network)
}

// GetSyncState retrieves the current sync state for a network
func (e *Explorer) GetSyncState(ctx context.Context, network string) (*NetworkSyncState, error) {
	return e.repo.GetSyncState(ctx, network)
}

// ============================================================================
// INDEXING (for use by the indexer)
// ============================================================================

// IndexBlock stores a block and its transactions
func (e *Explorer) IndexBlock(ctx context.Context, block *Block, txs []*Transaction, logs []*EventLog) error {
	// Insert block
	if err := e.repo.InsertBlock(ctx, block); err != nil {
		return fmt.Errorf("insert block: %w", err)
	}

	// Insert transactions
	if len(txs) > 0 {
		if err := e.repo.InsertTransactions(ctx, txs); err != nil {
			return fmt.Errorf("insert transactions: %w", err)
		}

		// Update address tx counts
		addressSet := make(map[string]bool)
		for _, tx := range txs {
			addressSet[tx.From] = true
			if tx.To != nil {
				addressSet[*tx.To] = true
			}
			if tx.ContractAddress != nil {
				addressSet[*tx.ContractAddress] = true
			}
		}

		addresses := make([]string, 0, len(addressSet))
		for addr := range addressSet {
			addresses = append(addresses, addr)
		}

		if err := e.repo.IncrementAddressTxCount(ctx, block.Network, addresses, block.Timestamp); err != nil {
			return fmt.Errorf("increment address tx counts: %w", err)
		}
	}

	// Insert event logs
	if len(logs) > 0 {
		if err := e.repo.InsertEventLogs(ctx, logs); err != nil {
			return fmt.Errorf("insert event logs: %w", err)
		}
	}

	// Update sync state
	if err := e.repo.UpdateSyncState(ctx, block.Network, block.BlockNumber, true, 0, nil); err != nil {
		return fmt.Errorf("update sync state: %w", err)
	}

	return nil
}

// IndexBlocks stores multiple blocks and their transactions in batch
func (e *Explorer) IndexBlocks(ctx context.Context, blocks []*Block, txs []*Transaction, logs []*EventLog) error {
	if len(blocks) == 0 {
		return nil
	}

	// Insert blocks
	if err := e.repo.InsertBlocks(ctx, blocks); err != nil {
		return fmt.Errorf("insert blocks: %w", err)
	}

	// Insert transactions
	if len(txs) > 0 {
		if err := e.repo.InsertTransactions(ctx, txs); err != nil {
			return fmt.Errorf("insert transactions: %w", err)
		}
	}

	// Insert event logs
	if len(logs) > 0 {
		if err := e.repo.InsertEventLogs(ctx, logs); err != nil {
			return fmt.Errorf("insert event logs: %w", err)
		}
	}

	// Update sync state with the latest block
	latestBlock := blocks[len(blocks)-1]
	if err := e.repo.UpdateSyncState(ctx, latestBlock.Network, latestBlock.BlockNumber, true, 0, nil); err != nil {
		return fmt.Errorf("update sync state: %w", err)
	}

	return nil
}

// ============================================================================
// HELPERS
// ============================================================================

var (
	txHashRegex    = regexp.MustCompile(`^0x[a-fA-F0-9]{64}$`)
	addressRegex   = regexp.MustCompile(`^0x[a-fA-F0-9]{40}$`)
	blockNumRegex  = regexp.MustCompile(`^[0-9]+$`)
)

func isTxHash(s string) bool {
	return txHashRegex.MatchString(s)
}

func isBlockHash(s string) bool {
	return txHashRegex.MatchString(s) // same format as tx hash
}

func isAddress(s string) bool {
	return addressRegex.MatchString(s)
}

func isBlockNumber(s string) bool {
	return blockNumRegex.MatchString(s)
}

func parseBlockNumber(s string) (int64, error) {
	var n int64
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}

// ParseValue parses a wei value string to big.Int
func ParseValue(s string) *big.Int {
	n := new(big.Int)
	n.SetString(s, 10)
	return n
}

// FormatValue formats a big.Int to string
func FormatValue(n *big.Int) string {
	if n == nil {
		return "0"
	}
	return n.String()
}

// SupportedNetworks returns a list of supported networks
func SupportedNetworks() []string {
	return []string{
		"ethereum",
		"polygon",
		"arbitrum",
		"optimism",
		"base",
		"bsc",
		"avalanche",
	}
}

// IsValidNetwork checks if a network is supported
func IsValidNetwork(network string) bool {
	for _, n := range SupportedNetworks() {
		if n == network {
			return true
		}
	}
	return false
}
