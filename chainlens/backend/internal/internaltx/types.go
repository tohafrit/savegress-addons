// Package internaltx provides internal transaction tracking and storage
package internaltx

import (
	"time"
)

// TraceType constants for internal transaction types
const (
	TraceTypeCall         = "CALL"
	TraceTypeStaticCall   = "STATICCALL"
	TraceTypeDelegateCall = "DELEGATECALL"
	TraceTypeCreate       = "CREATE"
	TraceTypeCreate2      = "CREATE2"
	TraceTypeSuicide      = "SUICIDE" // selfdestruct
)

// ProcessingStatus constants
const (
	StatusPending    = "pending"
	StatusProcessing = "processing"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
	StatusSkipped    = "skipped"
)

// InternalTransaction represents an internal transaction from a call trace
type InternalTransaction struct {
	ID               int64     `json:"-" db:"id"`
	Network          string    `json:"network" db:"network"`
	TxHash           string    `json:"txHash" db:"tx_hash"`
	TraceIndex       int       `json:"traceIndex" db:"trace_index"`
	BlockNumber      int64     `json:"blockNumber" db:"block_number"`
	ParentTraceIndex *int      `json:"parentTraceIndex,omitempty" db:"parent_trace_index"`
	Depth            int       `json:"depth" db:"depth"`
	TraceType        string    `json:"traceType" db:"trace_type"`
	FromAddress      string    `json:"from" db:"from_address"`
	ToAddress        *string   `json:"to,omitempty" db:"to_address"`
	Value            string    `json:"value" db:"value"`
	Gas              *int64    `json:"gas,omitempty" db:"gas"`
	GasUsed          *int64    `json:"gasUsed,omitempty" db:"gas_used"`
	InputData        *string   `json:"input,omitempty" db:"input_data"`
	OutputData       *string   `json:"output,omitempty" db:"output_data"`
	Error            *string   `json:"error,omitempty" db:"error"`
	Reverted         bool      `json:"reverted" db:"reverted"`
	CreatedContract  *string   `json:"createdContract,omitempty" db:"created_contract"`
	Timestamp        time.Time `json:"timestamp" db:"timestamp"`
	CreatedAt        time.Time `json:"-" db:"created_at"`
}

// TraceProcessingStatus tracks which transactions have been traced
type TraceProcessingStatus struct {
	ID            int64      `json:"-" db:"id"`
	Network       string     `json:"network" db:"network"`
	BlockNumber   int64      `json:"blockNumber" db:"block_number"`
	TxHash        string     `json:"txHash" db:"tx_hash"`
	Status        string     `json:"status" db:"status"`
	ErrorMessage  *string    `json:"errorMessage,omitempty" db:"error_message"`
	RetryCount    int        `json:"retryCount" db:"retry_count"`
	LastAttemptAt *time.Time `json:"lastAttemptAt,omitempty" db:"last_attempt_at"`
	CreatedAt     time.Time  `json:"-" db:"created_at"`
	UpdatedAt     time.Time  `json:"-" db:"updated_at"`
}

// InternalTxFilter represents filter options for querying internal transactions
type InternalTxFilter struct {
	Network     string
	TxHash      string
	FromAddress string
	ToAddress   string
	TraceType   string
	MinValue    string
	BlockFrom   int64
	BlockTo     int64
	Page        int
	PageSize    int
}

// InternalTxSummary represents a summary of internal transactions for an address
type InternalTxSummary struct {
	Network            string    `json:"network"`
	Address            string    `json:"address"`
	TotalInternalTxs   int64     `json:"totalInternalTxs"`
	TotalValueReceived string    `json:"totalValueReceived"`
	TotalValueSent     string    `json:"totalValueSent"`
	LastInternalTx     time.Time `json:"lastInternalTx,omitempty"`
}

// TraceTree represents a hierarchical view of internal calls
type TraceTree struct {
	Call     *InternalTransaction `json:"call"`
	Children []*TraceTree         `json:"children,omitempty"`
}

// CallStats provides statistics about internal calls in a transaction
type CallStats struct {
	TotalCalls       int            `json:"totalCalls"`
	MaxDepth         int            `json:"maxDepth"`
	TotalValueMoved  string         `json:"totalValueMoved"`
	TotalGasUsed     int64          `json:"totalGasUsed"`
	CallsByType      map[string]int `json:"callsByType"`
	UniqueAddresses  int            `json:"uniqueAddresses"`
	ContractsCreated int            `json:"contractsCreated"`
	FailedCalls      int            `json:"failedCalls"`
}

// PendingTraceJob represents a transaction that needs to be traced
type PendingTraceJob struct {
	Network     string
	TxHash      string
	BlockNumber int64
	RetryCount  int
}

// ValidTraceTypes returns all valid trace types
func ValidTraceTypes() []string {
	return []string{
		TraceTypeCall,
		TraceTypeStaticCall,
		TraceTypeDelegateCall,
		TraceTypeCreate,
		TraceTypeCreate2,
		TraceTypeSuicide,
	}
}

// IsValidTraceType checks if a trace type is valid
func IsValidTraceType(t string) bool {
	for _, valid := range ValidTraceTypes() {
		if valid == t {
			return true
		}
	}
	return false
}

// IsContractCreation checks if the trace type creates a contract
func IsContractCreation(t string) bool {
	return t == TraceTypeCreate || t == TraceTypeCreate2
}

// NormalizeTraceType normalizes trace type to uppercase
func NormalizeTraceType(t string) string {
	switch t {
	case "call", "CALL":
		return TraceTypeCall
	case "staticcall", "STATICCALL":
		return TraceTypeStaticCall
	case "delegatecall", "DELEGATECALL":
		return TraceTypeDelegateCall
	case "create", "CREATE":
		return TraceTypeCreate
	case "create2", "CREATE2":
		return TraceTypeCreate2
	case "suicide", "SUICIDE", "selfdestruct", "SELFDESTRUCT":
		return TraceTypeSuicide
	default:
		return t
	}
}
