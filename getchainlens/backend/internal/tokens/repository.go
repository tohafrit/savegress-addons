package tokens

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository provides database operations for tokens
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new token repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// ============================================================================
// TOKENS
// ============================================================================

// UpsertToken creates or updates a token
func (r *Repository) UpsertToken(ctx context.Context, token *Token) error {
	query := `
		INSERT INTO tokens (
			network, contract_address, name, symbol, decimals, total_supply,
			holder_count, transfer_count, logo_url, website, description,
			social_links, is_verified, coingecko_id, coinmarketcap_id,
			token_type, is_proxy, implementation_address, first_block,
			first_tx_hash, deployer_address
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)
		ON CONFLICT (network, contract_address) DO UPDATE SET
			name = COALESCE(EXCLUDED.name, tokens.name),
			symbol = COALESCE(EXCLUDED.symbol, tokens.symbol),
			decimals = EXCLUDED.decimals,
			total_supply = COALESCE(EXCLUDED.total_supply, tokens.total_supply),
			holder_count = EXCLUDED.holder_count,
			transfer_count = EXCLUDED.transfer_count,
			logo_url = COALESCE(EXCLUDED.logo_url, tokens.logo_url),
			updated_at = NOW()
		RETURNING id`

	var totalSupply *string
	if token.TotalSupply != nil {
		totalSupply = token.TotalSupply
	}

	return r.db.QueryRow(ctx, query,
		token.Network, strings.ToLower(token.ContractAddress), token.Name, token.Symbol, token.Decimals, totalSupply,
		token.HolderCount, token.TransferCount, token.LogoURL, token.Website, token.Description,
		token.SocialLinks, token.IsVerified, token.CoingeckoID, token.CoinmarketcapID,
		token.TokenType, token.IsProxy, token.Implementation, token.FirstBlock,
		token.FirstTxHash, token.DeployerAddress,
	).Scan(&token.ID)
}

// GetToken retrieves a token by network and address
func (r *Repository) GetToken(ctx context.Context, network, address string) (*Token, error) {
	query := `
		SELECT id, network, contract_address, name, symbol, decimals, total_supply,
			holder_count, transfer_count, logo_url, website, description,
			social_links, is_verified, coingecko_id, coinmarketcap_id,
			token_type, is_proxy, implementation_address, first_block,
			first_tx_hash, deployer_address, created_at, updated_at
		FROM tokens
		WHERE network = $1 AND LOWER(contract_address) = LOWER($2)`

	var token Token
	err := r.db.QueryRow(ctx, query, network, address).Scan(
		&token.ID, &token.Network, &token.ContractAddress, &token.Name, &token.Symbol, &token.Decimals, &token.TotalSupply,
		&token.HolderCount, &token.TransferCount, &token.LogoURL, &token.Website, &token.Description,
		&token.SocialLinks, &token.IsVerified, &token.CoingeckoID, &token.CoinmarketcapID,
		&token.TokenType, &token.IsProxy, &token.Implementation, &token.FirstBlock,
		&token.FirstTxHash, &token.DeployerAddress, &token.CreatedAt, &token.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get token: %w", err)
	}

	return &token, nil
}

