package nfts

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// MetadataFetcher fetches and parses NFT metadata from various sources
type MetadataFetcher struct {
	httpClient   *http.Client
	ipfsGateways []string
	arweaveURL   string
	maxSize      int64
}

// NewMetadataFetcher creates a new metadata fetcher
func NewMetadataFetcher() *MetadataFetcher {
	return &MetadataFetcher{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		ipfsGateways: []string{
			"https://ipfs.io/ipfs/",
			"https://cloudflare-ipfs.com/ipfs/",
			"https://gateway.pinata.cloud/ipfs/",
			"https://dweb.link/ipfs/",
		},
		arweaveURL: "https://arweave.net/",
		maxSize:    10 * 1024 * 1024, // 10MB max
	}
}

// WithHTTPClient sets a custom HTTP client
func (f *MetadataFetcher) WithHTTPClient(client *http.Client) *MetadataFetcher {
	f.httpClient = client
	return f
}

// WithIPFSGateways sets custom IPFS gateways
func (f *MetadataFetcher) WithIPFSGateways(gateways []string) *MetadataFetcher {
	f.ipfsGateways = gateways
	return f
}

// FetchMetadata fetches metadata from a URI
func (f *MetadataFetcher) FetchMetadata(ctx context.Context, uri string) (*NFTMetadata, error) {
	if uri == "" {
		return nil, fmt.Errorf("empty URI")
	}

	// Handle different URI schemes
	var content []byte
	var err error

	switch {
	case strings.HasPrefix(uri, "data:"):
		content, err = f.parseDataURI(uri)
	case strings.HasPrefix(uri, "ipfs://"):
		content, err = f.fetchIPFS(ctx, uri)
	case strings.HasPrefix(uri, "ar://"):
		content, err = f.fetchArweave(ctx, uri)
	case strings.HasPrefix(uri, "http://") || strings.HasPrefix(uri, "https://"):
		content, err = f.fetchHTTP(ctx, uri)
	default:
		// Try as IPFS hash directly
		if isIPFSHash(uri) {
			content, err = f.fetchIPFS(ctx, "ipfs://"+uri)
		} else {
			return nil, fmt.Errorf("unsupported URI scheme: %s", uri)
		}
	}

	if err != nil {
		return nil, err
	}

	// Parse JSON metadata
	var metadata NFTMetadata
	if err := json.Unmarshal(content, &metadata); err != nil {
		return nil, fmt.Errorf("parse metadata: %w", err)
	}

	// Normalize image URL
	if metadata.Image == "" && metadata.ImageURL != "" {
		metadata.Image = metadata.ImageURL
	}

	return &metadata, nil
}

// parseDataURI parses a data URI (base64 or plain)
func (f *MetadataFetcher) parseDataURI(uri string) ([]byte, error) {
	// Format: data:[<mediatype>][;base64],<data>
	if !strings.HasPrefix(uri, "data:") {
		return nil, fmt.Errorf("invalid data URI")
	}

	uri = uri[5:] // Remove "data:"

	// Find comma separator
	commaIdx := strings.Index(uri, ",")
	if commaIdx == -1 {
		return nil, fmt.Errorf("invalid data URI format")
	}

	header := uri[:commaIdx]
	data := uri[commaIdx+1:]

	// Check if base64 encoded
	if strings.Contains(header, ";base64") {
		decoded, err := base64.StdEncoding.DecodeString(data)
		if err != nil {
			// Try URL-safe base64
			decoded, err = base64.URLEncoding.DecodeString(data)
			if err != nil {
				return nil, fmt.Errorf("decode base64: %w", err)
			}
		}
		return decoded, nil
	}

	// URL-decode plain text
	decoded, err := url.QueryUnescape(data)
	if err != nil {
		return []byte(data), nil
	}
	return []byte(decoded), nil
}

// fetchIPFS fetches content from IPFS
func (f *MetadataFetcher) fetchIPFS(ctx context.Context, uri string) ([]byte, error) {
	// Extract CID from URI
	cid := strings.TrimPrefix(uri, "ipfs://")
	cid = strings.TrimPrefix(cid, "/ipfs/")

	// Try each gateway
	var lastErr error
	for _, gateway := range f.ipfsGateways {
		url := gateway + cid
		content, err := f.fetchHTTP(ctx, url)
		if err == nil {
			return content, nil
		}
		lastErr = err
	}

	return nil, fmt.Errorf("all IPFS gateways failed: %w", lastErr)
}

// fetchArweave fetches content from Arweave
func (f *MetadataFetcher) fetchArweave(ctx context.Context, uri string) ([]byte, error) {
	txID := strings.TrimPrefix(uri, "ar://")
	url := f.arweaveURL + txID
	return f.fetchHTTP(ctx, url)
}

