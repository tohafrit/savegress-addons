package nfts

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository provides database operations for NFTs
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new NFT repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// ============================================================================
// COLLECTIONS
// ============================================================================

// UpsertCollection creates or updates a collection
func (r *Repository) UpsertCollection(ctx context.Context, c *NFTCollection) error {
	query := `
		INSERT INTO nft_collections (
			network, contract_address, name, symbol, standard, description,
			total_supply, owner_count, transfer_count, floor_price, volume_total,
			base_uri, contract_uri, website, twitter, discord, opensea_slug,
			image_url, banner_url, royalty_recipient, royalty_bps,
			is_verified, is_spam, supports_eip2981,
			deployer_address, deploy_block, deploy_tx_hash, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28)
		ON CONFLICT (network, contract_address) DO UPDATE SET
			name = COALESCE(EXCLUDED.name, nft_collections.name),
			symbol = COALESCE(EXCLUDED.symbol, nft_collections.symbol),
			description = COALESCE(EXCLUDED.description, nft_collections.description),
			total_supply = COALESCE(EXCLUDED.total_supply, nft_collections.total_supply),
			owner_count = EXCLUDED.owner_count,
			transfer_count = EXCLUDED.transfer_count,
			base_uri = COALESCE(EXCLUDED.base_uri, nft_collections.base_uri),
			image_url = COALESCE(EXCLUDED.image_url, nft_collections.image_url),
			updated_at = NOW()
		RETURNING id`

	return r.db.QueryRow(ctx, query,
		c.Network, strings.ToLower(c.ContractAddress), c.Name, c.Symbol, c.Standard, c.Description,
		c.TotalSupply, c.OwnerCount, c.TransferCount, c.FloorPrice, c.VolumeTotal,
		c.BaseURI, c.ContractURI, c.Website, c.Twitter, c.Discord, c.OpenseaSlug,
		c.ImageURL, c.BannerURL, c.RoyaltyRecipient, c.RoyaltyBPS,
		c.IsVerified, c.IsSpam, c.SupportsEIP2981,
		c.DeployerAddress, c.DeployBlock, c.DeployTxHash, c.Metadata,
	).Scan(&c.ID)
}

// GetCollection retrieves a collection by network and address
func (r *Repository) GetCollection(ctx context.Context, network, address string) (*NFTCollection, error) {
	query := `
		SELECT id, network, contract_address, name, symbol, standard, description,
			total_supply, owner_count, transfer_count, floor_price, volume_total,
			base_uri, contract_uri, website, twitter, discord, opensea_slug,
			image_url, banner_url, royalty_recipient, royalty_bps,
			is_verified, is_spam, supports_eip2981,
			deployer_address, deploy_block, deploy_tx_hash, metadata,
			created_at, updated_at
		FROM nft_collections
		WHERE network = $1 AND LOWER(contract_address) = LOWER($2)`

	var c NFTCollection
	err := r.db.QueryRow(ctx, query, network, address).Scan(
		&c.ID, &c.Network, &c.ContractAddress, &c.Name, &c.Symbol, &c.Standard, &c.Description,
		&c.TotalSupply, &c.OwnerCount, &c.TransferCount, &c.FloorPrice, &c.VolumeTotal,
		&c.BaseURI, &c.ContractURI, &c.Website, &c.Twitter, &c.Discord, &c.OpenseaSlug,
		&c.ImageURL, &c.BannerURL, &c.RoyaltyRecipient, &c.RoyaltyBPS,
		&c.IsVerified, &c.IsSpam, &c.SupportsEIP2981,
		&c.DeployerAddress, &c.DeployBlock, &c.DeployTxHash, &c.Metadata,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get collection: %w", err)
	}

	return &c, nil
}

