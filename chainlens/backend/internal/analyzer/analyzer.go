// Package analyzer provides smart contract analysis using worker pool
package analyzer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/chainlens/chainlens/pkg/workerpool"
)

// Severity levels for security issues
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
	SeverityInfo     Severity = "info"
)

// Issue represents a detected security issue
type Issue struct {
	ID         string   `json:"id"`
	Type       string   `json:"type"`
	Severity   Severity `json:"severity"`
	Line       int      `json:"line"`
	Column     int      `json:"column"`
	Message    string   `json:"message"`
	Suggestion string   `json:"suggestion,omitempty"`
	Code       string   `json:"code,omitempty"`
}

// GasEstimate represents gas estimation for a function
type GasEstimate struct {
	Min       uint64 `json:"min"`
	Max       uint64 `json:"max"`
	Typical   uint64 `json:"typical"`
	Level     string `json:"level"` // low, medium, high
}

// AnalysisResult is the complete analysis result
type AnalysisResult struct {
	ID           string                 `json:"id"`
	SourceHash   string                 `json:"source_hash"`
	Status       string                 `json:"status"`
	Issues       []Issue                `json:"issues"`
	GasEstimates map[string]GasEstimate `json:"gas_estimates"`
	Score        Score                  `json:"score"`
	Duration     time.Duration          `json:"duration_ms"`
}

// Score represents analysis scores
type Score struct {
	Security      int `json:"security"`
	GasEfficiency int `json:"gas_efficiency"`
	CodeQuality   int `json:"code_quality"`
}

// Analyzer is the main contract analysis service
type Analyzer struct {
	pool     *workerpool.WorkerPool
	patterns []SecurityPattern
	mu       sync.RWMutex
	cache    map[string]*AnalysisResult // sourceHash -> result
}

// SecurityPattern interface for pluggable security checks
type SecurityPattern interface {
	Name() string
	Check(source string) []Issue
}