// fetchHTTP fetches content via HTTP(S)
func (f *MetadataFetcher) fetchHTTP(ctx context.Context, uri string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/json, */*")
	req.Header.Set("User-Agent", "ChainLens/1.0")

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Limit response size
	limitedReader := io.LimitReader(resp.Body, f.maxSize)
	content, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	return content, nil
}

// ResolveImageURL resolves an image URL to a fetchable HTTP URL
func (f *MetadataFetcher) ResolveImageURL(imageURI string) string {
	if imageURI == "" {
		return ""
	}

	switch {
	case strings.HasPrefix(imageURI, "ipfs://"):
		cid := strings.TrimPrefix(imageURI, "ipfs://")
		return f.ipfsGateways[0] + cid
	case strings.HasPrefix(imageURI, "ar://"):
		txID := strings.TrimPrefix(imageURI, "ar://")
		return f.arweaveURL + txID
	case strings.HasPrefix(imageURI, "data:"):
		return imageURI // Data URIs are self-contained
	case strings.HasPrefix(imageURI, "http://") || strings.HasPrefix(imageURI, "https://"):
		return imageURI
	default:
		if isIPFSHash(imageURI) {
			return f.ipfsGateways[0] + imageURI
		}
		return imageURI
	}
}

// ResolveTokenURI resolves a token URI, handling base URI + token ID patterns
func (f *MetadataFetcher) ResolveTokenURI(baseURI, tokenID string) string {
	if baseURI == "" {
		return ""
	}

	// Common patterns:
	// 1. {baseURI}{tokenId}
	// 2. {baseURI}/{tokenId}
	// 3. {baseURI}/{tokenId}.json

	uri := baseURI

	// Replace {id} placeholder (ERC-1155 style)
	if strings.Contains(uri, "{id}") {
		// Pad token ID to 64 hex chars for ERC-1155
		paddedID := fmt.Sprintf("%064s", tokenID)
		uri = strings.ReplaceAll(uri, "{id}", paddedID)
		return uri
	}

	// Append token ID if not already included
	if !strings.HasSuffix(uri, "/") && !strings.HasSuffix(uri, tokenID) {
		uri = uri + "/"
	}
	uri = uri + tokenID

	// Add .json extension if not present and looks like it needs it
	if !strings.HasSuffix(uri, ".json") && !strings.Contains(uri, "?") {
		// Check if it's likely an API that doesn't need extension
		if !strings.Contains(uri, "/api/") && !strings.Contains(uri, "metadata") {
			// Could add .json but many APIs work without it
		}
	}

	return uri
}

// HashURI returns a SHA256 hash of the URI for caching
func HashURI(uri string) string {
	hash := sha256.Sum256([]byte(uri))
	return hex.EncodeToString(hash[:])
}

// isIPFSHash checks if a string looks like an IPFS CID
func isIPFSHash(s string) bool {
	// CIDv0: Qm... (46 chars, base58)
	// CIDv1: b... (starts with b, variable length)
	if len(s) < 10 {
		return false
	}
	if strings.HasPrefix(s, "Qm") && len(s) == 46 {
		return true
	}
	if strings.HasPrefix(s, "bafy") || strings.HasPrefix(s, "bafk") {
		return true
	}
	return false
}

// ExtractIPFSCID extracts IPFS CID from various URI formats
func ExtractIPFSCID(uri string) string {
	uri = strings.TrimPrefix(uri, "ipfs://")
	uri = strings.TrimPrefix(uri, "/ipfs/")
	uri = strings.TrimPrefix(uri, "https://ipfs.io/ipfs/")
	uri = strings.TrimPrefix(uri, "https://gateway.pinata.cloud/ipfs/")
	uri = strings.TrimPrefix(uri, "https://cloudflare-ipfs.com/ipfs/")

	// Remove path after CID
	if idx := strings.Index(uri, "/"); idx > 0 {
		uri = uri[:idx]
	}
	if idx := strings.Index(uri, "?"); idx > 0 {
		uri = uri[:idx]
	}

	if isIPFSHash(uri) {
		return uri
	}
	return ""
}

// NormalizeMetadata normalizes metadata fields
func NormalizeMetadata(metadata *NFTMetadata) {
	// Ensure image field is set
	if metadata.Image == "" && metadata.ImageURL != "" {
		metadata.Image = metadata.ImageURL
	}

	// Clean up description
	if metadata.Description != "" {
		metadata.Description = strings.TrimSpace(metadata.Description)
	}

	// Clean up name
	if metadata.Name != "" {
		metadata.Name = strings.TrimSpace(metadata.Name)
	}
}

// ParseAttributes parses attributes from various formats
func ParseAttributes(raw json.RawMessage) []NFTAttribute {
	if len(raw) == 0 {
		return nil
	}

	// Try standard format: [{"trait_type": "...", "value": "..."}]
	var attrs []NFTAttribute
	if err := json.Unmarshal(raw, &attrs); err == nil {
		return attrs
	}

	// Try object format: {"trait": "value", ...}
	var objAttrs map[string]interface{}
	if err := json.Unmarshal(raw, &objAttrs); err == nil {
		for k, v := range objAttrs {
			attrs = append(attrs, NFTAttribute{
				TraitType: k,
				Value:     v,
			})
		}
		return attrs
	}

	return nil
}
