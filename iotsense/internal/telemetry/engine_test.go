package telemetry

import (
	"context"
	"testing"
	"time"

	"github.com/savegress/iotsense/internal/config"
	"github.com/savegress/iotsense/pkg/models"
)

func newTestConfig() *config.TelemetryConfig {
	return &config.TelemetryConfig{
		BatchSize:         10,
		FlushInterval:     100 * time.Millisecond,
		BufferSize:        100,
		MaxDataAge:        24 * time.Hour,
		ValidationEnabled: true,
	}
}

func TestNewEngine(t *testing.T) {
	cfg := newTestConfig()
	e := NewEngine(cfg)

	if e == nil {
		t.Fatal("expected non-nil engine")
	}
	if e.config != cfg {
		t.Error("config not set correctly")
	}
	if e.storage == nil {
		t.Error("storage should be initialized")
	}
	if e.buffer == nil {
		t.Error("buffer should be initialized")
	}
	if e.aggregator == nil {
		t.Error("aggregator should be initialized")
	}
}

func TestEngine_StartStop(t *testing.T) {
	cfg := newTestConfig()
	e := NewEngine(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := e.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Start again should be idempotent
	err = e.Start(ctx)
	if err != nil {
		t.Fatalf("Second Start should not fail: %v", err)
	}

	e.Stop()

	// Stop again should be idempotent
	e.Stop()
}

func TestEngine_SetDataCallback(t *testing.T) {
	cfg := newTestConfig()
	e := NewEngine(cfg)

	e.SetDataCallback(func(p *models.TelemetryPoint) {
		// Callback set
	})

	if e.onData == nil {
		t.Error("expected callback to be set")
	}
}

func TestEngine_Ingest(t *testing.T) {
	cfg := newTestConfig()
	e := NewEngine(cfg)

	point := &models.TelemetryPoint{
		DeviceID:  "device-001",
		Metric:    "temperature",
		Value:     25.5,
		Timestamp: time.Now(),
	}

	err := e.Ingest(point)
	if err != nil {
		t.Fatalf("Ingest failed: %v", err)
	}

	// Verify timestamp is set if zero
	point2 := &models.TelemetryPoint{
		DeviceID: "device-001",
		Metric:   "humidity",
		Value:    60.0,
	}
	err = e.Ingest(point2)
	if err != nil {
		t.Fatalf("Ingest failed: %v", err)
	}
	if point2.Timestamp.IsZero() {
		t.Error("Timestamp should be set")
	}

	// Verify quality is set if empty
	if point2.Quality != models.DataQualityGood {
		t.Errorf("Quality = %s, want %s", point2.Quality, models.DataQualityGood)
	}
}

func TestEngine_Ingest_BufferFull(t *testing.T) {
	cfg := &config.TelemetryConfig{
		BufferSize: 1,
	}
	e := NewEngine(cfg)

	// Fill buffer
	e.Ingest(&models.TelemetryPoint{DeviceID: "d1", Metric: "m1", Value: 1.0})

	// Next should fail
	err := e.Ingest(&models.TelemetryPoint{DeviceID: "d2", Metric: "m2", Value: 2.0})
	if err != ErrBufferFull {
		t.Errorf("expected ErrBufferFull, got %v", err)
	}
}

func TestEngine_IngestBatch(t *testing.T) {
	cfg := newTestConfig()
	e := NewEngine(cfg)

	batch := &models.TelemetryBatch{
		DeviceID:  "device-001",
		Timestamp: time.Now(),
		Points: []models.TelemetryPoint{
			{Metric: "temperature", Value: 25.0},
			{Metric: "humidity", Value: 60.0},
			{DeviceID: "other-device", Metric: "pressure", Value: 1013.0},
		},
	}

	err := e.IngestBatch(batch)
	if err != nil {
		t.Fatalf("IngestBatch failed: %v", err)
	}

	// Check that device IDs were populated
	if batch.Points[0].DeviceID != "device-001" {
		t.Error("DeviceID should be inherited from batch")
	}
	if batch.Points[2].DeviceID != "other-device" {
		t.Error("Existing DeviceID should be preserved")
	}
}

func TestEngine_ValidatePoint(t *testing.T) {
	cfg := newTestConfig()
	e := NewEngine(cfg)

	tests := []struct {
		name  string
		point *models.TelemetryPoint
		valid bool
	}{
		{
			name:  "valid point",
			point: &models.TelemetryPoint{DeviceID: "d1", Metric: "temp", Value: 25.0, Timestamp: time.Now()},
			valid: true,
		},
		{
			name:  "missing device ID",
			point: &models.TelemetryPoint{Metric: "temp", Value: 25.0, Timestamp: time.Now()},
			valid: false,
		},
		{
			name:  "missing metric",
			point: &models.TelemetryPoint{DeviceID: "d1", Value: 25.0, Timestamp: time.Now()},
			valid: false,
		},
		{
			name:  "missing timestamp",
			point: &models.TelemetryPoint{DeviceID: "d1", Metric: "temp", Value: 25.0},
			valid: false,
		},
		{
			name:  "data too old",
			point: &models.TelemetryPoint{DeviceID: "d1", Metric: "temp", Value: 25.0, Timestamp: time.Now().Add(-48 * time.Hour)},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := e.validatePoint(tt.point)
			if got != tt.valid {
				t.Errorf("validatePoint() = %v, want %v", got, tt.valid)
			}
		})
	}
}

