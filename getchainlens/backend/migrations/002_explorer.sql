-- ChainLens Explorer Schema
-- Version: 1.1.0
-- Description: Block explorer tables for blocks, transactions, addresses, and event logs

-- ============================================================================
-- BLOCKS
-- ============================================================================
CREATE TABLE IF NOT EXISTS blocks (
    id BIGSERIAL PRIMARY KEY,
    network VARCHAR(50) NOT NULL,
    block_number BIGINT NOT NULL,
    block_hash VARCHAR(66) NOT NULL,
    parent_hash VARCHAR(66) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    miner VARCHAR(42),
    gas_used BIGINT,
    gas_limit BIGINT,
    base_fee_per_gas BIGINT,
    transaction_count INT DEFAULT 0,
    size INT,
    extra_data TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(network, block_number),
    UNIQUE(network, block_hash)
);

CREATE INDEX IF NOT EXISTS idx_blocks_network_number ON blocks(network, block_number DESC);
CREATE INDEX IF NOT EXISTS idx_blocks_timestamp ON blocks(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_blocks_miner ON blocks(miner);
CREATE INDEX IF NOT EXISTS idx_blocks_hash ON blocks(block_hash);

-- ============================================================================
-- TRANSACTIONS
-- ============================================================================
CREATE TABLE IF NOT EXISTS transactions (
    id BIGSERIAL PRIMARY KEY,
    network VARCHAR(50) NOT NULL,
    tx_hash VARCHAR(66) NOT NULL,
    block_number BIGINT NOT NULL,
    block_hash VARCHAR(66) NOT NULL,
    tx_index INT NOT NULL,
    from_address VARCHAR(42) NOT NULL,
    to_address VARCHAR(42),
    value NUMERIC(78, 0) NOT NULL DEFAULT 0,
    gas_price BIGINT,
    gas_limit BIGINT NOT NULL,
    gas_used BIGINT,
    max_fee_per_gas BIGINT,
    max_priority_fee_per_gas BIGINT,
    input_data TEXT,
    nonce BIGINT NOT NULL,
    tx_type SMALLINT DEFAULT 0,
    status SMALLINT, -- 0=failed, 1=success, NULL=pending
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    contract_address VARCHAR(42), -- if contract creation
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(network, tx_hash)
);

CREATE INDEX IF NOT EXISTS idx_txs_network_block ON transactions(network, block_number DESC);
CREATE INDEX IF NOT EXISTS idx_txs_from ON transactions(from_address, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_txs_to ON transactions(to_address, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_txs_timestamp ON transactions(network, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_txs_hash ON transactions(tx_hash);
CREATE INDEX IF NOT EXISTS idx_txs_contract ON transactions(contract_address) WHERE contract_address IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_txs_block_index ON transactions(network, block_number, tx_index);

-- ============================================================================
-- ADDRESSES / ACCOUNTS
-- ============================================================================
CREATE TABLE IF NOT EXISTS addresses (
    id BIGSERIAL PRIMARY KEY,
    network VARCHAR(50) NOT NULL,
    address VARCHAR(42) NOT NULL,
    balance NUMERIC(78, 0) DEFAULT 0,
    tx_count BIGINT DEFAULT 0,
    is_contract BOOLEAN DEFAULT FALSE,
    contract_creator VARCHAR(42),
    contract_creation_tx VARCHAR(66),
    code_hash VARCHAR(66),
    first_seen_at TIMESTAMP WITH TIME ZONE,
    last_seen_at TIMESTAMP WITH TIME ZONE,
    label VARCHAR(255), -- ENS, known addresses
    tags TEXT[], -- ['exchange', 'defi', 'nft']
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(network, address)
);

CREATE INDEX IF NOT EXISTS idx_addresses_balance ON addresses(network, balance DESC);
CREATE INDEX IF NOT EXISTS idx_addresses_tx_count ON addresses(network, tx_count DESC);
CREATE INDEX IF NOT EXISTS idx_addresses_contract ON addresses(network, is_contract) WHERE is_contract = TRUE;
CREATE INDEX IF NOT EXISTS idx_addresses_label ON addresses(label) WHERE label IS NOT NULL;

-- ============================================================================
-- EVENT LOGS
-- ============================================================================
CREATE TABLE IF NOT EXISTS event_logs (
    id BIGSERIAL PRIMARY KEY,
    network VARCHAR(50) NOT NULL,
    tx_hash VARCHAR(66) NOT NULL,
    log_index INT NOT NULL,
    block_number BIGINT NOT NULL,
    contract_address VARCHAR(42) NOT NULL,
    topic0 VARCHAR(66), -- event signature hash
    topic1 VARCHAR(66),
    topic2 VARCHAR(66),
    topic3 VARCHAR(66),
    data TEXT,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    decoded_name VARCHAR(100),
    decoded_args JSONB,
    removed BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(network, tx_hash, log_index)
);

CREATE INDEX IF NOT EXISTS idx_logs_contract ON event_logs(contract_address, block_number DESC);
CREATE INDEX IF NOT EXISTS idx_logs_topic0 ON event_logs(topic0);
CREATE INDEX IF NOT EXISTS idx_logs_topic1 ON event_logs(topic1) WHERE topic1 IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_logs_block ON event_logs(network, block_number DESC);
CREATE INDEX IF NOT EXISTS idx_logs_tx ON event_logs(tx_hash);
CREATE INDEX IF NOT EXISTS idx_logs_decoded ON event_logs(decoded_name) WHERE decoded_name IS NOT NULL;

-- Composite index for common queries
CREATE INDEX IF NOT EXISTS idx_logs_contract_topic ON event_logs(contract_address, topic0, block_number DESC);

-- ============================================================================
-- INTERNAL TRANSACTIONS (from traces)
-- ============================================================================
CREATE TABLE IF NOT EXISTS internal_transactions (
    id BIGSERIAL PRIMARY KEY,
    network VARCHAR(50) NOT NULL,
    tx_hash VARCHAR(66) NOT NULL,
    trace_address TEXT NOT NULL, -- e.g., "0,1,0" for nested calls
    block_number BIGINT NOT NULL,
    trace_type VARCHAR(20) NOT NULL, -- CALL, CREATE, CREATE2, DELEGATECALL, STATICCALL, SELFDESTRUCT
    call_type VARCHAR(20), -- call, delegatecall, staticcall
    from_address VARCHAR(42) NOT NULL,
    to_address VARCHAR(42),
    value NUMERIC(78, 0) DEFAULT 0,
    gas BIGINT,
    gas_used BIGINT,
    input_data TEXT,
    output_data TEXT,
    error TEXT,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(network, tx_hash, trace_address)
);

CREATE INDEX IF NOT EXISTS idx_internal_txs_from ON internal_transactions(from_address, block_number DESC);
CREATE INDEX IF NOT EXISTS idx_internal_txs_to ON internal_transactions(to_address, block_number DESC);
CREATE INDEX IF NOT EXISTS idx_internal_txs_block ON internal_transactions(network, block_number DESC);
CREATE INDEX IF NOT EXISTS idx_internal_txs_tx ON internal_transactions(tx_hash);
CREATE INDEX IF NOT EXISTS idx_internal_txs_type ON internal_transactions(trace_type);

-- ============================================================================
-- NETWORK SYNC STATE
-- ============================================================================
CREATE TABLE IF NOT EXISTS network_sync_state (
    id SERIAL PRIMARY KEY,
    network VARCHAR(50) UNIQUE NOT NULL,
    last_indexed_block BIGINT NOT NULL DEFAULT 0,
    last_indexed_at TIMESTAMP WITH TIME ZONE,
    is_syncing BOOLEAN DEFAULT FALSE,
    sync_started_at TIMESTAMP WITH TIME ZONE,
    blocks_behind BIGINT DEFAULT 0,
    error_message TEXT,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sync_state_network ON network_sync_state(network);

-- ============================================================================
-- UNCLE BLOCKS (for Ethereum)
-- ============================================================================
CREATE TABLE IF NOT EXISTS uncle_blocks (
    id BIGSERIAL PRIMARY KEY,
    network VARCHAR(50) NOT NULL,
    block_number BIGINT NOT NULL, -- main chain block number
    uncle_index INT NOT NULL,
    uncle_hash VARCHAR(66) NOT NULL,
    uncle_number BIGINT NOT NULL,
    miner VARCHAR(42),
    timestamp TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(network, block_number, uncle_index)
);

CREATE INDEX IF NOT EXISTS idx_uncles_block ON uncle_blocks(network, block_number);
CREATE INDEX IF NOT EXISTS idx_uncles_miner ON uncle_blocks(miner);

-- ============================================================================
-- TRIGGERS
-- ============================================================================

-- Update addresses.updated_at
CREATE TRIGGER update_addresses_updated_at
    BEFORE UPDATE ON addresses
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Update network_sync_state.updated_at
CREATE TRIGGER update_network_sync_state_updated_at
    BEFORE UPDATE ON network_sync_state
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- HELPER FUNCTIONS
-- ============================================================================

-- Function to get address transaction count
CREATE OR REPLACE FUNCTION get_address_tx_count(p_network VARCHAR, p_address VARCHAR)
RETURNS BIGINT AS $$
BEGIN
    RETURN (
        SELECT COUNT(*)
        FROM transactions
        WHERE network = p_network
          AND (from_address = p_address OR to_address = p_address)
    );
END;
$$ LANGUAGE plpgsql;

-- Function to calculate address balance from transactions (for verification)
CREATE OR REPLACE FUNCTION calculate_address_balance(p_network VARCHAR, p_address VARCHAR)
RETURNS NUMERIC AS $$
DECLARE
    received NUMERIC;
    sent NUMERIC;
BEGIN
    SELECT COALESCE(SUM(value), 0) INTO received
    FROM transactions
    WHERE network = p_network AND to_address = p_address AND status = 1;

    SELECT COALESCE(SUM(value), 0) INTO sent
    FROM transactions
    WHERE network = p_network AND from_address = p_address AND status = 1;

    RETURN received - sent;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- INITIAL DATA
-- ============================================================================

-- Pre-populate supported networks
INSERT INTO network_sync_state (network, last_indexed_block) VALUES
    ('ethereum', 0),
    ('polygon', 0),
    ('arbitrum', 0),
    ('optimism', 0),
    ('base', 0),
    ('bsc', 0),
    ('avalanche', 0)
ON CONFLICT (network) DO NOTHING;

-- ============================================================================
-- COMMENTS
-- ============================================================================

COMMENT ON TABLE blocks IS 'Indexed blockchain blocks';
COMMENT ON TABLE transactions IS 'Indexed blockchain transactions';
COMMENT ON TABLE addresses IS 'Known addresses with aggregated stats';
COMMENT ON TABLE event_logs IS 'Contract event logs';
COMMENT ON TABLE internal_transactions IS 'Internal transactions from trace data';
COMMENT ON TABLE network_sync_state IS 'Indexer sync state per network';

COMMENT ON COLUMN blocks.base_fee_per_gas IS 'EIP-1559 base fee, NULL for legacy chains';
COMMENT ON COLUMN transactions.tx_type IS '0=legacy, 1=access list, 2=EIP-1559, 3=blob';
COMMENT ON COLUMN transactions.status IS '0=failed, 1=success, NULL=pending';
COMMENT ON COLUMN addresses.tags IS 'Array of tags like exchange, defi, nft, bridge';
COMMENT ON COLUMN event_logs.topic0 IS 'Keccak256 hash of event signature';
COMMENT ON COLUMN internal_transactions.trace_address IS 'Trace path like 0,1,0 for nested calls';