// ListTokens lists tokens with filtering and pagination
func (r *Repository) ListTokens(ctx context.Context, filter TokenFilter) ([]*Token, int64, error) {
	// Build query
	var conditions []string
	var args []interface{}
	argNum := 1

	conditions = append(conditions, fmt.Sprintf("network = $%d", argNum))
	args = append(args, filter.Network)
	argNum++

	if filter.TokenType != nil {
		conditions = append(conditions, fmt.Sprintf("token_type = $%d", argNum))
		args = append(args, *filter.TokenType)
		argNum++
	}

	if filter.Query != nil && *filter.Query != "" {
		conditions = append(conditions, fmt.Sprintf("(LOWER(name) LIKE LOWER($%d) OR LOWER(symbol) LIKE LOWER($%d))", argNum, argNum))
		args = append(args, "%"+*filter.Query+"%")
		argNum++
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	// Count total
	countQuery := "SELECT COUNT(*) FROM tokens " + whereClause
	var total int64
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count tokens: %w", err)
	}

	// Build sort
	sortBy := "holder_count"
	if filter.SortBy != "" {
		switch filter.SortBy {
		case "holder_count", "transfer_count", "name", "symbol":
			sortBy = filter.SortBy
		}
	}
	sortOrder := "DESC"
	if filter.SortOrder == "asc" {
		sortOrder = "ASC"
	}

	// Fetch page
	offset := (filter.Page - 1) * filter.PageSize
	query := fmt.Sprintf(`
		SELECT id, network, contract_address, name, symbol, decimals, total_supply,
			holder_count, transfer_count, logo_url, website, description,
			social_links, is_verified, coingecko_id, coinmarketcap_id,
			token_type, is_proxy, implementation_address, first_block,
			first_tx_hash, deployer_address, created_at, updated_at
		FROM tokens
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d`, whereClause, sortBy, sortOrder, argNum, argNum+1)

	args = append(args, filter.PageSize, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list tokens: %w", err)
	}
	defer rows.Close()

	var tokens []*Token
	for rows.Next() {
		var token Token
		if err := rows.Scan(
			&token.ID, &token.Network, &token.ContractAddress, &token.Name, &token.Symbol, &token.Decimals, &token.TotalSupply,
			&token.HolderCount, &token.TransferCount, &token.LogoURL, &token.Website, &token.Description,
			&token.SocialLinks, &token.IsVerified, &token.CoingeckoID, &token.CoinmarketcapID,
			&token.TokenType, &token.IsProxy, &token.Implementation, &token.FirstBlock,
			&token.FirstTxHash, &token.DeployerAddress, &token.CreatedAt, &token.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan token: %w", err)
		}
		tokens = append(tokens, &token)
	}

	return tokens, total, nil
}

// UpdateTokenStats updates token holder and transfer counts
func (r *Repository) UpdateTokenStats(ctx context.Context, network, address string) error {
	query := `
		UPDATE tokens SET
			holder_count = (SELECT COUNT(*) FROM token_balances WHERE network = $1 AND token_address = $2 AND balance > 0),
			transfer_count = (SELECT COUNT(*) FROM token_transfers WHERE network = $1 AND token_address = $2),
			updated_at = NOW()
		WHERE network = $1 AND LOWER(contract_address) = LOWER($2)`

	_, err := r.db.Exec(ctx, query, network, address)
	return err
}

// ============================================================================
// TOKEN TRANSFERS
// ============================================================================

// InsertTransfer inserts a token transfer
func (r *Repository) InsertTransfer(ctx context.Context, transfer *TokenTransfer) error {
	query := `
		INSERT INTO token_transfers (
			network, tx_hash, log_index, block_number, token_address,
			from_address, to_address, value, token_symbol, token_decimals,
			token_id, timestamp
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (network, tx_hash, log_index) DO NOTHING
		RETURNING id`

	err := r.db.QueryRow(ctx, query,
		transfer.Network, transfer.TxHash, transfer.LogIndex, transfer.BlockNumber, strings.ToLower(transfer.TokenAddress),
		strings.ToLower(transfer.FromAddress), strings.ToLower(transfer.ToAddress), transfer.Value, transfer.TokenSymbol, transfer.TokenDecimals,
		transfer.TokenID, transfer.Timestamp,
	).Scan(&transfer.ID)

	if err == pgx.ErrNoRows {
		return nil // Already exists
	}
	return err
}

// InsertTransfers batch inserts transfers
func (r *Repository) InsertTransfers(ctx context.Context, transfers []*TokenTransfer) error {
	if len(transfers) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	query := `
		INSERT INTO token_transfers (
			network, tx_hash, log_index, block_number, token_address,
			from_address, to_address, value, token_symbol, token_decimals,
			token_id, timestamp
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (network, tx_hash, log_index) DO NOTHING`

	for _, t := range transfers {
		batch.Queue(query,
			t.Network, t.TxHash, t.LogIndex, t.BlockNumber, strings.ToLower(t.TokenAddress),
			strings.ToLower(t.FromAddress), strings.ToLower(t.ToAddress), t.Value, t.TokenSymbol, t.TokenDecimals,
			t.TokenID, t.Timestamp,
		)
	}

	br := r.db.SendBatch(ctx, batch)
	defer br.Close()

	for range transfers {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("insert transfer: %w", err)
		}
	}

	return nil
}

