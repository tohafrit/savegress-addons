// Package nfts provides NFT (ERC-721, ERC-1155) tracking functionality
package nfts

import (
	"encoding/json"
	"math/big"
	"time"
)

// NFTCollection represents an NFT collection contract
type NFTCollection struct {
	ID               int64           `json:"-" db:"id"`
	Network          string          `json:"network" db:"network"`
	ContractAddress  string          `json:"contractAddress" db:"contract_address"`
	Name             *string         `json:"name,omitempty" db:"name"`
	Symbol           *string         `json:"symbol,omitempty" db:"symbol"`
	Standard         string          `json:"standard" db:"standard"` // ERC721, ERC1155
	Description      *string         `json:"description,omitempty" db:"description"`
	TotalSupply      *int64          `json:"totalSupply,omitempty" db:"total_supply"`
	OwnerCount       int64           `json:"ownerCount" db:"owner_count"`
	TransferCount    int64           `json:"transferCount" db:"transfer_count"`
	FloorPrice       *string         `json:"floorPrice,omitempty" db:"floor_price"`
	VolumeTotal      *string         `json:"volumeTotal,omitempty" db:"volume_total"`
	BaseURI          *string         `json:"baseUri,omitempty" db:"base_uri"`
	ContractURI      *string         `json:"contractUri,omitempty" db:"contract_uri"`
	Website          *string         `json:"website,omitempty" db:"website"`
	Twitter          *string         `json:"twitter,omitempty" db:"twitter"`
	Discord          *string         `json:"discord,omitempty" db:"discord"`
	OpenseaSlug      *string         `json:"openseaSlug,omitempty" db:"opensea_slug"`
	ImageURL         *string         `json:"imageUrl,omitempty" db:"image_url"`
	BannerURL        *string         `json:"bannerUrl,omitempty" db:"banner_url"`
	RoyaltyRecipient *string         `json:"royaltyRecipient,omitempty" db:"royalty_recipient"`
	RoyaltyBPS       *int            `json:"royaltyBps,omitempty" db:"royalty_bps"`
	IsVerified       bool            `json:"isVerified" db:"is_verified"`
	IsSpam           bool            `json:"isSpam" db:"is_spam"`
	SupportsEIP2981  bool            `json:"supportsEip2981" db:"supports_eip2981"`
	DeployerAddress  *string         `json:"deployerAddress,omitempty" db:"deployer_address"`
	DeployBlock      *int64          `json:"deployBlock,omitempty" db:"deploy_block"`
	DeployTxHash     *string         `json:"deployTxHash,omitempty" db:"deploy_tx_hash"`
	Metadata         json.RawMessage `json:"metadata,omitempty" db:"metadata"`
	CreatedAt        time.Time       `json:"-" db:"created_at"`
	UpdatedAt        time.Time       `json:"-" db:"updated_at"`
}

// NFTItem represents an individual NFT token
type NFTItem struct {
	ID                int64           `json:"-" db:"id"`
	Network           string          `json:"network" db:"network"`
	ContractAddress   string          `json:"contractAddress" db:"contract_address"`
	TokenID           string          `json:"tokenId" db:"token_id"`
	OwnerAddress      *string         `json:"owner,omitempty" db:"owner_address"`
	TokenURI          *string         `json:"tokenUri,omitempty" db:"token_uri"`
	Metadata          json.RawMessage `json:"metadata,omitempty" db:"metadata"`
	MetadataFetchedAt *time.Time      `json:"-" db:"metadata_fetched_at"`
	MetadataError     *string         `json:"-" db:"metadata_error"`
	Name              *string         `json:"name,omitempty" db:"name"`
	Description       *string         `json:"description,omitempty" db:"description"`
	ImageURL          *string         `json:"imageUrl,omitempty" db:"image_url"`
	AnimationURL      *string         `json:"animationUrl,omitempty" db:"animation_url"`
	ExternalURL       *string         `json:"externalUrl,omitempty" db:"external_url"`
	BackgroundColor   *string         `json:"backgroundColor,omitempty" db:"background_color"`
	Attributes        json.RawMessage `json:"attributes,omitempty" db:"attributes"`
	RarityScore       *float64        `json:"rarityScore,omitempty" db:"rarity_score"`
	RarityRank        *int            `json:"rarityRank,omitempty" db:"rarity_rank"`
	TransferCount     int             `json:"transferCount" db:"transfer_count"`
	LastSalePrice     *string         `json:"lastSalePrice,omitempty" db:"last_sale_price"`
	LastSaleCurrency  *string         `json:"lastSaleCurrency,omitempty" db:"last_sale_currency"`
	LastSaleAt        *time.Time      `json:"lastSaleAt,omitempty" db:"last_sale_at"`
	TotalSupply       *string         `json:"totalSupply,omitempty" db:"total_supply"` // For ERC-1155
	MintedAt          *time.Time      `json:"mintedAt,omitempty" db:"minted_at"`
	LastTransferAt    *time.Time      `json:"lastTransferAt,omitempty" db:"last_transfer_at"`
	BurnedAt          *time.Time      `json:"burnedAt,omitempty" db:"burned_at"`
	CreatedAt         time.Time       `json:"-" db:"created_at"`
	UpdatedAt         time.Time       `json:"-" db:"updated_at"`
}

