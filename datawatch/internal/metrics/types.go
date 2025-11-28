package metrics

import (
	"time"
)

// MetricType represents the type of metric
type MetricType string

const (
	MetricTypeCounter   MetricType = "counter"
	MetricTypeGauge     MetricType = "gauge"
	MetricTypeHistogram MetricType = "histogram"
	MetricTypeSummary   MetricType = "summary"
)

// AggregationType represents how to aggregate metric values
type AggregationType string

const (
	AggregationSum   AggregationType = "sum"
	AggregationAvg   AggregationType = "avg"
	AggregationMin   AggregationType = "min"
	AggregationMax   AggregationType = "max"
	AggregationCount AggregationType = "count"
	AggregationP50   AggregationType = "p50"
	AggregationP90   AggregationType = "p90"
	AggregationP95   AggregationType = "p95"
	AggregationP99   AggregationType = "p99"
	AggregationRate  AggregationType = "rate"
)

// FieldType represents the inferred type of a database field
type FieldType string

const (
	FieldTypeNumeric   FieldType = "numeric"
	FieldTypeStatus    FieldType = "status"
	FieldTypeTimestamp FieldType = "timestamp"
	FieldTypeID        FieldType = "id"
	FieldTypeText      FieldType = "text"
	FieldTypeBoolean   FieldType = "boolean"
	FieldTypeUnknown   FieldType = "unknown"
)

// Metric represents a metric definition
type Metric struct {
	Name        string            `json:"name"`
	Type        MetricType        `json:"type"`
	Description string            `json:"description"`
	Table       string            `json:"table"`
	Field       string            `json:"field,omitempty"`
	Labels      []string          `json:"labels,omitempty"`
	Unit        string            `json:"unit,omitempty"`
	AutoGen     bool              `json:"auto_generated"`
	CreatedAt   time.Time         `json:"created_at"`
}

// MetricMeta contains metadata about a metric
type MetricMeta struct {
	Name        string            `json:"name"`
	Type        MetricType        `json:"type"`
	Description string            `json:"description"`
	Labels      []string          `json:"labels"`
	Unit        string            `json:"unit"`
	FirstSeen   time.Time         `json:"first_seen"`
	LastSeen    time.Time         `json:"last_seen"`
	DataPoints  int64             `json:"data_points"`
}

// DataPoint represents a single metric data point
type DataPoint struct {
	Timestamp time.Time         `json:"timestamp"`
	Value     float64           `json:"value"`
	Labels    map[string]string `json:"labels,omitempty"`
}

// TimeSeries represents a series of data points
type TimeSeries struct {
	Metric     string            `json:"metric"`
	Labels     map[string]string `json:"labels,omitempty"`
	DataPoints []DataPoint       `json:"data_points"`
}

// QueryResult represents the result of a metric query
type QueryResult struct {
	Metric      string       `json:"metric"`
	Aggregation string       `json:"aggregation"`
	From        time.Time    `json:"from"`
	To          time.Time    `json:"to"`
	Step        string       `json:"step,omitempty"`
	Series      []TimeSeries `json:"series"`
}

// CDCEvent represents a CDC event from Savegress
type CDCEvent struct {
	ID        string                 `json:"id"`
	Type      CDCEventType           `json:"type"` // INSERT, UPDATE, DELETE, DDL
	Table     string                 `json:"table"`
	Schema    string                 `json:"schema"`
	Timestamp time.Time              `json:"timestamp"`
	Before    map[string]interface{} `json:"before,omitempty"`
	After     map[string]interface{} `json:"after,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// CDCEventType represents the type of CDC event
type CDCEventType string

const (
	CDCEventInsert CDCEventType = "INSERT"
	CDCEventUpdate CDCEventType = "UPDATE"
	CDCEventDelete CDCEventType = "DELETE"
	CDCEventDDL    CDCEventType = "DDL"
)

// TableSchema represents the schema of a database table
type TableSchema struct {
	Name    string        `json:"name"`
	Schema  string        `json:"schema"`
	Columns []ColumnInfo  `json:"columns"`
}

// ColumnInfo represents information about a database column
type ColumnInfo struct {
	Name         string    `json:"name"`
	DataType     string    `json:"data_type"`
	IsNullable   bool      `json:"is_nullable"`
	IsPrimaryKey bool      `json:"is_primary_key"`
	DefaultValue *string   `json:"default_value,omitempty"`
	InferredType FieldType `json:"inferred_type"`
}

// AutoMetricConfig represents configuration for auto-generated metrics
type AutoMetricConfig struct {
	Table            string   `json:"table"`
	EnabledMetrics   []string `json:"enabled_metrics,omitempty"`
	DisabledMetrics  []string `json:"disabled_metrics,omitempty"`
	NumericFields    []string `json:"numeric_fields,omitempty"`
	StatusFields     []string `json:"status_fields,omitempty"`
	TimestampFields  []string `json:"timestamp_fields,omitempty"`
}
