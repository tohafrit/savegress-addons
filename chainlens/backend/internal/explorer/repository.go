package explorer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles database operations for explorer data
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new explorer repository
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// ============================================================================
// BLOCKS
// ============================================================================

// InsertBlock inserts a new block into the database
func (r *Repository) InsertBlock(ctx context.Context, block *Block) error {
	query := `
		INSERT INTO blocks (
			network, block_number, block_hash, parent_hash, timestamp,
			miner, gas_used, gas_limit, base_fee_per_gas, transaction_count,
			size, extra_data
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (network, block_number) DO UPDATE SET
			block_hash = EXCLUDED.block_hash,
			parent_hash = EXCLUDED.parent_hash,
			timestamp = EXCLUDED.timestamp,
			miner = EXCLUDED.miner,
			gas_used = EXCLUDED.gas_used,
			gas_limit = EXCLUDED.gas_limit,
			base_fee_per_gas = EXCLUDED.base_fee_per_gas,
			transaction_count = EXCLUDED.transaction_count,
			size = EXCLUDED.size,
			extra_data = EXCLUDED.extra_data
		RETURNING id`

	return r.pool.QueryRow(ctx, query,
		block.Network, block.BlockNumber, block.BlockHash, block.ParentHash, block.Timestamp,
		block.Miner, block.GasUsed, block.GasLimit, block.BaseFeePerGas, block.TransactionCount,
		block.Size, block.ExtraData,
	).Scan(&block.ID)
}

// InsertBlocks inserts multiple blocks in a batch
func (r *Repository) InsertBlocks(ctx context.Context, blocks []*Block) error {
	if len(blocks) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	query := `
		INSERT INTO blocks (
			network, block_number, block_hash, parent_hash, timestamp,
			miner, gas_used, gas_limit, base_fee_per_gas, transaction_count,
			size, extra_data
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (network, block_number) DO UPDATE SET
			block_hash = EXCLUDED.block_hash,
			transaction_count = EXCLUDED.transaction_count`

	for _, block := range blocks {
		batch.Queue(query,
			block.Network, block.BlockNumber, block.BlockHash, block.ParentHash, block.Timestamp,
			block.Miner, block.GasUsed, block.GasLimit, block.BaseFeePerGas, block.TransactionCount,
			block.Size, block.ExtraData,
		)
	}

	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()

	for i := 0; i < len(blocks); i++ {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("batch insert block %d: %w", i, err)
		}
	}
	return nil
}

