-- ChainLens Contract Verification Schema
-- Version: 1.2.0
-- Description: Tables for verified contracts and source code

-- ============================================================================
-- VERIFIED CONTRACTS
-- ============================================================================
CREATE TABLE IF NOT EXISTS verified_contracts (
    id BIGSERIAL PRIMARY KEY,
    network VARCHAR(50) NOT NULL,
    address VARCHAR(42) NOT NULL,

    -- Contract info
    contract_name VARCHAR(255) NOT NULL,
    compiler_version VARCHAR(50) NOT NULL,
    optimization_enabled BOOLEAN DEFAULT FALSE,
    optimization_runs INT,
    evm_version VARCHAR(20),
    license VARCHAR(50),

    -- Source code
    source_code TEXT NOT NULL,
    abi JSONB NOT NULL,
    bytecode TEXT,
    deployed_bytecode TEXT,
    constructor_args TEXT,

    -- Metadata
    metadata JSONB,
    source_files JSONB, -- for multi-file contracts: {"Contract.sol": "...", "Library.sol": "..."}

    -- Verification info
    verification_source VARCHAR(20) NOT NULL, -- 'sourcify', 'manual', 'etherscan'
    verification_status VARCHAR(20) DEFAULT 'full', -- 'full', 'partial'
    verified_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    verified_by VARCHAR(255), -- user who verified

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(network, address)
);

CREATE INDEX IF NOT EXISTS idx_verified_name ON verified_contracts(contract_name);
CREATE INDEX IF NOT EXISTS idx_verified_network ON verified_contracts(network);
CREATE INDEX IF NOT EXISTS idx_verified_source ON verified_contracts(verification_source);
CREATE INDEX IF NOT EXISTS idx_verified_at ON verified_contracts(verified_at DESC);

