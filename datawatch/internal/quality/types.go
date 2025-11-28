package quality

import (
	"time"
)

// RuleType defines the type of data quality rule
type RuleType string

const (
	RuleTypeCompleteness  RuleType = "completeness"
	RuleTypeValidity      RuleType = "validity"
	RuleTypeFreshness     RuleType = "freshness"
	RuleTypeConsistency   RuleType = "consistency"
	RuleTypeUniqueness    RuleType = "uniqueness"
	RuleTypeAccuracy      RuleType = "accuracy"
)

// Rule defines a data quality rule
type Rule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Table       string                 `json:"table"`
	Field       string                 `json:"field,omitempty"`
	Type        RuleType               `json:"type"`
	Condition   string                 `json:"condition"`
	Threshold   float64                `json:"threshold"`
	Severity    string                 `json:"severity"` // low, medium, high, critical
	Enabled     bool                   `json:"enabled"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// Violation represents a data quality violation
type Violation struct {
	ID           string                 `json:"id"`
	RuleID       string                 `json:"rule_id"`
	RuleName     string                 `json:"rule_name"`
	Table        string                 `json:"table"`
	Field        string                 `json:"field,omitempty"`
	RecordID     string                 `json:"record_id,omitempty"`
	Type         RuleType               `json:"type"`
	Severity     string                 `json:"severity"`
	Message      string                 `json:"message"`
	ExpectedVal  interface{}            `json:"expected_value,omitempty"`
	ActualVal    interface{}            `json:"actual_value,omitempty"`
	Context      map[string]interface{} `json:"context,omitempty"`
	DetectedAt   time.Time              `json:"detected_at"`
	Acknowledged bool                   `json:"acknowledged"`
	AckedBy      string                 `json:"acknowledged_by,omitempty"`
	AckedAt      *time.Time             `json:"acknowledged_at,omitempty"`
}

// Score represents data quality score for a table
type Score struct {
	Table          string             `json:"table"`
	OverallScore   float64            `json:"overall_score"`
	Completeness   float64            `json:"completeness"`
	Validity       float64            `json:"validity"`
	Freshness      float64            `json:"freshness"`
	Consistency    float64            `json:"consistency"`
	Uniqueness     float64            `json:"uniqueness"`
	TotalRecords   int64              `json:"total_records"`
	ValidRecords   int64              `json:"valid_records"`
	InvalidRecords int64              `json:"invalid_records"`
	RuleResults    map[string]float64 `json:"rule_results"` // rule_id -> pass rate
	CalculatedAt   time.Time          `json:"calculated_at"`
}

// Report represents a data quality report
type Report struct {
	ID             string            `json:"id"`
	Period         ReportPeriod      `json:"period"`
	OverallScore   float64           `json:"overall_score"`
	TableScores    map[string]*Score `json:"table_scores"`
	TotalRules     int               `json:"total_rules"`
	PassingRules   int               `json:"passing_rules"`
	FailingRules   int               `json:"failing_rules"`
	Violations     []*Violation      `json:"violations"`
	Trends         []TrendPoint      `json:"trends"`
	Recommendations []string         `json:"recommendations"`
	GeneratedAt    time.Time         `json:"generated_at"`
}

// ReportPeriod defines the report time period
type ReportPeriod struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// TrendPoint represents a point in the quality trend
type TrendPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Score     float64   `json:"score"`
}

// TableStats contains quality statistics for a table
type TableStats struct {
	Table            string    `json:"table"`
	TotalRecords     int64     `json:"total_records"`
	NullCounts       map[string]int64 `json:"null_counts"`
	UniqueValues     map[string]int64 `json:"unique_values"`
	InvalidCounts    map[string]int64 `json:"invalid_counts"`
	LastRecordTime   time.Time `json:"last_record_time"`
	RecordsPerMinute float64   `json:"records_per_minute"`
}

// ValidationResult represents the result of validating a record
type ValidationResult struct {
	Valid      bool         `json:"valid"`
	Table      string       `json:"table"`
	RecordID   string       `json:"record_id,omitempty"`
	Violations []*Violation `json:"violations,omitempty"`
	Score      float64      `json:"score"`
	ValidatedAt time.Time   `json:"validated_at"`
}

// Config contains quality monitoring configuration
type Config struct {
	Enabled        bool    `json:"enabled"`
	DefaultRules   bool    `json:"default_rules"`
	ScoreThreshold float64 `json:"score_threshold"`
}
