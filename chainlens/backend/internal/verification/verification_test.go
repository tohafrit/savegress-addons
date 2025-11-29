package verification

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetChainID(t *testing.T) {
	tests := []struct {
		network  string
		expected string
	}{
		{"ethereum", "1"},
		{"polygon", "137"},
		{"arbitrum", "42161"},
		{"optimism", "10"},
		{"base", "8453"},
		{"bsc", "56"},
		{"avalanche", "43114"},
		{"sepolia", "11155111"},
		{"mumbai", "80001"},
		{"unknown", "unknown"}, // Should return as-is
		{"12345", "12345"},     // Chain ID directly
	}

	for _, tt := range tests {
		t.Run(tt.network, func(t *testing.T) {
			result := GetChainID(tt.network)
			if result != tt.expected {
				t.Errorf("GetChainID(%s) = %s, want %s", tt.network, result, tt.expected)
			}
		})
	}
}

func TestIsNetworkSupported(t *testing.T) {
	tests := []struct {
		network  string
		expected bool
	}{
		{"ethereum", true},
		{"polygon", true},
		{"arbitrum", true},
		{"unknown", false},
		{"12345", false},
	}

	for _, tt := range tests {
		t.Run(tt.network, func(t *testing.T) {
			result := IsNetworkSupported(tt.network)
			if result != tt.expected {
				t.Errorf("IsNetworkSupported(%s) = %v, want %v", tt.network, result, tt.expected)
			}
		})
	}
}

func TestComputeFunctionSelector(t *testing.T) {
	tests := []struct {
		signature string
		expected  string
	}{
		{"transfer(address,uint256)", "0xa9059cbb"},
		{"approve(address,uint256)", "0x095ea7b3"},
		{"transferFrom(address,address,uint256)", "0x23b872dd"},
		{"balanceOf(address)", "0x70a08231"},
		{"allowance(address,address)", "0xdd62ed3e"},
		{"totalSupply()", "0x18160ddd"},
		{"name()", "0x06fdde03"},
		{"symbol()", "0x95d89b41"},
		{"decimals()", "0x313ce567"},
	}

	for _, tt := range tests {
		t.Run(tt.signature, func(t *testing.T) {
			result := ComputeFunctionSelector(tt.signature)
			if result != tt.expected {
				t.Errorf("ComputeFunctionSelector(%s) = %s, want %s", tt.signature, result, tt.expected)
			}
		})
	}
}

func TestComputeEventTopic(t *testing.T) {
	tests := []struct {
		signature string
		expected  string
	}{
		{"Transfer(address,address,uint256)", "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"},
		{"Approval(address,address,uint256)", "0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925"},
		{"OwnershipTransferred(address,address)", "0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0"},
	}

	for _, tt := range tests {
		t.Run(tt.signature, func(t *testing.T) {
			result := ComputeEventTopic(tt.signature)
			if result != tt.expected {
				t.Errorf("ComputeEventTopic(%s) = %s, want %s", tt.signature, result, tt.expected)
			}
		})
	}
}

func TestExtractFunctionName(t *testing.T) {
	tests := []struct {
		signature string
		expected  string
	}{
		{"transfer(address,uint256)", "transfer"},
		{"approve(address,uint256)", "approve"},
		{"balanceOf(address)", "balanceOf"},
		{"totalSupply()", "totalSupply"},
		{"noParens", "noParens"},
	}

	for _, tt := range tests {
		t.Run(tt.signature, func(t *testing.T) {
			result := extractFunctionName(tt.signature)
			if result != tt.expected {
				t.Errorf("extractFunctionName(%s) = %s, want %s", tt.signature, result, tt.expected)
			}
		})
	}
}

func TestSourcifyClient_CheckVerification(t *testing.T) {
	// Mock Sourcify server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/check-by-addresses" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		addresses := r.URL.Query().Get("addresses")
		chainIds := r.URL.Query().Get("chainIds")

		if addresses == "0x1234567890123456789012345678901234567890" && chainIds == "1" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]SourcifyCheckResponse{
				{
					Address: addresses,
					ChainID: chainIds,
					Status:  "perfect",
				},
			})
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewSourcifyClient().WithBaseURL(server.URL)

	// Test verified contract
	result, err := client.CheckVerification(context.Background(), "ethereum", "0x1234567890123456789012345678901234567890")
	if err != nil {
		t.Fatalf("CheckVerification failed: %v", err)
	}

	if result.Status != "perfect" {
		t.Errorf("expected status 'perfect', got '%s'", result.Status)
	}

	// Test unverified contract
	result, err = client.CheckVerification(context.Background(), "ethereum", "0x0000000000000000000000000000000000000000")
	if err != nil {
		t.Fatalf("CheckVerification failed: %v", err)
	}

	if result.Status != "false" {
		t.Errorf("expected status 'false', got '%s'", result.Status)
	}
}

