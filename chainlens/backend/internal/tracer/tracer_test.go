package tracer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewTracer(t *testing.T) {
	tracer := NewTracer()
	if tracer == nil {
		t.Fatal("NewTracer returned nil")
	}
	if tracer.httpClient == nil {
		t.Error("httpClient should not be nil")
	}
}

func TestParseHexUint64(t *testing.T) {
	tests := []struct {
		input    string
		expected uint64
	}{
		{"0x1", 1},
		{"0x10", 16},
		{"0xff", 255},
		{"0x100", 256},
		{"0x0", 0},
		{"", 0},
		{"0xf4240", 1000000},
		{"1", 1}, // without 0x prefix
	}

	for _, tt := range tests {
		result := parseHexUint64(tt.input)
		if result != tt.expected {
			t.Errorf("parseHexUint64(%s) = %d, want %d", tt.input, result, tt.expected)
		}
	}
}

func TestGetStatus(t *testing.T) {
	tests := []struct {
		status   uint64
		expected string
	}{
		{1, "success"},
		{0, "failed"},
		{2, "failed"},
	}

	for _, tt := range tests {
		result := getStatus(tt.status)
		if result != tt.expected {
			t.Errorf("getStatus(%d) = %s, want %s", tt.status, result, tt.expected)
		}
	}
}

func TestParseEventLogs(t *testing.T) {
	logs := []logEntry{
		{
			Address:  "0x1234567890abcdef1234567890abcdef12345678",
			Topics:   []string{"0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"},
			Data:     "0x0000000000000000000000000000000000000000000000000de0b6b3a7640000",
			LogIndex: "0x0",
		},
		{
			Address:  "0xabcdef1234567890abcdef1234567890abcdef12",
			Topics:   []string{"0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925"},
			Data:     "0x0000000000000000000000000000000000000000000000000000000000000000",
			LogIndex: "0x1",
		},
	}

	result := parseEventLogs(logs)

	if len(result) != 2 {
		t.Fatalf("Expected 2 logs, got %d", len(result))
	}

	if result[0].Address != logs[0].Address {
		t.Errorf("Address[0] = %s, want %s", result[0].Address, logs[0].Address)
	}
	if result[0].LogIndex != 0 {
		t.Errorf("LogIndex[0] = %d, want 0", result[0].LogIndex)
	}
	if result[1].LogIndex != 1 {
		t.Errorf("LogIndex[1] = %d, want 1", result[1].LogIndex)
	}
}

func TestParseEventLogs_Empty(t *testing.T) {
	result := parseEventLogs(nil)
	if len(result) != 0 {
		t.Errorf("Expected empty result, got %d logs", len(result))
	}
}

func TestParseInternalCalls(t *testing.T) {
	trace := json.RawMessage(`{
		"type": "CALL",
		"from": "0x1111111111111111111111111111111111111111",
		"to": "0x2222222222222222222222222222222222222222",
		"value": "0x0",
		"gas": "0x5208",
		"gasUsed": "0x5208",
		"input": "0x",
		"output": "0x",
		"calls": [
			{
				"type": "DELEGATECALL",
				"from": "0x2222222222222222222222222222222222222222",
				"to": "0x3333333333333333333333333333333333333333",
				"gasUsed": "0x1000",
				"input": "0x1234",
				"output": "0x5678"
			}
		]
	}`)

	result := parseInternalCalls(trace)

	if len(result) != 1 {
		t.Fatalf("Expected 1 call, got %d", len(result))
	}

	call := result[0]
	if call.Type != "call" {
		t.Errorf("Type = %s, want call", call.Type)
	}
	if call.From != "0x1111111111111111111111111111111111111111" {
		t.Errorf("From = %s, want 0x1111...", call.From)
	}
	if call.GasUsed != 21000 {
		t.Errorf("GasUsed = %d, want 21000", call.GasUsed)
	}

	// Check nested calls
	if len(call.Children) != 1 {
		t.Fatalf("Expected 1 child call, got %d", len(call.Children))
	}
	if call.Children[0].Type != "delegatecall" {
		t.Errorf("Child Type = %s, want delegatecall", call.Children[0].Type)
	}
}

