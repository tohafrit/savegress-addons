package nfts

import (
	"encoding/json"
	"testing"
)

func TestParseERC721Transfer(t *testing.T) {
	tests := []struct {
		name     string
		topics   []string
		wantFrom string
		wantTo   string
		wantID   string
		wantNil  bool
	}{
		{
			name: "valid transfer",
			topics: []string{
				ERC721TransferTopic,
				"0x000000000000000000000000a0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
				"0x000000000000000000000000dac17f958d2ee523a2206206994597c13d831ec7",
				"0x0000000000000000000000000000000000000000000000000000000000000001",
			},
			wantFrom: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			wantTo:   "0xdac17f958d2ee523a2206206994597c13d831ec7",
			wantID:   "1",
			wantNil:  false,
		},
		{
			name: "mint (from zero)",
			topics: []string{
				ERC721TransferTopic,
				"0x0000000000000000000000000000000000000000000000000000000000000000",
				"0x000000000000000000000000a0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
				"0x00000000000000000000000000000000000000000000000000000000000003e8",
			},
			wantFrom: ZeroAddress,
			wantTo:   "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			wantID:   "1000",
			wantNil:  false,
		},
		{
			name: "burn (to zero)",
			topics: []string{
				ERC721TransferTopic,
				"0x000000000000000000000000a0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
				"0x0000000000000000000000000000000000000000000000000000000000000000",
				"0x0000000000000000000000000000000000000000000000000000000000000064",
			},
			wantFrom: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			wantTo:   ZeroAddress,
			wantID:   "100",
			wantNil:  false,
		},
		{
			name: "wrong topic",
			topics: []string{
				ApprovalForAllTopic,
				"0x000000000000000000000000a0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
				"0x000000000000000000000000dac17f958d2ee523a2206206994597c13d831ec7",
				"0x0000000000000000000000000000000000000000000000000000000000000001",
			},
			wantNil: true,
		},
		{
			name:    "not enough topics",
			topics:  []string{ERC721TransferTopic},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transfer, err := ParseERC721Transfer(tt.topics)
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
			if transfer.TokenID != tt.wantID {
				t.Errorf("tokenID = %s, want %s", transfer.TokenID, tt.wantID)
			}
		})
	}
}

func TestParseERC1155TransferSingle(t *testing.T) {
	tests := []struct {
		name       string
		topics     []string
		data       string
		wantOp     string
		wantFrom   string
		wantTo     string
		wantID     string
		wantAmount string
		wantNil    bool
	}{
		{
			name: "valid single transfer",
			topics: []string{
				TransferSingleTopic,
				"0x000000000000000000000000operator0000000000000000000000000000000",
				"0x000000000000000000000000a0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
				"0x000000000000000000000000dac17f958d2ee523a2206206994597c13d831ec7",
			},
			data:       "0x00000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000064",
			wantOp:     "0x0operator0000000000000000000000000000000",
			wantFrom:   "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			wantTo:     "0xdac17f958d2ee523a2206206994597c13d831ec7",
			wantID:     "1",
			wantAmount: "100",
			wantNil:    false,
		},
		{
			name: "wrong topic",
			topics: []string{
				TransferBatchTopic,
				"0x000000000000000000000000operator0000000000000000000000000000000",
				"0x000000000000000000000000a0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
				"0x000000000000000000000000dac17f958d2ee523a2206206994597c13d831ec7",
			},
			data:    "0x00000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000064",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transfer, err := ParseERC1155TransferSingle(tt.topics, tt.data)
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
			if transfer.TokenID != tt.wantID {
				t.Errorf("tokenID = %s, want %s", transfer.TokenID, tt.wantID)
			}
			if transfer.Amount != tt.wantAmount {
				t.Errorf("amount = %s, want %s", transfer.Amount, tt.wantAmount)
			}
		})
	}
}

func TestNFTTransferIsMint(t *testing.T) {
	mint := &NFTTransfer{FromAddress: ZeroAddress, ToAddress: "0x123"}
	if !mint.IsMint() {
		t.Error("expected IsMint() to be true")
	}

	notMint := &NFTTransfer{FromAddress: "0x456", ToAddress: "0x123"}
	if notMint.IsMint() {
		t.Error("expected IsMint() to be false")
	}
}

