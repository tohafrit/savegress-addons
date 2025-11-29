-- ChainLens Internal Transactions Schema
-- Version: 1.0.0
-- Description: Tables for storing internal transactions from call traces

-- ============================================================================
-- INTERNAL TRANSACTIONS
-- ============================================================================
CREATE TABLE IF NOT EXISTS internal_transactions (
    id BIGSERIAL PRIMARY KEY,
    network VARCHAR(50) NOT NULL,
    tx_hash VARCHAR(66) NOT NULL,
    trace_index INT NOT NULL,
    block_number BIGINT NOT NULL,

    -- Trace hierarchy
    parent_trace_index INT, -- NULL for root call
    depth INT DEFAULT 0,

    -- Call info
    trace_type VARCHAR(20) NOT NULL, -- CALL, STATICCALL, DELEGATECALL, CREATE, CREATE2, SUICIDE
    from_address VARCHAR(42) NOT NULL,
    to_address VARCHAR(42), -- NULL for CREATE

    -- Value and gas
    value NUMERIC(78, 0) DEFAULT 0,
    gas BIGINT,
    gas_used BIGINT,

    -- Data
    input_data TEXT,
    output_data TEXT,

    -- Status
    error TEXT,
    reverted BOOLEAN DEFAULT FALSE,

    -- Contract creation specific
    created_contract VARCHAR(42), -- for CREATE/CREATE2

    -- Timestamp from parent transaction
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(network, tx_hash, trace_index)
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_internal_txs_hash ON internal_transactions(network, tx_hash);
CREATE INDEX IF NOT EXISTS idx_internal_txs_from ON internal_transactions(from_address, block_number DESC);
CREATE INDEX IF NOT EXISTS idx_internal_txs_to ON internal_transactions(to_address, block_number DESC) WHERE to_address IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_internal_txs_block ON internal_transactions(network, block_number DESC);
CREATE INDEX IF NOT EXISTS idx_internal_txs_type ON internal_transactions(trace_type);
CREATE INDEX IF NOT EXISTS idx_internal_txs_created ON internal_transactions(created_contract) WHERE created_contract IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_internal_txs_value ON internal_transactions(network, value DESC) WHERE value > 0;

-- ============================================================================
-- TRACE PROCESSING STATUS
-- ============================================================================
CREATE TABLE IF NOT EXISTS trace_processing_status (
    id BIGSERIAL PRIMARY KEY,
    network VARCHAR(50) NOT NULL,
    block_number BIGINT NOT NULL,
    tx_hash VARCHAR(66) NOT NULL,

    -- Processing status
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending, processing, completed, failed, skipped
    error_message TEXT,

    -- Retry tracking
    retry_count INT DEFAULT 0,
    last_attempt_at TIMESTAMP WITH TIME ZONE,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(network, tx_hash)
);

CREATE INDEX IF NOT EXISTS idx_trace_status_pending ON trace_processing_status(network, status, block_number)
    WHERE status IN ('pending', 'failed');
CREATE INDEX IF NOT EXISTS idx_trace_status_block ON trace_processing_status(network, block_number);

-- ============================================================================
-- AGGREGATE VIEW FOR ADDRESS INTERNAL TRANSACTIONS
-- ============================================================================
CREATE OR REPLACE VIEW address_internal_tx_summary AS
SELECT
    network,
    address,
    SUM(tx_count) as total_internal_txs,
    SUM(value_in) as total_value_received,
    SUM(value_out) as total_value_sent,
    MAX(last_seen) as last_internal_tx
FROM (
    -- Incoming transactions
    SELECT
        network,
        to_address as address,
        COUNT(*) as tx_count,
        SUM(value) as value_in,
        0::NUMERIC as value_out,
        MAX(timestamp) as last_seen
    FROM internal_transactions
    WHERE to_address IS NOT NULL
    GROUP BY network, to_address

    UNION ALL

    -- Outgoing transactions
    SELECT
        network,
        from_address as address,
        COUNT(*) as tx_count,
        0::NUMERIC as value_in,
        SUM(value) as value_out,
        MAX(timestamp) as last_seen
    FROM internal_transactions
    GROUP BY network, from_address
) sub
GROUP BY network, address;

-- ============================================================================
-- TRIGGERS
-- ============================================================================

-- Update trace_processing_status updated_at
CREATE OR REPLACE FUNCTION update_trace_status_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_trace_status_updated_at
    BEFORE UPDATE ON trace_processing_status
    FOR EACH ROW EXECUTE FUNCTION update_trace_status_timestamp();

-- ============================================================================
-- FUNCTIONS
-- ============================================================================

-- Get internal transactions for a specific transaction
CREATE OR REPLACE FUNCTION get_internal_transactions(
    p_network VARCHAR,
    p_tx_hash VARCHAR
)
RETURNS TABLE (
    trace_index INT,
    parent_trace_index INT,
    depth INT,
    trace_type VARCHAR,
    from_address VARCHAR,
    to_address VARCHAR,
    value NUMERIC,
    gas_used BIGINT,
    input_data TEXT,
    output_data TEXT,
    error TEXT,
    reverted BOOLEAN,
    created_contract VARCHAR
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        it.trace_index,
        it.parent_trace_index,
        it.depth,
        it.trace_type,
        it.from_address,
        it.to_address,
        it.value,
        it.gas_used,
        it.input_data,
        it.output_data,
        it.error,
        it.reverted,
        it.created_contract
    FROM internal_transactions it
    WHERE it.network = p_network
      AND it.tx_hash = p_tx_hash
    ORDER BY it.trace_index;
END;
$$ LANGUAGE plpgsql;

-- Get address internal transactions with pagination
CREATE OR REPLACE FUNCTION get_address_internal_transactions(
    p_network VARCHAR,
    p_address VARCHAR,
    p_limit INT DEFAULT 100,
    p_offset INT DEFAULT 0
)
RETURNS TABLE (
    tx_hash VARCHAR,
    trace_index INT,
    block_number BIGINT,
    trace_type VARCHAR,
    from_address VARCHAR,
    to_address VARCHAR,
    value NUMERIC,
    gas_used BIGINT,
    error TEXT,
    timestamp TIMESTAMP WITH TIME ZONE
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        it.tx_hash,
        it.trace_index,
        it.block_number,
        it.trace_type,
        it.from_address,
        it.to_address,
        it.value,
        it.gas_used,
        it.error,
        it.timestamp
    FROM internal_transactions it
    WHERE it.network = p_network
      AND (it.from_address = p_address OR it.to_address = p_address)
    ORDER BY it.block_number DESC, it.trace_index
    LIMIT p_limit
    OFFSET p_offset;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- COMMENTS
-- ============================================================================
COMMENT ON TABLE internal_transactions IS 'Internal transactions extracted from call traces (debug_traceTransaction)';
COMMENT ON TABLE trace_processing_status IS 'Tracks which transactions have been traced for internal transactions';
COMMENT ON COLUMN internal_transactions.trace_index IS 'Sequential index of this call within the transaction trace';
COMMENT ON COLUMN internal_transactions.parent_trace_index IS 'Index of parent call, NULL for root call';
COMMENT ON COLUMN internal_transactions.trace_type IS 'Type of call: CALL, STATICCALL, DELEGATECALL, CREATE, CREATE2, SUICIDE';
COMMENT ON COLUMN internal_transactions.created_contract IS 'Address of contract created (for CREATE/CREATE2 operations)';