// ListCollections lists collections with filtering
func (r *Repository) ListCollections(ctx context.Context, filter CollectionFilter) ([]*NFTCollection, int64, error) {
	var conditions []string
	var args []interface{}
	argNum := 1

	conditions = append(conditions, fmt.Sprintf("network = $%d", argNum))
	args = append(args, filter.Network)
	argNum++

	if filter.Standard != nil {
		conditions = append(conditions, fmt.Sprintf("standard = $%d", argNum))
		args = append(args, *filter.Standard)
		argNum++
	}

	if filter.Query != nil && *filter.Query != "" {
		conditions = append(conditions, fmt.Sprintf("(LOWER(name) LIKE LOWER($%d) OR LOWER(symbol) LIKE LOWER($%d))", argNum, argNum))
		args = append(args, "%"+*filter.Query+"%")
		argNum++
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	// Count
	var total int64
	countQuery := "SELECT COUNT(*) FROM nft_collections " + whereClause
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count collections: %w", err)
	}

	// Sort
	sortBy := "owner_count"
	if filter.SortBy != "" {
		switch filter.SortBy {
		case "owner_count", "transfer_count", "volume_total", "name":
			sortBy = filter.SortBy
		}
	}
	sortOrder := "DESC"
	if filter.SortOrder == "asc" {
		sortOrder = "ASC"
	}

	offset := (filter.Page - 1) * filter.PageSize
	query := fmt.Sprintf(`
		SELECT id, network, contract_address, name, symbol, standard, description,
			total_supply, owner_count, transfer_count, floor_price, volume_total,
			base_uri, contract_uri, website, twitter, discord, opensea_slug,
			image_url, banner_url, royalty_recipient, royalty_bps,
			is_verified, is_spam, supports_eip2981,
			deployer_address, deploy_block, deploy_tx_hash, metadata,
			created_at, updated_at
		FROM nft_collections
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d`, whereClause, sortBy, sortOrder, argNum, argNum+1)

	args = append(args, filter.PageSize, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list collections: %w", err)
	}
	defer rows.Close()

	var collections []*NFTCollection
	for rows.Next() {
		var c NFTCollection
		if err := rows.Scan(
			&c.ID, &c.Network, &c.ContractAddress, &c.Name, &c.Symbol, &c.Standard, &c.Description,
			&c.TotalSupply, &c.OwnerCount, &c.TransferCount, &c.FloorPrice, &c.VolumeTotal,
			&c.BaseURI, &c.ContractURI, &c.Website, &c.Twitter, &c.Discord, &c.OpenseaSlug,
			&c.ImageURL, &c.BannerURL, &c.RoyaltyRecipient, &c.RoyaltyBPS,
			&c.IsVerified, &c.IsSpam, &c.SupportsEIP2981,
			&c.DeployerAddress, &c.DeployBlock, &c.DeployTxHash, &c.Metadata,
			&c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan collection: %w", err)
		}
		collections = append(collections, &c)
	}

	return collections, total, nil
}

// ============================================================================
// NFT ITEMS
// ============================================================================

// UpsertItem creates or updates an NFT item
func (r *Repository) UpsertItem(ctx context.Context, item *NFTItem) error {
	query := `
		INSERT INTO nft_items (
			network, contract_address, token_id, owner_address,
			token_uri, metadata, metadata_fetched_at, metadata_error,
			name, description, image_url, animation_url, external_url, background_color,
			attributes, rarity_score, rarity_rank, transfer_count,
			last_sale_price, last_sale_currency, last_sale_at,
			total_supply, minted_at, last_transfer_at, burned_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25)
		ON CONFLICT (network, contract_address, token_id) DO UPDATE SET
			owner_address = COALESCE(EXCLUDED.owner_address, nft_items.owner_address),
			metadata = COALESCE(EXCLUDED.metadata, nft_items.metadata),
			metadata_fetched_at = COALESCE(EXCLUDED.metadata_fetched_at, nft_items.metadata_fetched_at),
			name = COALESCE(EXCLUDED.name, nft_items.name),
			description = COALESCE(EXCLUDED.description, nft_items.description),
			image_url = COALESCE(EXCLUDED.image_url, nft_items.image_url),
			attributes = COALESCE(EXCLUDED.attributes, nft_items.attributes),
			transfer_count = EXCLUDED.transfer_count,
			last_transfer_at = EXCLUDED.last_transfer_at,
			burned_at = EXCLUDED.burned_at,
			updated_at = NOW()
		RETURNING id`

	return r.db.QueryRow(ctx, query,
		item.Network, strings.ToLower(item.ContractAddress), item.TokenID, item.OwnerAddress,
		item.TokenURI, item.Metadata, item.MetadataFetchedAt, item.MetadataError,
		item.Name, item.Description, item.ImageURL, item.AnimationURL, item.ExternalURL, item.BackgroundColor,
		item.Attributes, item.RarityScore, item.RarityRank, item.TransferCount,
		item.LastSalePrice, item.LastSaleCurrency, item.LastSaleAt,
		item.TotalSupply, item.MintedAt, item.LastTransferAt, item.BurnedAt,
	).Scan(&item.ID)
}