// GetBlockByNumber retrieves a block by network and number
func (r *Repository) GetBlockByNumber(ctx context.Context, network string, number int64) (*Block, error) {
	query := `
		SELECT id, network, block_number, block_hash, parent_hash, timestamp,
			   miner, gas_used, gas_limit, base_fee_per_gas, transaction_count,
			   size, extra_data, created_at
		FROM blocks
		WHERE network = $1 AND block_number = $2`

	block := &Block{}
	err := r.pool.QueryRow(ctx, query, network, number).Scan(
		&block.ID, &block.Network, &block.BlockNumber, &block.BlockHash, &block.ParentHash, &block.Timestamp,
		&block.Miner, &block.GasUsed, &block.GasLimit, &block.BaseFeePerGas, &block.TransactionCount,
		&block.Size, &block.ExtraData, &block.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return block, nil
}

// GetBlockByHash retrieves a block by hash
func (r *Repository) GetBlockByHash(ctx context.Context, network, hash string) (*Block, error) {
	query := `
		SELECT id, network, block_number, block_hash, parent_hash, timestamp,
			   miner, gas_used, gas_limit, base_fee_per_gas, transaction_count,
			   size, extra_data, created_at
		FROM blocks
		WHERE network = $1 AND block_hash = $2`

	block := &Block{}
	err := r.pool.QueryRow(ctx, query, network, hash).Scan(
		&block.ID, &block.Network, &block.BlockNumber, &block.BlockHash, &block.ParentHash, &block.Timestamp,
		&block.Miner, &block.GasUsed, &block.GasLimit, &block.BaseFeePerGas, &block.TransactionCount,
		&block.Size, &block.ExtraData, &block.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return block, nil
}

// ListBlocks retrieves blocks with filtering and pagination
func (r *Repository) ListBlocks(ctx context.Context, filter BlockFilter) (*ListResult[Block], error) {
	var conditions []string
	var args []interface{}
	argNum := 1

	conditions = append(conditions, fmt.Sprintf("network = $%d", argNum))
	args = append(args, filter.Network)
	argNum++

	if filter.FromBlock != nil {
		conditions = append(conditions, fmt.Sprintf("block_number >= $%d", argNum))
		args = append(args, *filter.FromBlock)
		argNum++
	}
	if filter.ToBlock != nil {
		conditions = append(conditions, fmt.Sprintf("block_number <= $%d", argNum))
		args = append(args, *filter.ToBlock)
		argNum++
	}
	if filter.Miner != nil {
		conditions = append(conditions, fmt.Sprintf("miner = $%d", argNum))
		args = append(args, *filter.Miner)
		argNum++
	}

	where := strings.Join(conditions, " AND ")

	// Count query
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM blocks WHERE %s", where)
	var total int64
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, err
	}

	// Data query
	query := fmt.Sprintf(`
		SELECT id, network, block_number, block_hash, parent_hash, timestamp,
			   miner, gas_used, gas_limit, base_fee_per_gas, transaction_count,
			   size, extra_data, created_at
		FROM blocks
		WHERE %s
		ORDER BY block_number DESC
		LIMIT $%d OFFSET $%d`, where, argNum, argNum+1)

	args = append(args, filter.PageSize, filter.Offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var blocks []Block
	for rows.Next() {
		var block Block
		if err := rows.Scan(
			&block.ID, &block.Network, &block.BlockNumber, &block.BlockHash, &block.ParentHash, &block.Timestamp,
			&block.Miner, &block.GasUsed, &block.GasLimit, &block.BaseFeePerGas, &block.TransactionCount,
			&block.Size, &block.ExtraData, &block.CreatedAt,
		); err != nil {
			return nil, err
		}
		blocks = append(blocks, block)
	}

	totalPages := int(total) / filter.PageSize
	if int(total)%filter.PageSize > 0 {
		totalPages++
	}

	return &ListResult[Block]{
		Items:      blocks,
		Total:      total,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalPages: totalPages,
	}, nil
}

// GetLatestBlock returns the latest indexed block for a network
func (r *Repository) GetLatestBlock(ctx context.Context, network string) (*Block, error) {
	query := `
		SELECT id, network, block_number, block_hash, parent_hash, timestamp,
			   miner, gas_used, gas_limit, base_fee_per_gas, transaction_count,
			   size, extra_data, created_at
		FROM blocks
		WHERE network = $1
		ORDER BY block_number DESC
		LIMIT 1`

	block := &Block{}
	err := r.pool.QueryRow(ctx, query, network).Scan(
		&block.ID, &block.Network, &block.BlockNumber, &block.BlockHash, &block.ParentHash, &block.Timestamp,
		&block.Miner, &block.GasUsed, &block.GasLimit, &block.BaseFeePerGas, &block.TransactionCount,
		&block.Size, &block.ExtraData, &block.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return block, nil
}

// ============================================================================
// TRANSACTIONS
// ============================================================================

// InsertTransaction inserts a new transaction
func (r *Repository) InsertTransaction(ctx context.Context, tx *Transaction) error {
	query := `
		INSERT INTO transactions (
			network, tx_hash, block_number, block_hash, tx_index,
			from_address, to_address, value, gas_price, gas_limit, gas_used,
			max_fee_per_gas, max_priority_fee_per_gas, input_data, nonce,
			tx_type, status, timestamp, contract_address, error_message
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
		ON CONFLICT (network, tx_hash) DO UPDATE SET
			status = EXCLUDED.status,
			gas_used = EXCLUDED.gas_used,
			error_message = EXCLUDED.error_message
		RETURNING id`

	return r.pool.QueryRow(ctx, query,
		tx.Network, tx.TxHash, tx.BlockNumber, tx.BlockHash, tx.TxIndex,
		tx.From, tx.To, tx.Value, tx.GasPrice, tx.GasLimit, tx.GasUsed,
		tx.MaxFeePerGas, tx.MaxPriorityFeePerGas, tx.InputData, tx.Nonce,
		tx.TxType, tx.Status, tx.Timestamp, tx.ContractAddress, tx.ErrorMessage,
	).Scan(&tx.ID)
}

// InsertTransactions inserts multiple transactions in a batch
func (r *Repository) InsertTransactions(ctx context.Context, txs []*Transaction) error {
	if len(txs) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	query := `
		INSERT INTO transactions (
			network, tx_hash, block_number, block_hash, tx_index,
			from_address, to_address, value, gas_price, gas_limit, gas_used,
			max_fee_per_gas, max_priority_fee_per_gas, input_data, nonce,
			tx_type, status, timestamp, contract_address, error_message
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
		ON CONFLICT (network, tx_hash) DO NOTHING`

	for _, tx := range txs {
		batch.Queue(query,
			tx.Network, tx.TxHash, tx.BlockNumber, tx.BlockHash, tx.TxIndex,
			tx.From, tx.To, tx.Value, tx.GasPrice, tx.GasLimit, tx.GasUsed,
			tx.MaxFeePerGas, tx.MaxPriorityFeePerGas, tx.InputData, tx.Nonce,
			tx.TxType, tx.Status, tx.Timestamp, tx.ContractAddress, tx.ErrorMessage,
		)
	}

	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()

	for i := 0; i < len(txs); i++ {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("batch insert transaction %d: %w", i, err)
		}
	}
	return nil
}

// GetTransactionByHash retrieves a transaction by hash
func (r *Repository) GetTransactionByHash(ctx context.Context, network, hash string) (*Transaction, error) {
	query := `
		SELECT id, network, tx_hash, block_number, block_hash, tx_index,
			   from_address, to_address, value, gas_price, gas_limit, gas_used,
			   max_fee_per_gas, max_priority_fee_per_gas, input_data, nonce,
			   tx_type, status, timestamp, contract_address, error_message, created_at
		FROM transactions
		WHERE network = $1 AND tx_hash = $2`

	tx := &Transaction{}
	err := r.pool.QueryRow(ctx, query, network, hash).Scan(
		&tx.ID, &tx.Network, &tx.TxHash, &tx.BlockNumber, &tx.BlockHash, &tx.TxIndex,
		&tx.From, &tx.To, &tx.Value, &tx.GasPrice, &tx.GasLimit, &tx.GasUsed,
		&tx.MaxFeePerGas, &tx.MaxPriorityFeePerGas, &tx.InputData, &tx.Nonce,
		&tx.TxType, &tx.Status, &tx.Timestamp, &tx.ContractAddress, &tx.ErrorMessage, &tx.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return tx, nil
}

// ListTransactions retrieves transactions with filtering
func (r *Repository) ListTransactions(ctx context.Context, filter TransactionFilter) (*ListResult[Transaction], error) {
	var conditions []string
	var args []interface{}
	argNum := 1

	conditions = append(conditions, fmt.Sprintf("network = $%d", argNum))
	args = append(args, filter.Network)
	argNum++

	if filter.BlockNumber != nil {
		conditions = append(conditions, fmt.Sprintf("block_number = $%d", argNum))
		args = append(args, *filter.BlockNumber)
		argNum++
	}
	if filter.FromAddress != nil {
		conditions = append(conditions, fmt.Sprintf("from_address = $%d", argNum))
		args = append(args, *filter.FromAddress)
		argNum++
	}
	if filter.ToAddress != nil {
		conditions = append(conditions, fmt.Sprintf("to_address = $%d", argNum))
		args = append(args, *filter.ToAddress)
		argNum++
	}
	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argNum))
		args = append(args, *filter.Status)
		argNum++
	}
	if filter.FromTime != nil {
		conditions = append(conditions, fmt.Sprintf("timestamp >= $%d", argNum))
		args = append(args, *filter.FromTime)
		argNum++
	}
	if filter.ToTime != nil {
		conditions = append(conditions, fmt.Sprintf("timestamp <= $%d", argNum))
		args = append(args, *filter.ToTime)
		argNum++
	}

	where := strings.Join(conditions, " AND ")

	// Count query
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM transactions WHERE %s", where)
	var total int64
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, err
	}

	// Data query
	query := fmt.Sprintf(`
		SELECT id, network, tx_hash, block_number, block_hash, tx_index,
			   from_address, to_address, value, gas_price, gas_limit, gas_used,
			   max_fee_per_gas, max_priority_fee_per_gas, input_data, nonce,
			   tx_type, status, timestamp, contract_address, error_message, created_at
		FROM transactions
		WHERE %s
		ORDER BY block_number DESC, tx_index DESC
		LIMIT $%d OFFSET $%d`, where, argNum, argNum+1)

	args = append(args, filter.PageSize, filter.Offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txs []Transaction
	for rows.Next() {
		var tx Transaction
		if err := rows.Scan(
			&tx.ID, &tx.Network, &tx.TxHash, &tx.BlockNumber, &tx.BlockHash, &tx.TxIndex,
			&tx.From, &tx.To, &tx.Value, &tx.GasPrice, &tx.GasLimit, &tx.GasUsed,
			&tx.MaxFeePerGas, &tx.MaxPriorityFeePerGas, &tx.InputData, &tx.Nonce,
			&tx.TxType, &tx.Status, &tx.Timestamp, &tx.ContractAddress, &tx.ErrorMessage, &tx.CreatedAt,
		); err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}

	totalPages := int(total) / filter.PageSize
	if int(total)%filter.PageSize > 0 {
		totalPages++
	}

	return &ListResult[Transaction]{
		Items:      txs,
		Total:      total,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalPages: totalPages,
	}, nil
}