// ListTransfers lists token transfers with filtering
func (r *Repository) ListTransfers(ctx context.Context, filter TransferFilter) ([]*TokenTransfer, int64, error) {
	var conditions []string
	var args []interface{}
	argNum := 1

	conditions = append(conditions, fmt.Sprintf("network = $%d", argNum))
	args = append(args, filter.Network)
	argNum++

	if filter.TokenAddress != nil {
		conditions = append(conditions, fmt.Sprintf("LOWER(token_address) = LOWER($%d)", argNum))
		args = append(args, *filter.TokenAddress)
		argNum++
	}

	if filter.FromAddress != nil {
		conditions = append(conditions, fmt.Sprintf("LOWER(from_address) = LOWER($%d)", argNum))
		args = append(args, *filter.FromAddress)
		argNum++
	}

	if filter.ToAddress != nil {
		conditions = append(conditions, fmt.Sprintf("LOWER(to_address) = LOWER($%d)", argNum))
		args = append(args, *filter.ToAddress)
		argNum++
	}

	if filter.BlockNumber != nil {
		conditions = append(conditions, fmt.Sprintf("block_number = $%d", argNum))
		args = append(args, *filter.BlockNumber)
		argNum++
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	// Count
	var total int64
	countQuery := "SELECT COUNT(*) FROM token_transfers " + whereClause
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count transfers: %w", err)
	}

	// Fetch
	offset := (filter.Page - 1) * filter.PageSize
	query := fmt.Sprintf(`
		SELECT id, network, tx_hash, log_index, block_number, token_address,
			from_address, to_address, value, token_symbol, token_decimals,
			token_id, timestamp, created_at
		FROM token_transfers
		%s
		ORDER BY block_number DESC, log_index DESC
		LIMIT $%d OFFSET $%d`, whereClause, argNum, argNum+1)

	args = append(args, filter.PageSize, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list transfers: %w", err)
	}
	defer rows.Close()

	var transfers []*TokenTransfer
	for rows.Next() {
		var t TokenTransfer
		if err := rows.Scan(
			&t.ID, &t.Network, &t.TxHash, &t.LogIndex, &t.BlockNumber, &t.TokenAddress,
			&t.FromAddress, &t.ToAddress, &t.Value, &t.TokenSymbol, &t.TokenDecimals,
			&t.TokenID, &t.Timestamp, &t.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan transfer: %w", err)
		}
		transfers = append(transfers, &t)
	}

	return transfers, total, nil
}

// GetTransfersByTxHash returns all transfers in a transaction
func (r *Repository) GetTransfersByTxHash(ctx context.Context, network, txHash string) ([]*TokenTransfer, error) {
	query := `
		SELECT id, network, tx_hash, log_index, block_number, token_address,
			from_address, to_address, value, token_symbol, token_decimals,
			token_id, timestamp, created_at
		FROM token_transfers
		WHERE network = $1 AND tx_hash = $2
		ORDER BY log_index`

	rows, err := r.db.Query(ctx, query, network, txHash)
	if err != nil {
		return nil, fmt.Errorf("get transfers: %w", err)
	}
	defer rows.Close()

	var transfers []*TokenTransfer
	for rows.Next() {
		var t TokenTransfer
		if err := rows.Scan(
			&t.ID, &t.Network, &t.TxHash, &t.LogIndex, &t.BlockNumber, &t.TokenAddress,
			&t.FromAddress, &t.ToAddress, &t.Value, &t.TokenSymbol, &t.TokenDecimals,
			&t.TokenID, &t.Timestamp, &t.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan transfer: %w", err)
		}
		transfers = append(transfers, &t)
	}

	return transfers, nil
}

