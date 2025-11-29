-- ChainLens Token Tracking Schema
-- Version: 1.0.0
-- Description: Tables for ERC-20 tokens, transfers, and balances

-- ============================================================================
-- TOKENS (ERC-20)
-- ============================================================================
CREATE TABLE IF NOT EXISTS tokens (
    id BIGSERIAL PRIMARY KEY,
    network VARCHAR(50) NOT NULL,
    contract_address VARCHAR(42) NOT NULL,

    -- Token metadata
    name VARCHAR(255),
    symbol VARCHAR(50),
    decimals SMALLINT DEFAULT 18,
    total_supply NUMERIC(78, 0),

    -- Statistics
    holder_count BIGINT DEFAULT 0,
    transfer_count BIGINT DEFAULT 0,

    -- Additional info
    logo_url TEXT,
    website TEXT,
    description TEXT,
    social_links JSONB, -- {"twitter": "...", "telegram": "...", "discord": "..."}

    -- Verification
    is_verified BOOLEAN DEFAULT FALSE,
    coingecko_id VARCHAR(100),
    coinmarketcap_id INT,

    -- Token type
    token_type VARCHAR(20) DEFAULT 'ERC20', -- ERC20, ERC721, ERC1155

    -- Contract info
    is_proxy BOOLEAN DEFAULT FALSE,
    implementation_address VARCHAR(42),

    -- First seen
    first_block BIGINT,
    first_tx_hash VARCHAR(66),
    deployer_address VARCHAR(42),

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(network, contract_address)
);

CREATE INDEX IF NOT EXISTS idx_tokens_symbol ON tokens(symbol);
CREATE INDEX IF NOT EXISTS idx_tokens_name ON tokens(name);
CREATE INDEX IF NOT EXISTS idx_tokens_network ON tokens(network);
CREATE INDEX IF NOT EXISTS idx_tokens_holders ON tokens(network, holder_count DESC);
CREATE INDEX IF NOT EXISTS idx_tokens_transfers ON tokens(network, transfer_count DESC);
CREATE INDEX IF NOT EXISTS idx_tokens_type ON tokens(token_type);

-- ============================================================================
-- TOKEN TRANSFERS
-- ============================================================================
CREATE TABLE IF NOT EXISTS token_transfers (
    id BIGSERIAL PRIMARY KEY,
    network VARCHAR(50) NOT NULL,
    tx_hash VARCHAR(66) NOT NULL,
    log_index INT NOT NULL,
    block_number BIGINT NOT NULL,

    -- Transfer details
    token_address VARCHAR(42) NOT NULL,
    from_address VARCHAR(42) NOT NULL,
    to_address VARCHAR(42) NOT NULL,
    value NUMERIC(78, 0) NOT NULL,

    -- Decoded info (cached for performance)
    token_symbol VARCHAR(50),
    token_decimals SMALLINT,

    -- For ERC-1155 batch transfers
    token_id NUMERIC(78, 0), -- NULL for ERC-20

    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(network, tx_hash, log_index)
);

