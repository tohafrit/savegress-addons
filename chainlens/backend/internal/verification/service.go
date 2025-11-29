package verification

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/sha3"
)

// Service provides contract verification functionality
type Service struct {
	repo     *Repository
	sourcify *SourcifyClient
}

// NewService creates a new verification service
func NewService(db *pgxpool.Pool) *Service {
	return &Service{
		repo:     NewRepository(db),
		sourcify: NewSourcifyClient(),
	}
}

// WithSourcifyClient sets a custom Sourcify client (for testing)
func (s *Service) WithSourcifyClient(client *SourcifyClient) *Service {
	s.sourcify = client
	return s
}

// ============================================================================
// CONTRACT RETRIEVAL
// ============================================================================

// GetVerifiedContract retrieves a verified contract, fetching from Sourcify if not cached
func (s *Service) GetVerifiedContract(ctx context.Context, network, address string) (*VerifiedContract, error) {
	// Check local database first
	contract, err := s.repo.GetVerifiedContract(ctx, network, address)
	if err != nil {
		return nil, fmt.Errorf("get from db: %w", err)
	}

	if contract != nil {
		return contract, nil
	}

	// Fetch from Sourcify
	contract, err = s.sourcify.GetVerifiedContract(ctx, network, address)
	if err != nil {
		return nil, fmt.Errorf("fetch from sourcify: %w", err)
	}

	if contract == nil {
		return nil, nil // Not verified
	}

	// Cache in database
	if err := s.repo.SaveVerifiedContract(ctx, contract); err != nil {
		// Log but don't fail - we can still return the contract
		fmt.Printf("warning: failed to cache verified contract: %v\n", err)
	}

	return contract, nil
}

// GetContractABI retrieves the ABI for a contract (verified or cached)
func (s *Service) GetContractABI(ctx context.Context, network, address string) (json.RawMessage, error) {
	// Try verified contract first
	contract, err := s.GetVerifiedContract(ctx, network, address)
	if err != nil {
		return nil, err
	}

	if contract != nil && len(contract.ABI) > 0 {
		return contract.ABI, nil
	}

	// Check ABI cache
	cache, err := s.repo.GetABICache(ctx, network, address)
	if err != nil {
		return nil, err
	}

	if cache != nil && len(cache.ABI) > 0 {
		return cache.ABI, nil
	}

	return nil, nil
}

// ListVerifiedContracts lists verified contracts with pagination
func (s *Service) ListVerifiedContracts(ctx context.Context, network string, page, pageSize int) ([]*VerifiedContract, int64, error) {
	return s.repo.ListVerifiedContracts(ctx, network, page, pageSize)
}

// SearchContracts searches for verified contracts
func (s *Service) SearchContracts(ctx context.Context, network, query string) ([]*VerifiedContract, error) {
	return s.repo.SearchVerifiedContracts(ctx, network, query, 20)
}

// ============================================================================
// INTERFACE DETECTION
// ============================================================================

// DetectInterfaces detects which interfaces a contract implements based on its ABI
func (s *Service) DetectInterfaces(ctx context.Context, abi json.RawMessage) ([]DetectedInterface, error) {
	var abiItems []ABIItem
	if err := json.Unmarshal(abi, &abiItems); err != nil {
		return nil, fmt.Errorf("parse abi: %w", err)
	}

	// Get all known interfaces
	interfaces, err := s.repo.ListContractInterfaces(ctx)
	if err != nil {
		return nil, fmt.Errorf("list interfaces: %w", err)
	}

	var detected []DetectedInterface
	for _, iface := range interfaces {
		match := s.matchInterface(abiItems, iface)
		if match != nil && match.Confidence >= 0.5 {
			detected = append(detected, *match)
		}
	}

	return detected, nil
}

func (s *Service) matchInterface(contractABI []ABIItem, iface *ContractInterface) *DetectedInterface {
	var ifaceABI []ABIItem
	if err := json.Unmarshal(iface.ABI, &ifaceABI); err != nil {
		return nil
	}

	// Count matching functions and events
	requiredFuncs := 0
	matchedFuncs := 0
	requiredEvents := 0
	matchedEvents := 0

	var matchedFuncNames []string
	var matchedEventNames []string

	for _, required := range ifaceABI {
		if required.Type == "function" {
			requiredFuncs++
			for _, actual := range contractABI {
				if actual.Type == "function" && actual.Name == required.Name {
					if s.matchParams(required.Inputs, actual.Inputs) {
						matchedFuncs++
						matchedFuncNames = append(matchedFuncNames, required.Name)
						break
					}
				}
			}
		} else if required.Type == "event" {
			requiredEvents++
			for _, actual := range contractABI {
				if actual.Type == "event" && actual.Name == required.Name {
					if s.matchParams(required.Inputs, actual.Inputs) {
						matchedEvents++
						matchedEventNames = append(matchedEventNames, required.Name)
						break
					}
				}
			}
		}
	}

	totalRequired := requiredFuncs + requiredEvents
	totalMatched := matchedFuncs + matchedEvents

	if totalRequired == 0 {
		return nil
	}

	confidence := float64(totalMatched) / float64(totalRequired)

	standard := ""
	if iface.Standard != nil {
		standard = *iface.Standard
	}

	return &DetectedInterface{
		Name:       iface.Name,
		Standard:   standard,
		Confidence: confidence,
		Functions:  matchedFuncNames,
		Events:     matchedEventNames,
	}
}

