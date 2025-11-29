// Package verification provides smart contract verification functionality
package verification

import (
	"encoding/json"
	"time"
)

// VerifiedContract represents a verified smart contract
type VerifiedContract struct {
	ID                  int64           `json:"-" db:"id"`
	Network             string          `json:"network" db:"network"`
	Address             string          `json:"address" db:"address"`
	ContractName        string          `json:"contractName" db:"contract_name"`
	CompilerVersion     string          `json:"compilerVersion" db:"compiler_version"`
	OptimizationEnabled bool            `json:"optimizationEnabled" db:"optimization_enabled"`
	OptimizationRuns    *int            `json:"optimizationRuns,omitempty" db:"optimization_runs"`
	EVMVersion          *string         `json:"evmVersion,omitempty" db:"evm_version"`
	License             *string         `json:"license,omitempty" db:"license"`
	SourceCode          string          `json:"sourceCode" db:"source_code"`
	ABI                 json.RawMessage `json:"abi" db:"abi"`
	Bytecode            *string         `json:"bytecode,omitempty" db:"bytecode"`
	DeployedBytecode    *string         `json:"deployedBytecode,omitempty" db:"deployed_bytecode"`
	ConstructorArgs     *string         `json:"constructorArgs,omitempty" db:"constructor_args"`
	Metadata            json.RawMessage `json:"metadata,omitempty" db:"metadata"`
	SourceFiles         json.RawMessage `json:"sourceFiles,omitempty" db:"source_files"`
	VerificationSource  string          `json:"verificationSource" db:"verification_source"`
	VerificationStatus  string          `json:"verificationStatus" db:"verification_status"`
	VerifiedAt          time.Time       `json:"verifiedAt" db:"verified_at"`
	VerifiedBy          *string         `json:"verifiedBy,omitempty" db:"verified_by"`
	CreatedAt           time.Time       `json:"-" db:"created_at"`
	UpdatedAt           time.Time       `json:"-" db:"updated_at"`
}

// ContractInterface represents a known contract interface (ERC standard)
type ContractInterface struct {
	ID                 int             `json:"-" db:"id"`
	Name               string          `json:"name" db:"name"`
	Standard           *string         `json:"standard,omitempty" db:"standard"`
	ABI                json.RawMessage `json:"abi" db:"abi"`
	EventSignatures    json.RawMessage `json:"eventSignatures,omitempty" db:"event_signatures"`
	FunctionSignatures json.RawMessage `json:"functionSignatures,omitempty" db:"function_signatures"`
	CreatedAt          time.Time       `json:"-" db:"created_at"`
}

// VerificationRequest represents a pending verification request
type VerificationRequest struct {
	ID                  int64     `json:"id" db:"id"`
	Network             string    `json:"network" db:"network"`
	Address             string    `json:"address" db:"address"`
	SourceCode          *string   `json:"sourceCode,omitempty" db:"source_code"`
	CompilerVersion     *string   `json:"compilerVersion,omitempty" db:"compiler_version"`
	OptimizationEnabled *bool     `json:"optimizationEnabled,omitempty" db:"optimization_enabled"`
	OptimizationRuns    *int      `json:"optimizationRuns,omitempty" db:"optimization_runs"`
	ConstructorArgs     *string   `json:"constructorArgs,omitempty" db:"constructor_args"`
	Status              string    `json:"status" db:"status"`
	ErrorMessage        *string   `json:"errorMessage,omitempty" db:"error_message"`
	Attempts            int       `json:"attempts" db:"attempts"`
	RequestedBy         *string   `json:"requestedBy,omitempty" db:"requested_by"`
	CreatedAt           time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt           time.Time `json:"updatedAt" db:"updated_at"`
}

// FunctionSignature represents a known function signature
type FunctionSignature struct {
	ID        int             `json:"-" db:"id"`
	Selector  string          `json:"selector" db:"selector"`
	Signature string          `json:"signature" db:"signature"`
	Name      string          `json:"name" db:"name"`
	Inputs    json.RawMessage `json:"inputs,omitempty" db:"inputs"`
	CreatedAt time.Time       `json:"-" db:"created_at"`
}

// EventSignature represents a known event signature
type EventSignature struct {
	ID        int             `json:"-" db:"id"`
	Topic     string          `json:"topic" db:"topic"`
	Signature string          `json:"signature" db:"signature"`
	Name      string          `json:"name" db:"name"`
	Inputs    json.RawMessage `json:"inputs,omitempty" db:"inputs"`
	CreatedAt time.Time       `json:"-" db:"created_at"`
}

// ABIItem represents a single item in the ABI
type ABIItem struct {
	Type            string     `json:"type"`
	Name            string     `json:"name,omitempty"`
	Inputs          []ABIParam `json:"inputs,omitempty"`
	Outputs         []ABIParam `json:"outputs,omitempty"`
	StateMutability string     `json:"stateMutability,omitempty"`
	Anonymous       bool       `json:"anonymous,omitempty"`
}