CREATE INDEX IF NOT EXISTS idx_transfers_token ON token_transfers(token_address, block_number DESC);
CREATE INDEX IF NOT EXISTS idx_transfers_from ON token_transfers(from_address, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_transfers_to ON token_transfers(to_address, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_transfers_block ON token_transfers(network, block_number DESC);
CREATE INDEX IF NOT EXISTS idx_transfers_timestamp ON token_transfers(timestamp DESC);

-- ============================================================================
-- TOKEN BALANCES
-- ============================================================================
CREATE TABLE IF NOT EXISTS token_balances (
    id BIGSERIAL PRIMARY KEY,
    network VARCHAR(50) NOT NULL,
    token_address VARCHAR(42) NOT NULL,
    holder_address VARCHAR(42) NOT NULL,

    balance NUMERIC(78, 0) NOT NULL DEFAULT 0,

    -- For tracking
    first_transfer_at TIMESTAMP WITH TIME ZONE,
    last_transfer_at TIMESTAMP WITH TIME ZONE,
    transfer_count INT DEFAULT 0,

    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(network, token_address, holder_address)
);

CREATE INDEX IF NOT EXISTS idx_balances_holder ON token_balances(holder_address);
CREATE INDEX IF NOT EXISTS idx_balances_token ON token_balances(token_address);
CREATE INDEX IF NOT EXISTS idx_balances_token_balance ON token_balances(token_address, balance DESC);
CREATE INDEX IF NOT EXISTS idx_balances_network ON token_balances(network);

-- Partial index for non-zero balances (most queries)
CREATE INDEX IF NOT EXISTS idx_balances_nonzero ON token_balances(token_address, balance DESC)
    WHERE balance > 0;

-- ============================================================================
-- TOKEN APPROVALS (for tracking allowances)
-- ============================================================================
CREATE TABLE IF NOT EXISTS token_approvals (
    id BIGSERIAL PRIMARY KEY,
    network VARCHAR(50) NOT NULL,
    tx_hash VARCHAR(66) NOT NULL,
    log_index INT NOT NULL,
    block_number BIGINT NOT NULL,

    token_address VARCHAR(42) NOT NULL,
    owner_address VARCHAR(42) NOT NULL,
    spender_address VARCHAR(42) NOT NULL,
    value NUMERIC(78, 0) NOT NULL, -- max uint256 for unlimited

    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(network, tx_hash, log_index)
);

CREATE INDEX IF NOT EXISTS idx_approvals_owner ON token_approvals(owner_address, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_approvals_spender ON token_approvals(spender_address, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_approvals_token ON token_approvals(token_address, block_number DESC);

-- ============================================================================
-- CURRENT ALLOWANCES (derived from approvals)
-- ============================================================================
CREATE TABLE IF NOT EXISTS token_allowances (
    id BIGSERIAL PRIMARY KEY,
    network VARCHAR(50) NOT NULL,
    token_address VARCHAR(42) NOT NULL,
    owner_address VARCHAR(42) NOT NULL,
    spender_address VARCHAR(42) NOT NULL,

    allowance NUMERIC(78, 0) NOT NULL DEFAULT 0,

    last_updated_block BIGINT,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(network, token_address, owner_address, spender_address)
);

CREATE INDEX IF NOT EXISTS idx_allowances_owner ON token_allowances(owner_address);
CREATE INDEX IF NOT EXISTS idx_allowances_spender ON token_allowances(spender_address);

-- ============================================================================
-- TOKEN PRICE HISTORY (optional, for analytics)
-- ============================================================================
CREATE TABLE IF NOT EXISTS token_prices (
    id BIGSERIAL PRIMARY KEY,
    network VARCHAR(50) NOT NULL,
    token_address VARCHAR(42) NOT NULL,

    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    price_usd NUMERIC(30, 18),
    price_eth NUMERIC(30, 18),
    market_cap_usd NUMERIC(30, 2),
    volume_24h_usd NUMERIC(30, 2),

    source VARCHAR(50), -- 'coingecko', 'dex', etc

    UNIQUE(network, token_address, timestamp)
);

CREATE INDEX IF NOT EXISTS idx_prices_token_time ON token_prices(token_address, timestamp DESC);

-- ============================================================================
-- WELL-KNOWN TOKENS (pre-populated list)
-- ============================================================================
CREATE TABLE IF NOT EXISTS well_known_tokens (
    id SERIAL PRIMARY KEY,
    network VARCHAR(50) NOT NULL,
    contract_address VARCHAR(42) NOT NULL,
    name VARCHAR(255) NOT NULL,
    symbol VARCHAR(50) NOT NULL,
    decimals SMALLINT NOT NULL,
    logo_url TEXT,
    coingecko_id VARCHAR(100),
    is_stablecoin BOOLEAN DEFAULT FALSE,
    is_wrapped_native BOOLEAN DEFAULT FALSE, -- WETH, WMATIC, etc

    UNIQUE(network, contract_address)
);

-- ============================================================================
-- TRIGGERS
-- ============================================================================

CREATE TRIGGER update_tokens_updated_at
    BEFORE UPDATE ON tokens
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_token_balances_updated_at
    BEFORE UPDATE ON token_balances
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_token_allowances_updated_at
    BEFORE UPDATE ON token_allowances
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- FUNCTIONS
-- ============================================================================

-- Function to update token holder count
CREATE OR REPLACE FUNCTION update_token_holder_count(p_network VARCHAR, p_token_address VARCHAR)
RETURNS VOID AS $$
BEGIN
    UPDATE tokens
    SET holder_count = (
        SELECT COUNT(*) FROM token_balances
        WHERE network = p_network
        AND token_address = p_token_address
        AND balance > 0
    ),
    updated_at = NOW()
    WHERE network = p_network AND contract_address = p_token_address;
END;
$$ LANGUAGE plpgsql;

-- Function to update token transfer count
CREATE OR REPLACE FUNCTION update_token_transfer_count(p_network VARCHAR, p_token_address VARCHAR)
RETURNS VOID AS $$
BEGIN
    UPDATE tokens
    SET transfer_count = (
        SELECT COUNT(*) FROM token_transfers
        WHERE network = p_network
        AND token_address = p_token_address
    ),
    updated_at = NOW()
    WHERE network = p_network AND contract_address = p_token_address;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- INITIAL DATA: Well-known tokens
-- ============================================================================

-- Ethereum Mainnet
INSERT INTO well_known_tokens (network, contract_address, name, symbol, decimals, coingecko_id, is_stablecoin, is_wrapped_native) VALUES
('ethereum', '0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2', 'Wrapped Ether', 'WETH', 18, 'weth', false, true),
('ethereum', '0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48', 'USD Coin', 'USDC', 6, 'usd-coin', true, false),
('ethereum', '0xdAC17F958D2ee523a2206206994597C13D831ec7', 'Tether USD', 'USDT', 6, 'tether', true, false),
('ethereum', '0x6B175474E89094C44Da98b954EescdeCB5f20000', 'Dai Stablecoin', 'DAI', 18, 'dai', true, false),
('ethereum', '0x2260FAC5E5542a773Aa44fBCfeDf7C193bc2C599', 'Wrapped BTC', 'WBTC', 8, 'wrapped-bitcoin', false, false),
('ethereum', '0x514910771AF9Ca656af840dff83E8264EcF986CA', 'ChainLink Token', 'LINK', 18, 'chainlink', false, false),
('ethereum', '0x1f9840a85d5aF5bf1D1762F925BDADdC4201F984', 'Uniswap', 'UNI', 18, 'uniswap', false, false),
('ethereum', '0x7Fc66500c84A76Ad7e9c93437bFc5Ac33E2DDaE9', 'Aave Token', 'AAVE', 18, 'aave', false, false)
ON CONFLICT (network, contract_address) DO NOTHING;

-- Polygon
INSERT INTO well_known_tokens (network, contract_address, name, symbol, decimals, coingecko_id, is_stablecoin, is_wrapped_native) VALUES
('polygon', '0x0d500B1d8E8eF31E21C99d1Db9A6444d3ADf1270', 'Wrapped Matic', 'WMATIC', 18, 'wmatic', false, true),
('polygon', '0x2791Bca1f2de4661ED88A30C99A7a9449Aa84174', 'USD Coin (PoS)', 'USDC', 6, 'usd-coin', true, false),
('polygon', '0xc2132D05D31c914a87C6611C10748AEb04B58e8F', 'Tether USD (PoS)', 'USDT', 6, 'tether', true, false),
('polygon', '0x7ceB23fD6bC0adD59E62ac25578270cFf1b9f619', 'Wrapped Ether (PoS)', 'WETH', 18, 'weth', false, false)
ON CONFLICT (network, contract_address) DO NOTHING;

-- Arbitrum
INSERT INTO well_known_tokens (network, contract_address, name, symbol, decimals, coingecko_id, is_stablecoin, is_wrapped_native) VALUES
('arbitrum', '0x82aF49447D8a07e3bd95BD0d56f35241523fBab1', 'Wrapped Ether', 'WETH', 18, 'weth', false, true),
('arbitrum', '0xFF970A61A04b1cA14834A43f5dE4533eBDDB5CC8', 'USD Coin (Arb1)', 'USDC', 6, 'usd-coin', true, false),
('arbitrum', '0xFd086bC7CD5C481DCC9C85ebE478A1C0b69FCbb9', 'Tether USD', 'USDT', 6, 'tether', true, false),
('arbitrum', '0x912CE59144191C1204E64559FE8253a0e49E6548', 'Arbitrum', 'ARB', 18, 'arbitrum', false, false)
ON CONFLICT (network, contract_address) DO NOTHING;

-- Base
INSERT INTO well_known_tokens (network, contract_address, name, symbol, decimals, coingecko_id, is_stablecoin, is_wrapped_native) VALUES
('base', '0x4200000000000000000000000000000000000006', 'Wrapped Ether', 'WETH', 18, 'weth', false, true),
('base', '0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913', 'USD Coin', 'USDC', 6, 'usd-coin', true, false)
ON CONFLICT (network, contract_address) DO NOTHING;

-- Optimism
INSERT INTO well_known_tokens (network, contract_address, name, symbol, decimals, coingecko_id, is_stablecoin, is_wrapped_native) VALUES
('optimism', '0x4200000000000000000000000000000000000006', 'Wrapped Ether', 'WETH', 18, 'weth', false, true),
('optimism', '0x7F5c764cBc14f9669B88837ca1490cCa17c31607', 'USD Coin', 'USDC', 6, 'usd-coin', true, false),
('optimism', '0x4200000000000000000000000000000000000042', 'Optimism', 'OP', 18, 'optimism', false, false)
ON CONFLICT (network, contract_address) DO NOTHING;

-- BSC
INSERT INTO well_known_tokens (network, contract_address, name, symbol, decimals, coingecko_id, is_stablecoin, is_wrapped_native) VALUES
('bsc', '0xbb4CdB9CBd36B01bD1cBaEBF2De08d9173bc095c', 'Wrapped BNB', 'WBNB', 18, 'wbnb', false, true),
('bsc', '0x8AC76a51cc950d9822D68b83fE1Ad97B32Cd580d', 'USD Coin', 'USDC', 18, 'usd-coin', true, false),
('bsc', '0x55d398326f99059fF775485246999027B3197955', 'Tether USD', 'USDT', 18, 'tether', true, false),
('bsc', '0xe9e7CEA3DedcA5984780Bafc599bD69ADd087D56', 'BUSD Token', 'BUSD', 18, 'binance-usd', true, false)
ON CONFLICT (network, contract_address) DO NOTHING;

-- Avalanche
INSERT INTO well_known_tokens (network, contract_address, name, symbol, decimals, coingecko_id, is_stablecoin, is_wrapped_native) VALUES
('avalanche', '0xB31f66AA3C1e785363F0875A1B74E27b85FD66c7', 'Wrapped AVAX', 'WAVAX', 18, 'wrapped-avax', false, true),
('avalanche', '0xB97EF9Ef8734C71904D8002F8b6Bc66Dd9c48a6E', 'USD Coin', 'USDC', 6, 'usd-coin', true, false),
('avalanche', '0x9702230A8Ea53601f5cD2dc00fDBc13d4dF4A8c7', 'TetherToken', 'USDT', 6, 'tether', true, false)
ON CONFLICT (network, contract_address) DO NOTHING;

-- ============================================================================
-- COMMENTS
-- ============================================================================
COMMENT ON TABLE tokens IS 'ERC-20 token contracts and metadata';
COMMENT ON TABLE token_transfers IS 'Token transfer events (Transfer event)';
COMMENT ON TABLE token_balances IS 'Current token balances per holder';
COMMENT ON TABLE token_approvals IS 'Token approval events (Approval event)';
COMMENT ON TABLE token_allowances IS 'Current token allowances (owner -> spender)';
COMMENT ON TABLE token_prices IS 'Historical token price data';
COMMENT ON TABLE well_known_tokens IS 'Pre-defined list of popular tokens';
