// Package tracer provides Ethereum transaction tracing and simulation
package tracer

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
)

// TraceResult represents the result of transaction tracing
type TraceResult struct {
	TxHash       string          `json:"tx_hash"`
	Chain        string          `json:"chain"`
	BlockNumber  uint64          `json:"block_number"`
	From         string          `json:"from"`
	To           string          `json:"to"`
	Value        string          `json:"value"`
	GasUsed      uint64          `json:"gas_used"`
	GasPrice     string          `json:"gas_price"`
	Status       string          `json:"status"` // success, failed
	Error        string          `json:"error,omitempty"`
	Calls        []InternalCall  `json:"calls"`
	Logs         []EventLog      `json:"logs"`
	StateChanges []StateChange   `json:"state_changes"`
	GasBreakdown GasBreakdown    `json:"gas_breakdown"`
}

// InternalCall represents an internal call during transaction execution
type InternalCall struct {
	Type     string         `json:"type"` // call, delegatecall, staticcall, create, create2
	From     string         `json:"from"`
	To       string         `json:"to"`
	Value    string         `json:"value"`
	GasUsed  uint64         `json:"gas_used"`
	Input    string         `json:"input"`
	Output   string         `json:"output"`
	Error    string         `json:"error,omitempty"`
	Depth    int            `json:"depth"`
	Children []InternalCall `json:"calls,omitempty"`
}

// EventLog represents an emitted event
type EventLog struct {
	Address     string   `json:"address"`
	Topics      []string `json:"topics"`
	Data        string   `json:"data"`
	Decoded     *DecodedEvent `json:"decoded,omitempty"`
	LogIndex    uint     `json:"log_index"`
}

// DecodedEvent represents a decoded event with human-readable data
type DecodedEvent struct {
	Name   string                 `json:"name"`
	Params map[string]interface{} `json:"params"`
}

// StateChange represents a storage state change
type StateChange struct {
	Address  string `json:"address"`
	Slot     string `json:"slot"`
	OldValue string `json:"old_value"`
	NewValue string `json:"new_value"`
}

// GasBreakdown provides detailed gas usage information
type GasBreakdown struct {
	Intrinsic   uint64            `json:"intrinsic"`
	Execution   uint64            `json:"execution"`
	Storage     uint64            `json:"storage"`
	Refund      uint64            `json:"refund"`
	ByOperation map[string]uint64 `json:"by_operation"`
}

// SimulationResult represents the result of a transaction simulation
type SimulationResult struct {
	Success      bool           `json:"success"`
	GasUsed      uint64         `json:"gas_used"`
	GasEstimate  uint64         `json:"gas_estimate"`
	ReturnData   string         `json:"return_data"`
	Error        string         `json:"error,omitempty"`
	Revert       *RevertReason  `json:"revert,omitempty"`
	Logs         []EventLog     `json:"logs"`
	StateChanges []StateChange  `json:"state_changes"`
	Trace        []InternalCall `json:"trace"`
}

// RevertReason provides decoded revert information
type RevertReason struct {
	Message  string `json:"message"`
	Selector string `json:"selector,omitempty"`
	Params   string `json:"params,omitempty"`
}

// Tracer provides transaction tracing and simulation capabilities
type Tracer struct {
	httpClient *http.Client
}