func TestNFTTransferIsBurn(t *testing.T) {
	burn := &NFTTransfer{FromAddress: "0x123", ToAddress: ZeroAddress}
	if !burn.IsBurn() {
		t.Error("expected IsBurn() to be true")
	}

	notBurn := &NFTTransfer{FromAddress: "0x123", ToAddress: "0x456"}
	if notBurn.IsBurn() {
		t.Error("expected IsBurn() to be false")
	}
}

func TestEventSignatures(t *testing.T) {
	// Verify the hardcoded event signatures are correct
	if ERC721TransferTopic != "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef" {
		t.Errorf("wrong ERC721 Transfer topic: %s", ERC721TransferTopic)
	}
	if TransferSingleTopic != "0xc3d58168c5ae7397731d063d5bbf3d657854427343f4c083240f7aacaa2d0f62" {
		t.Errorf("wrong TransferSingle topic: %s", TransferSingleTopic)
	}
	if TransferBatchTopic != "0x4a39dc06d4c0dbc64b70af90fd698a233a518aa5d07e595d983b8c0526c8f7fb" {
		t.Errorf("wrong TransferBatch topic: %s", TransferBatchTopic)
	}
	if ApprovalForAllTopic != "0x17307eab39ab6107e8899845ad3d59bd9653f200f220920489ca2b5937696c31" {
		t.Errorf("wrong ApprovalForAll topic: %s", ApprovalForAllTopic)
	}
}

func TestIsIPFSHash(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG", true},
		{"bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi", true},
		{"bafkreih5aznjvttude6c3wbvqeebb6rlx5wkbzyppv7garber7tctctmsu", true},
		{"short", false},
		{"not-a-hash", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := isIPFSHash(tt.input); got != tt.want {
				t.Errorf("isIPFSHash(%s) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestExtractIPFSCID(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"ipfs://QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG", "QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG"},
		{"/ipfs/QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG", "QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG"},
		{"https://ipfs.io/ipfs/QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG", "QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG"},
		{"QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG/metadata.json", "QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG"},
		{"https://example.com/nft", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := ExtractIPFSCID(tt.input); got != tt.want {
				t.Errorf("ExtractIPFSCID(%s) = %s, want %s", tt.input, got, tt.want)
			}
		})
	}
}

func TestHashURI(t *testing.T) {
	uri := "ipfs://QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG"
	hash := HashURI(uri)

	if len(hash) != 64 {
		t.Errorf("expected 64 char hash, got %d", len(hash))
	}

	// Same URI should produce same hash
	hash2 := HashURI(uri)
	if hash != hash2 {
		t.Error("same URI should produce same hash")
	}

	// Different URI should produce different hash
	hash3 := HashURI(uri + "/different")
	if hash == hash3 {
		t.Error("different URIs should produce different hashes")
	}
}

func TestMetadataFetcherResolveImageURL(t *testing.T) {
	fetcher := NewMetadataFetcher()

	tests := []struct {
		input string
		want  string
	}{
		{"ipfs://QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG", "https://ipfs.io/ipfs/QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG"},
		{"ar://abc123", "https://arweave.net/abc123"},
		{"https://example.com/image.png", "https://example.com/image.png"},
		{"data:image/png;base64,abc123", "data:image/png;base64,abc123"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := fetcher.ResolveImageURL(tt.input); got != tt.want {
				t.Errorf("ResolveImageURL(%s) = %s, want %s", tt.input, got, tt.want)
			}
		})
	}
}

func TestMetadataFetcherResolveTokenURI(t *testing.T) {
	fetcher := NewMetadataFetcher()

	tests := []struct {
		baseURI string
		tokenID string
		want    string
	}{
		{"https://api.example.com/token/{id}", "1", "https://api.example.com/token/0000000000000000000000000000000000000000000000000000000000000001"},
		{"https://api.example.com/token/", "123", "https://api.example.com/token/123"},
		{"https://api.example.com/token", "456", "https://api.example.com/token/456"},
		{"", "1", ""},
	}

	for _, tt := range tests {
		t.Run(tt.baseURI+"/"+tt.tokenID, func(t *testing.T) {
			if got := fetcher.ResolveTokenURI(tt.baseURI, tt.tokenID); got != tt.want {
				t.Errorf("ResolveTokenURI(%s, %s) = %s, want %s", tt.baseURI, tt.tokenID, got, tt.want)
			}
		})
	}
}

