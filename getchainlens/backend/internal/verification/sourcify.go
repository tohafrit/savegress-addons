// Package verification provides smart contract verification functionality
package verification

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// SourcifyClient provides access to Sourcify API for contract verification
type SourcifyClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewSourcifyClient creates a new Sourcify API client
func NewSourcifyClient() *SourcifyClient {
	return &SourcifyClient{
		baseURL: "https://sourcify.dev/server",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// WithBaseURL sets a custom base URL for testing
func (c *SourcifyClient) WithBaseURL(url string) *SourcifyClient {
	c.baseURL = url
	return c
}

// WithHTTPClient sets a custom HTTP client
func (c *SourcifyClient) WithHTTPClient(client *http.Client) *SourcifyClient {
	c.httpClient = client
	return c
}

// ChainID mapping for network names
var chainIDMap = map[string]string{
	"ethereum":  "1",
	"polygon":   "137",
	"arbitrum":  "42161",
	"optimism":  "10",
	"base":      "8453",
	"bsc":       "56",
	"avalanche": "43114",
	"sepolia":   "11155111",
	"mumbai":    "80001",
}

// GetChainID returns the chain ID for a network name
func GetChainID(network string) string {
	if id, ok := chainIDMap[network]; ok {
		return id
	}
	return network // assume it's already a chain ID
}

// CheckVerification checks if a contract is verified on Sourcify
func (c *SourcifyClient) CheckVerification(ctx context.Context, network, address string) (*SourcifyCheckResponse, error) {
	chainID := GetChainID(network)
	url := fmt.Sprintf("%s/check-by-addresses?addresses=%s&chainIds=%s", c.baseURL, address, chainID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("check verification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return &SourcifyCheckResponse{
			Address: address,
			ChainID: chainID,
			Status:  "false",
		}, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("sourcify API error: %s - %s", resp.Status, string(body))
	}

	// Sourcify returns array of results
	var results []SourcifyCheckResponse
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(results) == 0 {
		return &SourcifyCheckResponse{
			Address: address,
			ChainID: chainID,
			Status:  "false",
		}, nil
	}

	return &results[0], nil
}

// SourcifyFilesResponse represents the files response from Sourcify
type SourcifyFilesResponse struct {
	Status string         `json:"status"`
	Files  []SourcifyFile `json:"files"`
}

// GetVerifiedContract fetches the verified contract source code from Sourcify
func (c *SourcifyClient) GetVerifiedContract(ctx context.Context, network, address string) (*VerifiedContract, error) {
	chainID := GetChainID(network)

	// First check if it's verified
	check, err := c.CheckVerification(ctx, network, address)
	if err != nil {
		return nil, fmt.Errorf("check verification: %w", err)
	}

	if check.Status == "false" {
		return nil, nil // not verified
	}

	// Determine match type (full or partial)
	matchType := "full_match"
	if check.Status == "partial" {
		matchType = "partial_match"
	}

	// Fetch all files
	url := fmt.Sprintf("%s/files/%s/%s/%s", c.baseURL, matchType, chainID, address)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch files: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("sourcify API error: %s - %s", resp.Status, string(body))
	}

	var files []SourcifyFile
	if err := json.NewDecoder(resp.Body).Decode(&files); err != nil {
		return nil, fmt.Errorf("decode files: %w", err)
	}

	return c.parseVerifiedContract(network, address, check.Status, files)
}

// parseVerifiedContract parses Sourcify files into a VerifiedContract
func (c *SourcifyClient) parseVerifiedContract(network, address, status string, files []SourcifyFile) (*VerifiedContract, error) {
	contract := &VerifiedContract{
		Network:            network,
		Address:            strings.ToLower(address),
		VerificationSource: SourceSourcify,
		VerificationStatus: StatusFull,
		VerifiedAt:         time.Now(),
	}

	if status == "partial" {
		contract.VerificationStatus = StatusPartial
	}

	sourceFiles := make(map[string]string)
	var metadata *SourcifyMetadata

	for _, file := range files {
		name := file.Name
		if name == "" {
			// Extract name from path
			parts := strings.Split(file.Path, "/")
			name = parts[len(parts)-1]
		}

		if name == "metadata.json" {
			if err := json.Unmarshal([]byte(file.Content), &metadata); err != nil {
				continue // skip malformed metadata
			}
			continue
		}

		if strings.HasSuffix(name, ".sol") {
			sourceFiles[name] = file.Content

			// Use the first .sol file as main source if no metadata
			if contract.SourceCode == "" {
				contract.SourceCode = file.Content
			}
		}
	}

	// Parse metadata if available
	if metadata != nil {
		contract.CompilerVersion = metadata.Compiler.Version

		// Get contract name from compilation target
		for filename, contractName := range metadata.Settings.CompilationTarget {
			contract.ContractName = contractName
			// Set the main source file
			if src, ok := sourceFiles[filename]; ok {
				contract.SourceCode = src
			}
			break
		}

		contract.OptimizationEnabled = metadata.Settings.Optimizer.Enabled
		if metadata.Settings.Optimizer.Runs > 0 {
			runs := metadata.Settings.Optimizer.Runs
			contract.OptimizationRuns = &runs
		}

		if metadata.Settings.EvmVersion != "" {
			evmVersion := metadata.Settings.EvmVersion
			contract.EVMVersion = &evmVersion
		}

		// Extract ABI
		if len(metadata.Output.ABI) > 0 {
			contract.ABI = metadata.Output.ABI
		}

		// Store full metadata
		metadataBytes, _ := json.Marshal(metadata)
		contract.Metadata = metadataBytes

		// Get license from first source
		for _, src := range metadata.Sources {
			if src.License != "" {
				license := src.License
				contract.License = &license
				break
			}
		}
	}

	// Store source files as JSON
	if len(sourceFiles) > 0 {
		filesBytes, _ := json.Marshal(sourceFiles)
		contract.SourceFiles = filesBytes
	}

	return contract, nil
}

// GetContractFiles fetches just the source files without parsing
func (c *SourcifyClient) GetContractFiles(ctx context.Context, network, address string) ([]SourcifyFile, error) {
	chainID := GetChainID(network)

	// Try full match first
	url := fmt.Sprintf("%s/files/%s/%s/%s", c.baseURL, "full_match", chainID, address)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch files: %w", err)
	}
	defer resp.Body.Close()

	// Try partial match if full match not found
	if resp.StatusCode == http.StatusNotFound {
		url = fmt.Sprintf("%s/files/%s/%s/%s", c.baseURL, "partial_match", chainID, address)
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}

		resp, err = c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("fetch files: %w", err)
		}
		defer resp.Body.Close()
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil // not verified
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("sourcify API error: %s - %s", resp.Status, string(body))
	}

	var files []SourcifyFile
	if err := json.NewDecoder(resp.Body).Decode(&files); err != nil {
		return nil, fmt.Errorf("decode files: %w", err)
	}

	return files, nil
}

// SupportedChains returns the list of supported chain IDs
func (c *SourcifyClient) SupportedChains() []string {
	chains := make([]string, 0, len(chainIDMap))
	for _, id := range chainIDMap {
		chains = append(chains, id)
	}
	return chains
}

// IsNetworkSupported checks if a network is supported by Sourcify
func IsNetworkSupported(network string) bool {
	_, ok := chainIDMap[network]
	return ok
}
