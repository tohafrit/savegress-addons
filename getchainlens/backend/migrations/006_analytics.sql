-- ChainLens Analytics Schema
-- Version: 1.0.0
-- Description: Tables for network analytics, statistics, and gas tracking

-- ============================================================================
-- DAILY STATISTICS
-- ============================================================================
CREATE TABLE IF NOT EXISTS daily_stats (
    id BIGSERIAL PRIMARY KEY,
    network VARCHAR(50) NOT NULL,
    date DATE NOT NULL,

    -- Block metrics
    block_count INT DEFAULT 0,
    first_block BIGINT,
    last_block BIGINT,
    avg_block_time FLOAT, -- seconds

    -- Transaction metrics
    transaction_count BIGINT DEFAULT 0,
    successful_tx_count BIGINT DEFAULT 0,
    failed_tx_count BIGINT DEFAULT 0,
    contract_creation_count INT DEFAULT 0,

    -- Address metrics
    unique_senders BIGINT DEFAULT 0,
    unique_receivers BIGINT DEFAULT 0,
    new_addresses BIGINT DEFAULT 0,

    -- Value metrics
    total_value_transferred NUMERIC(78, 0) DEFAULT 0, -- in wei
    avg_value_per_tx NUMERIC(78, 0),

    -- Gas metrics
    total_gas_used NUMERIC(78, 0) DEFAULT 0,
    avg_gas_per_tx BIGINT,
    avg_gas_price NUMERIC(78, 0),
    min_gas_price NUMERIC(78, 0),
    max_gas_price NUMERIC(78, 0),
    avg_base_fee NUMERIC(78, 0),
    total_fees_burned NUMERIC(78, 0) DEFAULT 0, -- for EIP-1559

    -- Token metrics
    token_transfer_count BIGINT DEFAULT 0,
    unique_tokens_transferred INT DEFAULT 0,

    -- NFT metrics
    nft_transfer_count BIGINT DEFAULT 0,
    nft_mint_count BIGINT DEFAULT 0,
    unique_nft_collections INT DEFAULT 0,

    -- Contract metrics
    contract_deploy_count INT DEFAULT 0,
    verified_contracts_count INT DEFAULT 0,
    contract_call_count BIGINT DEFAULT 0,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(network, date)
);

CREATE INDEX IF NOT EXISTS idx_daily_stats_network_date ON daily_stats(network, date DESC);
CREATE INDEX IF NOT EXISTS idx_daily_stats_tx_count ON daily_stats(network, transaction_count DESC);

-- ============================================================================
-- HOURLY STATISTICS (for more granular data)
-- ============================================================================
CREATE TABLE IF NOT EXISTS hourly_stats (
    id BIGSERIAL PRIMARY KEY,
    network VARCHAR(50) NOT NULL,
    hour TIMESTAMP WITH TIME ZONE NOT NULL, -- truncated to hour

    block_count INT DEFAULT 0,
    transaction_count BIGINT DEFAULT 0,
    unique_addresses INT DEFAULT 0,
    total_gas_used NUMERIC(78, 0) DEFAULT 0,
    avg_gas_price NUMERIC(78, 0),
    token_transfer_count BIGINT DEFAULT 0,
    nft_transfer_count BIGINT DEFAULT 0,
    total_value_transferred NUMERIC(78, 0) DEFAULT 0,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(network, hour)
);

CREATE INDEX IF NOT EXISTS idx_hourly_stats_network_hour ON hourly_stats(network, hour DESC);

-- ============================================================================
-- GAS PRICES
-- ============================================================================
CREATE TABLE IF NOT EXISTS gas_prices (
    id BIGSERIAL PRIMARY KEY,
    network VARCHAR(50) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,

    -- Gas prices in wei
    slow BIGINT,              -- ~10 min confirmation
    standard BIGINT,          -- ~3 min confirmation
    fast BIGINT,              -- ~1 min confirmation
    instant BIGINT,           -- next block

    -- EIP-1559 data
    base_fee BIGINT,
    priority_fee_slow BIGINT,
    priority_fee_standard BIGINT,
    priority_fee_fast BIGINT,

    -- Block info
    block_number BIGINT,
    pending_tx_count INT,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(network, timestamp)
);

CREATE INDEX IF NOT EXISTS idx_gas_prices_network_time ON gas_prices(network, timestamp DESC);