// NewAnalyzer creates a new analyzer with worker pool
func NewAnalyzer(workers int) (*Analyzer, error) {
	pool, err := workerpool.NewWorkerPool(workerpool.Config{
		Workers:         workers,
		QueueSize:       1000,
		ShutdownTimeout: 30 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create worker pool: %w", err)
	}

	a := &Analyzer{
		pool:     pool,
		patterns: defaultPatterns(),
		cache:    make(map[string]*AnalysisResult),
	}

	return a, nil
}

// defaultPatterns returns built-in security patterns
func defaultPatterns() []SecurityPattern {
	return []SecurityPattern{
		&ReentrancyPattern{},
		&UncheckedCallPattern{},
		&TxOriginPattern{},
		&TimestampPattern{},
		&OverflowPattern{},
		&AccessControlPattern{},
		&SelfdestructPattern{},
		&DelegatecallPattern{},
	}
}

// Analyze performs security and gas analysis on contract source
func (a *Analyzer) Analyze(ctx context.Context, source string, compiler string) (*AnalysisResult, error) {
	start := time.Now()

	// Generate source hash for caching
	hash := sha256.Sum256([]byte(source))
	sourceHash := hex.EncodeToString(hash[:])

	// Check cache
	a.mu.RLock()
	if cached, ok := a.cache[sourceHash]; ok {
		a.mu.RUnlock()
		return cached, nil
	}
	a.mu.RUnlock()

	// Run analysis in parallel using worker pool
	var allIssues []Issue
	var issuesMu sync.Mutex
	var wg sync.WaitGroup

	for _, pattern := range a.patterns {
		p := pattern // capture for closure
		wg.Add(1)

		err := a.pool.Submit(func() error {
			defer wg.Done()

			issues := p.Check(source)

			issuesMu.Lock()
			allIssues = append(allIssues, issues...)
			issuesMu.Unlock()

			return nil
		})

		if err != nil {
			wg.Done()
			return nil, fmt.Errorf("failed to submit analysis task: %w", err)
		}
	}

	// Wait for all patterns to complete
	wg.Wait()

	// Gas analysis
	gasEstimates := analyzeGas(source)

	// Calculate scores
	score := calculateScore(allIssues, gasEstimates)

	result := &AnalysisResult{
		ID:           fmt.Sprintf("analysis_%s", sourceHash[:8]),
		SourceHash:   sourceHash,
		Status:       "completed",
		Issues:       allIssues,
		GasEstimates: gasEstimates,
		Score:        score,
		Duration:     time.Since(start),
	}

	// Cache result
	a.mu.Lock()
	a.cache[sourceHash] = result
	a.mu.Unlock()

	return result, nil
}

// AnalyzeBatch analyzes multiple contracts in parallel
func (a *Analyzer) AnalyzeBatch(ctx context.Context, sources []string) ([]*AnalysisResult, error) {
	results := make([]*AnalysisResult, len(sources))
	var wg sync.WaitGroup
	var errOnce sync.Once
	var firstErr error

	for i, source := range sources {
		idx := i
		src := source
		wg.Add(1)

		err := a.pool.Submit(func() error {
			defer wg.Done()

			result, err := a.Analyze(ctx, src, "0.8.19")
			if err != nil {
				errOnce.Do(func() { firstErr = err })
				return err
			}

			results[idx] = result
			return nil
		})

		if err != nil {
			wg.Done()
			return nil, err
		}
	}

	wg.Wait()

	if firstErr != nil {
		return nil, firstErr
	}

	return results, nil
}

// Stop shuts down the analyzer
func (a *Analyzer) Stop() error {
	return a.pool.Stop()
}

// Stats returns worker pool statistics
func (a *Analyzer) Stats() workerpool.Stats {
	return a.pool.Stats()
}

// analyzeGas performs gas estimation by analyzing Solidity source code
func analyzeGas(source string) map[string]GasEstimate {
	estimates := make(map[string]GasEstimate)
	lines := splitLines(source)

	// Track current function context
	var currentFunction string
	var functionBraces int
	var functionGas gasAccumulator

	for _, line := range lines {
		trimmed := trimSpace(line)

		// Detect function declaration
		if contains(trimmed, "function ") && (contains(trimmed, "public") || contains(trimmed, "external")) {
			// Extract function name
			start := findSubstring(trimmed, "function ") + 9
			end := start
			for end < len(trimmed) && trimmed[end] != '(' && trimmed[end] != ' ' {
				end++
			}
			if end > start {
				currentFunction = trimmed[start:end]
				functionBraces = countChar(line, '{') - countChar(line, '}')
				functionGas = gasAccumulator{}
			}
			continue
		}

		// Track braces for function scope
		if currentFunction != "" {
			functionBraces += countChar(line, '{') - countChar(line, '}')

			// Analyze gas costs within function
			analyzeLineGas(trimmed, &functionGas)

			// Function ended
			if functionBraces <= 0 {
				estimates[currentFunction] = calculateFunctionGas(functionGas)
				currentFunction = ""
			}
		}
	}

	return estimates
}

// gasAccumulator tracks gas costs during analysis
type gasAccumulator struct {
	storageReads   int
	storageWrites  int
	externalCalls  int
	memoryOps      int
	loops          int
	conditionals   int
	eventEmits     int
	mappingAccess  int
	arrayOps       int
	stringOps      int
}

// analyzeLineGas analyzes a line for gas-consuming operations
func analyzeLineGas(line string, gas *gasAccumulator) {
	// Storage reads (state variable access)
	if contains(line, "storage") || (contains(line, "=") && !contains(line, "memory") && !contains(line, "calldata")) {
		if contains(line, "=") && !contains(line, "==") {
			gas.storageWrites++
		} else {
			gas.storageReads++
		}
	}

	// External calls
	if contains(line, ".call") || contains(line, ".transfer") || contains(line, ".send") {
		gas.externalCalls++
	}

	// Contract calls
	if contains(line, "(") && contains(line, ")") && !contains(line, "function") && !contains(line, "if") && !contains(line, "require") {
		if contains(line, ".") {
			gas.externalCalls++
		}
	}

	// Memory operations
	if contains(line, "memory") || contains(line, "new ") {
		gas.memoryOps++
	}

	// Loops
	if contains(line, "for") || contains(line, "while") {
		gas.loops++
	}

	// Conditionals
	if contains(line, "if") || contains(line, "require") || contains(line, "assert") {
		gas.conditionals++
	}

	// Events
	if contains(line, "emit ") {
		gas.eventEmits++
	}

	// Mapping access
	if contains(line, "[") && contains(line, "]") {
		gas.mappingAccess++
	}

	// Array operations
	if contains(line, ".push") || contains(line, ".pop") || contains(line, ".length") {
		gas.arrayOps++
	}

	// String operations
	if contains(line, "string") && (contains(line, "concat") || contains(line, "bytes(")) {
		gas.stringOps++
	}
}

// calculateFunctionGas calculates gas estimate for a function
func calculateFunctionGas(gas gasAccumulator) GasEstimate {
	// Base gas costs (approximate values for EVM)
	const (
		baseGas            = 21000 // Transaction base cost
		sloadGas           = 2100  // Cold storage read
		sstoreGas          = 20000 // Storage write (new value)
		callGas            = 2600  // External call base
		memoryGas          = 3     // Per word
		loopOverhead       = 100   // Loop iteration overhead
		conditionalGas     = 10    // Jump cost
		eventGas           = 375   // Base log cost
		mappingGas         = 200   // Mapping hash computation
		arrayGas           = 100   // Array operation overhead
		stringGas          = 200   // String operation overhead
	)

	minGas := uint64(baseGas)
	minGas += uint64(gas.storageReads) * sloadGas
	minGas += uint64(gas.storageWrites) * sstoreGas
	minGas += uint64(gas.externalCalls) * callGas
	minGas += uint64(gas.memoryOps) * memoryGas * 32 // Assume 32 bytes per op
	minGas += uint64(gas.conditionals) * conditionalGas
	minGas += uint64(gas.eventEmits) * eventGas
	minGas += uint64(gas.mappingAccess) * mappingGas
	minGas += uint64(gas.arrayOps) * arrayGas
	minGas += uint64(gas.stringOps) * stringGas

	// Calculate typical (add loop overhead with average iterations)
	typicalGas := minGas
	if gas.loops > 0 {
		typicalGas += uint64(gas.loops) * loopOverhead * 10 // Assume ~10 iterations
	}

	// Calculate max (worst case with more loop iterations)
	maxGas := minGas
	if gas.loops > 0 {
		maxGas += uint64(gas.loops) * loopOverhead * 100 // Assume ~100 iterations worst case
	}

	// Add 20% buffer for max
	maxGas = maxGas * 120 / 100

	// Determine level
	level := "low"
	if typicalGas > 100000 {
		level = "high"
	} else if typicalGas > 50000 {
		level = "medium"
	}

	return GasEstimate{
		Min:     minGas,
		Max:     maxGas,
		Typical: typicalGas,
		Level:   level,
	}
}

func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

func countChar(s string, c byte) int {
	count := 0
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			count++
		}
	}
	return count
}