// GetTransactionsByBlock retrieves all transactions for a block
func (r *Repository) GetTransactionsByBlock(ctx context.Context, network string, blockNumber int64) ([]Transaction, error) {
	query := `
		SELECT id, network, tx_hash, block_number, block_hash, tx_index,
			   from_address, to_address, value, gas_price, gas_limit, gas_used,
			   max_fee_per_gas, max_priority_fee_per_gas, input_data, nonce,
			   tx_type, status, timestamp, contract_address, error_message, created_at
		FROM transactions
		WHERE network = $1 AND block_number = $2
		ORDER BY tx_index ASC`

	rows, err := r.pool.Query(ctx, query, network, blockNumber)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txs []Transaction
	for rows.Next() {
		var tx Transaction
		if err := rows.Scan(
			&tx.ID, &tx.Network, &tx.TxHash, &tx.BlockNumber, &tx.BlockHash, &tx.TxIndex,
			&tx.From, &tx.To, &tx.Value, &tx.GasPrice, &tx.GasLimit, &tx.GasUsed,
			&tx.MaxFeePerGas, &tx.MaxPriorityFeePerGas, &tx.InputData, &tx.Nonce,
			&tx.TxType, &tx.Status, &tx.Timestamp, &tx.ContractAddress, &tx.ErrorMessage, &tx.CreatedAt,
		); err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}
	return txs, nil
}