func TestParseDataURI(t *testing.T) {
	fetcher := NewMetadataFetcher()

	tests := []struct {
		name    string
		uri     string
		want    string
		wantErr bool
	}{
		{
			name: "base64 json",
			uri:  "data:application/json;base64,eyJuYW1lIjogIlRlc3QgTkZUIn0=",
			want: `{"name": "Test NFT"}`,
		},
		{
			name: "plain json",
			uri:  "data:application/json,{\"name\": \"Test NFT\"}",
			want: `{"name": "Test NFT"}`,
		},
		{
			name:    "invalid data URI",
			uri:     "not-a-data-uri",
			wantErr: true,
		},
		{
			name:    "no comma",
			uri:     "data:application/json",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := fetcher.parseDataURI(tt.uri)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseDataURI() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && string(got) != tt.want {
				t.Errorf("parseDataURI() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestParseAttributes(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{
			name:  "standard format",
			input: `[{"trait_type": "Color", "value": "Blue"}, {"trait_type": "Size", "value": "Large"}]`,
			want:  2,
		},
		{
			name:  "object format",
			input: `{"Color": "Blue", "Size": "Large"}`,
			want:  2,
		},
		{
			name:  "empty",
			input: ``,
			want:  0,
		},
		{
			name:  "null",
			input: `null`,
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attrs := ParseAttributes(json.RawMessage(tt.input))
			if len(attrs) != tt.want {
				t.Errorf("ParseAttributes() returned %d attrs, want %d", len(attrs), tt.want)
			}
		})
	}
}

func TestNormalizeMetadata(t *testing.T) {
	metadata := &NFTMetadata{
		Name:        "  Test NFT  ",
		Description: "  Test description  ",
		ImageURL:    "https://example.com/image.png",
	}

	NormalizeMetadata(metadata)

	if metadata.Name != "Test NFT" {
		t.Errorf("expected trimmed name, got %q", metadata.Name)
	}
	if metadata.Description != "Test description" {
		t.Errorf("expected trimmed description, got %q", metadata.Description)
	}
	if metadata.Image != "https://example.com/image.png" {
		t.Errorf("expected image from imageURL, got %q", metadata.Image)
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

func TestParseUint256(t *testing.T) {
	tests := []struct {
		hex  string
		want string
	}{
		// parseUint256 expects hex WITHOUT 0x prefix
		{"0000000000000000000000000000000000000000000000000000000000000001", "1"},
		{"0000000000000000000000000000000000000000000000000000000000000064", "100"},
		{"00000000000000000000000000000000000000000000000000000000000003e8", "1000"},
		{"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "115792089237316195423570985008687907853269984665640564039457584007913129639935"},
	}

	for _, tt := range tests {
		t.Run(tt.hex, func(t *testing.T) {
			result := parseUint256(tt.hex)
			if result != tt.want {
				t.Errorf("parseUint256(%s) = %s, want %s", tt.hex, result, tt.want)
			}
		})
	}
}

func TestNFTConstants(t *testing.T) {
	if StandardERC721 != "ERC721" {
		t.Errorf("wrong StandardERC721: %s", StandardERC721)
	}
	if StandardERC1155 != "ERC1155" {
		t.Errorf("wrong StandardERC1155: %s", StandardERC1155)
	}
	if ZeroAddress != "0x0000000000000000000000000000000000000000" {
		t.Errorf("wrong ZeroAddress: %s", ZeroAddress)
	}
}

func BenchmarkParseERC721Transfer(b *testing.B) {
	topics := []string{
		ERC721TransferTopic,
		"0x000000000000000000000000a0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
		"0x000000000000000000000000dac17f958d2ee523a2206206994597c13d831ec7",
		"0x0000000000000000000000000000000000000000000000000000000000000001",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParseERC721Transfer(topics)
	}
}

func BenchmarkHashURI(b *testing.B) {
	uri := "ipfs://QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG/metadata.json"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		HashURI(uri)
	}
}

func BenchmarkResolveImageURL(b *testing.B) {
	fetcher := NewMetadataFetcher()
	imageURI := "ipfs://QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fetcher.ResolveImageURL(imageURI)
	}
}