func TestEngine_GetStats(t *testing.T) {
	cfg := newTestConfig()
	e := NewEngine(cfg)

	stats := e.GetStats()
	if stats == nil {
		t.Fatal("expected non-nil stats")
	}
	if stats.BufferCapacity != cfg.BufferSize {
		t.Errorf("BufferCapacity = %d, want %d", stats.BufferCapacity, cfg.BufferSize)
	}
}

func TestTimeSeriesStorage_Store(t *testing.T) {
	s := NewTimeSeriesStorage()

	point := &models.TelemetryPoint{
		DeviceID:  "device-001",
		Metric:    "temperature",
		Value:     25.5,
		Timestamp: time.Now(),
	}

	s.Store(point)

	if s.TotalPoints() != 1 {
		t.Errorf("TotalPoints = %d, want 1", s.TotalPoints())
	}
	if s.DeviceCount() != 1 {
		t.Errorf("DeviceCount = %d, want 1", s.DeviceCount())
	}
	if s.MetricCount() != 1 {
		t.Errorf("MetricCount = %d, want 1", s.MetricCount())
	}
}

func TestTimeSeriesStorage_GetLatest(t *testing.T) {
	s := NewTimeSeriesStorage()

	// Not found
	_, ok := s.GetLatest("device-001", "temperature")
	if ok {
		t.Error("expected not found")
	}

	// Store and get
	point := &models.TelemetryPoint{
		DeviceID:  "device-001",
		Metric:    "temperature",
		Value:     25.5,
		Timestamp: time.Now(),
	}
	s.Store(point)

	got, ok := s.GetLatest("device-001", "temperature")
	if !ok {
		t.Fatal("expected to find point")
	}
	if got.Value != 25.5 {
		t.Errorf("Value = %v, want 25.5", got.Value)
	}
}

func TestTimeSeriesStorage_GetDeviceMetrics(t *testing.T) {
	s := NewTimeSeriesStorage()

	s.Store(&models.TelemetryPoint{DeviceID: "device-001", Metric: "temp", Value: 25.0, Timestamp: time.Now()})
	s.Store(&models.TelemetryPoint{DeviceID: "device-001", Metric: "humidity", Value: 60.0, Timestamp: time.Now()})

	metrics := s.GetDeviceMetrics("device-001")
	if len(metrics) != 2 {
		t.Errorf("expected 2 metrics, got %d", len(metrics))
	}

	// Non-existent device
	metrics = s.GetDeviceMetrics("nonexistent")
	if len(metrics) != 0 {
		t.Errorf("expected 0 metrics for nonexistent device, got %d", len(metrics))
	}
}

