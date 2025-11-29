package nfts

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

// Service provides NFT business logic
type Service struct {
	repo            *Repository
	metadataFetcher *MetadataFetcher
	rpcClients      map[string]RPCClient

	// Background metadata fetching
	metadataQueue chan metadataJob
	wg            sync.WaitGroup
}

// RPCClient interface for blockchain RPC calls
type RPCClient interface {
	Call(ctx context.Context, method string, params ...interface{}) (json.RawMessage, error)
}

type metadataJob struct {
	network         string
	contractAddress string
	tokenID         string
	tokenURI        string
}

// NewService creates a new NFT service
func NewService(repo *Repository) *Service {
	s := &Service{
		repo:            repo,
		metadataFetcher: NewMetadataFetcher(),
		rpcClients:      make(map[string]RPCClient),
		metadataQueue:   make(chan metadataJob, 1000),
	}

	// Start background metadata workers
	for i := 0; i < 5; i++ {
		s.wg.Add(1)
		go s.metadataWorker()
	}

	return s
}

// SetRPCClient sets the RPC client for a network
func (s *Service) SetRPCClient(network string, client RPCClient) {
	s.rpcClients[network] = client
}

// Close stops background workers
func (s *Service) Close() {
	close(s.metadataQueue)
	s.wg.Wait()
}

// ProcessERC721Transfer processes an ERC-721 transfer event
func (s *Service) ProcessERC721Transfer(ctx context.Context, transfer *NFTTransfer) error {
	// Ensure collection exists
	if err := s.ensureCollection(ctx, transfer.Network, transfer.ContractAddress, StandardERC721); err != nil {
		log.Printf("Warning: failed to ensure collection: %v", err)
	}

	// Update item ownership
	item := &NFTItem{
		Network:         transfer.Network,
		ContractAddress: transfer.ContractAddress,
		TokenID:         transfer.TokenID,
		OwnerAddress:    &transfer.ToAddress,
		LastTransferAt:  &transfer.Timestamp,
	}

	// Check if mint
	if transfer.IsMint() {
		item.MintedAt = &transfer.Timestamp
		transfer.TransferType = TransferTypeMint
	}

	// Check if burn
	if transfer.IsBurn() {
		now := transfer.Timestamp
		item.BurnedAt = &now
		item.OwnerAddress = nil
		transfer.TransferType = TransferTypeBurn
	}

	// Upsert item
	if err := s.repo.UpsertItem(ctx, item); err != nil {
		return fmt.Errorf("upsert item: %w", err)
	}

	// Insert transfer record
	if err := s.repo.InsertTransfer(ctx, transfer); err != nil {
		return fmt.Errorf("insert transfer: %w", err)
	}

	// Queue metadata fetch if new mint
	if transfer.IsMint() {
		s.queueMetadataFetch(transfer.Network, transfer.ContractAddress, transfer.TokenID, "")
	}

	return nil
}

// ProcessERC1155TransferSingle processes an ERC-1155 TransferSingle event
func (s *Service) ProcessERC1155TransferSingle(ctx context.Context, transfer *NFTTransfer) error {
	// Ensure collection exists
	if err := s.ensureCollection(ctx, transfer.Network, transfer.ContractAddress, StandardERC1155); err != nil {
		log.Printf("Warning: failed to ensure collection: %v", err)
	}

	// Determine transfer type
	if transfer.IsMint() {
		transfer.TransferType = TransferTypeMint
	} else if transfer.IsBurn() {
		transfer.TransferType = TransferTypeBurn
	}

	// Update balances
	if err := s.updateERC1155Balances(ctx, transfer); err != nil {
		return fmt.Errorf("update balances: %w", err)
	}

	// Insert transfer record
	if err := s.repo.InsertTransfer(ctx, transfer); err != nil {
		return fmt.Errorf("insert transfer: %w", err)
	}

	// Ensure item exists
	item := &NFTItem{
		Network:         transfer.Network,
		ContractAddress: transfer.ContractAddress,
		TokenID:         transfer.TokenID,
	}
	if transfer.IsMint() {
		item.MintedAt = &transfer.Timestamp
	}
	if err := s.repo.UpsertItem(ctx, item); err != nil {
		log.Printf("Warning: failed to upsert item: %v", err)
	}

	// Queue metadata fetch if new mint
	if transfer.IsMint() {
		s.queueMetadataFetch(transfer.Network, transfer.ContractAddress, transfer.TokenID, "")
	}

	return nil
}

