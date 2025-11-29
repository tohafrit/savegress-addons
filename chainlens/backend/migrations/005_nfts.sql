-- ChainLens NFT Tracking Schema
-- Version: 1.0.0
-- Description: Tables for ERC-721 and ERC-1155 NFTs

-- ============================================================================
-- NFT COLLECTIONS
-- ============================================================================
CREATE TABLE IF NOT EXISTS nft_collections (
    id BIGSERIAL PRIMARY KEY,
    network VARCHAR(50) NOT NULL,
    contract_address VARCHAR(42) NOT NULL,

    -- Collection metadata
    name VARCHAR(255),
    symbol VARCHAR(50),
    standard VARCHAR(20) NOT NULL, -- ERC721, ERC1155
    description TEXT,

    -- Stats
    total_supply BIGINT,
    owner_count BIGINT DEFAULT 0,
    transfer_count BIGINT DEFAULT 0,
    floor_price NUMERIC(78, 0), -- in wei
    volume_total NUMERIC(78, 0) DEFAULT 0,

    -- URIs
    base_uri TEXT,
    contract_uri TEXT,

    -- External links
    website TEXT,
    twitter TEXT,
    discord TEXT,
    opensea_slug VARCHAR(255),

    -- Images
    image_url TEXT,
    banner_url TEXT,

    -- Royalties (EIP-2981)
    royalty_recipient VARCHAR(42),
    royalty_bps INT, -- basis points (100 = 1%)

    -- Flags
    is_verified BOOLEAN DEFAULT FALSE,
    is_spam BOOLEAN DEFAULT FALSE,
    supports_eip2981 BOOLEAN DEFAULT FALSE,

    -- Deployment info
    deployer_address VARCHAR(42),
    deploy_block BIGINT,
    deploy_tx_hash VARCHAR(66),

    -- Metadata
    metadata JSONB,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(network, contract_address)
);

CREATE INDEX IF NOT EXISTS idx_nft_collections_network ON nft_collections(network);
CREATE INDEX IF NOT EXISTS idx_nft_collections_name ON nft_collections(name);
CREATE INDEX IF NOT EXISTS idx_nft_collections_standard ON nft_collections(standard);
CREATE INDEX IF NOT EXISTS idx_nft_collections_owner_count ON nft_collections(network, owner_count DESC);
CREATE INDEX IF NOT EXISTS idx_nft_collections_volume ON nft_collections(network, volume_total DESC);

-- ============================================================================
-- NFT ITEMS (Individual tokens)
-- ============================================================================
CREATE TABLE IF NOT EXISTS nft_items (
    id BIGSERIAL PRIMARY KEY,
    network VARCHAR(50) NOT NULL,
    contract_address VARCHAR(42) NOT NULL,
    token_id VARCHAR(78) NOT NULL, -- stored as string for large uint256

    -- Ownership (for ERC-721, single owner; for ERC-1155, see nft_balances)
    owner_address VARCHAR(42),

    -- Metadata
    token_uri TEXT,
    metadata JSONB,
    metadata_fetched_at TIMESTAMP WITH TIME ZONE,
    metadata_error TEXT,

    -- Parsed metadata fields (for search/filter)
    name VARCHAR(500),
    description TEXT,
    image_url TEXT,
    animation_url TEXT,
    external_url TEXT,
    background_color VARCHAR(10),

    -- Attributes (denormalized for query performance)
    attributes JSONB, -- [{"trait_type": "Color", "value": "Blue"}, ...]

    -- Rarity (if computed)
    rarity_score FLOAT,
    rarity_rank INT,

    -- Stats
    transfer_count INT DEFAULT 0,
    last_sale_price NUMERIC(78, 0),
    last_sale_currency VARCHAR(42),
    last_sale_at TIMESTAMP WITH TIME ZONE,

    -- For ERC-1155
    total_supply NUMERIC(78, 0), -- how many of this token exist

    -- Timestamps
    minted_at TIMESTAMP WITH TIME ZONE,
    last_transfer_at TIMESTAMP WITH TIME ZONE,
    burned_at TIMESTAMP WITH TIME ZONE,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(network, contract_address, token_id)
);

CREATE INDEX IF NOT EXISTS idx_nft_items_owner ON nft_items(owner_address) WHERE owner_address IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_nft_items_collection ON nft_items(contract_address);
CREATE INDEX IF NOT EXISTS idx_nft_items_name ON nft_items(name);
CREATE INDEX IF NOT EXISTS idx_nft_items_rarity ON nft_items(contract_address, rarity_rank) WHERE rarity_rank IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_nft_items_minted ON nft_items(contract_address, minted_at DESC);

-- GIN index for attribute queries
CREATE INDEX IF NOT EXISTS idx_nft_items_attributes ON nft_items USING GIN(attributes);