func TestParseInternalCalls_Nil(t *testing.T) {
	result := parseInternalCalls(nil)
	if result != nil {
		t.Errorf("Expected nil result for nil trace")
	}
}

func TestParseInternalCalls_InvalidJSON(t *testing.T) {
	result := parseInternalCalls(json.RawMessage(`invalid`))
	if result != nil {
		t.Errorf("Expected nil result for invalid JSON")
	}
}

func TestParseStateChanges(t *testing.T) {
	// parseStateChanges currently returns nil
	result := parseStateChanges(json.RawMessage(`{}`))
	if result != nil {
		t.Errorf("Expected nil result")
	}
}

func TestCalculateGasBreakdown(t *testing.T) {
	result := calculateGasBreakdown(21000, nil)

	if result.Execution != 21000 {
		t.Errorf("Execution = %d, want 21000", result.Execution)
	}
	if result.ByOperation == nil {
		t.Error("ByOperation should not be nil")
	}
}

func TestExtractRevertReason(t *testing.T) {
	tests := []struct {
		name     string
		errMsg   string
		hasError bool
	}{
		{
			name:     "simple revert",
			errMsg:   "execution reverted",
			hasError: true,
		},
		{
			name:     "revert with data",
			errMsg:   "execution reverted: 0x08c379a000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000011496e73756666696369656e742066756e6473000000000000000000000000000000",
			hasError: true,
		},
		{
			name:     "no revert",
			errMsg:   "connection refused",
			hasError: false,
		},
		{
			name:     "empty",
			errMsg:   "",
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractRevertReason(tt.errMsg)
			if tt.hasError && result == nil {
				t.Error("Expected revert reason, got nil")
			}
			if !tt.hasError && result != nil {
				t.Errorf("Expected nil, got %+v", result)
			}
		})
	}
}

func TestExtractRevertReason_WithSelector(t *testing.T) {
	// Error(string) ABI encoded: "test error"
	errMsg := "execution reverted: 0x08c379a000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000005746573743100000000000000000000000000000000000000000000000000000000"
	result := extractRevertReason(errMsg)

	if result == nil {
		t.Fatal("Expected revert reason")
	}
	if result.Selector != "0x08c379a0" {
		t.Errorf("Selector = %s, want 0x08c379a0", result.Selector)
	}
	if result.Message != "test1" {
		t.Errorf("Message = %s, want 'test1'", result.Message)
	}
}