// NFTBalance represents NFT ownership (for ERC-1155)
type NFTBalance struct {
	ID              int64      `json:"-" db:"id"`
	Network         string     `json:"network" db:"network"`
	ContractAddress string     `json:"contractAddress" db:"contract_address"`
	TokenID         string     `json:"tokenId" db:"token_id"`
	HolderAddress   string     `json:"holder" db:"holder_address"`
	Balance         string     `json:"balance" db:"balance"`
	FirstAcquiredAt *time.Time `json:"firstAcquiredAt,omitempty" db:"first_acquired_at"`
	LastUpdatedAt   time.Time  `json:"-" db:"last_updated_at"`
}

// NFTTransfer represents an NFT transfer event
type NFTTransfer struct {
	ID              int64      `json:"-" db:"id"`
	Network         string     `json:"network" db:"network"`
	TxHash          string     `json:"txHash" db:"tx_hash"`
	LogIndex        int        `json:"logIndex" db:"log_index"`
	BlockNumber     int64      `json:"blockNumber" db:"block_number"`
	ContractAddress string     `json:"contractAddress" db:"contract_address"`
	TokenID         string     `json:"tokenId" db:"token_id"`
	FromAddress     string     `json:"from" db:"from_address"`
	ToAddress       string     `json:"to" db:"to_address"`
	Amount          string     `json:"amount" db:"amount"` // For ERC-1155
	Operator        *string    `json:"operator,omitempty" db:"operator"`
	TransferType    string     `json:"transferType" db:"transfer_type"` // transfer, mint, burn, sale
	SalePrice       *string    `json:"salePrice,omitempty" db:"sale_price"`
	SaleCurrency    *string    `json:"saleCurrency,omitempty" db:"sale_currency"`
	Marketplace     *string    `json:"marketplace,omitempty" db:"marketplace"`
	Timestamp       time.Time  `json:"timestamp" db:"timestamp"`
	CreatedAt       time.Time  `json:"-" db:"created_at"`
}

// NFTApproval represents an NFT approval event
type NFTApproval struct {
	ID              int64     `json:"-" db:"id"`
	Network         string    `json:"network" db:"network"`
	TxHash          string    `json:"txHash" db:"tx_hash"`
	LogIndex        int       `json:"logIndex" db:"log_index"`
	BlockNumber     int64     `json:"blockNumber" db:"block_number"`
	ContractAddress string    `json:"contractAddress" db:"contract_address"`
	TokenID         *string   `json:"tokenId,omitempty" db:"token_id"` // NULL for ApprovalForAll
	OwnerAddress    string    `json:"owner" db:"owner_address"`
	ApprovedAddress string    `json:"approved" db:"approved_address"`
	Approved        bool      `json:"approved" db:"approved"` // For ApprovalForAll
	Timestamp       time.Time `json:"timestamp" db:"timestamp"`
	CreatedAt       time.Time `json:"-" db:"created_at"`
}

// NFTMetadata represents parsed NFT metadata
type NFTMetadata struct {
	Name            string          `json:"name,omitempty"`
	Description     string          `json:"description,omitempty"`
	Image           string          `json:"image,omitempty"`
	ImageURL        string          `json:"image_url,omitempty"`
	AnimationURL    string          `json:"animation_url,omitempty"`
	ExternalURL     string          `json:"external_url,omitempty"`
	BackgroundColor string          `json:"background_color,omitempty"`
	Attributes      []NFTAttribute  `json:"attributes,omitempty"`
	Properties      json.RawMessage `json:"properties,omitempty"`
}

// NFTAttribute represents a single NFT attribute/trait
type NFTAttribute struct {
	TraitType   string      `json:"trait_type,omitempty"`
	Value       interface{} `json:"value"`
	DisplayType string      `json:"display_type,omitempty"`
	MaxValue    interface{} `json:"max_value,omitempty"`
}

// WellKnownNFTCollection represents a pre-defined popular collection
type WellKnownNFTCollection struct {
	ID              int     `json:"-" db:"id"`
	Network         string  `json:"network" db:"network"`
	ContractAddress string  `json:"contractAddress" db:"contract_address"`
	Name            string  `json:"name" db:"name"`
	Symbol          *string `json:"symbol,omitempty" db:"symbol"`
	Standard        string  `json:"standard" db:"standard"`
	ImageURL        *string `json:"imageUrl,omitempty" db:"image_url"`
	OpenseaSlug     *string `json:"openseaSlug,omitempty" db:"opensea_slug"`
	IsVerified      bool    `json:"isVerified" db:"is_verified"`
}