// ============================================================================
// TOKEN BALANCES
// ============================================================================

// UpdateBalance updates a token balance after a transfer
func (r *Repository) UpdateBalance(ctx context.Context, network, tokenAddress, holderAddress string, delta *big.Int, timestamp time.Time) error {
	query := `
		INSERT INTO token_balances (network, token_address, holder_address, balance, first_transfer_at, last_transfer_at, transfer_count)
		VALUES ($1, $2, $3, $4, $5, $5, 1)
		ON CONFLICT (network, token_address, holder_address) DO UPDATE SET
			balance = token_balances.balance + $4,
			last_transfer_at = $5,
			transfer_count = token_balances.transfer_count + 1,
			updated_at = NOW()`

	_, err := r.db.Exec(ctx, query,
		network, strings.ToLower(tokenAddress), strings.ToLower(holderAddress),
		delta.String(), timestamp,
	)
	return err
}

// SetBalance sets the exact balance for a holder
func (r *Repository) SetBalance(ctx context.Context, balance *TokenBalance) error {
	query := `
		INSERT INTO token_balances (network, token_address, holder_address, balance, first_transfer_at, last_transfer_at, transfer_count)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (network, token_address, holder_address) DO UPDATE SET
			balance = $4,
			last_transfer_at = COALESCE($6, token_balances.last_transfer_at),
			updated_at = NOW()`

	_, err := r.db.Exec(ctx, query,
		balance.Network, strings.ToLower(balance.TokenAddress), strings.ToLower(balance.HolderAddress),
		balance.Balance, balance.FirstTransferAt, balance.LastTransferAt, balance.TransferCount,
	)
	return err
}

// GetBalance retrieves a holder's balance
func (r *Repository) GetBalance(ctx context.Context, network, tokenAddress, holderAddress string) (*TokenBalance, error) {
	query := `
		SELECT id, network, token_address, holder_address, balance,
			first_transfer_at, last_transfer_at, transfer_count, updated_at
		FROM token_balances
		WHERE network = $1 AND LOWER(token_address) = LOWER($2) AND LOWER(holder_address) = LOWER($3)`

	var b TokenBalance
	err := r.db.QueryRow(ctx, query, network, tokenAddress, holderAddress).Scan(
		&b.ID, &b.Network, &b.TokenAddress, &b.HolderAddress, &b.Balance,
		&b.FirstTransferAt, &b.LastTransferAt, &b.TransferCount, &b.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get balance: %w", err)
	}

	return &b, nil
}

// ListHolders lists token holders sorted by balance
func (r *Repository) ListHolders(ctx context.Context, network, tokenAddress string, page, pageSize int) ([]*TokenBalance, int64, error) {
	// Count
	var total int64
	countQuery := `SELECT COUNT(*) FROM token_balances WHERE network = $1 AND LOWER(token_address) = LOWER($2) AND balance > 0`
	if err := r.db.QueryRow(ctx, countQuery, network, tokenAddress).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count holders: %w", err)
	}

	// Fetch
	offset := (page - 1) * pageSize
	query := `
		SELECT id, network, token_address, holder_address, balance,
			first_transfer_at, last_transfer_at, transfer_count, updated_at
		FROM token_balances
		WHERE network = $1 AND LOWER(token_address) = LOWER($2) AND balance > 0
		ORDER BY balance DESC
		LIMIT $3 OFFSET $4`

	rows, err := r.db.Query(ctx, query, network, tokenAddress, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list holders: %w", err)
	}
	defer rows.Close()

	var balances []*TokenBalance
	for rows.Next() {
		var b TokenBalance
		if err := rows.Scan(
			&b.ID, &b.Network, &b.TokenAddress, &b.HolderAddress, &b.Balance,
			&b.FirstTransferAt, &b.LastTransferAt, &b.TransferCount, &b.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan balance: %w", err)
		}
		balances = append(balances, &b)
	}

	return balances, total, nil
}