-- ============================================================================
-- TOP TOKENS (aggregated rankings)
-- ============================================================================
CREATE TABLE IF NOT EXISTS top_tokens (
    id BIGSERIAL PRIMARY KEY,
    network VARCHAR(50) NOT NULL,
    date DATE NOT NULL,
    rank INT NOT NULL,

    token_address VARCHAR(42) NOT NULL,
    token_name VARCHAR(255),
    token_symbol VARCHAR(50),

    transfer_count BIGINT DEFAULT 0,
    unique_holders INT DEFAULT 0,
    volume NUMERIC(78, 0) DEFAULT 0, -- in token's smallest unit

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(network, date, rank)
);

CREATE INDEX IF NOT EXISTS idx_top_tokens_network_date ON top_tokens(network, date DESC);

-- ============================================================================
-- TOP CONTRACTS (most active)
-- ============================================================================
CREATE TABLE IF NOT EXISTS top_contracts (
    id BIGSERIAL PRIMARY KEY,
    network VARCHAR(50) NOT NULL,
    date DATE NOT NULL,
    rank INT NOT NULL,

    contract_address VARCHAR(42) NOT NULL,
    contract_name VARCHAR(255),

    call_count BIGINT DEFAULT 0,
    unique_callers INT DEFAULT 0,
    gas_used NUMERIC(78, 0) DEFAULT 0,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(network, date, rank)
);

CREATE INDEX IF NOT EXISTS idx_top_contracts_network_date ON top_contracts(network, date DESC);

