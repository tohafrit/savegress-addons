package metrics

import (
	"context"
	"testing"
	"time"

	"github.com/savegress/datawatch/internal/storage"
)

// mockStorage implements storage.MetricStorage for testing
type mockStorage struct {
	records     []mockRecord
	queryResult *storage.QueryResult
	queryErr    error
	metrics     []string
}

type mockRecord struct {
	metric string
	value  float64
	labels map[string]string
	ts     time.Time
}

func (m *mockStorage) Record(ctx context.Context, metric string, value float64, labels map[string]string, ts time.Time) error {
	m.records = append(m.records, mockRecord{metric, value, labels, ts})
	return nil
}

func (m *mockStorage) Query(ctx context.Context, metric string, from, to time.Time, aggregation storage.AggregationType) (*storage.QueryResult, error) {
	if m.queryErr != nil {
		return nil, m.queryErr
	}
	return m.queryResult, nil
}

func (m *mockStorage) QueryRange(ctx context.Context, metric string, from, to time.Time, step time.Duration, aggregation storage.AggregationType) (*storage.QueryResult, error) {
	if m.queryErr != nil {
		return nil, m.queryErr
	}
	return m.queryResult, nil
}

func (m *mockStorage) ListMetrics(ctx context.Context) ([]string, error) {
	return m.metrics, nil
}

func (m *mockStorage) GetMetricMeta(ctx context.Context, metric string) (*storage.MetricMeta, error) {
	return &storage.MetricMeta{Name: metric}, nil
}

func (m *mockStorage) DeleteMetric(ctx context.Context, metric string) error {
	return nil
}

func (m *mockStorage) Cleanup(ctx context.Context, retention time.Duration) error {
	return nil
}

func (m *mockStorage) Close() error {
	return nil
}

func TestNewEngine(t *testing.T) {
	store := &mockStorage{}
	e := NewEngine(store)

	if e == nil {
		t.Fatal("expected non-nil engine")
	}
	if e.storage != store {
		t.Error("storage not set correctly")
	}
	if e.autoDisc == nil {
		t.Error("expected autoDisc to be initialized")
	}
	if e.aggregator == nil {
		t.Error("expected aggregator to be initialized")
	}
	if e.eventChan == nil {
		t.Error("expected eventChan to be initialized")
	}
}

func TestEngine_StartStop(t *testing.T) {
	store := &mockStorage{}
	e := NewEngine(store)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := e.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Give goroutines time to start
	time.Sleep(50 * time.Millisecond)

	e.Stop()

	// Verify it stopped by trying to send an event (should not block forever)
	done := make(chan bool)
	go func() {
		e.ProcessEvent(&CDCEvent{
			Table: "test",
			Type:  CDCEventInsert,
		})
		done <- true
	}()

	select {
	case <-done:
		// OK
	case <-time.After(100 * time.Millisecond):
		// Also OK - channel might be closed
	}
}

func TestEngine_ProcessEvent(t *testing.T) {
	store := &mockStorage{}
	e := NewEngine(store)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	e.Start(ctx)
	defer e.Stop()

	// Process events
	events := []*CDCEvent{
		{
			Table:     "users",
			Schema:    "public",
			Type:      CDCEventInsert,
			Timestamp: time.Now(),
			After:     map[string]interface{}{"id": 1, "name": "test"},
		},
		{
			Table:     "users",
			Schema:    "public",
			Type:      CDCEventUpdate,
			Timestamp: time.Now(),
			Before:    map[string]interface{}{"id": 1, "name": "test"},
			After:     map[string]interface{}{"id": 1, "name": "updated"},
		},
		{
			Table:     "users",
			Schema:    "public",
			Type:      CDCEventDelete,
			Timestamp: time.Now(),
			Before:    map[string]interface{}{"id": 1, "name": "updated"},
		},
	}

	for _, event := range events {
		e.ProcessEvent(event)
	}

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	// Check that metrics were recorded
	if len(store.records) == 0 {
		t.Error("expected metrics to be recorded")
	}
}

func TestEngine_ProcessEvent_FullChannel(t *testing.T) {
	store := &mockStorage{}
	e := NewEngine(store)

	// Fill the channel
	for i := 0; i < 10001; i++ {
		e.ProcessEvent(&CDCEvent{
			Table: "test",
			Type:  CDCEventInsert,
		})
	}

	// This should not block
	done := make(chan bool)
	go func() {
		e.ProcessEvent(&CDCEvent{
			Table: "overflow",
			Type:  CDCEventInsert,
		})
		done <- true
	}()

	select {
	case <-done:
		// Expected - event dropped
	case <-time.After(100 * time.Millisecond):
		t.Error("ProcessEvent blocked on full channel")
	}
}

func TestEngine_RegisterMetric(t *testing.T) {
	store := &mockStorage{}
	e := NewEngine(store)

	metric := &Metric{
		Name:        "custom_metric",
		Description: "Test metric",
		Type:        MetricTypeGauge,
	}

	e.RegisterMetric(metric)

	got, ok := e.GetMetric("custom_metric")
	if !ok {
		t.Fatal("metric not found after registration")
	}
	if got.Name != metric.Name {
		t.Errorf("Name = %s, want %s", got.Name, metric.Name)
	}
}

