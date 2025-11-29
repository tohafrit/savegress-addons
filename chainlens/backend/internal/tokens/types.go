// Package tokens provides ERC-20 token tracking functionality
package tokens

import (
	"encoding/json"
	"math/big"
	"time"
)

// Token represents an ERC-20 token contract
type Token struct {
	ID              int64           `json:"-" db:"id"`
	Network         string          `json:"network" db:"network"`
	ContractAddress string          `json:"contractAddress" db:"contract_address"`
	Name            *string         `json:"name,omitempty" db:"name"`
	Symbol          *string         `json:"symbol,omitempty" db:"symbol"`
	Decimals        int             `json:"decimals" db:"decimals"`
	TotalSupply     *string         `json:"totalSupply,omitempty" db:"total_supply"`
	HolderCount     int64           `json:"holderCount" db:"holder_count"`
	TransferCount   int64           `json:"transferCount" db:"transfer_count"`
	LogoURL         *string         `json:"logoUrl,omitempty" db:"logo_url"`
	Website         *string         `json:"website,omitempty" db:"website"`
	Description     *string         `json:"description,omitempty" db:"description"`
	SocialLinks     json.RawMessage `json:"socialLinks,omitempty" db:"social_links"`
	IsVerified      bool            `json:"isVerified" db:"is_verified"`
	CoingeckoID     *string         `json:"coingeckoId,omitempty" db:"coingecko_id"`
	CoinmarketcapID *int            `json:"coinmarketcapId,omitempty" db:"coinmarketcap_id"`
	TokenType       string          `json:"tokenType" db:"token_type"`
	IsProxy         bool            `json:"isProxy" db:"is_proxy"`
	Implementation  *string         `json:"implementationAddress,omitempty" db:"implementation_address"`
	FirstBlock      *int64          `json:"firstBlock,omitempty" db:"first_block"`
	FirstTxHash     *string         `json:"firstTxHash,omitempty" db:"first_tx_hash"`
	DeployerAddress *string         `json:"deployerAddress,omitempty" db:"deployer_address"`
	CreatedAt       time.Time       `json:"-" db:"created_at"`
	UpdatedAt       time.Time       `json:"-" db:"updated_at"`
}

// TokenTransfer represents a token transfer event
type TokenTransfer struct {
	ID            int64     `json:"-" db:"id"`
	Network       string    `json:"network" db:"network"`
	TxHash        string    `json:"txHash" db:"tx_hash"`
	LogIndex      int       `json:"logIndex" db:"log_index"`
	BlockNumber   int64     `json:"blockNumber" db:"block_number"`
	TokenAddress  string    `json:"tokenAddress" db:"token_address"`
	FromAddress   string    `json:"from" db:"from_address"`
	ToAddress     string    `json:"to" db:"to_address"`
	Value         string    `json:"value" db:"value"`
	TokenSymbol   *string   `json:"tokenSymbol,omitempty" db:"token_symbol"`
	TokenDecimals *int      `json:"tokenDecimals,omitempty" db:"token_decimals"`
	TokenID       *string   `json:"tokenId,omitempty" db:"token_id"` // For ERC-1155
	Timestamp     time.Time `json:"timestamp" db:"timestamp"`
	CreatedAt     time.Time `json:"-" db:"created_at"`
}

// TokenBalance represents a holder's balance of a specific token
type TokenBalance struct {
	ID              int64     `json:"-" db:"id"`
	Network         string    `json:"network" db:"network"`
	TokenAddress    string    `json:"tokenAddress" db:"token_address"`
	HolderAddress   string    `json:"holderAddress" db:"holder_address"`
	Balance         string    `json:"balance" db:"balance"`
	FirstTransferAt *time.Time `json:"firstTransferAt,omitempty" db:"first_transfer_at"`
	LastTransferAt  *time.Time `json:"lastTransferAt,omitempty" db:"last_transfer_at"`
	TransferCount   int       `json:"transferCount" db:"transfer_count"`
	UpdatedAt       time.Time `json:"-" db:"updated_at"`
}

