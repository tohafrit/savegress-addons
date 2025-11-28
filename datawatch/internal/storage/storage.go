package storage

import (
	"context"
	"time"
)

// MetricStorage is the interface for metric storage backends
type MetricStorage interface {
	// Record records a metric data point
	Record(ctx context.Context, metric string, value float64, labels map[string]string, ts time.Time) error

	// Query queries metric data with aggregation
	Query(ctx context.Context, metric string, from, to time.Time, aggregation AggregationType) (*QueryResult, error)

	// QueryRange queries metric data with time steps
	QueryRange(ctx context.Context, metric string, from, to time.Time, step time.Duration, aggregation AggregationType) (*QueryResult, error)

	// ListMetrics returns all metric names
	ListMetrics(ctx context.Context) ([]string, error)

	// GetMetricMeta returns metadata for a metric
	GetMetricMeta(ctx context.Context, metric string) (*MetricMeta, error)

	// DeleteMetric deletes all data for a metric
	DeleteMetric(ctx context.Context, metric string) error

	// Cleanup removes old data based on retention policy
	Cleanup(ctx context.Context, retention time.Duration) error

	// Close closes the storage
	Close() error
}

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
	AggregationLast  AggregationType = "last"
)

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

// MetricMeta contains metadata about a metric
type MetricMeta struct {
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Labels      []string  `json:"labels"`
	FirstSeen   time.Time `json:"first_seen"`
	LastSeen    time.Time `json:"last_seen"`
	DataPoints  int64     `json:"data_points"`
}