// GetHolderTokens lists all tokens held by an address
func (r *Repository) GetHolderTokens(ctx context.Context, network, holderAddress string, page, pageSize int) ([]*TokenWithBalance, int64, error) {
	// Count
	var total int64
	countQuery := `SELECT COUNT(*) FROM token_balances WHERE network = $1 AND LOWER(holder_address) = LOWER($2) AND balance > 0`
	if err := r.db.QueryRow(ctx, countQuery, network, holderAddress).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count tokens: %w", err)
	}

	// Fetch with token info
	offset := (page - 1) * pageSize
	query := `
		SELECT t.id, t.network, t.contract_address, t.name, t.symbol, t.decimals, t.total_supply,
			t.holder_count, t.transfer_count, t.logo_url, t.is_verified, t.token_type,
			b.balance
		FROM token_balances b
		JOIN tokens t ON t.network = b.network AND LOWER(t.contract_address) = LOWER(b.token_address)
		WHERE b.network = $1 AND LOWER(b.holder_address) = LOWER($2) AND b.balance > 0
		ORDER BY b.balance DESC
		LIMIT $3 OFFSET $4`

	rows, err := r.db.Query(ctx, query, network, holderAddress, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list holder tokens: %w", err)
	}
	defer rows.Close()

	var results []*TokenWithBalance
	for rows.Next() {
		var token Token
		var balance string
		if err := rows.Scan(
			&token.ID, &token.Network, &token.ContractAddress, &token.Name, &token.Symbol, &token.Decimals, &token.TotalSupply,
			&token.HolderCount, &token.TransferCount, &token.LogoURL, &token.IsVerified, &token.TokenType,
			&balance,
		); err != nil {
			return nil, 0, fmt.Errorf("scan token balance: %w", err)
		}

		formatted := FormatBalance(balance, token.Decimals)
		results = append(results, &TokenWithBalance{
			Token:            &token,
			Balance:          balance,
			FormattedBalance: formatted,
		})
	}

	return results, total, nil
}

// ============================================================================
// WELL-KNOWN TOKENS
// ============================================================================

// GetWellKnownToken retrieves a well-known token
func (r *Repository) GetWellKnownToken(ctx context.Context, network, address string) (*WellKnownToken, error) {
	query := `
		SELECT id, network, contract_address, name, symbol, decimals, logo_url,
			coingecko_id, is_stablecoin, is_wrapped_native
		FROM well_known_tokens
		WHERE network = $1 AND LOWER(contract_address) = LOWER($2)`

	var t WellKnownToken
	err := r.db.QueryRow(ctx, query, network, address).Scan(
		&t.ID, &t.Network, &t.ContractAddress, &t.Name, &t.Symbol, &t.Decimals, &t.LogoURL,
		&t.CoingeckoID, &t.IsStablecoin, &t.IsWrappedNative,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get well-known token: %w", err)
	}

	return &t, nil
}

// ListWellKnownTokens lists all well-known tokens for a network
func (r *Repository) ListWellKnownTokens(ctx context.Context, network string) ([]*WellKnownToken, error) {
	query := `
		SELECT id, network, contract_address, name, symbol, decimals, logo_url,
			coingecko_id, is_stablecoin, is_wrapped_native
		FROM well_known_tokens
		WHERE network = $1
		ORDER BY name`

	rows, err := r.db.Query(ctx, query, network)
	if err != nil {
		return nil, fmt.Errorf("list well-known tokens: %w", err)
	}
	defer rows.Close()

	var tokens []*WellKnownToken
	for rows.Next() {
		var t WellKnownToken
		if err := rows.Scan(
			&t.ID, &t.Network, &t.ContractAddress, &t.Name, &t.Symbol, &t.Decimals, &t.LogoURL,
			&t.CoingeckoID, &t.IsStablecoin, &t.IsWrappedNative,
		); err != nil {
			return nil, fmt.Errorf("scan well-known token: %w", err)
		}
		tokens = append(tokens, &t)
	}

	return tokens, nil
}