// NFTWithCollection combines NFT item with its collection info
type NFTWithCollection struct {
	Item       *NFTItem       `json:"item"`
	Collection *NFTCollection `json:"collection,omitempty"`
}

// NFTMetadataCache represents cached metadata for an NFT URI
type NFTMetadataCache struct {
	ID          int64           `json:"-" db:"id"`
	URIHash     string          `json:"uriHash" db:"uri_hash"`
	URI         string          `json:"uri" db:"uri"`
	Content     json.RawMessage `json:"content,omitempty" db:"content"`
	ContentType *string         `json:"contentType,omitempty" db:"content_type"`
	FetchedAt   time.Time       `json:"-" db:"fetched_at"`
	Error       *string         `json:"error,omitempty" db:"error"`
	RetryCount  int             `json:"-" db:"retry_count"`
}

// CollectionFilter contains filters for listing collections
type CollectionFilter struct {
	Network   string
	Standard  *string // ERC721, ERC1155
	Query     *string
	Page      int
	PageSize  int
	SortBy    string // owner_count, transfer_count, volume_total
	SortOrder string
}

// ItemFilter contains filters for listing NFT items
type ItemFilter struct {
	Network         string
	ContractAddress *string
	OwnerAddress    *string
	TokenID         *string
	Page            int
	PageSize        int
}

// TransferFilter contains filters for listing NFT transfers
type TransferFilter struct {
	Network         string
	ContractAddress *string
	TokenID         *string
	FromAddress     *string
	ToAddress       *string
	TransferType    *string
	Page            int
	PageSize        int
}

// ListResult is a generic paginated result
type ListResult[T any] struct {
	Items      []T   `json:"items"`
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	PageSize   int   `json:"pageSize"`
	TotalPages int   `json:"totalPages"`
}

// ERC-721 event signatures
const (
	// Transfer(address indexed from, address indexed to, uint256 indexed tokenId)
	ERC721TransferTopic = "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"

	// Approval(address indexed owner, address indexed approved, uint256 indexed tokenId)
	ERC721ApprovalTopic = "0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925"

	// ApprovalForAll(address indexed owner, address indexed operator, bool approved)
	ApprovalForAllTopic = "0x17307eab39ab6107e8899845ad3d59bd9653f200f220920489ca2b5937696c31"
)

// ERC-1155 event signatures
const (
	// TransferSingle(address indexed operator, address indexed from, address indexed to, uint256 id, uint256 value)
	TransferSingleTopic = "0xc3d58168c5ae7397731d063d5bbf3d657854427343f4c083240f7aacaa2d0f62"

	// TransferBatch(address indexed operator, address indexed from, address indexed to, uint256[] ids, uint256[] values)
	TransferBatchTopic = "0x4a39dc06d4c0dbc64b70af90fd698a233a518aa5d07e595d983b8c0526c8f7fb"
)

// NFT Standards
const (
	StandardERC721  = "ERC721"
	StandardERC1155 = "ERC1155"
)

// Transfer types
const (
	TransferTypeTransfer = "transfer"
	TransferTypeMint     = "mint"
	TransferTypeBurn     = "burn"
	TransferTypeSale     = "sale"
)

// Zero address
const ZeroAddress = "0x0000000000000000000000000000000000000000"

// ParseERC721Transfer parses an ERC-721 Transfer event
// Note: data parameter is unused for ERC-721 since all values are in topics
func ParseERC721Transfer(topics []string) (*NFTTransfer, error) {
	if len(topics) < 4 {
		return nil, nil
	}

	if topics[0] != ERC721TransferTopic {
		return nil, nil
	}

	transfer := &NFTTransfer{
		FromAddress: parseAddress(topics[1]),
		ToAddress:   parseAddress(topics[2]),
		TokenID:     parseTokenID(topics[3]),
		Amount:      "1",
	}

	// Determine transfer type
	transfer.TransferType = TransferTypeTransfer
	if transfer.FromAddress == ZeroAddress {
		transfer.TransferType = TransferTypeMint
	} else if transfer.ToAddress == ZeroAddress {
		transfer.TransferType = TransferTypeBurn
	}

	return transfer, nil
}