// ProcessERC1155TransferBatch processes an ERC-1155 TransferBatch event
func (s *Service) ProcessERC1155TransferBatch(ctx context.Context, transfers []*NFTTransfer) error {
	if len(transfers) == 0 {
		return nil
	}

	// Ensure collection exists (use first transfer for collection info)
	if err := s.ensureCollection(ctx, transfers[0].Network, transfers[0].ContractAddress, StandardERC1155); err != nil {
		log.Printf("Warning: failed to ensure collection: %v", err)
	}

	for _, transfer := range transfers {
		// Determine transfer type
		if transfer.IsMint() {
			transfer.TransferType = TransferTypeMint
		} else if transfer.IsBurn() {
			transfer.TransferType = TransferTypeBurn
		}

		// Update balances
		if err := s.updateERC1155Balances(ctx, transfer); err != nil {
			log.Printf("Warning: failed to update balances: %v", err)
		}

		// Ensure item exists
		item := &NFTItem{
			Network:         transfer.Network,
			ContractAddress: transfer.ContractAddress,
			TokenID:         transfer.TokenID,
		}
		if transfer.IsMint() {
			item.MintedAt = &transfer.Timestamp
			s.queueMetadataFetch(transfer.Network, transfer.ContractAddress, transfer.TokenID, "")
		}
		if err := s.repo.UpsertItem(ctx, item); err != nil {
			log.Printf("Warning: failed to upsert item: %v", err)
		}
	}

	// Batch insert transfers
	if err := s.repo.InsertTransfers(ctx, transfers); err != nil {
		return fmt.Errorf("insert transfers: %w", err)
	}

	return nil
}

// updateERC1155Balances updates balances for ERC-1155 transfer
func (s *Service) updateERC1155Balances(ctx context.Context, transfer *NFTTransfer) error {
	now := time.Now()

	// Decrease sender balance (if not mint)
	if !transfer.IsMint() {
		if err := s.repo.UpdateBalance(ctx, &NFTBalance{
			Network:         transfer.Network,
			ContractAddress: transfer.ContractAddress,
			TokenID:         transfer.TokenID,
			HolderAddress:   transfer.FromAddress,
			Balance:         "-" + transfer.Amount,
			LastUpdatedAt:   now,
		}); err != nil {
			return fmt.Errorf("decrease sender balance: %w", err)
		}
	}

	// Increase receiver balance (if not burn)
	if !transfer.IsBurn() {
		if err := s.repo.UpdateBalance(ctx, &NFTBalance{
			Network:          transfer.Network,
			ContractAddress:  transfer.ContractAddress,
			TokenID:          transfer.TokenID,
			HolderAddress:    transfer.ToAddress,
			Balance:          transfer.Amount,
			FirstAcquiredAt:  &now,
			LastUpdatedAt:    now,
		}); err != nil {
			return fmt.Errorf("increase receiver balance: %w", err)
		}
	}

	return nil
}

// ensureCollection ensures the collection exists in the database
func (s *Service) ensureCollection(ctx context.Context, network, contractAddress, standard string) error {
	// Check if collection exists
	_, err := s.repo.GetCollection(ctx, network, contractAddress)
	if err == nil {
		return nil // Already exists
	}

	// Create new collection
	collection := &NFTCollection{
		Network:         network,
		ContractAddress: contractAddress,
		Standard:        standard,
	}

	// Try to fetch collection metadata from chain
	if client, ok := s.rpcClients[network]; ok {
		s.fetchCollectionMetadata(ctx, client, collection)
	}

	return s.repo.UpsertCollection(ctx, collection)
}