// ============================================================================
// ADDRESSES
// ============================================================================

// UpsertAddress creates or updates an address
func (r *Repository) UpsertAddress(ctx context.Context, addr *Address) error {
	query := `
		INSERT INTO addresses (
			network, address, balance, tx_count, is_contract,
			contract_creator, contract_creation_tx, code_hash,
			first_seen_at, last_seen_at, label, tags
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (network, address) DO UPDATE SET
			balance = EXCLUDED.balance,
			tx_count = addresses.tx_count + 1,
			last_seen_at = EXCLUDED.last_seen_at,
			is_contract = COALESCE(EXCLUDED.is_contract, addresses.is_contract),
			updated_at = NOW()
		RETURNING id`

	return r.pool.QueryRow(ctx, query,
		addr.Network, addr.Address, addr.Balance, addr.TxCount, addr.IsContract,
		addr.ContractCreator, addr.ContractCreationTx, addr.CodeHash,
		addr.FirstSeenAt, addr.LastSeenAt, addr.Label, addr.Tags,
	).Scan(&addr.ID)
}

// GetAddress retrieves an address by network and address
func (r *Repository) GetAddress(ctx context.Context, network, address string) (*Address, error) {
	query := `
		SELECT id, network, address, balance, tx_count, is_contract,
			   contract_creator, contract_creation_tx, code_hash,
			   first_seen_at, last_seen_at, label, tags, updated_at
		FROM addresses
		WHERE network = $1 AND address = $2`

	addr := &Address{}
	err := r.pool.QueryRow(ctx, query, network, address).Scan(
		&addr.ID, &addr.Network, &addr.Address, &addr.Balance, &addr.TxCount, &addr.IsContract,
		&addr.ContractCreator, &addr.ContractCreationTx, &addr.CodeHash,
		&addr.FirstSeenAt, &addr.LastSeenAt, &addr.Label, &addr.Tags, &addr.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return addr, nil
}

// GetAddressTransactions retrieves transactions for an address
func (r *Repository) GetAddressTransactions(ctx context.Context, network, address string, opts PaginationOptions) (*ListResult[Transaction], error) {
	// Count query
	countQuery := `
		SELECT COUNT(*) FROM transactions
		WHERE network = $1 AND (from_address = $2 OR to_address = $2)`
	var total int64
	if err := r.pool.QueryRow(ctx, countQuery, network, address).Scan(&total); err != nil {
		return nil, err
	}

	// Data query
	query := `
		SELECT id, network, tx_hash, block_number, block_hash, tx_index,
			   from_address, to_address, value, gas_price, gas_limit, gas_used,
			   max_fee_per_gas, max_priority_fee_per_gas, input_data, nonce,
			   tx_type, status, timestamp, contract_address, error_message, created_at
		FROM transactions
		WHERE network = $1 AND (from_address = $2 OR to_address = $2)
		ORDER BY block_number DESC, tx_index DESC
		LIMIT $3 OFFSET $4`

	rows, err := r.pool.Query(ctx, query, network, address, opts.PageSize, opts.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txs []Transaction
	for rows.Next() {
		var tx Transaction
		if err := rows.Scan(
			&tx.ID, &tx.Network, &tx.TxHash, &tx.BlockNumber, &tx.BlockHash, &tx.TxIndex,
			&tx.From, &tx.To, &tx.Value, &tx.GasPrice, &tx.GasLimit, &tx.GasUsed,
			&tx.MaxFeePerGas, &tx.MaxPriorityFeePerGas, &tx.InputData, &tx.Nonce,
			&tx.TxType, &tx.Status, &tx.Timestamp, &tx.ContractAddress, &tx.ErrorMessage, &tx.CreatedAt,
		); err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}

	totalPages := int(total) / opts.PageSize
	if int(total)%opts.PageSize > 0 {
		totalPages++
	}

	return &ListResult[Transaction]{
		Items:      txs,
		Total:      total,
		Page:       opts.Page,
		PageSize:   opts.PageSize,
		TotalPages: totalPages,
	}, nil
}

// IncrementAddressTxCount increments the transaction count for addresses
func (r *Repository) IncrementAddressTxCount(ctx context.Context, network string, addresses []string, timestamp time.Time) error {
	if len(addresses) == 0 {
		return nil
	}

	query := `
		INSERT INTO addresses (network, address, tx_count, first_seen_at, last_seen_at)
		VALUES ($1, $2, 1, $3, $3)
		ON CONFLICT (network, address) DO UPDATE SET
			tx_count = addresses.tx_count + 1,
			last_seen_at = $3,
			updated_at = NOW()`

	batch := &pgx.Batch{}
	for _, addr := range addresses {
		batch.Queue(query, network, addr, timestamp)
	}

	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()

	for i := 0; i < len(addresses); i++ {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("increment address tx count %d: %w", i, err)
		}
	}
	return nil
}

// ============================================================================
// EVENT LOGS
// ============================================================================

// InsertEventLog inserts a new event log
func (r *Repository) InsertEventLog(ctx context.Context, log *EventLog) error {
	decodedArgs, _ := json.Marshal(log.DecodedArgs)

	query := `
		INSERT INTO event_logs (
			network, tx_hash, log_index, block_number, contract_address,
			topic0, topic1, topic2, topic3, data, timestamp,
			decoded_name, decoded_args, removed
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (network, tx_hash, log_index) DO NOTHING
		RETURNING id`

	return r.pool.QueryRow(ctx, query,
		log.Network, log.TxHash, log.LogIndex, log.BlockNumber, log.ContractAddress,
		log.Topic0, log.Topic1, log.Topic2, log.Topic3, log.Data, log.Timestamp,
		log.DecodedName, decodedArgs, log.Removed,
	).Scan(&log.ID)
}

// InsertEventLogs inserts multiple event logs in a batch
func (r *Repository) InsertEventLogs(ctx context.Context, logs []*EventLog) error {
	if len(logs) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	query := `
		INSERT INTO event_logs (
			network, tx_hash, log_index, block_number, contract_address,
			topic0, topic1, topic2, topic3, data, timestamp,
			decoded_name, decoded_args, removed
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (network, tx_hash, log_index) DO NOTHING`

	for _, log := range logs {
		decodedArgs, _ := json.Marshal(log.DecodedArgs)
		batch.Queue(query,
			log.Network, log.TxHash, log.LogIndex, log.BlockNumber, log.ContractAddress,
			log.Topic0, log.Topic1, log.Topic2, log.Topic3, log.Data, log.Timestamp,
			log.DecodedName, decodedArgs, log.Removed,
		)
	}

	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()

	for i := 0; i < len(logs); i++ {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("batch insert event log %d: %w", i, err)
		}
	}
	return nil
}