-- ============================================================================
-- NFT BALANCES (for ERC-1155 multi-token ownership)
-- ============================================================================
CREATE TABLE IF NOT EXISTS nft_balances (
    id BIGSERIAL PRIMARY KEY,
    network VARCHAR(50) NOT NULL,
    contract_address VARCHAR(42) NOT NULL,
    token_id VARCHAR(78) NOT NULL,
    holder_address VARCHAR(42) NOT NULL,

    balance NUMERIC(78, 0) NOT NULL DEFAULT 0,

    first_acquired_at TIMESTAMP WITH TIME ZONE,
    last_updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(network, contract_address, token_id, holder_address)
);

CREATE INDEX IF NOT EXISTS idx_nft_balances_holder ON nft_balances(holder_address);
CREATE INDEX IF NOT EXISTS idx_nft_balances_token ON nft_balances(contract_address, token_id);
CREATE INDEX IF NOT EXISTS idx_nft_balances_nonzero ON nft_balances(contract_address, token_id, balance DESC)
    WHERE balance > 0;

-- ============================================================================
-- NFT TRANSFERS
-- ============================================================================
CREATE TABLE IF NOT EXISTS nft_transfers (
    id BIGSERIAL PRIMARY KEY,
    network VARCHAR(50) NOT NULL,
    tx_hash VARCHAR(66) NOT NULL,
    log_index INT NOT NULL,
    block_number BIGINT NOT NULL,

    contract_address VARCHAR(42) NOT NULL,
    token_id VARCHAR(78) NOT NULL,
    from_address VARCHAR(42) NOT NULL,
    to_address VARCHAR(42) NOT NULL,

    -- For ERC-1155
    amount NUMERIC(78, 0) DEFAULT 1,
    operator VARCHAR(42), -- for ERC-1155 transferFrom

    -- Transfer type
    transfer_type VARCHAR(20) DEFAULT 'transfer', -- transfer, mint, burn, sale

    -- Sale info (if applicable)
    sale_price NUMERIC(78, 0),
    sale_currency VARCHAR(42), -- token address or zero for native
    marketplace VARCHAR(50), -- opensea, blur, etc

    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(network, tx_hash, log_index)
);

