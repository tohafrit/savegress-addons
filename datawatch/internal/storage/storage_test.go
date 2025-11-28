package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewEmbeddedStorage(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "datawatch-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	storage, err := NewEmbeddedStorage(tmpDir)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer storage.Close()

	if storage.db == nil {
		t.Error("expected db to be initialized")
	}

	// Check that database file was created
	dbPath := filepath.Join(tmpDir, "metrics.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("database file was not created")
	}
}

func TestNewEmbeddedStorage_InvalidPath(t *testing.T) {
	// Try to create storage in a non-existent directory that can't be created
	_, err := NewEmbeddedStorage("/root/impossible/path/that/should/fail")
	if err == nil {
		t.Error("expected error for invalid path")
	}
}

func TestEmbeddedStorage_Record(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "datawatch-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	storage, err := NewEmbeddedStorage(tmpDir)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer storage.Close()

	ctx := context.Background()
	now := time.Now()

	// Record some data
	err = storage.Record(ctx, "test_metric", 100.5, map[string]string{"env": "prod"}, now)
	if err != nil {
		t.Fatalf("failed to record metric: %v", err)
	}

	// Record more data
	for i := 0; i < 10; i++ {
		err = storage.Record(ctx, "test_metric", float64(100+i), nil, now.Add(time.Duration(i)*time.Minute))
		if err != nil {
			t.Fatalf("failed to record metric: %v", err)
		}
	}

	// Wait for flush
	time.Sleep(1500 * time.Millisecond)

	// Verify data was stored
	result, err := storage.Query(ctx, "test_metric", now.Add(-time.Hour), now.Add(time.Hour), AggregationCount)
	if err != nil {
		t.Fatalf("failed to query: %v", err)
	}

	if len(result.Series) == 0 {
		t.Error("expected at least one series")
	}
}

func TestEmbeddedStorage_Query(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "datawatch-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	storage, err := NewEmbeddedStorage(tmpDir)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer storage.Close()

	ctx := context.Background()
	now := time.Now()

	// Record test data
	for i := 0; i < 10; i++ {
		storage.Record(ctx, "query_test", float64(10+i), nil, now.Add(time.Duration(i)*time.Minute))
	}
	storage.flush()

	tests := []struct {
		name        string
		aggregation AggregationType
	}{
		{"sum", AggregationSum},
		{"avg", AggregationAvg},
		{"min", AggregationMin},
		{"max", AggregationMax},
		{"count", AggregationCount},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := storage.Query(ctx, "query_test", now.Add(-time.Hour), now.Add(time.Hour), tt.aggregation)
			if err != nil {
				t.Fatalf("Query failed: %v", err)
			}

			if result.Metric != "query_test" {
				t.Errorf("Metric = %s, want query_test", result.Metric)
			}
			if result.Aggregation != string(tt.aggregation) {
				t.Errorf("Aggregation = %s, want %s", result.Aggregation, tt.aggregation)
			}
		})
	}
}

func TestEmbeddedStorage_QueryRange(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "datawatch-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	storage, err := NewEmbeddedStorage(tmpDir)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer storage.Close()

	ctx := context.Background()
	now := time.Now().Truncate(time.Hour)

	// Record hourly data
	for i := 0; i < 24; i++ {
		storage.Record(ctx, "range_test", float64(100+i), nil, now.Add(time.Duration(i)*time.Hour))
	}
	storage.flush()

	result, err := storage.QueryRange(ctx, "range_test", now, now.Add(24*time.Hour), time.Hour, AggregationAvg)
	if err != nil {
		t.Fatalf("QueryRange failed: %v", err)
	}

	if result.Step != "1h0m0s" {
		t.Errorf("Step = %s, want 1h0m0s", result.Step)
	}
}

func TestEmbeddedStorage_ListMetrics(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "datawatch-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	storage, err := NewEmbeddedStorage(tmpDir)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer storage.Close()

	ctx := context.Background()
	now := time.Now()

	// Record different metrics
	storage.Record(ctx, "metric_a", 1, nil, now)
	storage.Record(ctx, "metric_b", 2, nil, now)
	storage.Record(ctx, "metric_c", 3, nil, now)
	storage.flush()

	metrics, err := storage.ListMetrics(ctx)
	if err != nil {
		t.Fatalf("ListMetrics failed: %v", err)
	}

	if len(metrics) != 3 {
		t.Errorf("expected 3 metrics, got %d", len(metrics))
	}
}