-- ============================================================================
-- CONTRACT INTERFACES (for known ABIs)
-- ============================================================================
CREATE TABLE IF NOT EXISTS contract_interfaces (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    standard VARCHAR(20), -- ERC20, ERC721, ERC1155, etc
    abi JSONB NOT NULL,
    event_signatures JSONB, -- precomputed event signatures for matching
    function_signatures JSONB, -- precomputed function signatures
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- ============================================================================
-- CONTRACT ABI CACHE (for unverified contracts with known interfaces)
-- ============================================================================
CREATE TABLE IF NOT EXISTS contract_abi_cache (
    id BIGSERIAL PRIMARY KEY,
    network VARCHAR(50) NOT NULL,
    address VARCHAR(42) NOT NULL,
    detected_interfaces TEXT[], -- ['ERC20', 'Ownable']
    abi JSONB,
    last_checked_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(network, address)
);

CREATE INDEX IF NOT EXISTS idx_abi_cache_interfaces ON contract_abi_cache USING GIN(detected_interfaces);

-- ============================================================================
-- VERIFICATION REQUESTS (queue for pending verifications)
-- ============================================================================
CREATE TABLE IF NOT EXISTS verification_requests (
    id BIGSERIAL PRIMARY KEY,
    network VARCHAR(50) NOT NULL,
    address VARCHAR(42) NOT NULL,

    -- Request data
    source_code TEXT,
    compiler_version VARCHAR(50),
    optimization_enabled BOOLEAN,
    optimization_runs INT,
    constructor_args TEXT,

    -- Status
    status VARCHAR(20) DEFAULT 'pending', -- pending, processing, verified, failed
    error_message TEXT,
    attempts INT DEFAULT 0,

    -- User info
    requested_by UUID,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_verification_requests_status ON verification_requests(status);
CREATE INDEX IF NOT EXISTS idx_verification_requests_network ON verification_requests(network, address);

-- ============================================================================
-- FUNCTION/EVENT SIGNATURES DATABASE
-- ============================================================================
CREATE TABLE IF NOT EXISTS function_signatures (
    id SERIAL PRIMARY KEY,
    selector VARCHAR(10) NOT NULL UNIQUE, -- 4 bytes hex: 0xa9059cbb
    signature VARCHAR(500) NOT NULL, -- transfer(address,uint256)
    name VARCHAR(100) NOT NULL, -- transfer
    inputs JSONB, -- [{type: "address", name: "to"}, {type: "uint256", name: "amount"}]
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_func_sig_name ON function_signatures(name);

CREATE TABLE IF NOT EXISTS event_signatures (
    id SERIAL PRIMARY KEY,
    topic VARCHAR(66) NOT NULL UNIQUE, -- 32 bytes hex with 0x
    signature VARCHAR(500) NOT NULL, -- Transfer(address,address,uint256)
    name VARCHAR(100) NOT NULL, -- Transfer
    inputs JSONB, -- [{type: "address", name: "from", indexed: true}, ...]
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_event_sig_name ON event_signatures(name);

-- ============================================================================
-- TRIGGERS
-- ============================================================================

CREATE TRIGGER update_verified_contracts_updated_at
    BEFORE UPDATE ON verified_contracts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_verification_requests_updated_at
    BEFORE UPDATE ON verification_requests
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- INITIAL DATA: Common interfaces
-- ============================================================================

-- ERC-20
INSERT INTO contract_interfaces (name, standard, abi, event_signatures, function_signatures) VALUES
('ERC20', 'ERC20', '[
    {"type":"function","name":"name","inputs":[],"outputs":[{"type":"string"}],"stateMutability":"view"},
    {"type":"function","name":"symbol","inputs":[],"outputs":[{"type":"string"}],"stateMutability":"view"},
    {"type":"function","name":"decimals","inputs":[],"outputs":[{"type":"uint8"}],"stateMutability":"view"},
    {"type":"function","name":"totalSupply","inputs":[],"outputs":[{"type":"uint256"}],"stateMutability":"view"},
    {"type":"function","name":"balanceOf","inputs":[{"type":"address","name":"account"}],"outputs":[{"type":"uint256"}],"stateMutability":"view"},
    {"type":"function","name":"transfer","inputs":[{"type":"address","name":"to"},{"type":"uint256","name":"amount"}],"outputs":[{"type":"bool"}],"stateMutability":"nonpayable"},
    {"type":"function","name":"allowance","inputs":[{"type":"address","name":"owner"},{"type":"address","name":"spender"}],"outputs":[{"type":"uint256"}],"stateMutability":"view"},
    {"type":"function","name":"approve","inputs":[{"type":"address","name":"spender"},{"type":"uint256","name":"amount"}],"outputs":[{"type":"bool"}],"stateMutability":"nonpayable"},
    {"type":"function","name":"transferFrom","inputs":[{"type":"address","name":"from"},{"type":"address","name":"to"},{"type":"uint256","name":"amount"}],"outputs":[{"type":"bool"}],"stateMutability":"nonpayable"},
    {"type":"event","name":"Transfer","inputs":[{"type":"address","name":"from","indexed":true},{"type":"address","name":"to","indexed":true},{"type":"uint256","name":"value","indexed":false}]},
    {"type":"event","name":"Approval","inputs":[{"type":"address","name":"owner","indexed":true},{"type":"address","name":"spender","indexed":true},{"type":"uint256","name":"value","indexed":false}]}
]'::jsonb,
'{"Transfer":"0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef","Approval":"0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925"}'::jsonb,
'{"transfer":"0xa9059cbb","approve":"0x095ea7b3","transferFrom":"0x23b872dd","balanceOf":"0x70a08231","allowance":"0xdd62ed3e","totalSupply":"0x18160ddd"}'::jsonb)
ON CONFLICT (name) DO NOTHING;

-- ERC-721
INSERT INTO contract_interfaces (name, standard, abi, event_signatures, function_signatures) VALUES
('ERC721', 'ERC721', '[
    {"type":"function","name":"balanceOf","inputs":[{"type":"address","name":"owner"}],"outputs":[{"type":"uint256"}],"stateMutability":"view"},
    {"type":"function","name":"ownerOf","inputs":[{"type":"uint256","name":"tokenId"}],"outputs":[{"type":"address"}],"stateMutability":"view"},
    {"type":"function","name":"safeTransferFrom","inputs":[{"type":"address","name":"from"},{"type":"address","name":"to"},{"type":"uint256","name":"tokenId"}],"outputs":[],"stateMutability":"nonpayable"},
    {"type":"function","name":"transferFrom","inputs":[{"type":"address","name":"from"},{"type":"address","name":"to"},{"type":"uint256","name":"tokenId"}],"outputs":[],"stateMutability":"nonpayable"},
    {"type":"function","name":"approve","inputs":[{"type":"address","name":"to"},{"type":"uint256","name":"tokenId"}],"outputs":[],"stateMutability":"nonpayable"},
    {"type":"function","name":"setApprovalForAll","inputs":[{"type":"address","name":"operator"},{"type":"bool","name":"approved"}],"outputs":[],"stateMutability":"nonpayable"},
    {"type":"function","name":"getApproved","inputs":[{"type":"uint256","name":"tokenId"}],"outputs":[{"type":"address"}],"stateMutability":"view"},
    {"type":"function","name":"isApprovedForAll","inputs":[{"type":"address","name":"owner"},{"type":"address","name":"operator"}],"outputs":[{"type":"bool"}],"stateMutability":"view"},
    {"type":"event","name":"Transfer","inputs":[{"type":"address","name":"from","indexed":true},{"type":"address","name":"to","indexed":true},{"type":"uint256","name":"tokenId","indexed":true}]},
    {"type":"event","name":"Approval","inputs":[{"type":"address","name":"owner","indexed":true},{"type":"address","name":"approved","indexed":true},{"type":"uint256","name":"tokenId","indexed":true}]},
    {"type":"event","name":"ApprovalForAll","inputs":[{"type":"address","name":"owner","indexed":true},{"type":"address","name":"operator","indexed":true},{"type":"bool","name":"approved","indexed":false}]}
]'::jsonb,
'{"Transfer":"0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef","Approval":"0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925","ApprovalForAll":"0x17307eab39ab6107e8899845ad3d59bd9653f200f220920489ca2b5937696c31"}'::jsonb,
'{"balanceOf":"0x70a08231","ownerOf":"0x6352211e","transferFrom":"0x23b872dd","approve":"0x095ea7b3","setApprovalForAll":"0xa22cb465","getApproved":"0x081812fc","isApprovedForAll":"0xe985e9c5"}'::jsonb)
ON CONFLICT (name) DO NOTHING;

-- ERC-1155
INSERT INTO contract_interfaces (name, standard, abi, event_signatures, function_signatures) VALUES
('ERC1155', 'ERC1155', '[
    {"type":"function","name":"balanceOf","inputs":[{"type":"address","name":"account"},{"type":"uint256","name":"id"}],"outputs":[{"type":"uint256"}],"stateMutability":"view"},
    {"type":"function","name":"balanceOfBatch","inputs":[{"type":"address[]","name":"accounts"},{"type":"uint256[]","name":"ids"}],"outputs":[{"type":"uint256[]"}],"stateMutability":"view"},
    {"type":"function","name":"setApprovalForAll","inputs":[{"type":"address","name":"operator"},{"type":"bool","name":"approved"}],"outputs":[],"stateMutability":"nonpayable"},
    {"type":"function","name":"isApprovedForAll","inputs":[{"type":"address","name":"account"},{"type":"address","name":"operator"}],"outputs":[{"type":"bool"}],"stateMutability":"view"},
    {"type":"function","name":"safeTransferFrom","inputs":[{"type":"address","name":"from"},{"type":"address","name":"to"},{"type":"uint256","name":"id"},{"type":"uint256","name":"amount"},{"type":"bytes","name":"data"}],"outputs":[],"stateMutability":"nonpayable"},
    {"type":"function","name":"safeBatchTransferFrom","inputs":[{"type":"address","name":"from"},{"type":"address","name":"to"},{"type":"uint256[]","name":"ids"},{"type":"uint256[]","name":"amounts"},{"type":"bytes","name":"data"}],"outputs":[],"stateMutability":"nonpayable"},
    {"type":"event","name":"TransferSingle","inputs":[{"type":"address","name":"operator","indexed":true},{"type":"address","name":"from","indexed":true},{"type":"address","name":"to","indexed":true},{"type":"uint256","name":"id","indexed":false},{"type":"uint256","name":"value","indexed":false}]},
    {"type":"event","name":"TransferBatch","inputs":[{"type":"address","name":"operator","indexed":true},{"type":"address","name":"from","indexed":true},{"type":"address","name":"to","indexed":true},{"type":"uint256[]","name":"ids","indexed":false},{"type":"uint256[]","name":"values","indexed":false}]}
]'::jsonb,
'{"TransferSingle":"0xc3d58168c5ae7397731d063d5bbf3d657854427343f4c083240f7aacaa2d0f62","TransferBatch":"0x4a39dc06d4c0dbc64b70af90fd698a233a518aa5d07e595d983b8c0526c8f7fb"}'::jsonb,
'{"balanceOf":"0x00fdd58e","balanceOfBatch":"0x4e1273f4","setApprovalForAll":"0xa22cb465","isApprovedForAll":"0xe985e9c5","safeTransferFrom":"0xf242432a","safeBatchTransferFrom":"0x2eb2c2d6"}'::jsonb)
ON CONFLICT (name) DO NOTHING;

-- ============================================================================
-- COMMON FUNCTION SIGNATURES
-- ============================================================================
INSERT INTO function_signatures (selector, signature, name, inputs) VALUES
('0xa9059cbb', 'transfer(address,uint256)', 'transfer', '[{"type":"address","name":"to"},{"type":"uint256","name":"amount"}]'::jsonb),
('0x095ea7b3', 'approve(address,uint256)', 'approve', '[{"type":"address","name":"spender"},{"type":"uint256","name":"amount"}]'::jsonb),
('0x23b872dd', 'transferFrom(address,address,uint256)', 'transferFrom', '[{"type":"address","name":"from"},{"type":"address","name":"to"},{"type":"uint256","name":"amount"}]'::jsonb),
('0x70a08231', 'balanceOf(address)', 'balanceOf', '[{"type":"address","name":"account"}]'::jsonb),
('0xdd62ed3e', 'allowance(address,address)', 'allowance', '[{"type":"address","name":"owner"},{"type":"address","name":"spender"}]'::jsonb),
('0x18160ddd', 'totalSupply()', 'totalSupply', '[]'::jsonb),
('0x06fdde03', 'name()', 'name', '[]'::jsonb),
('0x95d89b41', 'symbol()', 'symbol', '[]'::jsonb),
('0x313ce567', 'decimals()', 'decimals', '[]'::jsonb),
('0x8da5cb5b', 'owner()', 'owner', '[]'::jsonb),
('0x715018a6', 'renounceOwnership()', 'renounceOwnership', '[]'::jsonb),
('0xf2fde38b', 'transferOwnership(address)', 'transferOwnership', '[{"type":"address","name":"newOwner"}]'::jsonb)
ON CONFLICT (selector) DO NOTHING;

-- ============================================================================
-- COMMON EVENT SIGNATURES
-- ============================================================================
INSERT INTO event_signatures (topic, signature, name, inputs) VALUES
('0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef', 'Transfer(address,address,uint256)', 'Transfer', '[{"type":"address","name":"from","indexed":true},{"type":"address","name":"to","indexed":true},{"type":"uint256","name":"value","indexed":false}]'::jsonb),
('0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925', 'Approval(address,address,uint256)', 'Approval', '[{"type":"address","name":"owner","indexed":true},{"type":"address","name":"spender","indexed":true},{"type":"uint256","name":"value","indexed":false}]'::jsonb),
('0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0', 'OwnershipTransferred(address,address)', 'OwnershipTransferred', '[{"type":"address","name":"previousOwner","indexed":true},{"type":"address","name":"newOwner","indexed":true}]'::jsonb),
('0xc3d58168c5ae7397731d063d5bbf3d657854427343f4c083240f7aacaa2d0f62', 'TransferSingle(address,address,address,uint256,uint256)', 'TransferSingle', '[{"type":"address","name":"operator","indexed":true},{"type":"address","name":"from","indexed":true},{"type":"address","name":"to","indexed":true},{"type":"uint256","name":"id"},{"type":"uint256","name":"value"}]'::jsonb)
ON CONFLICT (topic) DO NOTHING;

-- ============================================================================
-- COMMENTS
-- ============================================================================
COMMENT ON TABLE verified_contracts IS 'Verified smart contract source code and ABI';
COMMENT ON TABLE contract_interfaces IS 'Known contract interfaces (ERC standards)';
COMMENT ON TABLE contract_abi_cache IS 'Cached ABIs for unverified contracts based on detected interfaces';
COMMENT ON TABLE verification_requests IS 'Queue for contract verification requests';
COMMENT ON TABLE function_signatures IS 'Database of known function signatures (4-byte selectors)';
COMMENT ON TABLE event_signatures IS 'Database of known event signatures (topic0)';