// GetTransactionLogs retrieves event logs for a transaction
func (r *Repository) GetTransactionLogs(ctx context.Context, network, txHash string) ([]EventLog, error) {
	query := `
		SELECT id, network, tx_hash, log_index, block_number, contract_address,
			   topic0, topic1, topic2, topic3, data, timestamp,
			   decoded_name, decoded_args, removed, created_at
		FROM event_logs
		WHERE network = $1 AND tx_hash = $2
		ORDER BY log_index ASC`

	rows, err := r.pool.Query(ctx, query, network, txHash)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []EventLog
	for rows.Next() {
		var log EventLog
		var decodedArgs []byte
		if err := rows.Scan(
			&log.ID, &log.Network, &log.TxHash, &log.LogIndex, &log.BlockNumber, &log.ContractAddress,
			&log.Topic0, &log.Topic1, &log.Topic2, &log.Topic3, &log.Data, &log.Timestamp,
			&log.DecodedName, &decodedArgs, &log.Removed, &log.CreatedAt,
		); err != nil {
			return nil, err
		}
		if len(decodedArgs) > 0 {
			json.Unmarshal(decodedArgs, &log.DecodedArgs)
		}
		logs = append(logs, log)
	}
	return logs, nil
}

// GetAddressLogs retrieves event logs for an address (contract)
func (r *Repository) GetAddressLogs(ctx context.Context, network, address string, opts PaginationOptions) (*ListResult[EventLog], error) {
	// Count query
	countQuery := `SELECT COUNT(*) FROM event_logs WHERE network = $1 AND contract_address = $2`
	var total int64
	if err := r.pool.QueryRow(ctx, countQuery, network, address).Scan(&total); err != nil {
		return nil, err
	}

	// Data query
	query := `
		SELECT id, network, tx_hash, log_index, block_number, contract_address,
			   topic0, topic1, topic2, topic3, data, timestamp,
			   decoded_name, decoded_args, removed, created_at
		FROM event_logs
		WHERE network = $1 AND contract_address = $2
		ORDER BY block_number DESC, log_index DESC
		LIMIT $3 OFFSET $4`

	rows, err := r.pool.Query(ctx, query, network, address, opts.PageSize, opts.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []EventLog
	for rows.Next() {
		var log EventLog
		var decodedArgs []byte
		if err := rows.Scan(
			&log.ID, &log.Network, &log.TxHash, &log.LogIndex, &log.BlockNumber, &log.ContractAddress,
			&log.Topic0, &log.Topic1, &log.Topic2, &log.Topic3, &log.Data, &log.Timestamp,
			&log.DecodedName, &decodedArgs, &log.Removed, &log.CreatedAt,
		); err != nil {
			return nil, err
		}
		if len(decodedArgs) > 0 {
			json.Unmarshal(decodedArgs, &log.DecodedArgs)
		}
		logs = append(logs, log)
	}

	totalPages := int(total) / opts.PageSize
	if int(total)%opts.PageSize > 0 {
		totalPages++
	}

	return &ListResult[EventLog]{
		Items:      logs,
		Total:      total,
		Page:       opts.Page,
		PageSize:   opts.PageSize,
		TotalPages: totalPages,
	}, nil
}

// ============================================================================
// SYNC STATE
// ============================================================================

// GetSyncState retrieves the sync state for a network
func (r *Repository) GetSyncState(ctx context.Context, network string) (*NetworkSyncState, error) {
	query := `
		SELECT id, network, last_indexed_block, last_indexed_at, is_syncing,
			   sync_started_at, blocks_behind, error_message, updated_at
		FROM network_sync_state
		WHERE network = $1`

	state := &NetworkSyncState{}
	err := r.pool.QueryRow(ctx, query, network).Scan(
		&state.ID, &state.Network, &state.LastIndexedBlock, &state.LastIndexedAt, &state.IsSyncing,
		&state.SyncStartedAt, &state.BlocksBehind, &state.ErrorMessage, &state.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return state, nil
}

// UpdateSyncState updates the sync state for a network
func (r *Repository) UpdateSyncState(ctx context.Context, network string, lastBlock int64, isSyncing bool, blocksBehind int64, errMsg *string) error {
	now := time.Now()
	query := `
		INSERT INTO network_sync_state (network, last_indexed_block, last_indexed_at, is_syncing, blocks_behind, error_message)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (network) DO UPDATE SET
			last_indexed_block = EXCLUDED.last_indexed_block,
			last_indexed_at = EXCLUDED.last_indexed_at,
			is_syncing = EXCLUDED.is_syncing,
			blocks_behind = EXCLUDED.blocks_behind,
			error_message = EXCLUDED.error_message,
			updated_at = NOW()`

	_, err := r.pool.Exec(ctx, query, network, lastBlock, now, isSyncing, blocksBehind, errMsg)
	return err
}

// ============================================================================
// STATISTICS
// ============================================================================

// GetNetworkStats retrieves aggregated statistics for a network
func (r *Repository) GetNetworkStats(ctx context.Context, network string) (*NetworkStats, error) {
	stats := &NetworkStats{
		Network:     network,
		LastUpdated: time.Now(),
	}

	// Get latest block
	latestBlock, err := r.GetLatestBlock(ctx, network)
	if err != nil {
		return nil, err
	}
	if latestBlock != nil {
		stats.LatestBlock = latestBlock.BlockNumber
	}

	// Count transactions
	err = r.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM transactions WHERE network = $1", network,
	).Scan(&stats.TotalTransactions)
	if err != nil {
		return nil, err
	}

	// Count addresses
	err = r.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM addresses WHERE network = $1", network,
	).Scan(&stats.TotalAddresses)
	if err != nil {
		return nil, err
	}

	// Count contracts
	err = r.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM addresses WHERE network = $1 AND is_contract = TRUE", network,
	).Scan(&stats.TotalContracts)
	if err != nil {
		return nil, err
	}

	return stats, nil
}