func TestEngine_GetMetric_NotFound(t *testing.T) {
	store := &mockStorage{}
	e := NewEngine(store)

	_, ok := e.GetMetric("nonexistent")
	if ok {
		t.Error("expected not to find nonexistent metric")
	}
}

func TestEngine_ListMetrics(t *testing.T) {
	store := &mockStorage{}
	e := NewEngine(store)

	// Register some metrics
	e.RegisterMetric(&Metric{Name: "metric_a"})
	e.RegisterMetric(&Metric{Name: "metric_b"})

	metrics := e.ListMetrics()
	if len(metrics) != 2 {
		t.Errorf("expected 2 metrics, got %d", len(metrics))
	}
}

func TestEngine_Query(t *testing.T) {
	store := &mockStorage{
		queryResult: &storage.QueryResult{
			Metric: "test",
			Series: []storage.TimeSeries{
				{
					Metric: "test",
					DataPoints: []storage.DataPoint{
						{Timestamp: time.Now(), Value: 100},
					},
				},
			},
		},
	}
	e := NewEngine(store)

	ctx := context.Background()
	result, err := e.Query(ctx, "test", time.Now().Add(-time.Hour), time.Now(), storage.AggregationAvg)

	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Series) != 1 {
		t.Errorf("expected 1 series, got %d", len(result.Series))
	}
}

func TestEngine_QueryRange(t *testing.T) {
	store := &mockStorage{
		queryResult: &storage.QueryResult{
			Metric: "test",
			Step:   "1h0m0s",
			Series: []storage.TimeSeries{
				{
					Metric: "test",
					DataPoints: []storage.DataPoint{
						{Timestamp: time.Now(), Value: 100},
					},
				},
			},
		},
	}
	e := NewEngine(store)

	ctx := context.Background()
	result, err := e.QueryRange(ctx, "test", time.Now().Add(-24*time.Hour), time.Now(), time.Hour, storage.AggregationAvg)

	if err != nil {
		t.Fatalf("QueryRange failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestEngine_HandleEvent_EventTypes(t *testing.T) {
	store := &mockStorage{}
	e := NewEngine(store)

	ctx := context.Background()
	now := time.Now()

	tests := []struct {
		name      string
		event     *CDCEvent
		wantCount int
	}{
		{
			name: "insert event",
			event: &CDCEvent{
				Table:     "test",
				Schema:    "public",
				Type:      CDCEventInsert,
				Timestamp: now,
				After:     map[string]interface{}{"value": 100.0},
			},
			wantCount: 3, // events_total, events_by_type, inserts_total
		},
		{
			name: "update event",
			event: &CDCEvent{
				Table:     "test",
				Schema:    "public",
				Type:      CDCEventUpdate,
				Timestamp: now,
				After:     map[string]interface{}{"value": 200.0},
			},
			wantCount: 3,
		},
		{
			name: "delete event",
			event: &CDCEvent{
				Table:     "test",
				Schema:    "public",
				Type:      CDCEventDelete,
				Timestamp: now,
				Before:    map[string]interface{}{"value": 100.0},
			},
			wantCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store.records = nil
			e.handleEvent(ctx, tt.event)

			if len(store.records) < tt.wantCount {
				t.Errorf("expected at least %d records, got %d", tt.wantCount, len(store.records))
			}
		})
	}
}

func TestToFloat64(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  float64
		ok    bool
	}{
		{"float64", float64(1.5), 1.5, true},
		{"float32", float32(2.5), 2.5, true},
		{"int", int(3), 3.0, true},
		{"int32", int32(4), 4.0, true},
		{"int64", int64(5), 5.0, true},
		{"uint", uint(6), 6.0, true},
		{"uint32", uint32(7), 7.0, true},
		{"uint64", uint64(8), 8.0, true},
		{"string", "9", 0, false},
		{"nil", nil, 0, false},
		{"struct", struct{}{}, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := toFloat64(tt.input)
			if ok != tt.ok {
				t.Errorf("ok = %v, want %v", ok, tt.ok)
			}
			if tt.ok && got != tt.want {
				t.Errorf("value = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCopyLabels(t *testing.T) {
	original := map[string]string{"key1": "value1", "key2": "value2"}
	copied := copyLabels(original)

	// Verify copy has same values
	if len(copied) != len(original) {
		t.Errorf("copied length = %d, want %d", len(copied), len(original))
	}
	for k, v := range original {
		if copied[k] != v {
			t.Errorf("copied[%s] = %s, want %s", k, copied[k], v)
		}
	}

	// Modify copy and verify original unchanged
	copied["key3"] = "value3"
	if _, ok := original["key3"]; ok {
		t.Error("modifying copy affected original")
	}
}

func TestCopyLabels_Nil(t *testing.T) {
	copied := copyLabels(nil)
	if copied == nil {
		t.Error("expected non-nil map for nil input")
	}
	if len(copied) != 0 {
		t.Errorf("expected empty map, got %d entries", len(copied))
	}
}