func TestDecodeErrorString(t *testing.T) {
	tests := []struct {
		name     string
		hexData  string
		expected string
	}{
		{
			name:     "simple string",
			hexData:  "00000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000005746573743100000000000000000000000000000000000000000000000000000000",
			expected: "test1",
		},
		{
			name:     "with 0x prefix",
			hexData:  "0x00000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000005746573743100000000000000000000000000000000000000000000000000000000",
			expected: "test1",
		},
		{
			name:     "too short",
			hexData:  "0000",
			expected: "",
		},
		{
			name:     "invalid hex",
			hexData:  "zzzz",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := decodeErrorString(tt.hexData)
			if result != tt.expected {
				t.Errorf("decodeErrorString() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestTracer_TraceTransaction_MockServer(t *testing.T) {
	txHash := "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req rpcRequest
		json.NewDecoder(r.Body).Decode(&req)

		w.Header().Set("Content-Type", "application/json")

		switch req.Method {
		case "eth_getTransactionReceipt":
			json.NewEncoder(w).Encode(rpcResponse{
				JSONRPC: "2.0",
				ID:      1,
				Result: json.RawMessage(`{
					"blockNumber": "0x100",
					"gasUsed": "0x5208",
					"status": "0x1",
					"logs": []
				}`),
			})
		case "eth_getTransactionByHash":
			json.NewEncoder(w).Encode(rpcResponse{
				JSONRPC: "2.0",
				ID:      1,
				Result: json.RawMessage(`{
					"from": "0x1111111111111111111111111111111111111111",
					"to": "0x2222222222222222222222222222222222222222",
					"value": "0xde0b6b3a7640000",
					"gasPrice": "0x3b9aca00",
					"input": "0x"
				}`),
			})
		case "debug_traceTransaction":
			// Return error (not all nodes support this)
			json.NewEncoder(w).Encode(rpcResponse{
				JSONRPC: "2.0",
				ID:      1,
				Error: &rpcError{
					Code:    -32601,
					Message: "Method not found",
				},
			})
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer server.Close()

	tracer := NewTracer()
	result, err := tracer.TraceTransaction(context.Background(), server.URL, txHash)

	if err != nil {
		t.Fatalf("TraceTransaction failed: %v", err)
	}

	if result.TxHash != txHash {
		t.Errorf("TxHash = %s, want %s", result.TxHash, txHash)
	}
	if result.BlockNumber != 256 {
		t.Errorf("BlockNumber = %d, want 256", result.BlockNumber)
	}
	if result.GasUsed != 21000 {
		t.Errorf("GasUsed = %d, want 21000", result.GasUsed)
	}
	if result.Status != "success" {
		t.Errorf("Status = %s, want success", result.Status)
	}
}

func TestTracer_SimulateTransaction_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req rpcRequest
		json.NewDecoder(r.Body).Decode(&req)

		w.Header().Set("Content-Type", "application/json")

		switch req.Method {
		case "eth_call":
			json.NewEncoder(w).Encode(rpcResponse{
				JSONRPC: "2.0",
				ID:      1,
				Result:  json.RawMessage(`"0x0000000000000000000000000000000000000000000000000000000000000001"`),
			})
		case "eth_estimateGas":
			json.NewEncoder(w).Encode(rpcResponse{
				JSONRPC: "2.0",
				ID:      1,
				Result:  json.RawMessage(`"0x5208"`),
			})
		case "debug_traceCall":
			json.NewEncoder(w).Encode(rpcResponse{
				JSONRPC: "2.0",
				ID:      1,
				Error: &rpcError{
					Code:    -32601,
					Message: "Method not found",
				},
			})
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer server.Close()

	tracer := NewTracer()
	result, err := tracer.SimulateTransaction(
		context.Background(),
		server.URL,
		"0x1111111111111111111111111111111111111111",
		"0x2222222222222222222222222222222222222222",
		"0x",
		"0x0",
		100000,
	)

	if err != nil {
		t.Fatalf("SimulateTransaction failed: %v", err)
	}

	if !result.Success {
		t.Error("Expected simulation to succeed")
	}
	if result.GasEstimate != 21000 {
		t.Errorf("GasEstimate = %d, want 21000", result.GasEstimate)
	}
}

func TestTracer_SimulateTransaction_Revert(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rpcResponse{
			JSONRPC: "2.0",
			ID:      1,
			Error: &rpcError{
				Code:    3,
				Message: "execution reverted: 0x08c379a0...",
			},
		})
	}))
	defer server.Close()

	tracer := NewTracer()
	result, err := tracer.SimulateTransaction(
		context.Background(),
		server.URL,
		"0x1111111111111111111111111111111111111111",
		"0x2222222222222222222222222222222222222222",
		"0x",
		"0x0",
		100000,
	)

	if err != nil {
		t.Fatalf("SimulateTransaction should not return error for revert: %v", err)
	}

	if result.Success {
		t.Error("Expected simulation to fail (revert)")
	}
	if result.Revert == nil {
		t.Error("Expected revert reason")
	}
}

func TestTracer_RPCCall_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	tracer := &Tracer{
		httpClient: &http.Client{
			Timeout: 10 * time.Millisecond,
		},
	}

	ctx := context.Background()
	_, err := tracer.rpcCall(ctx, server.URL, "test_method", []interface{}{})

	if err == nil {
		t.Error("Expected timeout error")
	}
}

func TestTracer_RPCCall_InvalidURL(t *testing.T) {
	tracer := NewTracer()
	_, err := tracer.rpcCall(context.Background(), "invalid-url", "test", nil)

	if err == nil {
		t.Error("Expected error for invalid URL")
	}
}

