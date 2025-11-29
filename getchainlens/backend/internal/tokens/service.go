package tokens

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Service provides token tracking functionality
type Service struct {
	repo       *Repository
	erc20      *ERC20Client
}

// NewService creates a new token service
func NewService(db *pgxpool.Pool, rpcURLs map[string]string) *Service {
	return &Service{
		repo:  NewRepository(db),
		erc20: NewERC20Client(rpcURLs),
	}
}

// WithERC20Client sets a custom ERC20 client (for testing)
func (s *Service) WithERC20Client(client *ERC20Client) *Service {
	s.erc20 = client
	return s
}

// ============================================================================
// TOKEN MANAGEMENT
// ============================================================================

// GetToken retrieves a token, fetching metadata from chain if not cached
func (s *Service) GetToken(ctx context.Context, network, address string) (*Token, error) {
	// Check database first
	token, err := s.repo.GetToken(ctx, network, address)
	if err != nil {
		return nil, err
	}

	if token != nil {
		return token, nil
	}

	// Try to fetch from chain
	token, err = s.discoverToken(ctx, network, address)
	if err != nil {
		return nil, err
	}

	if token == nil {
		return nil, nil
	}

	// Save to database
	if err := s.repo.UpsertToken(ctx, token); err != nil {
		// Log but don't fail
		fmt.Printf("warning: failed to cache token: %v\n", err)
	}

	return token, nil
}

// discoverToken attempts to discover a token from the blockchain
func (s *Service) discoverToken(ctx context.Context, network, address string) (*Token, error) {
	// Check if it's an ERC-20 token
	if !s.erc20.IsERC20(ctx, network, address) {
		return nil, nil
	}

	// Fetch metadata
	metadata, err := s.erc20.GetTokenMetadata(ctx, network, address)
	if err != nil {
		return nil, fmt.Errorf("get metadata: %w", err)
	}

	token := &Token{
		Network:         network,
		ContractAddress: strings.ToLower(address),
		Decimals:        metadata.Decimals,
		TokenType:       TokenTypeERC20,
	}

	if metadata.Name != "" {
		token.Name = &metadata.Name
	}
	if metadata.Symbol != "" {
		token.Symbol = &metadata.Symbol
	}
	if metadata.TotalSupply != nil {
		supply := metadata.TotalSupply.String()
		token.TotalSupply = &supply
	}

	// Check for well-known token info
	wellKnown, _ := s.repo.GetWellKnownToken(ctx, network, address)
	if wellKnown != nil {
		token.Name = &wellKnown.Name
		token.Symbol = &wellKnown.Symbol
		token.Decimals = wellKnown.Decimals
		token.LogoURL = wellKnown.LogoURL
		token.CoingeckoID = wellKnown.CoingeckoID
		token.IsVerified = true
	}

	// Detect token type if not ERC-20
	detectedType := s.erc20.DetectTokenType(ctx, network, address)
	if detectedType != "" {
		token.TokenType = detectedType
	}

	return token, nil
}

// ListTokens lists tokens with filtering
func (s *Service) ListTokens(ctx context.Context, filter TokenFilter) (*ListResult[*Token], error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 || filter.PageSize > 100 {
		filter.PageSize = 20
	}

	tokens, total, err := s.repo.ListTokens(ctx, filter)
	if err != nil {
		return nil, err
	}

	totalPages := int(total) / filter.PageSize
	if int(total)%filter.PageSize > 0 {
		totalPages++
	}

	return &ListResult[*Token]{
		Items:      tokens,
		Total:      total,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalPages: totalPages,
	}, nil
}

// SearchTokens searches for tokens by name or symbol
func (s *Service) SearchTokens(ctx context.Context, network, query string) ([]*Token, error) {
	filter := TokenFilter{
		Network:  network,
		Query:    &query,
		Page:     1,
		PageSize: 20,
	}

	tokens, _, err := s.repo.ListTokens(ctx, filter)
	return tokens, err
}

// ============================================================================
// TRANSFER PROCESSING
// ============================================================================

// ProcessTransfer processes a token transfer event
func (s *Service) ProcessTransfer(ctx context.Context, transfer *TokenTransfer) error {
	// Ensure token exists
	token, err := s.GetToken(ctx, transfer.Network, transfer.TokenAddress)
	if err != nil {
		return fmt.Errorf("get token: %w", err)
	}

	// Add token info to transfer
	if token != nil {
		transfer.TokenSymbol = token.Symbol
		transfer.TokenDecimals = &token.Decimals
	}

	// Save transfer
	if err := s.repo.InsertTransfer(ctx, transfer); err != nil {
		return fmt.Errorf("insert transfer: %w", err)
	}

	// Update balances
	value := new(big.Int)
	value.SetString(transfer.Value, 10)

	// Decrease sender balance (unless mint)
	if transfer.FromAddress != ZeroAddress {
		negValue := new(big.Int).Neg(value)
		if err := s.repo.UpdateBalance(ctx, transfer.Network, transfer.TokenAddress, transfer.FromAddress, negValue, transfer.Timestamp); err != nil {
			return fmt.Errorf("update sender balance: %w", err)
		}
	}

	// Increase receiver balance (unless burn)
	if transfer.ToAddress != ZeroAddress {
		if err := s.repo.UpdateBalance(ctx, transfer.Network, transfer.TokenAddress, transfer.ToAddress, value, transfer.Timestamp); err != nil {
			return fmt.Errorf("update receiver balance: %w", err)
		}
	}

	return nil
}