// ParseERC1155TransferSingle parses an ERC-1155 TransferSingle event
func ParseERC1155TransferSingle(topics []string, data string) (*NFTTransfer, error) {
	if len(topics) < 4 {
		return nil, nil
	}

	if topics[0] != TransferSingleTopic {
		return nil, nil
	}

	operator := parseAddress(topics[1])
	transfer := &NFTTransfer{
		FromAddress: parseAddress(topics[2]),
		ToAddress:   parseAddress(topics[3]),
		Operator:    &operator,
	}

	// Parse id and value from data
	if len(data) >= 130 { // 0x + 64 + 64
		data = data[2:] // Remove 0x
		transfer.TokenID = parseUint256(data[:64])
		transfer.Amount = parseUint256(data[64:128])
	}

	// Determine transfer type
	transfer.TransferType = TransferTypeTransfer
	if transfer.FromAddress == ZeroAddress {
		transfer.TransferType = TransferTypeMint
	} else if transfer.ToAddress == ZeroAddress {
		transfer.TransferType = TransferTypeBurn
	}

	return transfer, nil
}

// ParseERC1155TransferBatch parses an ERC-1155 TransferBatch event
func ParseERC1155TransferBatch(topics []string, data string) ([]*NFTTransfer, error) {
	if len(topics) < 4 {
		return nil, nil
	}

	if topics[0] != TransferBatchTopic {
		return nil, nil
	}

	operator := parseAddress(topics[1])
	from := parseAddress(topics[2])
	to := parseAddress(topics[3])

	transferType := TransferTypeTransfer
	if from == ZeroAddress {
		transferType = TransferTypeMint
	} else if to == ZeroAddress {
		transferType = TransferTypeBurn
	}

	// Parse arrays from data (complex ABI decoding)
	// Format: offset_ids (32) + offset_values (32) + len_ids (32) + ids... + len_values (32) + values...
	if len(data) < 130 {
		return nil, nil
	}

	data = data[2:] // Remove 0x

	// Skip offsets, get to first array
	// This is simplified - full implementation would properly decode dynamic arrays
	ids, values := parseArrays(data)

	var transfers []*NFTTransfer
	for i := 0; i < len(ids) && i < len(values); i++ {
		transfer := &NFTTransfer{
			FromAddress:  from,
			ToAddress:    to,
			TokenID:      ids[i],
			Amount:       values[i],
			Operator:     &operator,
			TransferType: transferType,
		}
		transfers = append(transfers, transfer)
	}

	return transfers, nil
}

// ParseApprovalForAll parses an ApprovalForAll event
func ParseApprovalForAll(topics []string, data string) (*NFTApproval, error) {
	if len(topics) < 3 {
		return nil, nil
	}

	if topics[0] != ApprovalForAllTopic {
		return nil, nil
	}

	approval := &NFTApproval{
		OwnerAddress:    parseAddress(topics[1]),
		ApprovedAddress: parseAddress(topics[2]),
		Approved:        true,
	}

	// Parse approved bool from data
	if len(data) >= 66 {
		approval.Approved = data[len(data)-1] == '1'
	}

	return approval, nil
}

// Helper functions

func parseAddress(topic string) string {
	if len(topic) < 42 {
		return ZeroAddress
	}
	return "0x" + topic[len(topic)-40:]
}

func parseTokenID(topic string) string {
	if len(topic) < 3 {
		return "0"
	}
	topic = topic[2:] // Remove 0x
	n := new(big.Int)
	n.SetString(topic, 16)
	return n.String()
}

func parseUint256(hex string) string {
	n := new(big.Int)
	n.SetString(hex, 16)
	return n.String()
}

func parseArrays(data string) ([]string, []string) {
	// Simplified array parsing
	// In production, you'd want full ABI decoding
	var ids, values []string

	if len(data) < 256 {
		return ids, values
	}

	// Skip to first array length (after two 32-byte offsets)
	offset := 128 // 64 * 2

	if offset+64 > len(data) {
		return ids, values
	}

	lenIdsHex := data[offset : offset+64]
	lenIds := new(big.Int)
	lenIds.SetString(lenIdsHex, 16)
	count := int(lenIds.Int64())

	offset += 64

	// Parse ids
	for i := 0; i < count && offset+64 <= len(data); i++ {
		ids = append(ids, parseUint256(data[offset:offset+64]))
		offset += 64
	}

	// Skip length of values array
	if offset+64 > len(data) {
		return ids, values
	}
	offset += 64

	// Parse values
	for i := 0; i < count && offset+64 <= len(data); i++ {
		values = append(values, parseUint256(data[offset:offset+64]))
		offset += 64
	}

	return ids, values
}

// IsMint checks if the transfer is a mint
func (t *NFTTransfer) IsMint() bool {
	return t.FromAddress == ZeroAddress
}

// IsBurn checks if the transfer is a burn
func (t *NFTTransfer) IsBurn() bool {
	return t.ToAddress == ZeroAddress
}

// GetImageURL returns the best available image URL
func (i *NFTItem) GetImageURL() string {
	if i.ImageURL != nil {
		return *i.ImageURL
	}
	if i.AnimationURL != nil {
		return *i.AnimationURL
	}
	return ""
}
