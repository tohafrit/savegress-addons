package verification

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

// ContractCaller provides functionality to read from and encode writes to contracts
type ContractCaller struct {
	rpcURLs    map[string]string
	httpClient *http.Client
	service    *Service
}

// NewContractCaller creates a new contract caller
func NewContractCaller(rpcURLs map[string]string, service *Service) *ContractCaller {
	return &ContractCaller{
		rpcURLs: rpcURLs,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		service: service,
	}
}

// WithHTTPClient sets a custom HTTP client
func (c *ContractCaller) WithHTTPClient(client *http.Client) *ContractCaller {
	c.httpClient = client
	return c
}

// ReadContract reads data from a contract using eth_call
func (c *ContractCaller) ReadContract(ctx context.Context, req *ReadContractRequest) (*ReadContractResponse, error) {
	// Get contract ABI
	abi, err := c.service.GetContractABI(ctx, req.Network, req.Address)
	if err != nil {
		return nil, fmt.Errorf("get abi: %w", err)
	}

	if abi == nil {
		return nil, fmt.Errorf("contract not verified, ABI not available")
	}

	// Parse ABI and find the function
	var abiItems []ABIItem
	if err := json.Unmarshal(abi, &abiItems); err != nil {
		return nil, fmt.Errorf("parse abi: %w", err)
	}

	var targetFunc *ABIItem
	for _, item := range abiItems {
		if item.Type == "function" && item.Name == req.Function {
			targetFunc = &item
			break
		}
	}

	if targetFunc == nil {
		return nil, fmt.Errorf("function '%s' not found in ABI", req.Function)
	}

	// Encode the function call
	data, err := c.encodeFunction(targetFunc, req.Args)
	if err != nil {
		return nil, fmt.Errorf("encode function: %w", err)
	}

	// Make eth_call
	rpcURL, ok := c.rpcURLs[req.Network]
	if !ok {
		return nil, fmt.Errorf("network '%s' not configured", req.Network)
	}

	result, err := c.ethCall(ctx, rpcURL, req.Address, data)
	if err != nil {
		return nil, fmt.Errorf("eth_call: %w", err)
	}

	// Decode the result
	decoded, err := c.decodeOutput(targetFunc.Outputs, result)
	if err != nil {
		return nil, fmt.Errorf("decode output: %w", err)
	}

	return &ReadContractResponse{
		Result: decoded,
		Raw:    result,
	}, nil
}

// EncodeWriteTransaction encodes a transaction for a write function
func (c *ContractCaller) EncodeWriteTransaction(ctx context.Context, req *WriteContractRequest) (*WriteContractResponse, error) {
	// Get contract ABI
	abi, err := c.service.GetContractABI(ctx, req.Network, req.Address)
	if err != nil {
		return nil, fmt.Errorf("get abi: %w", err)
	}

	if abi == nil {
		return nil, fmt.Errorf("contract not verified, ABI not available")
	}

	// Parse ABI and find the function
	var abiItems []ABIItem
	if err := json.Unmarshal(abi, &abiItems); err != nil {
		return nil, fmt.Errorf("parse abi: %w", err)
	}

	var targetFunc *ABIItem
	for _, item := range abiItems {
		if item.Type == "function" && item.Name == req.Function {
			targetFunc = &item
			break
		}
	}

	if targetFunc == nil {
		return nil, fmt.Errorf("function '%s' not found in ABI", req.Function)
	}

	// Encode the function call
	data, err := c.encodeFunction(targetFunc, req.Args)
	if err != nil {
		return nil, fmt.Errorf("encode function: %w", err)
	}

	resp := &WriteContractResponse{
		To:   req.Address,
		Data: data,
	}

	if req.Value != "" {
		resp.Value = req.Value
	}

	return resp, nil
}