func (s *Service) matchParams(required, actual []ABIParam) bool {
	if len(required) != len(actual) {
		return false
	}

	for i, req := range required {
		if req.Type != actual[i].Type {
			return false
		}
	}

	return true
}

// ============================================================================
// SIGNATURE DECODING
// ============================================================================

// DecodeFunction decodes a function call using the 4-byte selector
func (s *Service) DecodeFunction(ctx context.Context, data string) (*FunctionSignature, error) {
	if len(data) < 10 {
		return nil, fmt.Errorf("data too short for function selector")
	}

	selector := strings.ToLower(data[:10])
	return s.repo.GetFunctionSignature(ctx, selector)
}

// DecodeEvent decodes an event log using topic0
func (s *Service) DecodeEvent(ctx context.Context, topic0 string) (*EventSignature, error) {
	return s.repo.GetEventSignature(ctx, strings.ToLower(topic0))
}

// ComputeFunctionSelector computes the 4-byte selector for a function signature
func ComputeFunctionSelector(signature string) string {
	hash := sha3.NewLegacyKeccak256()
	hash.Write([]byte(signature))
	return "0x" + hex.EncodeToString(hash.Sum(nil)[:4])
}

// ComputeEventTopic computes the topic0 for an event signature
func ComputeEventTopic(signature string) string {
	hash := sha3.NewLegacyKeccak256()
	hash.Write([]byte(signature))
	return "0x" + hex.EncodeToString(hash.Sum(nil))
}

// ============================================================================
// VERIFICATION REQUESTS
// ============================================================================

// SubmitVerificationRequest submits a new verification request
func (s *Service) SubmitVerificationRequest(ctx context.Context, req *VerifyRequest) (*VerificationRequest, error) {
	// Check if already verified
	existing, err := s.repo.GetVerifiedContract(ctx, req.Network, req.Address)
	if err != nil {
		return nil, fmt.Errorf("check existing: %w", err)
	}

	if existing != nil {
		return nil, fmt.Errorf("contract already verified")
	}

	// Create request
	vr := &VerificationRequest{
		Network:   req.Network,
		Address:   req.Address,
		SourceCode: &req.SourceCode,
		CompilerVersion: &req.CompilerVersion,
		OptimizationEnabled: &req.OptimizationEnabled,
		OptimizationRuns: &req.OptimizationRuns,
		Status: RequestPending,
	}

	if req.ConstructorArgs != "" {
		vr.ConstructorArgs = &req.ConstructorArgs
	}

	if err := s.repo.CreateVerificationRequest(ctx, vr); err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	return vr, nil
}

// GetVerificationRequest retrieves a verification request
func (s *Service) GetVerificationRequest(ctx context.Context, id int64) (*VerificationRequest, error) {
	return s.repo.GetVerificationRequest(ctx, id)
}

// ============================================================================
// ANALYSIS
// ============================================================================

// AnalyzeContract analyzes a contract address
func (s *Service) AnalyzeContract(ctx context.Context, network, address string) (*ContractAnalysis, error) {
	analysis := &ContractAnalysis{
		Network: network,
		Address: address,
	}

	// Check if verified
	contract, err := s.GetVerifiedContract(ctx, network, address)
	if err != nil {
		return nil, fmt.Errorf("get verified: %w", err)
	}

	if contract != nil {
		analysis.IsContract = true
		analysis.IsVerified = true
		analysis.HasSourceCode = true

		// Detect interfaces
		if len(contract.ABI) > 0 {
			interfaces, err := s.DetectInterfaces(ctx, contract.ABI)
			if err == nil {
				analysis.DetectedInterfaces = interfaces
			}
		}
	}

	return analysis, nil
}

// ============================================================================
// CONTRACT INTERFACES
// ============================================================================

// GetContractInterface retrieves a known contract interface
func (s *Service) GetContractInterface(ctx context.Context, name string) (*ContractInterface, error) {
	return s.repo.GetContractInterface(ctx, name)
}

// ListContractInterfaces lists all known contract interfaces
func (s *Service) ListContractInterfaces(ctx context.Context) ([]*ContractInterface, error) {
	return s.repo.ListContractInterfaces(ctx)
}

// ============================================================================
// SIGNATURE MANAGEMENT
// ============================================================================

// AddFunctionSignature adds a function signature to the database
func (s *Service) AddFunctionSignature(ctx context.Context, signature string) (*FunctionSignature, error) {
	// Parse signature to get name
	name := extractFunctionName(signature)
	selector := ComputeFunctionSelector(signature)

	fs := &FunctionSignature{
		Selector:  selector,
		Signature: signature,
		Name:      name,
	}

	if err := s.repo.SaveFunctionSignature(ctx, fs); err != nil {
		return nil, err
	}

	return fs, nil
}

// AddEventSignature adds an event signature to the database
func (s *Service) AddEventSignature(ctx context.Context, signature string) (*EventSignature, error) {
	name := extractFunctionName(signature)
	topic := ComputeEventTopic(signature)

	es := &EventSignature{
		Topic:     topic,
		Signature: signature,
		Name:      name,
	}

	if err := s.repo.SaveEventSignature(ctx, es); err != nil {
		return nil, err
	}

	return es, nil
}

// extractFunctionName extracts the function/event name from a signature
func extractFunctionName(signature string) string {
	idx := strings.Index(signature, "(")
	if idx == -1 {
		return signature
	}
	return signature[:idx]
}