// NewTracer creates a new Tracer instance
func NewTracer() *Tracer {
	return &Tracer{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// TraceTransaction traces a transaction by its hash
func (t *Tracer) TraceTransaction(ctx context.Context, rpcURL, txHash string) (*TraceResult, error) {
	// First, get the transaction receipt
	receipt, err := t.getTransactionReceipt(ctx, rpcURL, txHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction receipt: %w", err)
	}

	// Get the transaction details
	tx, err := t.getTransaction(ctx, rpcURL, txHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	// Get debug trace if available (using debug_traceTransaction)
	trace, err := t.debugTraceTransaction(ctx, rpcURL, txHash)
	if err != nil {
		// Not all nodes support debug_traceTransaction, continue without detailed trace
		trace = nil
	}

	// Parse state changes if trace is available
	var stateChanges []StateChange
	if trace != nil {
		stateChanges = parseStateChanges(trace)
	}

	result := &TraceResult{
		TxHash:       txHash,
		BlockNumber:  receipt.BlockNumber,
		From:         tx.From,
		To:           tx.To,
		Value:        tx.Value,
		GasUsed:      receipt.GasUsed,
		GasPrice:     tx.GasPrice,
		Status:       getStatus(receipt.Status),
		Logs:         parseEventLogs(receipt.Logs),
		StateChanges: stateChanges,
		GasBreakdown: calculateGasBreakdown(receipt.GasUsed, trace),
	}

	// Parse internal calls from trace
	if trace != nil {
		result.Calls = parseInternalCalls(trace)
	}

	return result, nil
}

// SimulateTransaction simulates a transaction without executing it
func (t *Tracer) SimulateTransaction(ctx context.Context, rpcURL, from, to, data, value string, gasLimit uint64) (*SimulationResult, error) {
	// Use eth_call to simulate
	callResult, err := t.ethCall(ctx, rpcURL, from, to, data, value, gasLimit)
	if err != nil {
		// Check if it's a revert
		if revertErr := extractRevertReason(err.Error()); revertErr != nil {
			return &SimulationResult{
				Success: false,
				Error:   err.Error(),
				Revert:  revertErr,
			}, nil
		}
		return nil, fmt.Errorf("simulation failed: %w", err)
	}

	// Estimate gas
	gasEstimate, err := t.estimateGas(ctx, rpcURL, from, to, data, value)
	if err != nil {
		gasEstimate = gasLimit
	}

	// Trace the call for detailed execution info
	traceResult, _ := t.traceCall(ctx, rpcURL, from, to, data, value, gasLimit)

	result := &SimulationResult{
		Success:     true,
		ReturnData:  callResult,
		GasEstimate: gasEstimate,
	}

	if traceResult != nil {
		result.GasUsed = traceResult.GasUsed
		result.Trace = traceResult.Calls
		result.Logs = traceResult.Logs
		result.StateChanges = traceResult.StateChanges
	}

	return result, nil
}

// JSON-RPC types
type rpcRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
}

type transactionReceipt struct {
	BlockNumber uint64       `json:"-"`
	GasUsed     uint64       `json:"-"`
	Status      uint64       `json:"-"`
	Logs        []logEntry   `json:"logs"`
}

type logEntry struct {
	Address  string   `json:"address"`
	Topics   []string `json:"topics"`
	Data     string   `json:"data"`
	LogIndex string   `json:"logIndex"`
}

type transaction struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Value    string `json:"value"`
	GasPrice string `json:"gasPrice"`
	Input    string `json:"input"`
}

func (t *Tracer) rpcCall(ctx context.Context, rpcURL, method string, params []interface{}) (json.RawMessage, error) {
	reqBody, err := json.Marshal(rpcRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      1,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", rpcURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var rpcResp rpcResponse
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return nil, err
	}

	if rpcResp.Error != nil {
		return nil, fmt.Errorf("RPC error: %s (code: %d)", rpcResp.Error.Message, rpcResp.Error.Code)
	}

	return rpcResp.Result, nil
}

func (t *Tracer) getTransactionReceipt(ctx context.Context, rpcURL, txHash string) (*transactionReceipt, error) {
	result, err := t.rpcCall(ctx, rpcURL, "eth_getTransactionReceipt", []interface{}{txHash})
	if err != nil {
		return nil, err
	}

	var raw struct {
		BlockNumber string     `json:"blockNumber"`
		GasUsed     string     `json:"gasUsed"`
		Status      string     `json:"status"`
		Logs        []logEntry `json:"logs"`
	}
	if err := json.Unmarshal(result, &raw); err != nil {
		return nil, err
	}

	return &transactionReceipt{
		BlockNumber: parseHexUint64(raw.BlockNumber),
		GasUsed:     parseHexUint64(raw.GasUsed),
		Status:      parseHexUint64(raw.Status),
		Logs:        raw.Logs,
	}, nil
}

func (t *Tracer) getTransaction(ctx context.Context, rpcURL, txHash string) (*transaction, error) {
	result, err := t.rpcCall(ctx, rpcURL, "eth_getTransactionByHash", []interface{}{txHash})
	if err != nil {
		return nil, err
	}

	var tx transaction
	if err := json.Unmarshal(result, &tx); err != nil {
		return nil, err
	}

	return &tx, nil
}

func (t *Tracer) debugTraceTransaction(ctx context.Context, rpcURL, txHash string) (json.RawMessage, error) {
	tracerConfig := map[string]interface{}{
		"tracer": "callTracer",
		"tracerConfig": map[string]interface{}{
			"withLog": true,
		},
	}
	return t.rpcCall(ctx, rpcURL, "debug_traceTransaction", []interface{}{txHash, tracerConfig})
}

func (t *Tracer) ethCall(ctx context.Context, rpcURL, from, to, data, value string, gasLimit uint64) (string, error) {
	callObj := map[string]interface{}{
		"from": from,
		"to":   to,
		"data": data,
	}
	if value != "" && value != "0" && value != "0x0" {
		callObj["value"] = value
	}
	if gasLimit > 0 {
		callObj["gas"] = fmt.Sprintf("0x%x", gasLimit)
	}

	result, err := t.rpcCall(ctx, rpcURL, "eth_call", []interface{}{callObj, "latest"})
	if err != nil {
		return "", err
	}

	var returnData string
	if err := json.Unmarshal(result, &returnData); err != nil {
		return "", err
	}

	return returnData, nil
}

func (t *Tracer) estimateGas(ctx context.Context, rpcURL, from, to, data, value string) (uint64, error) {
	callObj := map[string]interface{}{
		"from": from,
		"to":   to,
		"data": data,
	}
	if value != "" && value != "0" && value != "0x0" {
		callObj["value"] = value
	}

	result, err := t.rpcCall(ctx, rpcURL, "eth_estimateGas", []interface{}{callObj})
	if err != nil {
		return 0, err
	}

	var gasHex string
	if err := json.Unmarshal(result, &gasHex); err != nil {
		return 0, err
	}

	return parseHexUint64(gasHex), nil
}

func (t *Tracer) traceCall(ctx context.Context, rpcURL, from, to, data, value string, gasLimit uint64) (*TraceResult, error) {
	callObj := map[string]interface{}{
		"from": from,
		"to":   to,
		"data": data,
	}
	if value != "" && value != "0" && value != "0x0" {
		callObj["value"] = value
	}
	if gasLimit > 0 {
		callObj["gas"] = fmt.Sprintf("0x%x", gasLimit)
	}

	tracerConfig := map[string]interface{}{
		"tracer": "callTracer",
		"tracerConfig": map[string]interface{}{
			"withLog": true,
		},
	}

	result, err := t.rpcCall(ctx, rpcURL, "debug_traceCall", []interface{}{callObj, "latest", tracerConfig})
	if err != nil {
		return nil, err
	}

	calls := parseInternalCalls(result)

	return &TraceResult{
		Calls: calls,
	}, nil
}

// Helper functions

func parseHexUint64(s string) uint64 {
	s = strings.TrimPrefix(s, "0x")
	if s == "" {
		return 0
	}
	n := new(big.Int)
	n.SetString(s, 16)
	return n.Uint64()
}

func getStatus(status uint64) string {
	if status == 1 {
		return "success"
	}
	return "failed"
}

func parseEventLogs(logs []logEntry) []EventLog {
	result := make([]EventLog, len(logs))
	for i, log := range logs {
		logIdx := parseHexUint64(log.LogIndex)
		result[i] = EventLog{
			Address:  log.Address,
			Topics:   log.Topics,
			Data:     log.Data,
			LogIndex: uint(logIdx),
		}
	}
	return result
}

func parseStateChanges(trace json.RawMessage) []StateChange {
	// Parse state changes from prestateTracer or similar
	// This requires additional RPC call with different tracer
	return nil
}

func calculateGasBreakdown(gasUsed uint64, trace json.RawMessage) GasBreakdown {
	// Calculate gas breakdown from trace
	return GasBreakdown{
		Execution:   gasUsed,
		ByOperation: make(map[string]uint64),
	}
}

func parseInternalCalls(trace json.RawMessage) []InternalCall {
	if trace == nil {
		return nil
	}

	var rawCall struct {
		Type    string          `json:"type"`
		From    string          `json:"from"`
		To      string          `json:"to"`
		Value   string          `json:"value"`
		Gas     string          `json:"gas"`
		GasUsed string          `json:"gasUsed"`
		Input   string          `json:"input"`
		Output  string          `json:"output"`
		Error   string          `json:"error"`
		Calls   json.RawMessage `json:"calls"`
	}

	if err := json.Unmarshal(trace, &rawCall); err != nil {
		return nil
	}

	call := InternalCall{
		Type:    strings.ToLower(rawCall.Type),
		From:    rawCall.From,
		To:      rawCall.To,
		Value:   rawCall.Value,
		GasUsed: parseHexUint64(rawCall.GasUsed),
		Input:   rawCall.Input,
		Output:  rawCall.Output,
		Error:   rawCall.Error,
		Depth:   0,
	}

	// Parse nested calls
	if rawCall.Calls != nil {
		var nestedCalls []json.RawMessage
		if err := json.Unmarshal(rawCall.Calls, &nestedCalls); err == nil {
			for _, nested := range nestedCalls {
				if childCalls := parseInternalCalls(nested); len(childCalls) > 0 {
					for i := range childCalls {
						childCalls[i].Depth = 1
					}
					call.Children = append(call.Children, childCalls...)
				}
			}
		}
	}

	return []InternalCall{call}
}

func extractRevertReason(errMsg string) *RevertReason {
	// Check for standard revert
	if strings.Contains(errMsg, "execution reverted") {
		reason := &RevertReason{}

		// Try to extract revert message
		if idx := strings.Index(errMsg, "0x"); idx != -1 {
			data := errMsg[idx:]
			if len(data) >= 10 {
				reason.Selector = data[:10]
				if len(data) > 10 {
					reason.Params = data[10:]
					// Try to decode Error(string) - 0x08c379a0
					if reason.Selector == "0x08c379a0" {
						reason.Message = decodeErrorString(reason.Params)
					}
				}
			}
		}

		return reason
	}

	return nil
}

func decodeErrorString(hexData string) string {
	// Decode ABI-encoded string from Error(string)
	data, err := hex.DecodeString(strings.TrimPrefix(hexData, "0x"))
	if err != nil || len(data) < 64 {
		return ""
	}

	// Skip offset (32 bytes) and length (32 bytes)
	length := new(big.Int).SetBytes(data[32:64]).Uint64()
	if len(data) < 64+int(length) {
		return ""
	}

	return string(data[64 : 64+length])
}
