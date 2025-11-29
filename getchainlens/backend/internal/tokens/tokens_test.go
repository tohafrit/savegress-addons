package tokens

import (
	"testing"
)

func TestParseTransferEvent(t *testing.T) {
	tests := []struct {
		name     string
		topics   []string
		data     string
		wantFrom string
		wantTo   string
		wantVal  string
		wantNil  bool
	}{
		{
			name: "valid transfer",
			topics: []string{
				TransferEventTopic,
				"0x000000000000000000000000a0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
				"0x000000000000000000000000dac17f958d2ee523a2206206994597c13d831ec7",
			},
			data:     "0x0000000000000000000000000000000000000000000000000000000005f5e100",
			wantFrom: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			wantTo:   "0xdac17f958d2ee523a2206206994597c13d831ec7",
			wantVal:  "100000000",
			wantNil:  false,
		},
		{
			name: "mint (from zero)",
			topics: []string{
				TransferEventTopic,
				"0x0000000000000000000000000000000000000000000000000000000000000000",
				"0x000000000000000000000000a0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			},
			data:     "0x00000000000000000000000000000000000000000000000000000000000f4240",
			wantFrom: ZeroAddress,
			wantTo:   "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			wantVal:  "1000000",
			wantNil:  false,
		},
		{
			name: "burn (to zero)",
			topics: []string{
				TransferEventTopic,
				"0x000000000000000000000000a0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
				"0x0000000000000000000000000000000000000000000000000000000000000000",
			},
			data:     "0x00000000000000000000000000000000000000000000000000000000000186a0",
			wantFrom: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			wantTo:   ZeroAddress,
			wantVal:  "100000",
			wantNil:  false,
		},
		{
			name: "wrong topic",
			topics: []string{
				ApprovalEventTopic,
				"0x000000000000000000000000a0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
				"0x000000000000000000000000dac17f958d2ee523a2206206994597c13d831ec7",
			},
			data:    "0x0000000000000000000000000000000000000000000000000000000005f5e100",
			wantNil: true,
		},
		{
			name:    "not enough topics",
			topics:  []string{TransferEventTopic},
			data:    "0x0000000000000000000000000000000000000000000000000000000005f5e100",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transfer, err := ParseTransferEvent(tt.topics, tt.data)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantNil {
				if transfer != nil {
					t.Error("expected nil transfer")
				}
				return
			}

			if transfer == nil {
				t.Fatal("expected non-nil transfer")
			}

			if transfer.FromAddress != tt.wantFrom {
				t.Errorf("from = %s, want %s", transfer.FromAddress, tt.wantFrom)
			}
			if transfer.ToAddress != tt.wantTo {
				t.Errorf("to = %s, want %s", transfer.ToAddress, tt.wantTo)
			}
			if transfer.Value != tt.wantVal {
				t.Errorf("value = %s, want %s", transfer.Value, tt.wantVal)
			}
		})
	}
}

func TestParseApprovalEvent(t *testing.T) {
	tests := []struct {
		name        string
		topics      []string
		data        string
		wantOwner   string
		wantSpender string
		wantVal     string
		wantNil     bool
	}{
		{
			name: "valid approval",
			topics: []string{
				ApprovalEventTopic,
				"0x000000000000000000000000a0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
				"0x000000000000000000000000dac17f958d2ee523a2206206994597c13d831ec7",
			},
			data:        "0x0000000000000000000000000000000000000000000000000000000005f5e100",
			wantOwner:   "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			wantSpender: "0xdac17f958d2ee523a2206206994597c13d831ec7",
			wantVal:     "100000000",
			wantNil:     false,
		},
		{
			name: "unlimited approval",
			topics: []string{
				ApprovalEventTopic,
				"0x000000000000000000000000a0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
				"0x000000000000000000000000dac17f958d2ee523a2206206994597c13d831ec7",
			},
			data:        "0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			wantOwner:   "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			wantSpender: "0xdac17f958d2ee523a2206206994597c13d831ec7",
			wantVal:     MaxUint256.String(),
			wantNil:     false,
		},
		{
			name: "wrong topic",
			topics: []string{
				TransferEventTopic,
				"0x000000000000000000000000a0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
				"0x000000000000000000000000dac17f958d2ee523a2206206994597c13d831ec7",
			},
			data:    "0x0000000000000000000000000000000000000000000000000000000005f5e100",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			approval, err := ParseApprovalEvent(tt.topics, tt.data)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantNil {
				if approval != nil {
					t.Error("expected nil approval")
				}
				return
			}

			if approval == nil {
				t.Fatal("expected non-nil approval")
			}

			if approval.OwnerAddress != tt.wantOwner {
				t.Errorf("owner = %s, want %s", approval.OwnerAddress, tt.wantOwner)
			}
			if approval.SpenderAddress != tt.wantSpender {
				t.Errorf("spender = %s, want %s", approval.SpenderAddress, tt.wantSpender)
			}
			if approval.Value != tt.wantVal {
				t.Errorf("value = %s, want %s", approval.Value, tt.wantVal)
			}
		})
	}
}