func TestSourcifyClient_GetVerifiedContract(t *testing.T) {
	// Mock Sourcify server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/check-by-addresses":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]SourcifyCheckResponse{
				{
					Address: "0x1234567890123456789012345678901234567890",
					ChainID: "1",
					Status:  "perfect",
				},
			})
		case "/files/full_match/1/0x1234567890123456789012345678901234567890":
			w.WriteHeader(http.StatusOK)
			files := []SourcifyFile{
				{
					Name:    "Token.sol",
					Path:    "sources/Token.sol",
					Content: "// SPDX-License-Identifier: MIT\npragma solidity ^0.8.0;\n\ncontract Token {}",
				},
				{
					Name: "metadata.json",
					Path: "metadata.json",
					Content: `{
						"compiler": {"version": "0.8.19+commit.7dd6d404"},
						"language": "Solidity",
						"output": {"abi": [{"type": "function", "name": "balanceOf"}]},
						"settings": {
							"compilationTarget": {"Token.sol": "Token"},
							"evmVersion": "paris",
							"optimizer": {"enabled": true, "runs": 200}
						},
						"sources": {
							"Token.sol": {"license": "MIT"}
						}
					}`,
				},
			}
			json.NewEncoder(w).Encode(files)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewSourcifyClient().WithBaseURL(server.URL)

	contract, err := client.GetVerifiedContract(context.Background(), "ethereum", "0x1234567890123456789012345678901234567890")
	if err != nil {
		t.Fatalf("GetVerifiedContract failed: %v", err)
	}

	if contract == nil {
		t.Fatal("expected contract, got nil")
	}

	if contract.ContractName != "Token" {
		t.Errorf("expected contract name 'Token', got '%s'", contract.ContractName)
	}

	if contract.CompilerVersion != "0.8.19+commit.7dd6d404" {
		t.Errorf("expected compiler version '0.8.19+commit.7dd6d404', got '%s'", contract.CompilerVersion)
	}

	if !contract.OptimizationEnabled {
		t.Error("expected optimization enabled")
	}

	if contract.OptimizationRuns == nil || *contract.OptimizationRuns != 200 {
		t.Error("expected optimization runs 200")
	}

	if contract.VerificationSource != SourceSourcify {
		t.Errorf("expected verification source '%s', got '%s'", SourceSourcify, contract.VerificationSource)
	}
}

func TestSourcifyClient_NotVerified(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewSourcifyClient().WithBaseURL(server.URL)

	contract, err := client.GetVerifiedContract(context.Background(), "ethereum", "0x0000000000000000000000000000000000000000")
	if err != nil {
		t.Fatalf("GetVerifiedContract failed: %v", err)
	}

	if contract != nil {
		t.Error("expected nil contract for unverified address")
	}
}