func TestTimeSeriesStorage_Query(t *testing.T) {
	s := NewTimeSeriesStorage()

	now := time.Now()
	s.Store(&models.TelemetryPoint{DeviceID: "device-001", Metric: "temp", Value: 25.0, Timestamp: now})
	s.Store(&models.TelemetryPoint{DeviceID: "device-001", Metric: "temp", Value: 26.0, Timestamp: now.Add(time.Minute)})
	s.Store(&models.TelemetryPoint{DeviceID: "device-002", Metric: "temp", Value: 20.0, Timestamp: now})

	// Query single device, single metric
	query := &models.TimeSeriesQuery{
		DeviceID:  "device-001",
		Metric:    "temp",
		StartTime: now.Add(-time.Hour),
		EndTime:   now.Add(time.Hour),
	}
	results, err := s.Query(query)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if len(results[0].Points) != 2 {
		t.Errorf("expected 2 points, got %d", len(results[0].Points))
	}

	// Query with DeviceIDs and Metrics
	query2 := &models.TimeSeriesQuery{
		DeviceIDs: []string{"device-001", "device-002"},
		Metrics:   []string{"temp"},
		StartTime: now.Add(-time.Hour),
		EndTime:   now.Add(time.Hour),
	}
	results, err = s.Query(query2)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	// Query with time filter
	query3 := &models.TimeSeriesQuery{
		DeviceID:  "device-001",
		Metric:    "temp",
		StartTime: now.Add(30 * time.Second),
		EndTime:   now.Add(2 * time.Minute),
	}
	results, err = s.Query(query3)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if len(results[0].Points) != 1 {
		t.Errorf("expected 1 point in time range, got %d", len(results[0].Points))
	}
}

func TestTimeSeriesStorage_Flush(t *testing.T) {
	s := NewTimeSeriesStorage()
	// Flush should not panic
	s.Flush()
}