-- ============================================================================
-- NETWORK OVERVIEW (current state snapshot)
-- ============================================================================
CREATE TABLE IF NOT EXISTS network_overview (
    id SERIAL PRIMARY KEY,
    network VARCHAR(50) NOT NULL UNIQUE,

    -- Current state
    latest_block BIGINT,
    latest_block_time TIMESTAMP WITH TIME ZONE,
    pending_tx_count INT DEFAULT 0,

    -- Cumulative stats
    total_blocks BIGINT DEFAULT 0,
    total_transactions BIGINT DEFAULT 0,
    total_addresses BIGINT DEFAULT 0,
    total_contracts BIGINT DEFAULT 0,
    total_tokens BIGINT DEFAULT 0,
    total_nft_collections BIGINT DEFAULT 0,

    -- 24h metrics
    tx_count_24h BIGINT DEFAULT 0,
    active_addresses_24h BIGINT DEFAULT 0,
    gas_used_24h NUMERIC(78, 0) DEFAULT 0,
    avg_gas_price_24h NUMERIC(78, 0),
    token_transfers_24h BIGINT DEFAULT 0,
    nft_transfers_24h BIGINT DEFAULT 0,

    -- Current gas prices
    gas_price_slow BIGINT,
    gas_price_standard BIGINT,
    gas_price_fast BIGINT,
    base_fee BIGINT,

    -- Network info
    chain_id INT,
    native_currency VARCHAR(10),
    native_currency_decimals INT DEFAULT 18,

    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- ============================================================================
-- ADDRESS RANKINGS
-- ============================================================================
CREATE TABLE IF NOT EXISTS address_rankings (
    id BIGSERIAL PRIMARY KEY,
    network VARCHAR(50) NOT NULL,
    date DATE NOT NULL,
    ranking_type VARCHAR(20) NOT NULL, -- 'balance', 'tx_count', 'gas_spent'
    rank INT NOT NULL,

    address VARCHAR(42) NOT NULL,
    value NUMERIC(78, 0) NOT NULL, -- balance/count/gas depending on type

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(network, date, ranking_type, rank)
);

CREATE INDEX IF NOT EXISTS idx_address_rankings_network_date ON address_rankings(network, date DESC, ranking_type);

-- ============================================================================
-- TRIGGERS
-- ============================================================================

-- Update daily_stats updated_at
CREATE OR REPLACE FUNCTION update_daily_stats_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_daily_stats_updated_at
    BEFORE UPDATE ON daily_stats
    FOR EACH ROW EXECUTE FUNCTION update_daily_stats_timestamp();

-- Update network_overview updated_at
CREATE OR REPLACE FUNCTION update_network_overview_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_network_overview_updated_at
    BEFORE UPDATE ON network_overview
    FOR EACH ROW EXECUTE FUNCTION update_network_overview_timestamp();

-- ============================================================================
-- FUNCTIONS
-- ============================================================================

-- Function to aggregate daily stats from transactions
CREATE OR REPLACE FUNCTION aggregate_daily_stats(p_network VARCHAR, p_date DATE)
RETURNS VOID AS $$
BEGIN
    INSERT INTO daily_stats (
        network, date,
        block_count, first_block, last_block,
        transaction_count, successful_tx_count, failed_tx_count,
        unique_senders, unique_receivers,
        total_value_transferred,
        total_gas_used, avg_gas_per_tx, avg_gas_price
    )
    SELECT
        p_network,
        p_date,
        COUNT(DISTINCT block_number),
        MIN(block_number),
        MAX(block_number),
        COUNT(*),
        SUM(CASE WHEN status = 1 THEN 1 ELSE 0 END),
        SUM(CASE WHEN status = 0 THEN 1 ELSE 0 END),
        COUNT(DISTINCT from_address),
        COUNT(DISTINCT to_address),
        COALESCE(SUM(value), 0),
        COALESCE(SUM(gas_used), 0),
        AVG(gas_used)::BIGINT,
        AVG(gas_price)
    FROM transactions
    WHERE network = p_network
      AND timestamp >= p_date
      AND timestamp < p_date + INTERVAL '1 day'
    ON CONFLICT (network, date) DO UPDATE SET
        block_count = EXCLUDED.block_count,
        first_block = EXCLUDED.first_block,
        last_block = EXCLUDED.last_block,
        transaction_count = EXCLUDED.transaction_count,
        successful_tx_count = EXCLUDED.successful_tx_count,
        failed_tx_count = EXCLUDED.failed_tx_count,
        unique_senders = EXCLUDED.unique_senders,
        unique_receivers = EXCLUDED.unique_receivers,
        total_value_transferred = EXCLUDED.total_value_transferred,
        total_gas_used = EXCLUDED.total_gas_used,
        avg_gas_per_tx = EXCLUDED.avg_gas_per_tx,
        avg_gas_price = EXCLUDED.avg_gas_price,
        updated_at = NOW();
END;
$$ LANGUAGE plpgsql;

-- Function to update network overview
CREATE OR REPLACE FUNCTION update_network_overview(p_network VARCHAR)
RETURNS VOID AS $$
DECLARE
    v_latest_block BIGINT;
    v_latest_time TIMESTAMP WITH TIME ZONE;
BEGIN
    -- Get latest block info
    SELECT block_number, timestamp INTO v_latest_block, v_latest_time
    FROM blocks
    WHERE network = p_network
    ORDER BY block_number DESC
    LIMIT 1;

    INSERT INTO network_overview (
        network, latest_block, latest_block_time,
        total_blocks, total_transactions, total_addresses,
        tx_count_24h, active_addresses_24h
    )
    VALUES (
        p_network, v_latest_block, v_latest_time,
        (SELECT COUNT(*) FROM blocks WHERE network = p_network),
        (SELECT COUNT(*) FROM transactions WHERE network = p_network),
        (SELECT COUNT(*) FROM addresses WHERE network = p_network),
        (SELECT COUNT(*) FROM transactions WHERE network = p_network AND timestamp > NOW() - INTERVAL '24 hours'),
        (SELECT COUNT(DISTINCT from_address) FROM transactions WHERE network = p_network AND timestamp > NOW() - INTERVAL '24 hours')
    )
    ON CONFLICT (network) DO UPDATE SET
        latest_block = EXCLUDED.latest_block,
        latest_block_time = EXCLUDED.latest_block_time,
        total_blocks = EXCLUDED.total_blocks,
        total_transactions = EXCLUDED.total_transactions,
        total_addresses = EXCLUDED.total_addresses,
        tx_count_24h = EXCLUDED.tx_count_24h,
        active_addresses_24h = EXCLUDED.active_addresses_24h;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- INITIAL DATA
-- ============================================================================

-- Initialize network overviews for supported networks
INSERT INTO network_overview (network, chain_id, native_currency, native_currency_decimals) VALUES
('ethereum', 1, 'ETH', 18),
('polygon', 137, 'MATIC', 18),
('arbitrum', 42161, 'ETH', 18),
('optimism', 10, 'ETH', 18),
('base', 8453, 'ETH', 18),
('bsc', 56, 'BNB', 18),
('avalanche', 43114, 'AVAX', 18)
ON CONFLICT (network) DO NOTHING;

-- ============================================================================
-- COMMENTS
-- ============================================================================
COMMENT ON TABLE daily_stats IS 'Daily aggregated statistics per network';
COMMENT ON TABLE hourly_stats IS 'Hourly aggregated statistics for more granular analysis';
COMMENT ON TABLE gas_prices IS 'Historical gas price data for each network';
COMMENT ON TABLE top_tokens IS 'Daily rankings of top tokens by activity';
COMMENT ON TABLE top_contracts IS 'Daily rankings of most active contracts';
COMMENT ON TABLE network_overview IS 'Current state snapshot for each network';
COMMENT ON TABLE address_rankings IS 'Daily rankings of top addresses by various metrics';
