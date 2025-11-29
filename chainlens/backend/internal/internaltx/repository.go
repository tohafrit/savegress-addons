package internaltx

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository provides database operations for internal transactions
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new internal transaction repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// InsertInternalTransaction inserts a single internal transaction
func (r *Repository) InsertInternalTransaction(ctx context.Context, tx *InternalTransaction) error {
	query := `
		INSERT INTO internal_transactions (
			network, tx_hash, trace_index, block_number,
			parent_trace_index, depth, trace_type,
			from_address, to_address, value,
			gas, gas_used, input_data, output_data,
			error, reverted, created_contract, timestamp
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18
		)
		ON CONFLICT (network, tx_hash, trace_index) DO UPDATE SET
			parent_trace_index = EXCLUDED.parent_trace_index,
			depth = EXCLUDED.depth,
			trace_type = EXCLUDED.trace_type,
			from_address = EXCLUDED.from_address,
			to_address = EXCLUDED.to_address,
			value = EXCLUDED.value,
			gas = EXCLUDED.gas,
			gas_used = EXCLUDED.gas_used,
			input_data = EXCLUDED.input_data,
			output_data = EXCLUDED.output_data,
			error = EXCLUDED.error,
			reverted = EXCLUDED.reverted,
			created_contract = EXCLUDED.created_contract`

	_, err := r.db.Exec(ctx, query,
		tx.Network, tx.TxHash, tx.TraceIndex, tx.BlockNumber,
		tx.ParentTraceIndex, tx.Depth, tx.TraceType,
		tx.FromAddress, tx.ToAddress, tx.Value,
		tx.Gas, tx.GasUsed, tx.InputData, tx.OutputData,
		tx.Error, tx.Reverted, tx.CreatedContract, tx.Timestamp,
	)
	return err
}

// InsertInternalTransactionsBatch inserts multiple internal transactions in a batch
func (r *Repository) InsertInternalTransactionsBatch(ctx context.Context, txs []*InternalTransaction) error {
	if len(txs) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	query := `
		INSERT INTO internal_transactions (
			network, tx_hash, trace_index, block_number,
			parent_trace_index, depth, trace_type,
			from_address, to_address, value,
			gas, gas_used, input_data, output_data,
			error, reverted, created_contract, timestamp
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18
		)
		ON CONFLICT (network, tx_hash, trace_index) DO NOTHING`

	for _, tx := range txs {
		batch.Queue(query,
			tx.Network, tx.TxHash, tx.TraceIndex, tx.BlockNumber,
			tx.ParentTraceIndex, tx.Depth, tx.TraceType,
			tx.FromAddress, tx.ToAddress, tx.Value,
			tx.Gas, tx.GasUsed, tx.InputData, tx.OutputData,
			tx.Error, tx.Reverted, tx.CreatedContract, tx.Timestamp,
		)
	}

	br := r.db.SendBatch(ctx, batch)
	defer br.Close()

	for range txs {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("batch insert failed: %w", err)
		}
	}

	return nil
}

// GetByTxHash returns all internal transactions for a transaction hash
func (r *Repository) GetByTxHash(ctx context.Context, network, txHash string) ([]*InternalTransaction, error) {
	query := `
		SELECT id, network, tx_hash, trace_index, block_number,
			parent_trace_index, depth, trace_type,
			from_address, to_address, value,
			gas, gas_used, input_data, output_data,
			error, reverted, created_contract, timestamp, created_at
		FROM internal_transactions
		WHERE network = $1 AND tx_hash = $2
		ORDER BY trace_index`

	rows, err := r.db.Query(ctx, query, network, txHash)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanInternalTransactions(rows)
}