func TestAggregator_Add(t *testing.T) {
	a := NewAggregator()

	now := time.Now()
	a.Add(&models.TelemetryPoint{DeviceID: "d1", Metric: "temp", Value: 25.0, Timestamp: now})
	a.Add(&models.TelemetryPoint{DeviceID: "d1", Metric: "temp", Value: 30.0, Timestamp: now})
	a.Add(&models.TelemetryPoint{DeviceID: "d1", Metric: "temp", Value: 20.0, Timestamp: now})

	results, err := a.GetAggregated("d1", "temp", now.Add(-time.Hour), now.Add(time.Hour), "1h")
	if err != nil {
		t.Fatalf("GetAggregated failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Count != 3 {
		t.Errorf("Count = %d, want 3", results[0].Count)
	}
	if results[0].Min != 20.0 {
		t.Errorf("Min = %v, want 20.0", results[0].Min)
	}
	if results[0].Max != 30.0 {
		t.Errorf("Max = %v, want 30.0", results[0].Max)
	}
	if results[0].Sum != 75.0 {
		t.Errorf("Sum = %v, want 75.0", results[0].Sum)
	}
	if results[0].Avg != 25.0 {
		t.Errorf("Avg = %v, want 25.0", results[0].Avg)
	}
}

func TestAggregator_Add_DifferentTypes(t *testing.T) {
	a := NewAggregator()
	now := time.Now()

	// float32
	a.Add(&models.TelemetryPoint{DeviceID: "d1", Metric: "m1", Value: float32(1.5), Timestamp: now})
	// int
	a.Add(&models.TelemetryPoint{DeviceID: "d1", Metric: "m2", Value: 10, Timestamp: now})
	// int64
	a.Add(&models.TelemetryPoint{DeviceID: "d1", Metric: "m3", Value: int64(100), Timestamp: now})
	// string (non-numeric, should be ignored)
	a.Add(&models.TelemetryPoint{DeviceID: "d1", Metric: "m4", Value: "string", Timestamp: now})

	// Verify float32 aggregation
	results, _ := a.GetAggregated("d1", "m1", now.Add(-time.Hour), now.Add(time.Hour), "1h")
	if len(results) == 0 || results[0].Count != 1 {
		t.Error("float32 should be aggregated")
	}

	// Verify int aggregation
	results, _ = a.GetAggregated("d1", "m2", now.Add(-time.Hour), now.Add(time.Hour), "1h")
	if len(results) == 0 || results[0].Count != 1 {
		t.Error("int should be aggregated")
	}

	// Verify int64 aggregation
	results, _ = a.GetAggregated("d1", "m3", now.Add(-time.Hour), now.Add(time.Hour), "1h")
	if len(results) == 0 || results[0].Count != 1 {
		t.Error("int64 should be aggregated")
	}

	// Verify string is ignored
	results, _ = a.GetAggregated("d1", "m4", now.Add(-time.Hour), now.Add(time.Hour), "1h")
	if len(results) != 0 {
		t.Error("string should not be aggregated")
	}
}

func TestAggregator_GetAggregated_Empty(t *testing.T) {
	a := NewAggregator()

	now := time.Now()
	results, err := a.GetAggregated("d1", "temp", now.Add(-time.Hour), now.Add(time.Hour), "1h")
	if err != nil {
		t.Fatalf("GetAggregated failed: %v", err)
	}
	if results != nil && len(results) != 0 {
		t.Errorf("expected empty results, got %d", len(results))
	}
}

func TestAggregator_GetAggregated_TimeFilter(t *testing.T) {
	a := NewAggregator()

	now := time.Now().Truncate(time.Hour)
	pastHour := now.Add(-time.Hour)
	nextHour := now.Add(time.Hour)

	a.Add(&models.TelemetryPoint{DeviceID: "d1", Metric: "temp", Value: 20.0, Timestamp: pastHour})
	a.Add(&models.TelemetryPoint{DeviceID: "d1", Metric: "temp", Value: 25.0, Timestamp: now})
	a.Add(&models.TelemetryPoint{DeviceID: "d1", Metric: "temp", Value: 30.0, Timestamp: nextHour})

	// Query only current hour
	results, _ := a.GetAggregated("d1", "temp", now, now.Add(30*time.Minute), "1h")
	if len(results) != 1 {
		t.Errorf("expected 1 result for current hour, got %d", len(results))
	}
}

func TestError_Error(t *testing.T) {
	err := ErrBufferFull
	if err.Error() != "Telemetry buffer is full" {
		t.Errorf("Error() = %s, want 'Telemetry buffer is full'", err.Error())
	}
}

func TestTelemetryStats_Struct(t *testing.T) {
	stats := &TelemetryStats{
		TotalPoints:    1000,
		BufferSize:     50,
		BufferCapacity: 100,
		DeviceCount:    10,
		MetricCount:    25,
	}

	if stats.TotalPoints != 1000 {
		t.Error("TotalPoints not set")
	}
	if stats.DeviceCount != 10 {
		t.Error("DeviceCount not set")
	}
}

func TestAggregationBucket_Struct(t *testing.T) {
	bucket := &AggregationBucket{
		Count:  100,
		Sum:    2500.0,
		Min:    10.0,
		Max:    50.0,
		Values: []float64{10.0, 25.0, 50.0},
	}

	if bucket.Count != 100 {
		t.Error("Count not set")
	}
}

func TestEngine_Query(t *testing.T) {
	cfg := newTestConfig()
	e := NewEngine(cfg)

	// Store directly via storage
	now := time.Now()
	e.storage.Store(&models.TelemetryPoint{DeviceID: "d1", Metric: "temp", Value: 25.0, Timestamp: now})
	e.storage.Store(&models.TelemetryPoint{DeviceID: "d1", Metric: "temp", Value: 26.0, Timestamp: now.Add(time.Minute)})

	query := &models.TimeSeriesQuery{
		DeviceID:  "d1",
		Metric:    "temp",
		StartTime: now.Add(-time.Hour),
		EndTime:   now.Add(time.Hour),
	}

	results, err := e.Query(query)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if len(results[0].Points) != 2 {
		t.Errorf("expected 2 points, got %d", len(results[0].Points))
	}
}

func TestEngine_GetLatest(t *testing.T) {
	cfg := newTestConfig()
	e := NewEngine(cfg)

	// Not found
	_, ok := e.GetLatest("d1", "temp")
	if ok {
		t.Error("expected not found")
	}

	// Store and get
	e.storage.Store(&models.TelemetryPoint{DeviceID: "d1", Metric: "temp", Value: 25.0, Timestamp: time.Now()})

	point, ok := e.GetLatest("d1", "temp")
	if !ok {
		t.Fatal("expected to find point")
	}
	if point.Value != 25.0 {
		t.Errorf("Value = %v, want 25.0", point.Value)
	}
}

func TestEngine_GetDeviceMetrics(t *testing.T) {
	cfg := newTestConfig()
	e := NewEngine(cfg)

	e.storage.Store(&models.TelemetryPoint{DeviceID: "d1", Metric: "temp", Value: 25.0, Timestamp: time.Now()})
	e.storage.Store(&models.TelemetryPoint{DeviceID: "d1", Metric: "humidity", Value: 60.0, Timestamp: time.Now()})

	metrics := e.GetDeviceMetrics("d1")
	if len(metrics) != 2 {
		t.Errorf("expected 2 metrics, got %d", len(metrics))
	}
}

func TestEngine_GetAggregated(t *testing.T) {
	cfg := newTestConfig()
	e := NewEngine(cfg)

	now := time.Now()
	e.aggregator.Add(&models.TelemetryPoint{DeviceID: "d1", Metric: "temp", Value: 25.0, Timestamp: now})
	e.aggregator.Add(&models.TelemetryPoint{DeviceID: "d1", Metric: "temp", Value: 30.0, Timestamp: now})

	results, err := e.GetAggregated("d1", "temp", now.Add(-time.Hour), now.Add(time.Hour), "1h")
	if err != nil {
		t.Fatalf("GetAggregated failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Count != 2 {
		t.Errorf("Count = %d, want 2", results[0].Count)
	}
}

func TestEngine_ProcessBatch(t *testing.T) {
	cfg := newTestConfig()
	cfg.ValidationEnabled = true
	e := NewEngine(cfg)

	var callbackCount int
	e.SetDataCallback(func(p *models.TelemetryPoint) {
		callbackCount++
	})

	batch := []*models.TelemetryPoint{
		{DeviceID: "d1", Metric: "temp", Value: 25.0, Timestamp: time.Now()},
		{DeviceID: "d2", Metric: "humidity", Value: 60.0, Timestamp: time.Now()},
	}

	e.processBatch(batch)

	if e.storage.TotalPoints() != 2 {
		t.Errorf("expected 2 points stored, got %d", e.storage.TotalPoints())
	}
	if callbackCount != 2 {
		t.Errorf("expected 2 callbacks, got %d", callbackCount)
	}
}

func TestEngine_ProcessBatch_ValidationFails(t *testing.T) {
	cfg := newTestConfig()
	cfg.ValidationEnabled = true
	e := NewEngine(cfg)

	batch := []*models.TelemetryPoint{
		{DeviceID: "", Metric: "temp", Value: 25.0, Timestamp: time.Now()}, // Invalid - no device ID
		{DeviceID: "d1", Metric: "", Value: 60.0, Timestamp: time.Now()},   // Invalid - no metric
	}

	e.processBatch(batch)

	if e.storage.TotalPoints() != 0 {
		t.Errorf("expected 0 points (validation failed), got %d", e.storage.TotalPoints())
	}
}

func TestEngine_ProcessBatch_ValidationDisabled(t *testing.T) {
	cfg := newTestConfig()
	cfg.ValidationEnabled = false
	e := NewEngine(cfg)

	batch := []*models.TelemetryPoint{
		{DeviceID: "", Metric: "temp", Value: 25.0, Timestamp: time.Now()}, // Would fail validation
	}

	e.processBatch(batch)

	// With validation disabled, all points should be stored
	if e.storage.TotalPoints() != 1 {
		t.Errorf("expected 1 point (validation disabled), got %d", e.storage.TotalPoints())
	}
}