// fetchCollectionMetadata fetches collection metadata from the blockchain
func (s *Service) fetchCollectionMetadata(ctx context.Context, client RPCClient, collection *NFTCollection) {
	// Try to get name
	if name := s.callString(ctx, client, collection.ContractAddress, "0x06fdde03"); name != "" {
		collection.Name = &name
	}

	// Try to get symbol
	if symbol := s.callString(ctx, client, collection.ContractAddress, "0x95d89b41"); symbol != "" {
		collection.Symbol = &symbol
	}

	// Try to get contractURI (ERC-721 Metadata extension)
	if contractURI := s.callString(ctx, client, collection.ContractAddress, "0xe8a3d485"); contractURI != "" {
		collection.ContractURI = &contractURI
	}
}

// callString makes an eth_call and decodes string result
func (s *Service) callString(ctx context.Context, client RPCClient, to, data string) string {
	result, err := client.Call(ctx, "eth_call", map[string]string{
		"to":   to,
		"data": data,
	}, "latest")
	if err != nil {
		return ""
	}

	var hex string
	if err := json.Unmarshal(result, &hex); err != nil {
		return ""
	}

	return decodeABIString(hex)
}

// decodeABIString decodes an ABI-encoded string from hex
func decodeABIString(hex string) string {
	if len(hex) < 2 || hex[:2] != "0x" {
		return ""
	}
	hex = hex[2:]
	if len(hex) < 128 {
		return ""
	}

	// Decode offset
	// offset := new(big.Int).SetBytes(fromHex(hex[:64]))

	// Decode length
	lengthHex := hex[64:128]
	length := 0
	for _, c := range lengthHex {
		length = length*16 + hexValue(c)
	}

	if length == 0 || len(hex) < 128+length*2 {
		return ""
	}

	// Decode string bytes
	strHex := hex[128 : 128+length*2]
	var result []byte
	for i := 0; i < len(strHex); i += 2 {
		b := hexValue(rune(strHex[i]))*16 + hexValue(rune(strHex[i+1]))
		if b == 0 {
			break
		}
		result = append(result, byte(b))
	}

	return string(result)
}

func hexValue(c rune) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c - 'a' + 10)
	case c >= 'A' && c <= 'F':
		return int(c - 'A' + 10)
	default:
		return 0
	}
}

// queueMetadataFetch adds a metadata fetch job to the queue
func (s *Service) queueMetadataFetch(network, contractAddress, tokenID, tokenURI string) {
	select {
	case s.metadataQueue <- metadataJob{
		network:         network,
		contractAddress: contractAddress,
		tokenID:         tokenID,
		tokenURI:        tokenURI,
	}:
	default:
		log.Printf("Metadata queue full, skipping %s/%s/%s", network, contractAddress, tokenID)
	}
}

// metadataWorker processes metadata fetch jobs in the background
func (s *Service) metadataWorker() {
	defer s.wg.Done()

	for job := range s.metadataQueue {
		s.processMetadataJob(job)
	}
}