// encodeFunction encodes a function call with arguments
func (c *ContractCaller) encodeFunction(fn *ABIItem, args []interface{}) (string, error) {
	// Compute function selector
	signature := c.buildSignature(fn.Name, fn.Inputs)
	selector := computeSelector(signature)

	if len(args) != len(fn.Inputs) {
		return "", fmt.Errorf("expected %d arguments, got %d", len(fn.Inputs), len(args))
	}

	// Encode arguments
	encoded, err := c.encodeArguments(fn.Inputs, args)
	if err != nil {
		return "", err
	}

	return "0x" + hex.EncodeToString(selector) + encoded, nil
}

func (c *ContractCaller) buildSignature(name string, inputs []ABIParam) string {
	var types []string
	for _, input := range inputs {
		types = append(types, c.formatType(input))
	}
	return fmt.Sprintf("%s(%s)", name, strings.Join(types, ","))
}

func (c *ContractCaller) formatType(param ABIParam) string {
	if len(param.Components) > 0 {
		// Tuple type
		var compTypes []string
		for _, comp := range param.Components {
			compTypes = append(compTypes, c.formatType(comp))
		}
		return "(" + strings.Join(compTypes, ",") + ")"
	}
	return param.Type
}

func computeSelector(signature string) []byte {
	hash := sha3.NewLegacyKeccak256()
	hash.Write([]byte(signature))
	return hash.Sum(nil)[:4]
}

// encodeArguments encodes function arguments using ABI encoding
func (c *ContractCaller) encodeArguments(inputs []ABIParam, args []interface{}) (string, error) {
	var result []byte

	for i, input := range inputs {
		encoded, err := c.encodeValue(input.Type, args[i])
		if err != nil {
			return "", fmt.Errorf("encode arg %d: %w", i, err)
		}
		result = append(result, encoded...)
	}

	return hex.EncodeToString(result), nil
}

// encodeValue encodes a single value according to its ABI type
func (c *ContractCaller) encodeValue(typ string, value interface{}) ([]byte, error) {
	// Pad to 32 bytes
	result := make([]byte, 32)

	switch {
	case typ == "address":
		addr, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("expected string for address")
		}
		addr = strings.TrimPrefix(addr, "0x")
		decoded, err := hex.DecodeString(addr)
		if err != nil {
			return nil, fmt.Errorf("invalid address: %w", err)
		}
		copy(result[12:], decoded)

	case strings.HasPrefix(typ, "uint") || strings.HasPrefix(typ, "int"):
		var n *big.Int
		switch v := value.(type) {
		case string:
			n = new(big.Int)
			if strings.HasPrefix(v, "0x") {
				n.SetString(v[2:], 16)
			} else {
				n.SetString(v, 10)
			}
		case float64:
			n = big.NewInt(int64(v))
		case int:
			n = big.NewInt(int64(v))
		case int64:
			n = big.NewInt(v)
		case *big.Int:
			n = v
		default:
			return nil, fmt.Errorf("cannot convert %T to uint", value)
		}
		b := n.Bytes()
		copy(result[32-len(b):], b)

	case typ == "bool":
		b, ok := value.(bool)
		if !ok {
			return nil, fmt.Errorf("expected bool")
		}
		if b {
			result[31] = 1
		}

	case strings.HasPrefix(typ, "bytes"):
		// Fixed bytes
		data, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("expected string for bytes")
		}
		data = strings.TrimPrefix(data, "0x")
		decoded, err := hex.DecodeString(data)
		if err != nil {
			return nil, fmt.Errorf("invalid bytes: %w", err)
		}
		copy(result[:len(decoded)], decoded)

	default:
		return nil, fmt.Errorf("unsupported type: %s", typ)
	}

	return result, nil
}