// ABIParam represents a parameter in an ABI item
type ABIParam struct {
	Type       string     `json:"type"`
	Name       string     `json:"name"`
	Indexed    bool       `json:"indexed,omitempty"`
	Components []ABIParam `json:"components,omitempty"` // for tuple types
}

// VerifyRequest represents a request to verify a contract
type VerifyRequest struct {
	Network             string            `json:"network"`
	Address             string            `json:"address"`
	SourceCode          string            `json:"sourceCode,omitempty"`
	SourceFiles         map[string]string `json:"sourceFiles,omitempty"` // filename -> content
	CompilerVersion     string            `json:"compilerVersion"`
	OptimizationEnabled bool              `json:"optimizationEnabled"`
	OptimizationRuns    int               `json:"optimizationRuns"`
	ConstructorArgs     string            `json:"constructorArgs,omitempty"`
	EVMVersion          string            `json:"evmVersion,omitempty"`
	License             string            `json:"license,omitempty"`
}

// ReadContractRequest represents a request to read from a contract
type ReadContractRequest struct {
	Network  string        `json:"network"`
	Address  string        `json:"address"`
	Function string        `json:"function"`
	Args     []interface{} `json:"args,omitempty"`
}

// ReadContractResponse represents the response from reading a contract
type ReadContractResponse struct {
	Result interface{} `json:"result"`
	Raw    string      `json:"raw,omitempty"`
}

// WriteContractRequest represents a request to encode a write transaction
type WriteContractRequest struct {
	Network  string        `json:"network"`
	Address  string        `json:"address"`
	Function string        `json:"function"`
	Args     []interface{} `json:"args,omitempty"`
	Value    string        `json:"value,omitempty"` // wei
}

// WriteContractResponse represents the encoded transaction data
type WriteContractResponse struct {
	To       string `json:"to"`
	Data     string `json:"data"`
	Value    string `json:"value,omitempty"`
	GasLimit string `json:"gasLimit,omitempty"`
}

// SourcifyMetadata represents the metadata returned from Sourcify
type SourcifyMetadata struct {
	Compiler struct {
		Version string `json:"version"`
	} `json:"compiler"`
	Language string `json:"language"`
	Output   struct {
		ABI json.RawMessage `json:"abi"`
	} `json:"output"`
	Settings struct {
		CompilationTarget map[string]string `json:"compilationTarget"`
		EvmVersion        string            `json:"evmVersion"`
		Libraries         map[string]string `json:"libraries"`
		Optimizer         struct {
			Enabled bool `json:"enabled"`
			Runs    int  `json:"runs"`
		} `json:"optimizer"`
	} `json:"settings"`
	Sources map[string]struct {
		Keccak256 string   `json:"keccak256"`
		License   string   `json:"license"`
		URLs      []string `json:"urls"`
		Content   string   `json:"content"`
	} `json:"sources"`
}

// SourcifyFile represents a file in the Sourcify response
type SourcifyFile struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Content string `json:"content"`
}

// SourcifyCheckResponse represents the response from Sourcify check endpoint
type SourcifyCheckResponse struct {
	Address  string `json:"address"`
	ChainID  string `json:"chainId"`
	Status   string `json:"status"` // "perfect", "partial", "false"
	Storages struct {
		SourceMap bool `json:"sourcemap"`
	} `json:"storages,omitempty"`
}

// DetectedInterface represents a detected contract interface
type DetectedInterface struct {
	Name       string   `json:"name"`
	Standard   string   `json:"standard,omitempty"`
	Confidence float64  `json:"confidence"` // 0-1
	Functions  []string `json:"functions,omitempty"`
	Events     []string `json:"events,omitempty"`
}

// ContractAnalysis contains analysis results for a contract
type ContractAnalysis struct {
	Network            string              `json:"network"`
	Address            string              `json:"address"`
	IsContract         bool                `json:"isContract"`
	IsVerified         bool                `json:"isVerified"`
	DetectedInterfaces []DetectedInterface `json:"detectedInterfaces,omitempty"`
	HasSourceCode      bool                `json:"hasSourceCode"`
	Bytecode           string              `json:"bytecode,omitempty"`
	BytecodeHash       string              `json:"bytecodeHash,omitempty"`
}

// Verification sources
const (
	SourceSourcify   = "sourcify"
	SourceManual     = "manual"
	SourceEtherscan  = "etherscan"
	SourceBlockscout = "blockscout"
)

// Verification statuses
const (
	StatusFull    = "full"
	StatusPartial = "partial"
)

// Request statuses
const (
	RequestPending    = "pending"
	RequestProcessing = "processing"
	RequestVerified   = "verified"
	RequestFailed     = "failed"
)