// TokenApproval represents a token approval event
type TokenApproval struct {
	ID             int64     `json:"-" db:"id"`
	Network        string    `json:"network" db:"network"`
	TxHash         string    `json:"txHash" db:"tx_hash"`
	LogIndex       int       `json:"logIndex" db:"log_index"`
	BlockNumber    int64     `json:"blockNumber" db:"block_number"`
	TokenAddress   string    `json:"tokenAddress" db:"token_address"`
	OwnerAddress   string    `json:"owner" db:"owner_address"`
	SpenderAddress string    `json:"spender" db:"spender_address"`
	Value          string    `json:"value" db:"value"`
	Timestamp      time.Time `json:"timestamp" db:"timestamp"`
	CreatedAt      time.Time `json:"-" db:"created_at"`
}

// TokenAllowance represents current allowance state
type TokenAllowance struct {
	ID               int64     `json:"-" db:"id"`
	Network          string    `json:"network" db:"network"`
	TokenAddress     string    `json:"tokenAddress" db:"token_address"`
	OwnerAddress     string    `json:"owner" db:"owner_address"`
	SpenderAddress   string    `json:"spender" db:"spender_address"`
	Allowance        string    `json:"allowance" db:"allowance"`
	LastUpdatedBlock *int64    `json:"lastUpdatedBlock,omitempty" db:"last_updated_block"`
	UpdatedAt        time.Time `json:"-" db:"updated_at"`
}

// TokenPrice represents token price at a point in time
type TokenPrice struct {
	ID            int64     `json:"-" db:"id"`
	Network       string    `json:"network" db:"network"`
	TokenAddress  string    `json:"tokenAddress" db:"token_address"`
	Timestamp     time.Time `json:"timestamp" db:"timestamp"`
	PriceUSD      *string   `json:"priceUsd,omitempty" db:"price_usd"`
	PriceETH      *string   `json:"priceEth,omitempty" db:"price_eth"`
	MarketCapUSD  *string   `json:"marketCapUsd,omitempty" db:"market_cap_usd"`
	Volume24hUSD  *string   `json:"volume24hUsd,omitempty" db:"volume_24h_usd"`
	Source        *string   `json:"source,omitempty" db:"source"`
}

// WellKnownToken represents a pre-defined popular token
type WellKnownToken struct {
	ID               int     `json:"-" db:"id"`
	Network          string  `json:"network" db:"network"`
	ContractAddress  string  `json:"contractAddress" db:"contract_address"`
	Name             string  `json:"name" db:"name"`
	Symbol           string  `json:"symbol" db:"symbol"`
	Decimals         int     `json:"decimals" db:"decimals"`
	LogoURL          *string `json:"logoUrl,omitempty" db:"logo_url"`
	CoingeckoID      *string `json:"coingeckoId,omitempty" db:"coingecko_id"`
	IsStablecoin     bool    `json:"isStablecoin" db:"is_stablecoin"`
	IsWrappedNative  bool    `json:"isWrappedNative" db:"is_wrapped_native"`
}

// TokenWithBalance combines token info with a specific holder's balance
type TokenWithBalance struct {
	Token   *Token  `json:"token"`
	Balance string  `json:"balance"`
	// Formatted balance using token decimals
	FormattedBalance string `json:"formattedBalance,omitempty"`
}

// TokenHolder represents a token holder for the holders list
type TokenHolder struct {
	Address    string  `json:"address"`
	Balance    string  `json:"balance"`
	Percentage float64 `json:"percentage"` // Percentage of total supply
	Rank       int     `json:"rank"`
}

// TransferFilter contains filters for listing transfers
type TransferFilter struct {
	Network      string
	TokenAddress *string
	FromAddress  *string
	ToAddress    *string
	BlockNumber  *int64
	Page         int
	PageSize     int
}

// TokenFilter contains filters for listing tokens
type TokenFilter struct {
	Network   string
	TokenType *string
	Query     *string // Search by name or symbol
	Page      int
	PageSize  int
	SortBy    string // holder_count, transfer_count, name
	SortOrder string // asc, desc
}

// ListResult is a generic paginated result
type ListResult[T any] struct {
	Items      []T   `json:"items"`
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	PageSize   int   `json:"pageSize"`
	TotalPages int   `json:"totalPages"`
}

// ERC-20 event signatures
const (
	// Transfer(address indexed from, address indexed to, uint256 value)
	TransferEventTopic = "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"

	// Approval(address indexed owner, address indexed spender, uint256 value)
	ApprovalEventTopic = "0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925"
)