// calculateScore calculates security/quality scores
func calculateScore(issues []Issue, gas map[string]GasEstimate) Score {
	securityScore := 100
	for _, issue := range issues {
		switch issue.Severity {
		case SeverityCritical:
			securityScore -= 25
		case SeverityHigh:
			securityScore -= 15
		case SeverityMedium:
			securityScore -= 8
		case SeverityLow:
			securityScore -= 3
		}
	}
	if securityScore < 0 {
		securityScore = 0
	}

	// Calculate gas efficiency score
	gasScore := 100
	highGasFunctions := 0
	mediumGasFunctions := 0
	for _, estimate := range gas {
		switch estimate.Level {
		case "high":
			highGasFunctions++
		case "medium":
			mediumGasFunctions++
		}
	}
	gasScore -= highGasFunctions * 15
	gasScore -= mediumGasFunctions * 5
	if gasScore < 0 {
		gasScore = 0
	}

	// Calculate code quality score based on patterns
	codeQuality := 100
	// Deduct for critical/high issues as they indicate quality problems
	criticalCount := 0
	for _, issue := range issues {
		if issue.Severity == SeverityCritical || issue.Severity == SeverityHigh {
			criticalCount++
		}
	}
	codeQuality -= criticalCount * 10
	if codeQuality < 0 {
		codeQuality = 0
	}

	return Score{
		Security:      securityScore,
		GasEfficiency: gasScore,
		CodeQuality:   codeQuality,
	}
}

