package tokens

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/sha3"
)

// ERC20Client provides methods to interact with ERC-20 contracts
type ERC20Client struct {
	rpcURLs    map[string]string
	httpClient *http.Client
}

// NewERC20Client creates a new ERC-20 client
func NewERC20Client(rpcURLs map[string]string) *ERC20Client {
	return &ERC20Client{
		rpcURLs: rpcURLs,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// WithHTTPClient sets a custom HTTP client
func (c *ERC20Client) WithHTTPClient(client *http.Client) *ERC20Client {
	c.httpClient = client
	return c
}

// ERC-20 function selectors
var (
	// name() returns (string)
	nameSelector = computeSelector("name()")
	// symbol() returns (string)
	symbolSelector = computeSelector("symbol()")
	// decimals() returns (uint8)
	decimalsSelector = computeSelector("decimals()")
	// totalSupply() returns (uint256)
	totalSupplySelector = computeSelector("totalSupply()")
	// balanceOf(address) returns (uint256)
	balanceOfSelector = computeSelector("balanceOf(address)")
	// allowance(address,address) returns (uint256)
	allowanceSelector = computeSelector("allowance(address,address)")
)

func computeSelector(signature string) string {
	hash := sha3.NewLegacyKeccak256()
	hash.Write([]byte(signature))
	return "0x" + hex.EncodeToString(hash.Sum(nil)[:4])
}

// TokenMetadata contains basic token information
type TokenMetadata struct {
	Name        string
	Symbol      string
	Decimals    int
	TotalSupply *big.Int
}

// GetTokenMetadata fetches token metadata from the contract
func (c *ERC20Client) GetTokenMetadata(ctx context.Context, network, address string) (*TokenMetadata, error) {
	rpcURL, ok := c.rpcURLs[network]
	if !ok {
		return nil, fmt.Errorf("network '%s' not configured", network)
	}

	metadata := &TokenMetadata{}

	// Fetch name
	nameResult, err := c.ethCall(ctx, rpcURL, address, nameSelector)
	if err == nil && len(nameResult) > 2 {
		metadata.Name = decodeString(nameResult)
	}

	// Fetch symbol
	symbolResult, err := c.ethCall(ctx, rpcURL, address, symbolSelector)
	if err == nil && len(symbolResult) > 2 {
		metadata.Symbol = decodeString(symbolResult)
	}

	// Fetch decimals
	decimalsResult, err := c.ethCall(ctx, rpcURL, address, decimalsSelector)
	if err == nil && len(decimalsResult) > 2 {
		metadata.Decimals = decodeUint8(decimalsResult)
	} else {
		metadata.Decimals = 18 // Default to 18
	}

	// Fetch total supply
	totalSupplyResult, err := c.ethCall(ctx, rpcURL, address, totalSupplySelector)
	if err == nil && len(totalSupplyResult) > 2 {
		metadata.TotalSupply = decodeUint256(totalSupplyResult)
	}

	return metadata, nil
}

// GetBalance fetches the token balance for an address
func (c *ERC20Client) GetBalance(ctx context.Context, network, tokenAddress, holderAddress string) (*big.Int, error) {
	rpcURL, ok := c.rpcURLs[network]
	if !ok {
		return nil, fmt.Errorf("network '%s' not configured", network)
	}

	// Encode balanceOf call
	data := balanceOfSelector + encodeAddress(holderAddress)

	result, err := c.ethCall(ctx, rpcURL, tokenAddress, data)
	if err != nil {
		return nil, err
	}

	return decodeUint256(result), nil
}

// GetAllowance fetches the allowance for owner/spender
func (c *ERC20Client) GetAllowance(ctx context.Context, network, tokenAddress, owner, spender string) (*big.Int, error) {
	rpcURL, ok := c.rpcURLs[network]
	if !ok {
		return nil, fmt.Errorf("network '%s' not configured", network)
	}

	// Encode allowance call
	data := allowanceSelector + encodeAddress(owner) + encodeAddress(spender)

	result, err := c.ethCall(ctx, rpcURL, tokenAddress, data)
	if err != nil {
		return nil, err
	}

	return decodeUint256(result), nil
}

// IsERC20 checks if a contract implements ERC-20 interface
func (c *ERC20Client) IsERC20(ctx context.Context, network, address string) bool {
	rpcURL, ok := c.rpcURLs[network]
	if !ok {
		return false
	}

	// Try to call balanceOf - most reliable ERC-20 check
	data := balanceOfSelector + encodeAddress(ZeroAddress)
	result, err := c.ethCall(ctx, rpcURL, address, data)
	if err != nil {
		return false
	}

	// Should return 32 bytes (uint256)
	return len(result) >= 66 // 0x + 64 hex chars
}

// ethCall makes an eth_call RPC request
func (c *ERC20Client) ethCall(ctx context.Context, rpcURL, to, data string) (string, error) {
	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "eth_call",
		"params": []interface{}{
			map[string]string{
				"to":   to,
				"data": data,
			},
			"latest",
		},
		"id": 1,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rpcURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	var rpcResp struct {
		Result string `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(respBody, &rpcResp); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if rpcResp.Error != nil {
		return "", fmt.Errorf("rpc error: %s", rpcResp.Error.Message)
	}

	return rpcResp.Result, nil
}

// encodeAddress encodes an address as a 32-byte hex string (for ABI encoding)
func encodeAddress(address string) string {
	addr := strings.TrimPrefix(address, "0x")
	// Pad to 32 bytes (64 hex chars)
	return strings.Repeat("0", 64-len(addr)) + strings.ToLower(addr)
}

// decodeString decodes an ABI-encoded string
func decodeString(data string) string {
	data = strings.TrimPrefix(data, "0x")
	if len(data) < 128 { // Need at least offset + length
		// Try as fixed bytes32 string
		return decodeBytes32String(data)
	}

	// Dynamic string: offset (32 bytes) + length (32 bytes) + data
	// Offset is at position 0-64
	// Length is at offset position
	offsetHex := data[0:64]
	offset := new(big.Int)
	offset.SetString(offsetHex, 16)
	offsetInt := int(offset.Int64()) * 2 // Convert bytes to hex chars

	if offsetInt+64 > len(data) {
		return decodeBytes32String(data)
	}

	lengthHex := data[offsetInt : offsetInt+64]
	length := new(big.Int)
	length.SetString(lengthHex, 16)
	lengthInt := int(length.Int64()) * 2 // Convert bytes to hex chars

	if offsetInt+64+lengthInt > len(data) {
		return decodeBytes32String(data)
	}

	stringHex := data[offsetInt+64 : offsetInt+64+lengthInt]
	decoded, err := hex.DecodeString(stringHex)
	if err != nil {
		return ""
	}

	return string(decoded)
}

// decodeBytes32String decodes a fixed bytes32 string (some tokens use this)
func decodeBytes32String(data string) string {
	if len(data) < 64 {
		return ""
	}

	// Take first 32 bytes
	decoded, err := hex.DecodeString(data[:64])
	if err != nil {
		return ""
	}

	// Remove trailing zeros
	result := bytes.TrimRight(decoded, "\x00")
	return string(result)
}

// decodeUint8 decodes an ABI-encoded uint8
func decodeUint8(data string) int {
	data = strings.TrimPrefix(data, "0x")
	if len(data) < 2 {
		return 0
	}

	n := new(big.Int)
	n.SetString(data, 16)
	return int(n.Int64())
}

// decodeUint256 decodes an ABI-encoded uint256
func decodeUint256(data string) *big.Int {
	data = strings.TrimPrefix(data, "0x")
	if len(data) == 0 {
		return big.NewInt(0)
	}

	n := new(big.Int)
	n.SetString(data, 16)
	return n
}

// DetectTokenType attempts to detect the token type (ERC20, ERC721, ERC1155)
func (c *ERC20Client) DetectTokenType(ctx context.Context, network, address string) string {
	rpcURL, ok := c.rpcURLs[network]
	if !ok {
		return ""
	}

	// Check for ERC-165 supportsInterface
	// ERC-721: 0x80ac58cd
	// ERC-1155: 0xd9b67a26

	// Try ERC-721 check
	erc721Check := "0x01ffc9a7" + "80ac58cd" + strings.Repeat("0", 56)
	result, err := c.ethCall(ctx, rpcURL, address, erc721Check)
	if err == nil && len(result) >= 66 {
		if result[len(result)-1] == '1' {
			return TokenTypeERC721
		}
	}

	// Try ERC-1155 check
	erc1155Check := "0x01ffc9a7" + "d9b67a26" + strings.Repeat("0", 56)
	result, err = c.ethCall(ctx, rpcURL, address, erc1155Check)
	if err == nil && len(result) >= 66 {
		if result[len(result)-1] == '1' {
			return TokenTypeERC1155
		}
	}

	// Check if it's ERC-20
	if c.IsERC20(ctx, network, address) {
		return TokenTypeERC20
	}

	return ""
}