// GetItem retrieves an NFT item
func (r *Repository) GetItem(ctx context.Context, network, contractAddress, tokenID string) (*NFTItem, error) {
	query := `
		SELECT id, network, contract_address, token_id, owner_address,
			token_uri, metadata, metadata_fetched_at, metadata_error,
			name, description, image_url, animation_url, external_url, background_color,
			attributes, rarity_score, rarity_rank, transfer_count,
			last_sale_price, last_sale_currency, last_sale_at,
			total_supply, minted_at, last_transfer_at, burned_at,
			created_at, updated_at
		FROM nft_items
		WHERE network = $1 AND LOWER(contract_address) = LOWER($2) AND token_id = $3`

	var item NFTItem
	err := r.db.QueryRow(ctx, query, network, contractAddress, tokenID).Scan(
		&item.ID, &item.Network, &item.ContractAddress, &item.TokenID, &item.OwnerAddress,
		&item.TokenURI, &item.Metadata, &item.MetadataFetchedAt, &item.MetadataError,
		&item.Name, &item.Description, &item.ImageURL, &item.AnimationURL, &item.ExternalURL, &item.BackgroundColor,
		&item.Attributes, &item.RarityScore, &item.RarityRank, &item.TransferCount,
		&item.LastSalePrice, &item.LastSaleCurrency, &item.LastSaleAt,
		&item.TotalSupply, &item.MintedAt, &item.LastTransferAt, &item.BurnedAt,
		&item.CreatedAt, &item.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get item: %w", err)
	}

	return &item, nil
}

// ListItems lists NFT items with filtering
func (r *Repository) ListItems(ctx context.Context, filter ItemFilter) ([]*NFTItem, int64, error) {
	var conditions []string
	var args []interface{}
	argNum := 1

	conditions = append(conditions, fmt.Sprintf("network = $%d", argNum))
	args = append(args, filter.Network)
	argNum++

	if filter.ContractAddress != nil {
		conditions = append(conditions, fmt.Sprintf("LOWER(contract_address) = LOWER($%d)", argNum))
		args = append(args, *filter.ContractAddress)
		argNum++
	}

	if filter.OwnerAddress != nil {
		conditions = append(conditions, fmt.Sprintf("LOWER(owner_address) = LOWER($%d)", argNum))
		args = append(args, *filter.OwnerAddress)
		argNum++
	}

	if filter.TokenID != nil {
		conditions = append(conditions, fmt.Sprintf("token_id = $%d", argNum))
		args = append(args, *filter.TokenID)
		argNum++
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	// Count
	var total int64
	countQuery := "SELECT COUNT(*) FROM nft_items " + whereClause
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count items: %w", err)
	}

	offset := (filter.Page - 1) * filter.PageSize
	query := fmt.Sprintf(`
		SELECT id, network, contract_address, token_id, owner_address,
			token_uri, metadata, metadata_fetched_at, metadata_error,
			name, description, image_url, animation_url, external_url, background_color,
			attributes, rarity_score, rarity_rank, transfer_count,
			last_sale_price, last_sale_currency, last_sale_at,
			total_supply, minted_at, last_transfer_at, burned_at,
			created_at, updated_at
		FROM nft_items
		%s
		ORDER BY token_id::numeric
		LIMIT $%d OFFSET $%d`, whereClause, argNum, argNum+1)

	args = append(args, filter.PageSize, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list items: %w", err)
	}
	defer rows.Close()

	var items []*NFTItem
	for rows.Next() {
		var item NFTItem
		if err := rows.Scan(
			&item.ID, &item.Network, &item.ContractAddress, &item.TokenID, &item.OwnerAddress,
			&item.TokenURI, &item.Metadata, &item.MetadataFetchedAt, &item.MetadataError,
			&item.Name, &item.Description, &item.ImageURL, &item.AnimationURL, &item.ExternalURL, &item.BackgroundColor,
			&item.Attributes, &item.RarityScore, &item.RarityRank, &item.TransferCount,
			&item.LastSalePrice, &item.LastSaleCurrency, &item.LastSaleAt,
			&item.TotalSupply, &item.MintedAt, &item.LastTransferAt, &item.BurnedAt,
			&item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan item: %w", err)
		}
		items = append(items, &item)
	}

	return items, total, nil
}