// ethCall makes an eth_call RPC request
func (c *ContractCaller) ethCall(ctx context.Context, rpcURL, to, data string) (string, error) {
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

// decodeOutput decodes the output of a function call
func (c *ContractCaller) decodeOutput(outputs []ABIParam, data string) (interface{}, error) {
	if len(outputs) == 0 {
		return nil, nil
	}

	data = strings.TrimPrefix(data, "0x")
	if len(data) == 0 {
		return nil, nil
	}

	decoded, err := hex.DecodeString(data)
	if err != nil {
		return nil, fmt.Errorf("decode hex: %w", err)
	}

	// Single return value
	if len(outputs) == 1 {
		return c.decodeValue(outputs[0].Type, decoded, 0)
	}

	// Multiple return values
	result := make(map[string]interface{})
	offset := 0
	for _, output := range outputs {
		val, err := c.decodeValue(output.Type, decoded, offset)
		if err != nil {
			return nil, fmt.Errorf("decode %s: %w", output.Name, err)
		}
		name := output.Name
		if name == "" {
			name = fmt.Sprintf("value%d", offset/32)
		}
		result[name] = val
		offset += 32 // Simple types are 32 bytes
	}

	return result, nil
}

// decodeValue decodes a single value from ABI-encoded data
func (c *ContractCaller) decodeValue(typ string, data []byte, offset int) (interface{}, error) {
	if offset+32 > len(data) {
		return nil, fmt.Errorf("data too short")
	}

	chunk := data[offset : offset+32]

	switch {
	case typ == "address":
		return "0x" + hex.EncodeToString(chunk[12:]), nil

	case strings.HasPrefix(typ, "uint"):
		n := new(big.Int).SetBytes(chunk)
		// Return as string for large numbers
		if n.BitLen() > 53 {
			return n.String(), nil
		}
		return n.Int64(), nil

	case strings.HasPrefix(typ, "int"):
		n := new(big.Int).SetBytes(chunk)
		// Handle signed integers
		if chunk[0]&0x80 != 0 {
			// Negative number
			n.Sub(n, new(big.Int).Lsh(big.NewInt(1), 256))
		}
		if n.BitLen() > 53 {
			return n.String(), nil
		}
		return n.Int64(), nil

	case typ == "bool":
		return chunk[31] != 0, nil

	case typ == "string":
		// Dynamic type - offset is stored in chunk
		strOffset := new(big.Int).SetBytes(chunk).Uint64()
		if int(strOffset)+64 > len(data) {
			return "", nil
		}
		length := new(big.Int).SetBytes(data[strOffset : strOffset+32]).Uint64()
		if int(strOffset)+32+int(length) > len(data) {
			return "", nil
		}
		return string(data[strOffset+32 : strOffset+32+length]), nil

	case strings.HasPrefix(typ, "bytes"):
		if typ == "bytes" {
			// Dynamic bytes
			bytesOffset := new(big.Int).SetBytes(chunk).Uint64()
			if int(bytesOffset)+32 > len(data) {
				return "0x", nil
			}
			length := new(big.Int).SetBytes(data[bytesOffset : bytesOffset+32]).Uint64()
			if int(bytesOffset)+32+int(length) > len(data) {
				return "0x", nil
			}
			return "0x" + hex.EncodeToString(data[bytesOffset+32:bytesOffset+32+length]), nil
		}
		// Fixed bytes
		return "0x" + hex.EncodeToString(chunk), nil

	default:
		// Return raw hex for unsupported types
		return "0x" + hex.EncodeToString(chunk), nil
	}
}

// EstimateGas estimates gas for a transaction
func (c *ContractCaller) EstimateGas(ctx context.Context, network, to, data, value string) (string, error) {
	rpcURL, ok := c.rpcURLs[network]
	if !ok {
		return "", fmt.Errorf("network '%s' not configured", network)
	}

	params := map[string]string{
		"to":   to,
		"data": data,
	}
	if value != "" {
		params["value"] = value
	}

	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "eth_estimateGas",
		"params":  []interface{}{params, "latest"},
		"id":      1,
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

	var rpcResp struct {
		Result string `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if rpcResp.Error != nil {
		return "", fmt.Errorf("rpc error: %s", rpcResp.Error.Message)
	}

	return rpcResp.Result, nil
}