// GetByAddress returns internal transactions involving an address
func (r *Repository) GetByAddress(ctx context.Context, filter *InternalTxFilter) ([]*InternalTransaction, error) {
	var conditions []string
	var args []interface{}
	argNum := 1

	conditions = append(conditions, fmt.Sprintf("network = $%d", argNum))
	args = append(args, filter.Network)
	argNum++

	if filter.FromAddress != "" && filter.ToAddress != "" && filter.FromAddress == filter.ToAddress {
		conditions = append(conditions, fmt.Sprintf("(from_address = $%d OR to_address = $%d)", argNum, argNum))
		args = append(args, filter.FromAddress)
		argNum++
	} else {
		if filter.FromAddress != "" {
			conditions = append(conditions, fmt.Sprintf("from_address = $%d", argNum))
			args = append(args, filter.FromAddress)
			argNum++
		}
		if filter.ToAddress != "" {
			conditions = append(conditions, fmt.Sprintf("to_address = $%d", argNum))
			args = append(args, filter.ToAddress)
			argNum++
		}
	}

	if filter.TraceType != "" {
		conditions = append(conditions, fmt.Sprintf("trace_type = $%d", argNum))
		args = append(args, filter.TraceType)
		argNum++
	}

	if filter.BlockFrom > 0 {
		conditions = append(conditions, fmt.Sprintf("block_number >= $%d", argNum))
		args = append(args, filter.BlockFrom)
		argNum++
	}

	if filter.BlockTo > 0 {
		conditions = append(conditions, fmt.Sprintf("block_number <= $%d", argNum))
		args = append(args, filter.BlockTo)
		argNum++
	}

	if filter.MinValue != "" && filter.MinValue != "0" {
		conditions = append(conditions, fmt.Sprintf("value >= $%d", argNum))
		args = append(args, filter.MinValue)
		argNum++
	}

	pageSize := filter.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	page := filter.Page
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * pageSize

	query := fmt.Sprintf(`
		SELECT id, network, tx_hash, trace_index, block_number,
			parent_trace_index, depth, trace_type,
			from_address, to_address, value,
			gas, gas_used, input_data, output_data,
			error, reverted, created_contract, timestamp, created_at
		FROM internal_transactions
		WHERE %s
		ORDER BY block_number DESC, trace_index
		LIMIT $%d OFFSET $%d`,
		strings.Join(conditions, " AND "), argNum, argNum+1)

	args = append(args, pageSize, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanInternalTransactions(rows)
}

// GetCreatedContracts returns contracts created via internal transactions
func (r *Repository) GetCreatedContracts(ctx context.Context, network string, limit, offset int) ([]*InternalTransaction, error) {
	query := `
		SELECT id, network, tx_hash, trace_index, block_number,
			parent_trace_index, depth, trace_type,
			from_address, to_address, value,
			gas, gas_used, input_data, output_data,
			error, reverted, created_contract, timestamp, created_at
		FROM internal_transactions
		WHERE network = $1
			AND created_contract IS NOT NULL
			AND reverted = false
		ORDER BY block_number DESC, trace_index
		LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(ctx, query, network, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanInternalTransactions(rows)
}

// GetCallStats returns statistics about internal calls for a transaction
func (r *Repository) GetCallStats(ctx context.Context, network, txHash string) (*CallStats, error) {
	query := `
		SELECT
			COUNT(*) as total_calls,
			MAX(depth) as max_depth,
			COALESCE(SUM(value), 0) as total_value,
			COALESCE(SUM(gas_used), 0) as total_gas_used,
			COUNT(DISTINCT from_address) + COUNT(DISTINCT to_address) as unique_addresses,
			COUNT(*) FILTER (WHERE created_contract IS NOT NULL) as contracts_created,
			COUNT(*) FILTER (WHERE reverted = true OR error IS NOT NULL) as failed_calls
		FROM internal_transactions
		WHERE network = $1 AND tx_hash = $2`

	var stats CallStats
	var totalValue string
	err := r.db.QueryRow(ctx, query, network, txHash).Scan(
		&stats.TotalCalls,
		&stats.MaxDepth,
		&totalValue,
		&stats.TotalGasUsed,
		&stats.UniqueAddresses,
		&stats.ContractsCreated,
		&stats.FailedCalls,
	)
	if err != nil {
		return nil, err
	}
	stats.TotalValueMoved = totalValue

	// Get calls by type
	typeQuery := `
		SELECT trace_type, COUNT(*) as count
		FROM internal_transactions
		WHERE network = $1 AND tx_hash = $2
		GROUP BY trace_type`

	rows, err := r.db.Query(ctx, typeQuery, network, txHash)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats.CallsByType = make(map[string]int)
	for rows.Next() {
		var traceType string
		var count int
		if err := rows.Scan(&traceType, &count); err != nil {
			return nil, err
		}
		stats.CallsByType[traceType] = count
	}

	return &stats, nil
}

// DeleteByTxHash deletes all internal transactions for a transaction
func (r *Repository) DeleteByTxHash(ctx context.Context, network, txHash string) error {
	_, err := r.db.Exec(ctx,
		"DELETE FROM internal_transactions WHERE network = $1 AND tx_hash = $2",
		network, txHash)
	return err
}

// Trace Processing Status methods

// GetOrCreateProcessingStatus gets or creates a processing status record
func (r *Repository) GetOrCreateProcessingStatus(ctx context.Context, network, txHash string, blockNumber int64) (*TraceProcessingStatus, error) {
	query := `
		INSERT INTO trace_processing_status (network, block_number, tx_hash, status)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (network, tx_hash) DO UPDATE SET
			block_number = EXCLUDED.block_number
		RETURNING id, network, block_number, tx_hash, status, error_message, retry_count, last_attempt_at, created_at, updated_at`

	var status TraceProcessingStatus
	err := r.db.QueryRow(ctx, query, network, blockNumber, txHash, StatusPending).Scan(
		&status.ID, &status.Network, &status.BlockNumber, &status.TxHash,
		&status.Status, &status.ErrorMessage, &status.RetryCount, &status.LastAttemptAt,
		&status.CreatedAt, &status.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &status, nil
}

// UpdateProcessingStatus updates the processing status
func (r *Repository) UpdateProcessingStatus(ctx context.Context, network, txHash, status string, errorMsg *string) error {
	now := time.Now().UTC()
	query := `
		UPDATE trace_processing_status
		SET status = $3, error_message = $4, last_attempt_at = $5,
			retry_count = CASE WHEN $3 = 'failed' THEN retry_count + 1 ELSE retry_count END
		WHERE network = $1 AND tx_hash = $2`

	_, err := r.db.Exec(ctx, query, network, txHash, status, errorMsg, now)
	return err
}

// GetPendingTraces returns transactions that need to be traced
func (r *Repository) GetPendingTraces(ctx context.Context, network string, limit int, maxRetries int) ([]*PendingTraceJob, error) {
	query := `
		SELECT network, tx_hash, block_number, retry_count
		FROM trace_processing_status
		WHERE network = $1
			AND status IN ('pending', 'failed')
			AND retry_count < $3
		ORDER BY block_number ASC
		LIMIT $2
		FOR UPDATE SKIP LOCKED`

	rows, err := r.db.Query(ctx, query, network, limit, maxRetries)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*PendingTraceJob
	for rows.Next() {
		var job PendingTraceJob
		if err := rows.Scan(&job.Network, &job.TxHash, &job.BlockNumber, &job.RetryCount); err != nil {
			return nil, err
		}
		jobs = append(jobs, &job)
	}

	return jobs, rows.Err()
}

// MarkAsProcessing marks transactions as being processed
func (r *Repository) MarkAsProcessing(ctx context.Context, network string, txHashes []string) error {
	if len(txHashes) == 0 {
		return nil
	}

	now := time.Now().UTC()
	query := `
		UPDATE trace_processing_status
		SET status = $1, last_attempt_at = $2
		WHERE network = $3 AND tx_hash = ANY($4)`

	_, err := r.db.Exec(ctx, query, StatusProcessing, now, network, txHashes)
	return err
}

// GetProcessingStatus gets the status for a specific transaction
func (r *Repository) GetProcessingStatus(ctx context.Context, network, txHash string) (*TraceProcessingStatus, error) {
	query := `
		SELECT id, network, block_number, tx_hash, status, error_message,
			retry_count, last_attempt_at, created_at, updated_at
		FROM trace_processing_status
		WHERE network = $1 AND tx_hash = $2`

	var status TraceProcessingStatus
	err := r.db.QueryRow(ctx, query, network, txHash).Scan(
		&status.ID, &status.Network, &status.BlockNumber, &status.TxHash,
		&status.Status, &status.ErrorMessage, &status.RetryCount, &status.LastAttemptAt,
		&status.CreatedAt, &status.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &status, nil
}

// CountByStatus counts transactions by processing status
func (r *Repository) CountByStatus(ctx context.Context, network string) (map[string]int64, error) {
	query := `
		SELECT status, COUNT(*) as count
		FROM trace_processing_status
		WHERE network = $1
		GROUP BY status`

	rows, err := r.db.Query(ctx, query, network)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[string]int64)
	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		counts[status] = count
	}

	return counts, rows.Err()
}

// Helper functions

func scanInternalTransactions(rows pgx.Rows) ([]*InternalTransaction, error) {
	var txs []*InternalTransaction
	for rows.Next() {
		var tx InternalTransaction
		err := rows.Scan(
			&tx.ID, &tx.Network, &tx.TxHash, &tx.TraceIndex, &tx.BlockNumber,
			&tx.ParentTraceIndex, &tx.Depth, &tx.TraceType,
			&tx.FromAddress, &tx.ToAddress, &tx.Value,
			&tx.Gas, &tx.GasUsed, &tx.InputData, &tx.OutputData,
			&tx.Error, &tx.Reverted, &tx.CreatedContract, &tx.Timestamp, &tx.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		txs = append(txs, &tx)
	}
	return txs, rows.Err()
}