func TestFormatBalance(t *testing.T) {
	tests := []struct {
		balance  string
		decimals int
		want     string
	}{
		{"1000000", 6, "1"},
		{"1500000", 6, "1.5"},
		{"1000000000000000000", 18, "1"},
		{"1500000000000000000", 18, "1.5"},
		{"100", 2, "1"},
		{"123", 2, "1.23"},
		{"123456789", 2, "1234567.89"},
		{"0", 18, "0"},
		{"100000000", 8, "1"},
		{"12345678", 8, "0.12345678"},
	}

	for _, tt := range tests {
		t.Run(tt.balance, func(t *testing.T) {
			result := FormatBalance(tt.balance, tt.decimals)
			if result != tt.want {
				t.Errorf("FormatBalance(%s, %d) = %s, want %s", tt.balance, tt.decimals, result, tt.want)
			}
		})
	}
}

func TestParseBalance(t *testing.T) {
	tests := []struct {
		formatted string
		decimals  int
		want      string
	}{
		{"1", 6, "1000000"},
		{"1.5", 6, "1500000"},
		{"1", 18, "1000000000000000000"},
		{"1.5", 18, "1500000000000000000"},
		{"0", 18, "0"},
		{"100", 6, "100000000"},
	}

	for _, tt := range tests {
		t.Run(tt.formatted, func(t *testing.T) {
			result := ParseBalance(tt.formatted, tt.decimals)
			if result != tt.want {
				t.Errorf("ParseBalance(%s, %d) = %s, want %s", tt.formatted, tt.decimals, result, tt.want)
			}
		})
	}
}

func TestTokenTransferIsMint(t *testing.T) {
	mint := &TokenTransfer{FromAddress: ZeroAddress, ToAddress: "0x123"}
	if !mint.IsMint() {
		t.Error("expected IsMint() to be true")
	}

	notMint := &TokenTransfer{FromAddress: "0x456", ToAddress: "0x123"}
	if notMint.IsMint() {
		t.Error("expected IsMint() to be false")
	}
}

func TestTokenTransferIsBurn(t *testing.T) {
	burn := &TokenTransfer{FromAddress: "0x123", ToAddress: ZeroAddress}
	if !burn.IsBurn() {
		t.Error("expected IsBurn() to be true")
	}

	notBurn := &TokenTransfer{FromAddress: "0x123", ToAddress: "0x456"}
	if notBurn.IsBurn() {
		t.Error("expected IsBurn() to be false")
	}
}

func TestIsUnlimitedApproval(t *testing.T) {
	unlimited := &TokenApproval{Value: MaxUint256.String()}
	if !unlimited.IsUnlimitedApproval() {
		t.Error("expected IsUnlimitedApproval() to be true")
	}

	limited := &TokenApproval{Value: "1000000"}
	if limited.IsUnlimitedApproval() {
		t.Error("expected IsUnlimitedApproval() to be false")
	}
}