// =============================================================================
// Security Patterns Implementation
// =============================================================================

// ReentrancyPattern detects reentrancy vulnerabilities
type ReentrancyPattern struct{}

func (p *ReentrancyPattern) Name() string { return "reentrancy" }

func (p *ReentrancyPattern) Check(source string) []Issue {
	var issues []Issue

	// Simple heuristic: look for .call{ before state changes
	// In production, use proper AST analysis

	lines := splitLines(source)
	for i, line := range lines {
		if contains(line, ".call{value") || contains(line, ".call{") {
			// Check if there's an assignment after this in the same function
			// This is a simplified check
			for j := i + 1; j < len(lines) && j < i+10; j++ {
				if contains(lines[j], "=") && !contains(lines[j], "==") && !contains(lines[j], "//") {
					issues = append(issues, Issue{
						ID:         fmt.Sprintf("REENTRANCY-%d", i+1),
						Type:       "reentrancy",
						Severity:   SeverityCritical,
						Line:       i + 1,
						Message:    "Potential reentrancy: external call before state update",
						Suggestion: "Apply checks-effects-interactions pattern",
						Code:       line,
					})
					break
				}
			}
		}
	}

	return issues
}

// UncheckedCallPattern detects unchecked low-level calls
type UncheckedCallPattern struct{}

func (p *UncheckedCallPattern) Name() string { return "unchecked_call" }

func (p *UncheckedCallPattern) Check(source string) []Issue {
	var issues []Issue

	lines := splitLines(source)
	for i, line := range lines {
		if contains(line, ".call") && !contains(line, "(bool") && !contains(line, "//") {
			issues = append(issues, Issue{
				ID:         fmt.Sprintf("UNCHECKED-%d", i+1),
				Type:       "unchecked_call",
				Severity:   SeverityHigh,
				Line:       i + 1,
				Message:    "Return value of low-level call not checked",
				Suggestion: "Check return value: (bool success, ) = addr.call{...}(...); require(success);",
				Code:       line,
			})
		}
	}

	return issues
}

// TxOriginPattern detects tx.origin usage
type TxOriginPattern struct{}

func (p *TxOriginPattern) Name() string { return "tx_origin" }

func (p *TxOriginPattern) Check(source string) []Issue {
	var issues []Issue

	lines := splitLines(source)
	for i, line := range lines {
		if contains(line, "tx.origin") && !contains(line, "//") {
			issues = append(issues, Issue{
				ID:         fmt.Sprintf("TXORIGIN-%d", i+1),
				Type:       "tx_origin",
				Severity:   SeverityHigh,
				Line:       i + 1,
				Message:    "Using tx.origin for authorization is vulnerable to phishing",
				Suggestion: "Use msg.sender instead",
				Code:       line,
			})
		}
	}

	return issues
}

// TimestampPattern detects block.timestamp dependencies
type TimestampPattern struct{}

func (p *TimestampPattern) Name() string { return "timestamp" }

func (p *TimestampPattern) Check(source string) []Issue {
	var issues []Issue

	lines := splitLines(source)
	for i, line := range lines {
		if (contains(line, "if") || contains(line, "require")) && contains(line, "block.timestamp") {
			issues = append(issues, Issue{
				ID:         fmt.Sprintf("TIMESTAMP-%d", i+1),
				Type:       "timestamp_dependency",
				Severity:   SeverityMedium,
				Line:       i + 1,
				Message:    "block.timestamp can be manipulated by miners (~15 seconds)",
				Suggestion: "Avoid using timestamp for critical logic",
				Code:       line,
			})
		}
	}

	return issues
}