// ProcessTransfers batch processes multiple transfers
func (s *Service) ProcessTransfers(ctx context.Context, transfers []*TokenTransfer) error {
	if len(transfers) == 0 {
		return nil
	}

	// Group by token to batch token lookups
	tokenAddresses := make(map[string]bool)
	for _, t := range transfers {
		key := t.Network + ":" + strings.ToLower(t.TokenAddress)
		tokenAddresses[key] = true
	}

	// Prefetch tokens
	tokenCache := make(map[string]*Token)
	for key := range tokenAddresses {
		parts := strings.SplitN(key, ":", 2)
		token, _ := s.GetToken(ctx, parts[0], parts[1])
		if token != nil {
			tokenCache[key] = token
		}
	}

	// Enrich transfers with token info
	for _, t := range transfers {
		key := t.Network + ":" + strings.ToLower(t.TokenAddress)
		if token, ok := tokenCache[key]; ok {
			t.TokenSymbol = token.Symbol
			t.TokenDecimals = &token.Decimals
		}
	}

	// Batch insert transfers
	if err := s.repo.InsertTransfers(ctx, transfers); err != nil {
		return fmt.Errorf("insert transfers: %w", err)
	}

	// Update balances for each transfer
	for _, t := range transfers {
		value := new(big.Int)
		value.SetString(t.Value, 10)

		if t.FromAddress != ZeroAddress {
			negValue := new(big.Int).Neg(value)
			if err := s.repo.UpdateBalance(ctx, t.Network, t.TokenAddress, t.FromAddress, negValue, t.Timestamp); err != nil {
				// Log but continue
				fmt.Printf("warning: failed to update sender balance: %v\n", err)
			}
		}

		if t.ToAddress != ZeroAddress {
			if err := s.repo.UpdateBalance(ctx, t.Network, t.TokenAddress, t.ToAddress, value, t.Timestamp); err != nil {
				fmt.Printf("warning: failed to update receiver balance: %v\n", err)
			}
		}
	}

	// Update token stats
	for key := range tokenAddresses {
		parts := strings.SplitN(key, ":", 2)
		_ = s.repo.UpdateTokenStats(ctx, parts[0], parts[1])
	}

	return nil
}

// ============================================================================
// TRANSFER QUERIES
// ============================================================================

// ListTransfers lists token transfers with filtering
func (s *Service) ListTransfers(ctx context.Context, filter TransferFilter) (*ListResult[*TokenTransfer], error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 || filter.PageSize > 100 {
		filter.PageSize = 20
	}

	transfers, total, err := s.repo.ListTransfers(ctx, filter)
	if err != nil {
		return nil, err
	}

	totalPages := int(total) / filter.PageSize
	if int(total)%filter.PageSize > 0 {
		totalPages++
	}

	return &ListResult[*TokenTransfer]{
		Items:      transfers,
		Total:      total,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalPages: totalPages,
	}, nil
}

// GetTransfersByTxHash returns all token transfers in a transaction
func (s *Service) GetTransfersByTxHash(ctx context.Context, network, txHash string) ([]*TokenTransfer, error) {
	return s.repo.GetTransfersByTxHash(ctx, network, txHash)
}

// GetTokenTransfers returns transfers for a specific token
func (s *Service) GetTokenTransfers(ctx context.Context, network, tokenAddress string, page, pageSize int) (*ListResult[*TokenTransfer], error) {
	filter := TransferFilter{
		Network:      network,
		TokenAddress: &tokenAddress,
		Page:         page,
		PageSize:     pageSize,
	}
	return s.ListTransfers(ctx, filter)
}

// GetAddressTransfers returns token transfers for an address (as sender or receiver)
func (s *Service) GetAddressTransfers(ctx context.Context, network, address string, page, pageSize int) (*ListResult[*TokenTransfer], error) {
	// We need to query both from and to
	// For simplicity, we'll do two queries and merge
	// In production, you might want a combined index

	fromFilter := TransferFilter{
		Network:     network,
		FromAddress: &address,
		Page:        page,
		PageSize:    pageSize,
	}

	toFilter := TransferFilter{
		Network:   network,
		ToAddress: &address,
		Page:      page,
		PageSize:  pageSize,
	}

	fromTransfers, fromTotal, err := s.repo.ListTransfers(ctx, fromFilter)
	if err != nil {
		return nil, err
	}

	toTransfers, toTotal, err := s.repo.ListTransfers(ctx, toFilter)
	if err != nil {
		return nil, err
	}

	// Merge and deduplicate
	seen := make(map[string]bool)
	var all []*TokenTransfer
	for _, t := range fromTransfers {
		key := fmt.Sprintf("%s:%d", t.TxHash, t.LogIndex)
		if !seen[key] {
			seen[key] = true
			all = append(all, t)
		}
	}
	for _, t := range toTransfers {
		key := fmt.Sprintf("%s:%d", t.TxHash, t.LogIndex)
		if !seen[key] {
			seen[key] = true
			all = append(all, t)
		}
	}

	// Sort by block number desc
	// Note: This is a simplified approach; for production you'd want a proper combined query
	total := fromTotal + toTotal

	return &ListResult[*TokenTransfer]{
		Items:      all,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: int(total)/pageSize + 1,
	}, nil
}