func TestEventSignatures(t *testing.T) {
	// Verify the hardcoded event signatures are correct
	if TransferEventTopic != "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef" {
		t.Errorf("wrong Transfer event topic: %s", TransferEventTopic)
	}
	if ApprovalEventTopic != "0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925" {
		t.Errorf("wrong Approval event topic: %s", ApprovalEventTopic)
	}
}

func TestTokenConstants(t *testing.T) {
	if TokenTypeERC20 != "ERC20" {
		t.Errorf("wrong TokenTypeERC20: %s", TokenTypeERC20)
	}
	if TokenTypeERC721 != "ERC721" {
		t.Errorf("wrong TokenTypeERC721: %s", TokenTypeERC721)
	}
	if TokenTypeERC1155 != "ERC1155" {
		t.Errorf("wrong TokenTypeERC1155: %s", TokenTypeERC1155)
	}
	if ZeroAddress != "0x0000000000000000000000000000000000000000" {
		t.Errorf("wrong ZeroAddress: %s", ZeroAddress)
	}
}

func TestParseAddress(t *testing.T) {
	tests := []struct {
		topic string
		want  string
	}{
		{"0x000000000000000000000000a0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"},
		{"0x0000000000000000000000000000000000000000000000000000000000000000", ZeroAddress},
		{"short", ZeroAddress},
	}

	for _, tt := range tests {
		t.Run(tt.topic, func(t *testing.T) {
			result := parseAddress(tt.topic)
			if result != tt.want {
				t.Errorf("parseAddress(%s) = %s, want %s", tt.topic, result, tt.want)
			}
		})
	}
}

func TestDecodeString(t *testing.T) {
	// Test ABI-encoded string "USDC"
	// offset (32 bytes) + length (32 bytes) + data
	encoded := "0000000000000000000000000000000000000000000000000000000000000020" + // offset = 32
		"0000000000000000000000000000000000000000000000000000000000000004" + // length = 4
		"5553444300000000000000000000000000000000000000000000000000000000" // "USDC"

	result := decodeString(encoded)
	if result != "USDC" {
		t.Errorf("decodeString() = %q, want %q", result, "USDC")
	}
}

func TestDecodeBytes32String(t *testing.T) {
	// Some old tokens use bytes32 for name/symbol
	// "MKR" as bytes32
	encoded := "4d4b5200000000000000000000000000000000000000000000000000000000000"

	result := decodeBytes32String(encoded)
	if result != "MKR" {
		t.Errorf("decodeBytes32String() = %q, want %q", result, "MKR")
	}
}

func TestDecodeUint8(t *testing.T) {
	tests := []struct {
		data string
		want int
	}{
		{"0x0000000000000000000000000000000000000000000000000000000000000012", 18},
		{"0x0000000000000000000000000000000000000000000000000000000000000006", 6},
		{"0x0000000000000000000000000000000000000000000000000000000000000008", 8},
	}

	for _, tt := range tests {
		t.Run(tt.data, func(t *testing.T) {
			result := decodeUint8(tt.data)
			if result != tt.want {
				t.Errorf("decodeUint8(%s) = %d, want %d", tt.data, result, tt.want)
			}
		})
	}
}

func TestEncodeAddress(t *testing.T) {
	tests := []struct {
		address string
		want    string
	}{
		{"0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", "000000000000000000000000a0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"},
		{"0x0000000000000000000000000000000000000000", "0000000000000000000000000000000000000000000000000000000000000000"},
	}

	for _, tt := range tests {
		t.Run(tt.address, func(t *testing.T) {
			result := encodeAddress(tt.address)
			if result != tt.want {
				t.Errorf("encodeAddress(%s) = %s, want %s", tt.address, result, tt.want)
			}
		})
	}
}

func BenchmarkParseTransferEvent(b *testing.B) {
	topics := []string{
		TransferEventTopic,
		"0x000000000000000000000000a0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
		"0x000000000000000000000000dac17f958d2ee523a2206206994597c13d831ec7",
	}
	data := "0x0000000000000000000000000000000000000000000000000000000005f5e100"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParseTransferEvent(topics, data)
	}
}

func BenchmarkFormatBalance(b *testing.B) {
	balance := "1500000000000000000"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FormatBalance(balance, 18)
	}
}
