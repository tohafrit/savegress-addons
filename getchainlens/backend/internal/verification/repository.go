package verification

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository provides database operations for contract verification
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new verification repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// ============================================================================
// VERIFIED CONTRACTS
// ============================================================================

// SaveVerifiedContract saves or updates a verified contract
func (r *Repository) SaveVerifiedContract(ctx context.Context, contract *VerifiedContract) error {
	query := `
		INSERT INTO verified_contracts (
			network, address, contract_name, compiler_version,
			optimization_enabled, optimization_runs, evm_version, license,
			source_code, abi, bytecode, deployed_bytecode, constructor_args,
			metadata, source_files, verification_source, verification_status,
			verified_at, verified_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
		ON CONFLICT (network, address) DO UPDATE SET
			contract_name = EXCLUDED.contract_name,
			compiler_version = EXCLUDED.compiler_version,
			optimization_enabled = EXCLUDED.optimization_enabled,
			optimization_runs = EXCLUDED.optimization_runs,
			evm_version = EXCLUDED.evm_version,
			license = EXCLUDED.license,
			source_code = EXCLUDED.source_code,
			abi = EXCLUDED.abi,
			bytecode = EXCLUDED.bytecode,
			deployed_bytecode = EXCLUDED.deployed_bytecode,
			constructor_args = EXCLUDED.constructor_args,
			metadata = EXCLUDED.metadata,
			source_files = EXCLUDED.source_files,
			verification_source = EXCLUDED.verification_source,
			verification_status = EXCLUDED.verification_status,
			verified_at = EXCLUDED.verified_at,
			verified_by = EXCLUDED.verified_by,
			updated_at = NOW()
		RETURNING id`

	return r.db.QueryRow(ctx, query,
		contract.Network, contract.Address, contract.ContractName, contract.CompilerVersion,
		contract.OptimizationEnabled, contract.OptimizationRuns, contract.EVMVersion, contract.License,
		contract.SourceCode, contract.ABI, contract.Bytecode, contract.DeployedBytecode, contract.ConstructorArgs,
		contract.Metadata, contract.SourceFiles, contract.VerificationSource, contract.VerificationStatus,
		contract.VerifiedAt, contract.VerifiedBy,
	).Scan(&contract.ID)
}

// GetVerifiedContract retrieves a verified contract by network and address
func (r *Repository) GetVerifiedContract(ctx context.Context, network, address string) (*VerifiedContract, error) {
	query := `
		SELECT id, network, address, contract_name, compiler_version,
			optimization_enabled, optimization_runs, evm_version, license,
			source_code, abi, bytecode, deployed_bytecode, constructor_args,
			metadata, source_files, verification_source, verification_status,
			verified_at, verified_by, created_at, updated_at
		FROM verified_contracts
		WHERE network = $1 AND LOWER(address) = LOWER($2)`

	var contract VerifiedContract
	err := r.db.QueryRow(ctx, query, network, address).Scan(
		&contract.ID, &contract.Network, &contract.Address, &contract.ContractName, &contract.CompilerVersion,
		&contract.OptimizationEnabled, &contract.OptimizationRuns, &contract.EVMVersion, &contract.License,
		&contract.SourceCode, &contract.ABI, &contract.Bytecode, &contract.DeployedBytecode, &contract.ConstructorArgs,
		&contract.Metadata, &contract.SourceFiles, &contract.VerificationSource, &contract.VerificationStatus,
		&contract.VerifiedAt, &contract.VerifiedBy, &contract.CreatedAt, &contract.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get verified contract: %w", err)
	}

	return &contract, nil
}