func TestABIItemTypes(t *testing.T) {
	// Test that ABIItem can be properly marshaled/unmarshaled
	abiJSON := `[
		{
			"type": "function",
			"name": "transfer",
			"inputs": [
				{"type": "address", "name": "to"},
				{"type": "uint256", "name": "amount"}
			],
			"outputs": [{"type": "bool"}],
			"stateMutability": "nonpayable"
		},
		{
			"type": "event",
			"name": "Transfer",
			"inputs": [
				{"type": "address", "name": "from", "indexed": true},
				{"type": "address", "name": "to", "indexed": true},
				{"type": "uint256", "name": "value", "indexed": false}
			]
		}
	]`

	var items []ABIItem
	if err := json.Unmarshal([]byte(abiJSON), &items); err != nil {
		t.Fatalf("failed to unmarshal ABI: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	// Check function
	if items[0].Type != "function" {
		t.Errorf("expected type 'function', got '%s'", items[0].Type)
	}
	if items[0].Name != "transfer" {
		t.Errorf("expected name 'transfer', got '%s'", items[0].Name)
	}
	if len(items[0].Inputs) != 2 {
		t.Errorf("expected 2 inputs, got %d", len(items[0].Inputs))
	}

	// Check event
	if items[1].Type != "event" {
		t.Errorf("expected type 'event', got '%s'", items[1].Type)
	}
	if items[1].Name != "Transfer" {
		t.Errorf("expected name 'Transfer', got '%s'", items[1].Name)
	}
	if !items[1].Inputs[0].Indexed {
		t.Error("expected first input to be indexed")
	}
}

func TestVerificationConstants(t *testing.T) {
	// Verify constants are defined correctly
	if SourceSourcify != "sourcify" {
		t.Errorf("expected SourceSourcify = 'sourcify', got '%s'", SourceSourcify)
	}
	if SourceManual != "manual" {
		t.Errorf("expected SourceManual = 'manual', got '%s'", SourceManual)
	}
	if SourceEtherscan != "etherscan" {
		t.Errorf("expected SourceEtherscan = 'etherscan', got '%s'", SourceEtherscan)
	}
	if SourceBlockscout != "blockscout" {
		t.Errorf("expected SourceBlockscout = 'blockscout', got '%s'", SourceBlockscout)
	}

	if StatusFull != "full" {
		t.Errorf("expected StatusFull = 'full', got '%s'", StatusFull)
	}
	if StatusPartial != "partial" {
		t.Errorf("expected StatusPartial = 'partial', got '%s'", StatusPartial)
	}

	if RequestPending != "pending" {
		t.Errorf("expected RequestPending = 'pending', got '%s'", RequestPending)
	}
	if RequestProcessing != "processing" {
		t.Errorf("expected RequestProcessing = 'processing', got '%s'", RequestProcessing)
	}
	if RequestVerified != "verified" {
		t.Errorf("expected RequestVerified = 'verified', got '%s'", RequestVerified)
	}
	if RequestFailed != "failed" {
		t.Errorf("expected RequestFailed = 'failed', got '%s'", RequestFailed)
	}
}

func TestContractCallerEncoding(t *testing.T) {
	caller := &ContractCaller{
		rpcURLs:    map[string]string{"ethereum": "http://localhost:8545"},
		httpClient: http.DefaultClient,
	}

	// Test building signatures
	testCases := []struct {
		name     string
		inputs   []ABIParam
		expected string
	}{
		{
			name:     "transfer",
			inputs:   []ABIParam{{Type: "address"}, {Type: "uint256"}},
			expected: "transfer(address,uint256)",
		},
		{
			name:     "approve",
			inputs:   []ABIParam{{Type: "address"}, {Type: "uint256"}},
			expected: "approve(address,uint256)",
		},
		{
			name:     "balanceOf",
			inputs:   []ABIParam{{Type: "address"}},
			expected: "balanceOf(address)",
		},
		{
			name:     "totalSupply",
			inputs:   []ABIParam{},
			expected: "totalSupply()",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := caller.buildSignature(tc.name, tc.inputs)
			if result != tc.expected {
				t.Errorf("buildSignature = %s, want %s", result, tc.expected)
			}
		})
	}
}

func TestContractCallerDecoding(t *testing.T) {
	caller := &ContractCaller{}

	// Test decoding address
	addressData := make([]byte, 32)
	copy(addressData[12:], []byte{0x12, 0x34, 0x56, 0x78, 0x90, 0x12, 0x34, 0x56, 0x78, 0x90, 0x12, 0x34, 0x56, 0x78, 0x90, 0x12, 0x34, 0x56, 0x78, 0x90})

	val, err := caller.decodeValue("address", addressData, 0)
	if err != nil {
		t.Fatalf("decodeValue failed: %v", err)
	}
	if val != "0x1234567890123456789012345678901234567890" {
		t.Errorf("expected address 0x1234567890123456789012345678901234567890, got %v", val)
	}

	// Test decoding uint256
	uintData := make([]byte, 32)
	uintData[31] = 100 // 100 in big endian

	val, err = caller.decodeValue("uint256", uintData, 0)
	if err != nil {
		t.Fatalf("decodeValue failed: %v", err)
	}
	if val != int64(100) {
		t.Errorf("expected 100, got %v", val)
	}

	// Test decoding bool true
	boolData := make([]byte, 32)
	boolData[31] = 1

	val, err = caller.decodeValue("bool", boolData, 0)
	if err != nil {
		t.Fatalf("decodeValue failed: %v", err)
	}
	if val != true {
		t.Errorf("expected true, got %v", val)
	}

	// Test decoding bool false
	boolData = make([]byte, 32)

	val, err = caller.decodeValue("bool", boolData, 0)
	if err != nil {
		t.Fatalf("decodeValue failed: %v", err)
	}
	if val != false {
		t.Errorf("expected false, got %v", val)
	}
}

func BenchmarkComputeFunctionSelector(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ComputeFunctionSelector("transfer(address,uint256)")
	}
}

func BenchmarkComputeEventTopic(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ComputeEventTopic("Transfer(address,address,uint256)")
	}
}