// Token types
const (
	TokenTypeERC20   = "ERC20"
	TokenTypeERC721  = "ERC721"
	TokenTypeERC1155 = "ERC1155"
)

// Zero address (burn/mint detection)
const ZeroAddress = "0x0000000000000000000000000000000000000000"

// MaxUint256 for unlimited approvals
var MaxUint256 = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))

// ParseTransferEvent parses a Transfer event from log data
func ParseTransferEvent(topics []string, data string) (*TokenTransfer, error) {
	if len(topics) < 3 {
		return nil, nil // Not a valid Transfer event
	}

	if topics[0] != TransferEventTopic {
		return nil, nil // Not a Transfer event
	}

	transfer := &TokenTransfer{
		FromAddress: parseAddress(topics[1]),
		ToAddress:   parseAddress(topics[2]),
	}

	// Parse value from data
	if len(data) >= 66 { // 0x + 64 hex chars
		value := new(big.Int)
		value.SetString(data[2:], 16)
		transfer.Value = value.String()
	} else {
		transfer.Value = "0"
	}

	return transfer, nil
}

// ParseApprovalEvent parses an Approval event from log data
func ParseApprovalEvent(topics []string, data string) (*TokenApproval, error) {
	if len(topics) < 3 {
		return nil, nil
	}

	if topics[0] != ApprovalEventTopic {
		return nil, nil
	}

	approval := &TokenApproval{
		OwnerAddress:   parseAddress(topics[1]),
		SpenderAddress: parseAddress(topics[2]),
	}

	// Parse value from data
	if len(data) >= 66 {
		value := new(big.Int)
		value.SetString(data[2:], 16)
		approval.Value = value.String()
	} else {
		approval.Value = "0"
	}

	return approval, nil
}

// parseAddress extracts address from a 32-byte topic
func parseAddress(topic string) string {
	if len(topic) < 42 {
		return ZeroAddress
	}
	// Topic is 66 chars (0x + 64 hex), address is last 40 chars
	return "0x" + topic[len(topic)-40:]
}

// IsMint checks if the transfer is a mint (from zero address)
func (t *TokenTransfer) IsMint() bool {
	return t.FromAddress == ZeroAddress
}

// IsBurn checks if the transfer is a burn (to zero address)
func (t *TokenTransfer) IsBurn() bool {
	return t.ToAddress == ZeroAddress
}

// IsUnlimitedApproval checks if the approval is for unlimited amount
func (a *TokenApproval) IsUnlimitedApproval() bool {
	value := new(big.Int)
	value.SetString(a.Value, 10)
	return value.Cmp(MaxUint256) == 0
}

// FormatBalance formats a token balance with decimals
func FormatBalance(balance string, decimals int) string {
	if decimals == 0 {
		return balance
	}

	value := new(big.Int)
	value.SetString(balance, 10)

	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	whole := new(big.Int).Div(value, divisor)
	remainder := new(big.Int).Mod(value, divisor)

	if remainder.Sign() == 0 {
		return whole.String()
	}

	// Format with decimal places
	remainderStr := remainder.String()
	// Pad with leading zeros
	for len(remainderStr) < decimals {
		remainderStr = "0" + remainderStr
	}
	// Trim trailing zeros
	for len(remainderStr) > 0 && remainderStr[len(remainderStr)-1] == '0' {
		remainderStr = remainderStr[:len(remainderStr)-1]
	}

	if len(remainderStr) == 0 {
		return whole.String()
	}

	return whole.String() + "." + remainderStr
}

// ParseBalance parses a formatted balance string back to raw value
func ParseBalance(formatted string, decimals int) string {
	// Simple implementation - handle decimal point
	parts := splitBalance(formatted)
	if len(parts) == 1 {
		// No decimal point
		value := new(big.Int)
		value.SetString(parts[0], 10)
		multiplier := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
		value.Mul(value, multiplier)
		return value.String()
	}

	whole := parts[0]
	frac := parts[1]

	// Pad or truncate fractional part
	for len(frac) < decimals {
		frac += "0"
	}
	frac = frac[:decimals]

	combined := whole + frac
	value := new(big.Int)
	value.SetString(combined, 10)
	return value.String()
}

func splitBalance(s string) []string {
	for i, c := range s {
		if c == '.' {
			return []string{s[:i], s[i+1:]}
		}
	}
	return []string{s}
}