// ============================================================================
// BALANCE QUERIES
// ============================================================================

// GetBalance retrieves a holder's balance for a specific token
func (s *Service) GetBalance(ctx context.Context, network, tokenAddress, holderAddress string) (*TokenBalance, error) {
	return s.repo.GetBalance(ctx, network, tokenAddress, holderAddress)
}

// GetBalanceFromChain fetches the current balance from the blockchain
func (s *Service) GetBalanceFromChain(ctx context.Context, network, tokenAddress, holderAddress string) (*big.Int, error) {
	return s.erc20.GetBalance(ctx, network, tokenAddress, holderAddress)
}

// SyncBalance syncs a holder's balance from the blockchain
func (s *Service) SyncBalance(ctx context.Context, network, tokenAddress, holderAddress string) (*TokenBalance, error) {
	balance, err := s.erc20.GetBalance(ctx, network, tokenAddress, holderAddress)
	if err != nil {
		return nil, fmt.Errorf("get balance from chain: %w", err)
	}

	now := time.Now()
	tb := &TokenBalance{
		Network:        network,
		TokenAddress:   tokenAddress,
		HolderAddress:  holderAddress,
		Balance:        balance.String(),
		LastTransferAt: &now,
	}

	if err := s.repo.SetBalance(ctx, tb); err != nil {
		return nil, fmt.Errorf("set balance: %w", err)
	}

	return tb, nil
}

// ListHolders lists token holders sorted by balance
func (s *Service) ListHolders(ctx context.Context, network, tokenAddress string, page, pageSize int) (*ListResult[*TokenHolder], error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	balances, total, err := s.repo.ListHolders(ctx, network, tokenAddress, page, pageSize)
	if err != nil {
		return nil, err
	}

	// Get token for total supply (for percentage calculation)
	token, _ := s.repo.GetToken(ctx, network, tokenAddress)
	var totalSupply *big.Int
	if token != nil && token.TotalSupply != nil {
		totalSupply = new(big.Int)
		totalSupply.SetString(*token.TotalSupply, 10)
	}

	// Convert to TokenHolder with rank and percentage
	holders := make([]*TokenHolder, len(balances))
	for i, b := range balances {
		holder := &TokenHolder{
			Address: b.HolderAddress,
			Balance: b.Balance,
			Rank:    (page-1)*pageSize + i + 1,
		}

		if totalSupply != nil && totalSupply.Sign() > 0 {
			balance := new(big.Int)
			balance.SetString(b.Balance, 10)
			// percentage = (balance / totalSupply) * 100
			percentBig := new(big.Float).Quo(
				new(big.Float).SetInt(balance),
				new(big.Float).SetInt(totalSupply),
			)
			percentBig.Mul(percentBig, big.NewFloat(100))
			holder.Percentage, _ = percentBig.Float64()
		}

		holders[i] = holder
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return &ListResult[*TokenHolder]{
		Items:      holders,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// GetHolderTokens lists all tokens held by an address
func (s *Service) GetHolderTokens(ctx context.Context, network, holderAddress string, page, pageSize int) (*ListResult[*TokenWithBalance], error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	tokens, total, err := s.repo.GetHolderTokens(ctx, network, holderAddress, page, pageSize)
	if err != nil {
		return nil, err
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return &ListResult[*TokenWithBalance]{
		Items:      tokens,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// ============================================================================
// EVENT PARSING
// ============================================================================

// ParseLogAsTransfer attempts to parse a log as a Transfer event
func (s *Service) ParseLogAsTransfer(network, txHash string, logIndex int, blockNumber int64, contractAddress string, topics []string, data string, timestamp time.Time) *TokenTransfer {
	transfer, err := ParseTransferEvent(topics, data)
	if err != nil || transfer == nil {
		return nil
	}

	transfer.Network = network
	transfer.TxHash = txHash
	transfer.LogIndex = logIndex
	transfer.BlockNumber = blockNumber
	transfer.TokenAddress = contractAddress
	transfer.Timestamp = timestamp

	return transfer
}

// ParseLogAsApproval attempts to parse a log as an Approval event
func (s *Service) ParseLogAsApproval(network, txHash string, logIndex int, blockNumber int64, contractAddress string, topics []string, data string, timestamp time.Time) *TokenApproval {
	approval, err := ParseApprovalEvent(topics, data)
	if err != nil || approval == nil {
		return nil
	}

	approval.Network = network
	approval.TxHash = txHash
	approval.LogIndex = logIndex
	approval.BlockNumber = blockNumber
	approval.TokenAddress = contractAddress
	approval.Timestamp = timestamp

	return approval
}