// ListVerifiedContracts lists verified contracts with pagination
func (r *Repository) ListVerifiedContracts(ctx context.Context, network string, page, pageSize int) ([]*VerifiedContract, int64, error) {
	offset := (page - 1) * pageSize

	// Count total
	var total int64
	countQuery := `SELECT COUNT(*) FROM verified_contracts WHERE network = $1`
	if err := r.db.QueryRow(ctx, countQuery, network).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count contracts: %w", err)
	}

	// Fetch page
	query := `
		SELECT id, network, address, contract_name, compiler_version,
			optimization_enabled, optimization_runs, evm_version, license,
			source_code, abi, bytecode, deployed_bytecode, constructor_args,
			metadata, source_files, verification_source, verification_status,
			verified_at, verified_by, created_at, updated_at
		FROM verified_contracts
		WHERE network = $1
		ORDER BY verified_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(ctx, query, network, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list contracts: %w", err)
	}
	defer rows.Close()

	var contracts []*VerifiedContract
	for rows.Next() {
		var contract VerifiedContract
		if err := rows.Scan(
			&contract.ID, &contract.Network, &contract.Address, &contract.ContractName, &contract.CompilerVersion,
			&contract.OptimizationEnabled, &contract.OptimizationRuns, &contract.EVMVersion, &contract.License,
			&contract.SourceCode, &contract.ABI, &contract.Bytecode, &contract.DeployedBytecode, &contract.ConstructorArgs,
			&contract.Metadata, &contract.SourceFiles, &contract.VerificationSource, &contract.VerificationStatus,
			&contract.VerifiedAt, &contract.VerifiedBy, &contract.CreatedAt, &contract.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan contract: %w", err)
		}
		contracts = append(contracts, &contract)
	}

	return contracts, total, nil
}

// SearchVerifiedContracts searches contracts by name or address
func (r *Repository) SearchVerifiedContracts(ctx context.Context, network, query string, limit int) ([]*VerifiedContract, error) {
	sqlQuery := `
		SELECT id, network, address, contract_name, compiler_version,
			optimization_enabled, optimization_runs, evm_version, license,
			source_code, abi, bytecode, deployed_bytecode, constructor_args,
			metadata, source_files, verification_source, verification_status,
			verified_at, verified_by, created_at, updated_at
		FROM verified_contracts
		WHERE network = $1 AND (
			LOWER(address) LIKE LOWER($2) OR
			LOWER(contract_name) LIKE LOWER($2)
		)
		ORDER BY verified_at DESC
		LIMIT $3`

	searchPattern := "%" + query + "%"
	rows, err := r.db.Query(ctx, sqlQuery, network, searchPattern, limit)
	if err != nil {
		return nil, fmt.Errorf("search contracts: %w", err)
	}
	defer rows.Close()

	var contracts []*VerifiedContract
	for rows.Next() {
		var contract VerifiedContract
		if err := rows.Scan(
			&contract.ID, &contract.Network, &contract.Address, &contract.ContractName, &contract.CompilerVersion,
			&contract.OptimizationEnabled, &contract.OptimizationRuns, &contract.EVMVersion, &contract.License,
			&contract.SourceCode, &contract.ABI, &contract.Bytecode, &contract.DeployedBytecode, &contract.ConstructorArgs,
			&contract.Metadata, &contract.SourceFiles, &contract.VerificationSource, &contract.VerificationStatus,
			&contract.VerifiedAt, &contract.VerifiedBy, &contract.CreatedAt, &contract.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan contract: %w", err)
		}
		contracts = append(contracts, &contract)
	}

	return contracts, nil
}

// ============================================================================
// CONTRACT INTERFACES
// ============================================================================

// GetContractInterface retrieves a contract interface by name
func (r *Repository) GetContractInterface(ctx context.Context, name string) (*ContractInterface, error) {
	query := `
		SELECT id, name, standard, abi, event_signatures, function_signatures, created_at
		FROM contract_interfaces
		WHERE name = $1`

	var ci ContractInterface
	err := r.db.QueryRow(ctx, query, name).Scan(
		&ci.ID, &ci.Name, &ci.Standard, &ci.ABI, &ci.EventSignatures, &ci.FunctionSignatures, &ci.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get contract interface: %w", err)
	}

	return &ci, nil
}

// ListContractInterfaces lists all known contract interfaces
func (r *Repository) ListContractInterfaces(ctx context.Context) ([]*ContractInterface, error) {
	query := `
		SELECT id, name, standard, abi, event_signatures, function_signatures, created_at
		FROM contract_interfaces
		ORDER BY name`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list interfaces: %w", err)
	}
	defer rows.Close()

	var interfaces []*ContractInterface
	for rows.Next() {
		var ci ContractInterface
		if err := rows.Scan(
			&ci.ID, &ci.Name, &ci.Standard, &ci.ABI, &ci.EventSignatures, &ci.FunctionSignatures, &ci.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan interface: %w", err)
		}
		interfaces = append(interfaces, &ci)
	}

	return interfaces, nil
}

// ============================================================================
// FUNCTION SIGNATURES
// ============================================================================

// GetFunctionSignature retrieves a function signature by selector
func (r *Repository) GetFunctionSignature(ctx context.Context, selector string) (*FunctionSignature, error) {
	query := `
		SELECT id, selector, signature, name, inputs, created_at
		FROM function_signatures
		WHERE selector = $1`

	var fs FunctionSignature
	err := r.db.QueryRow(ctx, query, selector).Scan(
		&fs.ID, &fs.Selector, &fs.Signature, &fs.Name, &fs.Inputs, &fs.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get function signature: %w", err)
	}

	return &fs, nil
}

// SaveFunctionSignature saves a function signature
func (r *Repository) SaveFunctionSignature(ctx context.Context, fs *FunctionSignature) error {
	query := `
		INSERT INTO function_signatures (selector, signature, name, inputs)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (selector) DO NOTHING
		RETURNING id`

	err := r.db.QueryRow(ctx, query, fs.Selector, fs.Signature, fs.Name, fs.Inputs).Scan(&fs.ID)
	if err == pgx.ErrNoRows {
		// Already exists, that's fine
		return nil
	}
	return err
}

// ============================================================================
// EVENT SIGNATURES
// ============================================================================

// GetEventSignature retrieves an event signature by topic
func (r *Repository) GetEventSignature(ctx context.Context, topic string) (*EventSignature, error) {
	query := `
		SELECT id, topic, signature, name, inputs, created_at
		FROM event_signatures
		WHERE topic = $1`

	var es EventSignature
	err := r.db.QueryRow(ctx, query, topic).Scan(
		&es.ID, &es.Topic, &es.Signature, &es.Name, &es.Inputs, &es.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get event signature: %w", err)
	}

	return &es, nil
}

// SaveEventSignature saves an event signature
func (r *Repository) SaveEventSignature(ctx context.Context, es *EventSignature) error {
	query := `
		INSERT INTO event_signatures (topic, signature, name, inputs)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (topic) DO NOTHING
		RETURNING id`

	err := r.db.QueryRow(ctx, query, es.Topic, es.Signature, es.Name, es.Inputs).Scan(&es.ID)
	if err == pgx.ErrNoRows {
		return nil
	}
	return err
}

// ============================================================================
// VERIFICATION REQUESTS
// ============================================================================

// CreateVerificationRequest creates a new verification request
func (r *Repository) CreateVerificationRequest(ctx context.Context, req *VerificationRequest) error {
	query := `
		INSERT INTO verification_requests (
			network, address, source_code, compiler_version,
			optimization_enabled, optimization_runs, constructor_args,
			status, requested_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at`

	return r.db.QueryRow(ctx, query,
		req.Network, req.Address, req.SourceCode, req.CompilerVersion,
		req.OptimizationEnabled, req.OptimizationRuns, req.ConstructorArgs,
		RequestPending, req.RequestedBy,
	).Scan(&req.ID, &req.CreatedAt, &req.UpdatedAt)
}

// GetVerificationRequest retrieves a verification request by ID
func (r *Repository) GetVerificationRequest(ctx context.Context, id int64) (*VerificationRequest, error) {
	query := `
		SELECT id, network, address, source_code, compiler_version,
			optimization_enabled, optimization_runs, constructor_args,
			status, error_message, attempts, requested_by, created_at, updated_at
		FROM verification_requests
		WHERE id = $1`

	var req VerificationRequest
	err := r.db.QueryRow(ctx, query, id).Scan(
		&req.ID, &req.Network, &req.Address, &req.SourceCode, &req.CompilerVersion,
		&req.OptimizationEnabled, &req.OptimizationRuns, &req.ConstructorArgs,
		&req.Status, &req.ErrorMessage, &req.Attempts, &req.RequestedBy, &req.CreatedAt, &req.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get verification request: %w", err)
	}

	return &req, nil
}

// UpdateVerificationRequest updates a verification request status
func (r *Repository) UpdateVerificationRequest(ctx context.Context, id int64, status string, errorMessage *string) error {
	query := `
		UPDATE verification_requests
		SET status = $2, error_message = $3, attempts = attempts + 1, updated_at = NOW()
		WHERE id = $1`

	_, err := r.db.Exec(ctx, query, id, status, errorMessage)
	return err
}

// GetPendingVerificationRequests retrieves pending requests for processing
func (r *Repository) GetPendingVerificationRequests(ctx context.Context, limit int) ([]*VerificationRequest, error) {
	query := `
		SELECT id, network, address, source_code, compiler_version,
			optimization_enabled, optimization_runs, constructor_args,
			status, error_message, attempts, requested_by, created_at, updated_at
		FROM verification_requests
		WHERE status = $1 AND attempts < 3
		ORDER BY created_at ASC
		LIMIT $2`

	rows, err := r.db.Query(ctx, query, RequestPending, limit)
	if err != nil {
		return nil, fmt.Errorf("get pending requests: %w", err)
	}
	defer rows.Close()

	var requests []*VerificationRequest
	for rows.Next() {
		var req VerificationRequest
		if err := rows.Scan(
			&req.ID, &req.Network, &req.Address, &req.SourceCode, &req.CompilerVersion,
			&req.OptimizationEnabled, &req.OptimizationRuns, &req.ConstructorArgs,
			&req.Status, &req.ErrorMessage, &req.Attempts, &req.RequestedBy, &req.CreatedAt, &req.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan request: %w", err)
		}
		requests = append(requests, &req)
	}

	return requests, nil
}

// ============================================================================
// ABI CACHE
// ============================================================================

// ABICache represents cached ABI for an unverified contract
type ABICache struct {
	ID                 int64           `json:"-" db:"id"`
	Network            string          `json:"network" db:"network"`
	Address            string          `json:"address" db:"address"`
	DetectedInterfaces []string        `json:"detectedInterfaces" db:"detected_interfaces"`
	ABI                json.RawMessage `json:"abi" db:"abi"`
}

// GetABICache retrieves cached ABI for a contract
func (r *Repository) GetABICache(ctx context.Context, network, address string) (*ABICache, error) {
	query := `
		SELECT id, network, address, detected_interfaces, abi
		FROM contract_abi_cache
		WHERE network = $1 AND LOWER(address) = LOWER($2)`

	var cache ABICache
	err := r.db.QueryRow(ctx, query, network, address).Scan(
		&cache.ID, &cache.Network, &cache.Address, &cache.DetectedInterfaces, &cache.ABI,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get abi cache: %w", err)
	}

	return &cache, nil
}

// SaveABICache saves ABI cache for a contract
func (r *Repository) SaveABICache(ctx context.Context, cache *ABICache) error {
	query := `
		INSERT INTO contract_abi_cache (network, address, detected_interfaces, abi, last_checked_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (network, address) DO UPDATE SET
			detected_interfaces = EXCLUDED.detected_interfaces,
			abi = EXCLUDED.abi,
			last_checked_at = NOW()
		RETURNING id`

	return r.db.QueryRow(ctx, query,
		cache.Network, cache.Address, cache.DetectedInterfaces, cache.ABI,
	).Scan(&cache.ID)
}