// processMetadataJob fetches and stores metadata for an NFT
func (s *Service) processMetadataJob(job metadataJob) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Get token URI if not provided
	tokenURI := job.tokenURI
	if tokenURI == "" {
		// Try to get from chain
		if client, ok := s.rpcClients[job.network]; ok {
			tokenURI = s.getTokenURI(ctx, client, job.contractAddress, job.tokenID)
		}
	}

	if tokenURI == "" {
		log.Printf("No token URI for %s/%s/%s", job.network, job.contractAddress, job.tokenID)
		return
	}

	// Check cache first
	uriHash := HashURI(tokenURI)
	cached, err := s.repo.GetMetadataCache(ctx, uriHash)
	if err == nil && cached != nil && cached.Error == nil {
		// Use cached metadata
		s.applyMetadata(ctx, job.network, job.contractAddress, job.tokenID, cached.Content)
		return
	}

	// Fetch metadata
	metadata, err := s.metadataFetcher.FetchMetadata(ctx, tokenURI)
	if err != nil {
		log.Printf("Failed to fetch metadata for %s: %v", tokenURI, err)
		// Cache the error
		errStr := err.Error()
		s.repo.UpsertMetadataCache(ctx, &NFTMetadataCache{
			URIHash: uriHash,
			URI:     tokenURI,
			Error:   &errStr,
		})
		return
	}

	// Cache successful fetch
	content, _ := json.Marshal(metadata)
	s.repo.UpsertMetadataCache(ctx, &NFTMetadataCache{
		URIHash:     uriHash,
		URI:         tokenURI,
		Content:     content,
		ContentType: stringPtr("application/json"),
	})

	// Apply to item
	s.applyMetadata(ctx, job.network, job.contractAddress, job.tokenID, content)
}

// getTokenURI fetches the token URI from the contract
func (s *Service) getTokenURI(ctx context.Context, client RPCClient, contractAddress, tokenID string) string {
	// Try ERC-721 tokenURI(uint256)
	// Function selector: 0xc87b56dd
	data := fmt.Sprintf("0xc87b56dd%064s", tokenID)
	uri := s.callString(ctx, client, contractAddress, data)
	if uri != "" {
		return uri
	}

	// Try ERC-1155 uri(uint256)
	// Function selector: 0x0e89341c
	data = fmt.Sprintf("0x0e89341c%064s", tokenID)
	uri = s.callString(ctx, client, contractAddress, data)
	if uri != "" {
		// ERC-1155 might have {id} placeholder
		return s.metadataFetcher.ResolveTokenURI(uri, tokenID)
	}

	return ""
}