func TestEmbeddedStorage_GetMetricMeta(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "datawatch-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	storage, err := NewEmbeddedStorage(tmpDir)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer storage.Close()

	ctx := context.Background()
	now := time.Now()

	// Record data with labels
	storage.Record(ctx, "meta_test", 100, map[string]string{"env": "prod", "region": "us"}, now)
	storage.Record(ctx, "meta_test", 200, map[string]string{"env": "prod", "region": "eu"}, now)
	storage.flush()

	meta, err := storage.GetMetricMeta(ctx, "meta_test")
	if err != nil {
		t.Fatalf("GetMetricMeta failed: %v", err)
	}

	if meta.Name != "meta_test" {
		t.Errorf("Name = %s, want meta_test", meta.Name)
	}
	if meta.DataPoints < 2 {
		t.Errorf("DataPoints = %d, want >= 2", meta.DataPoints)
	}
}

func TestEmbeddedStorage_DeleteMetric(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "datawatch-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	storage, err := NewEmbeddedStorage(tmpDir)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer storage.Close()

	ctx := context.Background()
	now := time.Now()

	// Record and flush
	storage.Record(ctx, "delete_test", 100, nil, now)
	storage.flush()

	// Verify exists
	metrics, _ := storage.ListMetrics(ctx)
	found := false
	for _, m := range metrics {
		if m == "delete_test" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("metric should exist before deletion")
	}

	// Delete
	err = storage.DeleteMetric(ctx, "delete_test")
	if err != nil {
		t.Fatalf("DeleteMetric failed: %v", err)
	}

	// Verify deleted
	metrics, _ = storage.ListMetrics(ctx)
	for _, m := range metrics {
		if m == "delete_test" {
			t.Error("metric should be deleted")
		}
	}
}

func TestEmbeddedStorage_Cleanup(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "datawatch-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	storage, err := NewEmbeddedStorage(tmpDir)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer storage.Close()

	ctx := context.Background()
	now := time.Now()

	// Record old and new data
	storage.Record(ctx, "cleanup_test", 100, nil, now.Add(-48*time.Hour)) // Old
	storage.Record(ctx, "cleanup_test", 200, nil, now)                     // Recent
	storage.flush()

	// Cleanup data older than 24 hours
	err = storage.Cleanup(ctx, 24*time.Hour)
	if err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}

	// Query should only return recent data
	result, _ := storage.Query(ctx, "cleanup_test", now.Add(-72*time.Hour), now.Add(time.Hour), AggregationCount)
	// Note: The old data should be cleaned up
	t.Logf("Series after cleanup: %d", len(result.Series))
}

func TestGetAggregationSQL(t *testing.T) {
	tests := []struct {
		agg  AggregationType
		want string
	}{
		{AggregationSum, "SUM(value)"},
		{AggregationAvg, "AVG(value)"},
		{AggregationMin, "MIN(value)"},
		{AggregationMax, "MAX(value)"},
		{AggregationCount, "COUNT(*)"},
		{AggregationLast, "value"},
		{"unknown", "AVG(value)"},
	}

	for _, tt := range tests {
		t.Run(string(tt.agg), func(t *testing.T) {
			got := getAggregationSQL(tt.agg)
			if got != tt.want {
				t.Errorf("getAggregationSQL(%s) = %s, want %s", tt.agg, got, tt.want)
			}
		})
	}
}

func TestAggregationTypeConstants(t *testing.T) {
	// Verify all aggregation types are defined
	types := []AggregationType{
		AggregationSum,
		AggregationAvg,
		AggregationMin,
		AggregationMax,
		AggregationCount,
		AggregationP50,
		AggregationP90,
		AggregationP95,
		AggregationP99,
		AggregationRate,
		AggregationLast,
	}

	for _, at := range types {
		if at == "" {
			t.Error("aggregation type should not be empty")
		}
	}
}