// UpdateItemOwner updates the owner of an NFT item
func (r *Repository) UpdateItemOwner(ctx context.Context, network, contractAddress, tokenID, newOwner string) error {
	query := `
		UPDATE nft_items SET
			owner_address = $4,
			last_transfer_at = NOW(),
			transfer_count = transfer_count + 1,
			updated_at = NOW()
		WHERE network = $1 AND LOWER(contract_address) = LOWER($2) AND token_id = $3`

	_, err := r.db.Exec(ctx, query, network, contractAddress, tokenID, strings.ToLower(newOwner))
	return err
}

// ============================================================================
// NFT TRANSFERS
// ============================================================================

// InsertTransfer inserts an NFT transfer
func (r *Repository) InsertTransfer(ctx context.Context, t *NFTTransfer) error {
	query := `
		INSERT INTO nft_transfers (
			network, tx_hash, log_index, block_number,
			contract_address, token_id, from_address, to_address,
			amount, operator, transfer_type,
			sale_price, sale_currency, marketplace, timestamp
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		ON CONFLICT (network, tx_hash, log_index) DO NOTHING
		RETURNING id`

	err := r.db.QueryRow(ctx, query,
		t.Network, t.TxHash, t.LogIndex, t.BlockNumber,
		strings.ToLower(t.ContractAddress), t.TokenID, strings.ToLower(t.FromAddress), strings.ToLower(t.ToAddress),
		t.Amount, t.Operator, t.TransferType,
		t.SalePrice, t.SaleCurrency, t.Marketplace, t.Timestamp,
	).Scan(&t.ID)

	if err == pgx.ErrNoRows {
		return nil
	}
	return err
}