// applyMetadata updates an NFT item with fetched metadata
func (s *Service) applyMetadata(ctx context.Context, network, contractAddress, tokenID string, content []byte) {
	var metadata NFTMetadata
	if err := json.Unmarshal(content, &metadata); err != nil {
		log.Printf("Failed to parse metadata: %v", err)
		return
	}

	NormalizeMetadata(&metadata)

	now := time.Now()
	item := &NFTItem{
		Network:           network,
		ContractAddress:   contractAddress,
		TokenID:           tokenID,
		Name:              nilIfEmpty(metadata.Name),
		Description:       nilIfEmpty(metadata.Description),
		ImageURL:          nilIfEmpty(s.metadataFetcher.ResolveImageURL(metadata.Image)),
		AnimationURL:      nilIfEmpty(metadata.AnimationURL),
		ExternalURL:       nilIfEmpty(metadata.ExternalURL),
		BackgroundColor:   nilIfEmpty(metadata.BackgroundColor),
		Metadata:          content,
		MetadataFetchedAt: &now,
	}

	// Parse attributes
	if len(metadata.Attributes) > 0 {
		attrs, _ := json.Marshal(metadata.Attributes)
		item.Attributes = attrs
	}

	if err := s.repo.UpsertItem(ctx, item); err != nil {
		log.Printf("Failed to update item metadata: %v", err)
	}
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func stringPtr(s string) *string {
	return &s
}

// GetCollection retrieves a collection
func (s *Service) GetCollection(ctx context.Context, network, contractAddress string) (*NFTCollection, error) {
	return s.repo.GetCollection(ctx, network, contractAddress)
}

// ListCollections lists collections with pagination
func (s *Service) ListCollections(ctx context.Context, network string, limit, offset int) ([]*NFTCollection, error) {
	page := 1
	if offset > 0 && limit > 0 {
		page = (offset / limit) + 1
	}
	filter := CollectionFilter{
		Network:  network,
		Page:     page,
		PageSize: limit,
	}
	collections, _, err := s.repo.ListCollections(ctx, filter)
	return collections, err
}

// GetItem retrieves an NFT item
func (s *Service) GetItem(ctx context.Context, network, contractAddress, tokenID string) (*NFTItem, error) {
	return s.repo.GetItem(ctx, network, contractAddress, tokenID)
}

// ListItems lists items in a collection
func (s *Service) ListItems(ctx context.Context, network, contractAddress string, limit, offset int) ([]*NFTItem, error) {
	page := 1
	if offset > 0 && limit > 0 {
		page = (offset / limit) + 1
	}
	filter := ItemFilter{
		Network:         network,
		ContractAddress: &contractAddress,
		Page:            page,
		PageSize:        limit,
	}
	items, _, err := s.repo.ListItems(ctx, filter)
	return items, err
}

// ListTransfers lists transfers with pagination
func (s *Service) ListTransfers(ctx context.Context, network, contractAddress string, tokenID *string, limit, offset int) ([]*NFTTransfer, error) {
	page := 1
	if offset > 0 && limit > 0 {
		page = (offset / limit) + 1
	}
	filter := TransferFilter{
		Network:         network,
		ContractAddress: &contractAddress,
		TokenID:         tokenID,
		Page:            page,
		PageSize:        limit,
	}
	transfers, _, err := s.repo.ListTransfers(ctx, filter)
	return transfers, err
}

// GetAddressNFTs retrieves all NFTs owned by an address
func (s *Service) GetAddressNFTs(ctx context.Context, network, address string, limit, offset int) ([]*NFTItem, error) {
	page := 1
	if offset > 0 && limit > 0 {
		page = (offset / limit) + 1
	}
	items, _, err := s.repo.GetAddressNFTs(ctx, network, address, page, limit)
	return items, err
}

// GetAddressTransfers retrieves NFT transfers for an address
func (s *Service) GetAddressTransfers(ctx context.Context, network, address string, limit, offset int) ([]*NFTTransfer, error) {
	page := 1
	if offset > 0 && limit > 0 {
		page = (offset / limit) + 1
	}
	transfers, _, err := s.repo.GetAddressTransfers(ctx, network, address, page, limit)
	return transfers, err
}

// RefreshMetadata forces a metadata refresh for an NFT
func (s *Service) RefreshMetadata(ctx context.Context, network, contractAddress, tokenID string) error {
	// Get current item to find token URI
	item, err := s.repo.GetItem(ctx, network, contractAddress, tokenID)
	if err != nil {
		return fmt.Errorf("get item: %w", err)
	}

	tokenURI := ""
	if item != nil && item.TokenURI != nil {
		tokenURI = *item.TokenURI
	}

	s.queueMetadataFetch(network, contractAddress, tokenID, tokenURI)
	return nil
}

// UpdateCollectionStats updates statistics for a collection
func (s *Service) UpdateCollectionStats(ctx context.Context, network, contractAddress string) error {
	return s.repo.UpdateCollectionStats(ctx, network, contractAddress)
}

// SearchCollections searches collections by name
func (s *Service) SearchCollections(ctx context.Context, network, query string, limit int) ([]*NFTCollection, error) {
	return s.repo.SearchCollections(ctx, network, query, limit)
}

// GetTokenHolders returns holders of an ERC-1155 token
func (s *Service) GetTokenHolders(ctx context.Context, network, contractAddress, tokenID string, limit, offset int) ([]*NFTBalance, error) {
	page := 1
	if offset > 0 && limit > 0 {
		page = (offset / limit) + 1
	}
	holders, _, err := s.repo.GetTokenHolders(ctx, network, contractAddress, tokenID, page, limit)
	return holders, err
}

// GetCollectionHolders returns all holders of a collection
func (s *Service) GetCollectionHolders(ctx context.Context, network, contractAddress string, limit, offset int) ([]*NFTBalance, error) {
	page := 1
	if offset > 0 && limit > 0 {
		page = (offset / limit) + 1
	}
	holders, _, err := s.repo.GetCollectionHolders(ctx, network, contractAddress, page, limit)
	return holders, err
}