// OverflowPattern detects potential integer overflows (pre-0.8.0)
type OverflowPattern struct{}

func (p *OverflowPattern) Name() string { return "overflow" }

func (p *OverflowPattern) Check(source string) []Issue {
	var issues []Issue

	// Check if Solidity version is < 0.8.0
	if !contains(source, "pragma solidity") {
		return issues
	}

	if contains(source, "^0.8") || contains(source, ">=0.8") || contains(source, "0.8.") {
		// Safe - has built-in overflow checks
		return issues
	}

	// Pre-0.8.0, check for SafeMath
	if contains(source, "SafeMath") {
		return issues
	}

	lines := splitLines(source)
	for i, line := range lines {
		if (contains(line, "+") || contains(line, "*")) && !contains(line, "//") && !contains(line, "pragma") {
			issues = append(issues, Issue{
				ID:         fmt.Sprintf("OVERFLOW-%d", i+1),
				Type:       "integer_overflow",
				Severity:   SeverityHigh,
				Line:       i + 1,
				Message:    "Potential integer overflow in Solidity < 0.8.0",
				Suggestion: "Use SafeMath or upgrade to Solidity 0.8.0+",
				Code:       line,
			})
		}
	}

	return issues
}

// AccessControlPattern detects missing access control
type AccessControlPattern struct{}

func (p *AccessControlPattern) Name() string { return "access_control" }

func (p *AccessControlPattern) Check(source string) []Issue {
	var issues []Issue

	criticalFunctions := []string{"withdraw", "transfer", "mint", "burn", "pause", "upgrade", "setOwner"}

	lines := splitLines(source)
	for i, line := range lines {
		if contains(line, "function") && (contains(line, "public") || contains(line, "external")) {
			for _, fn := range criticalFunctions {
				if contains(line, fn) && !contains(line, "onlyOwner") && !contains(line, "modifier") {
					issues = append(issues, Issue{
						ID:         fmt.Sprintf("ACCESS-%d", i+1),
						Type:       "missing_access_control",
						Severity:   SeverityMedium,
						Line:       i + 1,
						Message:    fmt.Sprintf("Function '%s' may need access control", fn),
						Suggestion: "Add onlyOwner or role-based access control",
						Code:       line,
					})
				}
			}
		}
	}

	return issues
}

// SelfdestructPattern detects selfdestruct usage
type SelfdestructPattern struct{}

func (p *SelfdestructPattern) Name() string { return "selfdestruct" }

func (p *SelfdestructPattern) Check(source string) []Issue {
	var issues []Issue

	lines := splitLines(source)
	for i, line := range lines {
		if contains(line, "selfdestruct") && !contains(line, "//") {
			issues = append(issues, Issue{
				ID:         fmt.Sprintf("SELFDESTRUCT-%d", i+1),
				Type:       "selfdestruct",
				Severity:   SeverityHigh,
				Line:       i + 1,
				Message:    "Contract can be destroyed - ensure proper access control",
				Suggestion: "Add strict access control or consider removing selfdestruct",
				Code:       line,
			})
		}
	}

	return issues
}

// DelegatecallPattern detects delegatecall usage
type DelegatecallPattern struct{}

func (p *DelegatecallPattern) Name() string { return "delegatecall" }

func (p *DelegatecallPattern) Check(source string) []Issue {
	var issues []Issue

	lines := splitLines(source)
	for i, line := range lines {
		if contains(line, "delegatecall") && !contains(line, "//") {
			issues = append(issues, Issue{
				ID:         fmt.Sprintf("DELEGATECALL-%d", i+1),
				Type:       "delegatecall",
				Severity:   SeverityHigh,
				Line:       i + 1,
				Message:    "delegatecall can modify storage - ensure target is trusted",
				Suggestion: "Only delegatecall to trusted, audited contracts",
				Code:       line,
			})
		}
	}

	return issues
}

// Helper functions
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr) >= 0
}

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