// InsertTransfers batch inserts transfers
func (r *Repository) InsertTransfers(ctx context.Context, transfers []*NFTTransfer) error {
	if len(transfers) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	query := `
		INSERT INTO nft_transfers (
			network, tx_hash, log_index, block_number,
			contract_address, token_id, from_address, to_address,
			amount, operator, transfer_type,
			sale_price, sale_currency, marketplace, timestamp
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		ON CONFLICT (network, tx_hash, log_index) DO NOTHING`

	for _, t := range transfers {
		batch.Queue(query,
			t.Network, t.TxHash, t.LogIndex, t.BlockNumber,
			strings.ToLower(t.ContractAddress), t.TokenID, strings.ToLower(t.FromAddress), strings.ToLower(t.ToAddress),
			t.Amount, t.Operator, t.TransferType,
			t.SalePrice, t.SaleCurrency, t.Marketplace, t.Timestamp,
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

// ListTransfers lists NFT transfers
func (r *Repository) ListTransfers(ctx context.Context, filter TransferFilter) ([]*NFTTransfer, int64, error) {
	var conditions []string
	var args []interface{}
	argNum := 1

	conditions = append(conditions, fmt.Sprintf("network = $%d", argNum))
	args = append(args, filter.Network)
	argNum++

	if filter.ContractAddress != nil {
		conditions = append(conditions, fmt.Sprintf("LOWER(contract_address) = LOWER($%d)", argNum))
		args = append(args, *filter.ContractAddress)
		argNum++
	}

	if filter.TokenID != nil {
		conditions = append(conditions, fmt.Sprintf("token_id = $%d", argNum))
		args = append(args, *filter.TokenID)
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

	if filter.TransferType != nil {
		conditions = append(conditions, fmt.Sprintf("transfer_type = $%d", argNum))
		args = append(args, *filter.TransferType)
		argNum++
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	// Count
	var total int64
	countQuery := "SELECT COUNT(*) FROM nft_transfers " + whereClause
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count transfers: %w", err)
	}

	offset := (filter.Page - 1) * filter.PageSize
	query := fmt.Sprintf(`
		SELECT id, network, tx_hash, log_index, block_number,
			contract_address, token_id, from_address, to_address,
			amount, operator, transfer_type,
			sale_price, sale_currency, marketplace, timestamp, created_at
		FROM nft_transfers
		%s
		ORDER BY block_number DESC, log_index DESC
		LIMIT $%d OFFSET $%d`, whereClause, argNum, argNum+1)

	args = append(args, filter.PageSize, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list transfers: %w", err)
	}
	defer rows.Close()

	var transfers []*NFTTransfer
	for rows.Next() {
		var t NFTTransfer
		if err := rows.Scan(
			&t.ID, &t.Network, &t.TxHash, &t.LogIndex, &t.BlockNumber,
			&t.ContractAddress, &t.TokenID, &t.FromAddress, &t.ToAddress,
			&t.Amount, &t.Operator, &t.TransferType,
			&t.SalePrice, &t.SaleCurrency, &t.Marketplace, &t.Timestamp, &t.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan transfer: %w", err)
		}
		transfers = append(transfers, &t)
	}

	return transfers, total, nil
}

// GetAddressNFTs returns all NFTs owned by an address
func (r *Repository) GetAddressNFTs(ctx context.Context, network, address string, page, pageSize int) ([]*NFTItem, int64, error) {
	filter := ItemFilter{
		Network:      network,
		OwnerAddress: &address,
		Page:         page,
		PageSize:     pageSize,
	}
	return r.ListItems(ctx, filter)
}

// UpdateCollectionStats updates collection statistics
func (r *Repository) UpdateCollectionStats(ctx context.Context, network, contractAddress string) error {
	_, err := r.db.Exec(ctx, "SELECT update_nft_collection_stats($1, $2)", network, strings.ToLower(contractAddress))
	return err
}

// SearchCollections searches collections by name or symbol
func (r *Repository) SearchCollections(ctx context.Context, network, query string, limit int) ([]*NFTCollection, error) {
	q := `
		SELECT id, network, contract_address, name, symbol, standard, description,
			total_supply, owner_count, transfer_count, floor_price, volume_total,
			base_uri, contract_uri, website, twitter, discord, opensea_slug,
			image_url, banner_url, royalty_recipient, royalty_bps,
			is_verified, is_spam, supports_eip2981,
			deployer_address, deploy_block, deploy_tx_hash, metadata,
			created_at, updated_at
		FROM nft_collections
		WHERE network = $1 AND (LOWER(name) LIKE LOWER($2) OR LOWER(symbol) LIKE LOWER($2))
		ORDER BY owner_count DESC
		LIMIT $3`

	rows, err := r.db.Query(ctx, q, network, "%"+query+"%", limit)
	if err != nil {
		return nil, fmt.Errorf("search collections: %w", err)
	}
	defer rows.Close()

	var collections []*NFTCollection
	for rows.Next() {
		var c NFTCollection
		if err := rows.Scan(
			&c.ID, &c.Network, &c.ContractAddress, &c.Name, &c.Symbol, &c.Standard, &c.Description,
			&c.TotalSupply, &c.OwnerCount, &c.TransferCount, &c.FloorPrice, &c.VolumeTotal,
			&c.BaseURI, &c.ContractURI, &c.Website, &c.Twitter, &c.Discord, &c.OpenseaSlug,
			&c.ImageURL, &c.BannerURL, &c.RoyaltyRecipient, &c.RoyaltyBPS,
			&c.IsVerified, &c.IsSpam, &c.SupportsEIP2981,
			&c.DeployerAddress, &c.DeployBlock, &c.DeployTxHash, &c.Metadata,
			&c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan collection: %w", err)
		}
		collections = append(collections, &c)
	}

	return collections, nil
}

// ============================================================================
// NFT BALANCES (for ERC-1155)
// ============================================================================

// UpdateBalance updates the balance for an ERC-1155 token holder
func (r *Repository) UpdateBalance(ctx context.Context, b *NFTBalance) error {
	query := `
		INSERT INTO nft_balances (network, contract_address, token_id, holder_address, balance, first_acquired_at, last_updated_at)
		VALUES ($1, $2, $3, $4, $5::numeric, $6, $7)
		ON CONFLICT (network, contract_address, token_id, holder_address) DO UPDATE SET
			balance = nft_balances.balance + EXCLUDED.balance::numeric,
			last_updated_at = EXCLUDED.last_updated_at
		RETURNING id`

	return r.db.QueryRow(ctx, query,
		b.Network, strings.ToLower(b.ContractAddress), b.TokenID, strings.ToLower(b.HolderAddress),
		b.Balance, b.FirstAcquiredAt, b.LastUpdatedAt,
	).Scan(&b.ID)
}

// GetBalance retrieves the balance for an ERC-1155 token holder
func (r *Repository) GetBalance(ctx context.Context, network, contractAddress, tokenID, holder string) (*NFTBalance, error) {
	query := `
		SELECT id, network, contract_address, token_id, holder_address, balance, first_acquired_at, last_updated_at
		FROM nft_balances
		WHERE network = $1 AND LOWER(contract_address) = LOWER($2) AND token_id = $3 AND LOWER(holder_address) = LOWER($4)`

	var b NFTBalance
	err := r.db.QueryRow(ctx, query, network, contractAddress, tokenID, holder).Scan(
		&b.ID, &b.Network, &b.ContractAddress, &b.TokenID, &b.HolderAddress, &b.Balance, &b.FirstAcquiredAt, &b.LastUpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get balance: %w", err)
	}
	return &b, nil
}

// GetTokenHolders returns holders of a specific ERC-1155 token
func (r *Repository) GetTokenHolders(ctx context.Context, network, contractAddress, tokenID string, page, pageSize int) ([]*NFTBalance, int64, error) {
	countQuery := `
		SELECT COUNT(*) FROM nft_balances
		WHERE network = $1 AND LOWER(contract_address) = LOWER($2) AND token_id = $3 AND balance > 0`

	var total int64
	if err := r.db.QueryRow(ctx, countQuery, network, contractAddress, tokenID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count holders: %w", err)
	}

	offset := (page - 1) * pageSize
	query := `
		SELECT id, network, contract_address, token_id, holder_address, balance, first_acquired_at, last_updated_at
		FROM nft_balances
		WHERE network = $1 AND LOWER(contract_address) = LOWER($2) AND token_id = $3 AND balance > 0
		ORDER BY balance::numeric DESC
		LIMIT $4 OFFSET $5`

	rows, err := r.db.Query(ctx, query, network, contractAddress, tokenID, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("get token holders: %w", err)
	}
	defer rows.Close()

	var balances []*NFTBalance
	for rows.Next() {
		var b NFTBalance
		if err := rows.Scan(&b.ID, &b.Network, &b.ContractAddress, &b.TokenID, &b.HolderAddress, &b.Balance, &b.FirstAcquiredAt, &b.LastUpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan balance: %w", err)
		}
		balances = append(balances, &b)
	}

	return balances, total, nil
}

// GetCollectionHolders returns all unique holders of a collection
func (r *Repository) GetCollectionHolders(ctx context.Context, network, contractAddress string, page, pageSize int) ([]*NFTBalance, int64, error) {
	countQuery := `
		SELECT COUNT(DISTINCT holder_address) FROM nft_balances
		WHERE network = $1 AND LOWER(contract_address) = LOWER($2) AND balance > 0`

	var total int64
	if err := r.db.QueryRow(ctx, countQuery, network, contractAddress).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count collection holders: %w", err)
	}

	offset := (page - 1) * pageSize
	query := `
		SELECT 0 as id, network, contract_address, '' as token_id, holder_address,
			   SUM(balance::numeric)::text as balance, MIN(first_acquired_at) as first_acquired_at, MAX(last_updated_at) as last_updated_at
		FROM nft_balances
		WHERE network = $1 AND LOWER(contract_address) = LOWER($2) AND balance > 0
		GROUP BY network, contract_address, holder_address
		ORDER BY SUM(balance::numeric) DESC
		LIMIT $3 OFFSET $4`

	rows, err := r.db.Query(ctx, query, network, contractAddress, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("get collection holders: %w", err)
	}
	defer rows.Close()

	var balances []*NFTBalance
	for rows.Next() {
		var b NFTBalance
		if err := rows.Scan(&b.ID, &b.Network, &b.ContractAddress, &b.TokenID, &b.HolderAddress, &b.Balance, &b.FirstAcquiredAt, &b.LastUpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan balance: %w", err)
		}
		balances = append(balances, &b)
	}

	return balances, total, nil
}

// ============================================================================
// METADATA CACHE
// ============================================================================

// GetMetadataCache retrieves cached metadata by URI hash
func (r *Repository) GetMetadataCache(ctx context.Context, uriHash string) (*NFTMetadataCache, error) {
	query := `
		SELECT id, uri_hash, uri, content, content_type, fetched_at, error, retry_count
		FROM nft_metadata_cache
		WHERE uri_hash = $1`

	var mc NFTMetadataCache
	err := r.db.QueryRow(ctx, query, uriHash).Scan(
		&mc.ID, &mc.URIHash, &mc.URI, &mc.Content, &mc.ContentType, &mc.FetchedAt, &mc.Error, &mc.RetryCount,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get metadata cache: %w", err)
	}
	return &mc, nil
}

// UpsertMetadataCache creates or updates cached metadata
func (r *Repository) UpsertMetadataCache(ctx context.Context, mc *NFTMetadataCache) error {
	query := `
		INSERT INTO nft_metadata_cache (uri_hash, uri, content, content_type, fetched_at, error, retry_count)
		VALUES ($1, $2, $3, $4, NOW(), $5, $6)
		ON CONFLICT (uri_hash) DO UPDATE SET
			content = EXCLUDED.content,
			content_type = EXCLUDED.content_type,
			fetched_at = NOW(),
			error = EXCLUDED.error,
			retry_count = CASE
				WHEN EXCLUDED.error IS NOT NULL THEN nft_metadata_cache.retry_count + 1
				ELSE 0
			END
		RETURNING id`

	return r.db.QueryRow(ctx, query,
		mc.URIHash, mc.URI, mc.Content, mc.ContentType, mc.Error, mc.RetryCount,
	).Scan(&mc.ID)
}

// ============================================================================
// ADDRESS QUERIES
// ============================================================================

// GetAddressTransfers returns NFT transfers for an address
func (r *Repository) GetAddressTransfers(ctx context.Context, network, address string, page, pageSize int) ([]*NFTTransfer, int64, error) {
	countQuery := `
		SELECT COUNT(*) FROM nft_transfers
		WHERE network = $1 AND (LOWER(from_address) = LOWER($2) OR LOWER(to_address) = LOWER($2))`

	var total int64
	if err := r.db.QueryRow(ctx, countQuery, network, address).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count transfers: %w", err)
	}

	offset := (page - 1) * pageSize
	query := `
		SELECT id, network, tx_hash, log_index, block_number,
			contract_address, token_id, from_address, to_address,
			amount, operator, transfer_type,
			sale_price, sale_currency, marketplace, timestamp, created_at
		FROM nft_transfers
		WHERE network = $1 AND (LOWER(from_address) = LOWER($2) OR LOWER(to_address) = LOWER($2))
		ORDER BY block_number DESC, log_index DESC
		LIMIT $3 OFFSET $4`

	rows, err := r.db.Query(ctx, query, network, address, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("get address transfers: %w", err)
	}
	defer rows.Close()

	var transfers []*NFTTransfer
	for rows.Next() {
		var t NFTTransfer
		if err := rows.Scan(
			&t.ID, &t.Network, &t.TxHash, &t.LogIndex, &t.BlockNumber,
			&t.ContractAddress, &t.TokenID, &t.FromAddress, &t.ToAddress,
			&t.Amount, &t.Operator, &t.TransferType,
			&t.SalePrice, &t.SaleCurrency, &t.Marketplace, &t.Timestamp, &t.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan transfer: %w", err)
		}
		transfers = append(transfers, &t)
	}

	return transfers, total, nil
}