CREATE INDEX IF NOT EXISTS idx_nft_transfers_collection ON nft_transfers(contract_address, block_number DESC);
CREATE INDEX IF NOT EXISTS idx_nft_transfers_token ON nft_transfers(contract_address, token_id, block_number DESC);
CREATE INDEX IF NOT EXISTS idx_nft_transfers_from ON nft_transfers(from_address, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_nft_transfers_to ON nft_transfers(to_address, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_nft_transfers_block ON nft_transfers(network, block_number DESC);
CREATE INDEX IF NOT EXISTS idx_nft_transfers_type ON nft_transfers(transfer_type);

-- ============================================================================
-- NFT APPROVALS
-- ============================================================================
CREATE TABLE IF NOT EXISTS nft_approvals (
    id BIGSERIAL PRIMARY KEY,
    network VARCHAR(50) NOT NULL,
    tx_hash VARCHAR(66) NOT NULL,
    log_index INT NOT NULL,
    block_number BIGINT NOT NULL,

    contract_address VARCHAR(42) NOT NULL,
    token_id VARCHAR(78), -- NULL for ApprovalForAll
    owner_address VARCHAR(42) NOT NULL,
    approved_address VARCHAR(42) NOT NULL, -- or operator for ApprovalForAll
    approved BOOLEAN DEFAULT TRUE, -- for ApprovalForAll

    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(network, tx_hash, log_index)
);

CREATE INDEX IF NOT EXISTS idx_nft_approvals_owner ON nft_approvals(owner_address);
CREATE INDEX IF NOT EXISTS idx_nft_approvals_approved ON nft_approvals(approved_address);

-- ============================================================================
-- NFT METADATA CACHE (for external URIs)
-- ============================================================================
CREATE TABLE IF NOT EXISTS nft_metadata_cache (
    id BIGSERIAL PRIMARY KEY,
    uri_hash VARCHAR(64) NOT NULL, -- SHA256 of URI
    uri TEXT NOT NULL,

    content JSONB,
    content_type VARCHAR(100),
    fetched_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    error TEXT,
    retry_count INT DEFAULT 0,

    UNIQUE(uri_hash)
);

CREATE INDEX IF NOT EXISTS idx_metadata_cache_uri ON nft_metadata_cache(uri_hash);

-- ============================================================================
-- TRIGGERS
-- ============================================================================

CREATE TRIGGER update_nft_collections_updated_at
    BEFORE UPDATE ON nft_collections
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_nft_items_updated_at
    BEFORE UPDATE ON nft_items
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- FUNCTIONS
-- ============================================================================

-- Update collection stats
CREATE OR REPLACE FUNCTION update_nft_collection_stats(p_network VARCHAR, p_contract_address VARCHAR)
RETURNS VOID AS $$
BEGIN
    UPDATE nft_collections SET
        owner_count = (
            SELECT COUNT(DISTINCT owner_address)
            FROM nft_items
            WHERE network = p_network
            AND contract_address = p_contract_address
            AND owner_address IS NOT NULL
            AND owner_address != '0x0000000000000000000000000000000000000000'
        ),
        transfer_count = (
            SELECT COUNT(*)
            FROM nft_transfers
            WHERE network = p_network
            AND contract_address = p_contract_address
        ),
        total_supply = (
            SELECT COUNT(*)
            FROM nft_items
            WHERE network = p_network
            AND contract_address = p_contract_address
            AND burned_at IS NULL
        ),
        updated_at = NOW()
    WHERE network = p_network AND contract_address = p_contract_address;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- WELL-KNOWN COLLECTIONS (popular NFTs)
-- ============================================================================
CREATE TABLE IF NOT EXISTS well_known_nft_collections (
    id SERIAL PRIMARY KEY,
    network VARCHAR(50) NOT NULL,
    contract_address VARCHAR(42) NOT NULL,
    name VARCHAR(255) NOT NULL,
    symbol VARCHAR(50),
    standard VARCHAR(20) NOT NULL,
    image_url TEXT,
    opensea_slug VARCHAR(255),
    is_verified BOOLEAN DEFAULT TRUE,

    UNIQUE(network, contract_address)
);

-- Popular Ethereum NFTs
INSERT INTO well_known_nft_collections (network, contract_address, name, symbol, standard, opensea_slug) VALUES
('ethereum', '0xBC4CA0EdA7647A8aB7C2061c2E118A18a936f13D', 'Bored Ape Yacht Club', 'BAYC', 'ERC721', 'boredapeyachtclub'),
('ethereum', '0xb47e3cd837dDF8e4c57F05d70Ab865de6e193BBB', 'CryptoPunks', 'PUNK', 'ERC721', 'cryptopunks'),
('ethereum', '0x60E4d786628Fea6478F785A6d7e704777c86a7c6', 'Mutant Ape Yacht Club', 'MAYC', 'ERC721', 'mutant-ape-yacht-club'),
('ethereum', '0xED5AF388653567Af2F388E6224dC7C4b3241C544', 'Azuki', 'AZUKI', 'ERC721', 'azuki'),
('ethereum', '0x49cF6f5d44E70224e2E23fDcdd2C053F30aDA28B', 'CloneX', 'CloneX', 'ERC721', 'clonex'),
('ethereum', '0x8a90CAb2b38dba80c64b7734e58Ee1dB38B8992e', 'Doodles', 'DOODLE', 'ERC721', 'doodles-official'),
('ethereum', '0x23581767a106ae21c074b2276D25e5C3e136a68b', 'Moonbirds', 'MOONBIRD', 'ERC721', 'proof-moonbirds'),
('ethereum', '0x34d85c9CDeB23FA97cb08333b511ac86E1C4E258', 'Otherdeed', 'OTHR', 'ERC721', 'otherdeed'),
('ethereum', '0x1A92f7381B9F03921564a437210bB9396471050C', 'Cool Cats', 'COOL', 'ERC721', 'cool-cats-nft'),
('ethereum', '0xa3AEe8BcE55BEeA1951EF834b99f3Ac60d1ABeeB', 'VeeFriends', 'VFT', 'ERC721', 'veefriends')
ON CONFLICT (network, contract_address) DO NOTHING;

-- Popular Polygon NFTs
INSERT INTO well_known_nft_collections (network, contract_address, name, symbol, standard, opensea_slug) VALUES
('polygon', '0x2953399124F0cBB46d2CbACD8A89cF0599974963', 'OpenSea Shared Storefront', 'OPENSTORE', 'ERC1155', 'opensea-collections'),
('polygon', '0x22d5f9B75c524Fec1D6619787e582644CD4D7422', 'y00ts', 'Y00T', 'ERC721', 'y00ts')
ON CONFLICT (network, contract_address) DO NOTHING;

-- ============================================================================
-- COMMENTS
-- ============================================================================
COMMENT ON TABLE nft_collections IS 'NFT collection contracts (ERC-721 and ERC-1155)';
COMMENT ON TABLE nft_items IS 'Individual NFT tokens';
COMMENT ON TABLE nft_balances IS 'NFT balances for ERC-1155 multi-token ownership';
COMMENT ON TABLE nft_transfers IS 'NFT transfer events';
COMMENT ON TABLE nft_approvals IS 'NFT approval events';
COMMENT ON TABLE nft_metadata_cache IS 'Cache for fetched NFT metadata from external URIs';
COMMENT ON TABLE well_known_nft_collections IS 'Pre-defined list of popular NFT collections';