func TestTracer_RPCCall_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer server.Close()

	tracer := NewTracer()
	_, err := tracer.rpcCall(context.Background(), server.URL, "test", nil)

	if err == nil {
		t.Error("Expected error for invalid JSON response")
	}
}

func TestTraceResult_Fields(t *testing.T) {
	result := TraceResult{
		TxHash:      "0x123",
		Chain:       "ethereum",
		BlockNumber: 12345678,
		From:        "0x111",
		To:          "0x222",
		Value:       "1000000000000000000",
		GasUsed:     21000,
		GasPrice:    "20000000000",
		Status:      "success",
		Calls:       []InternalCall{},
		Logs:        []EventLog{},
	}

	if result.TxHash != "0x123" {
		t.Error("TxHash mismatch")
	}
	if result.BlockNumber != 12345678 {
		t.Error("BlockNumber mismatch")
	}
}

func TestSimulationResult_Fields(t *testing.T) {
	result := SimulationResult{
		Success:     true,
		GasUsed:     50000,
		GasEstimate: 55000,
		ReturnData:  "0x0000000000000000000000000000000000000000000000000000000000000001",
	}

	if !result.Success {
		t.Error("Success should be true")
	}
	if result.GasEstimate != 55000 {
		t.Error("GasEstimate mismatch")
	}
}

func TestInternalCall_Fields(t *testing.T) {
	call := InternalCall{
		Type:    "call",
		From:    "0x111",
		To:      "0x222",
		Value:   "0",
		GasUsed: 1000,
		Input:   "0x1234",
		Output:  "0x5678",
		Depth:   0,
		Children: []InternalCall{
			{Type: "staticcall", Depth: 1},
		},
	}

	if call.Type != "call" {
		t.Error("Type mismatch")
	}
	if len(call.Children) != 1 {
		t.Error("Children count mismatch")
	}
}

func TestEventLog_Fields(t *testing.T) {
	log := EventLog{
		Address:  "0x123",
		Topics:   []string{"0xabc", "0xdef"},
		Data:     "0x1234",
		LogIndex: 5,
		Decoded: &DecodedEvent{
			Name:   "Transfer",
			Params: map[string]interface{}{"from": "0x111", "to": "0x222"},
		},
	}

	if log.LogIndex != 5 {
		t.Error("LogIndex mismatch")
	}
	if log.Decoded.Name != "Transfer" {
		t.Error("Decoded event name mismatch")
	}
}

func TestGasBreakdown_Fields(t *testing.T) {
	breakdown := GasBreakdown{
		Intrinsic:   21000,
		Execution:   5000,
		Storage:     20000,
		Refund:      1000,
		ByOperation: map[string]uint64{"SSTORE": 20000, "ADD": 3},
	}

	if breakdown.Intrinsic != 21000 {
		t.Error("Intrinsic mismatch")
	}
	if breakdown.ByOperation["SSTORE"] != 20000 {
		t.Error("ByOperation SSTORE mismatch")
	}
}

func TestRevertReason_Fields(t *testing.T) {
	reason := RevertReason{
		Message:  "Insufficient balance",
		Selector: "0x08c379a0",
		Params:   "0x...",
	}

	if reason.Message != "Insufficient balance" {
		t.Error("Message mismatch")
	}
}

// Benchmarks

func BenchmarkParseHexUint64(b *testing.B) {
	for i := 0; i < b.N; i++ {
		parseHexUint64("0xf4240")
	}
}

func BenchmarkParseEventLogs(b *testing.B) {
	logs := []logEntry{
		{Address: "0x123", Topics: []string{"0xabc"}, Data: "0x", LogIndex: "0x0"},
		{Address: "0x456", Topics: []string{"0xdef"}, Data: "0x", LogIndex: "0x1"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parseEventLogs(logs)
	}
}

func BenchmarkParseInternalCalls(b *testing.B) {
	trace := json.RawMessage(`{"type":"CALL","from":"0x111","to":"0x222","gasUsed":"0x5208"}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parseInternalCalls(trace)
	}
}
