package metrics

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/savegress/datawatch/internal/storage"
)

// Engine is the main metrics processing engine
type Engine struct {
	storage     storage.MetricStorage
	autoDisc    *AutoDiscovery
	aggregator  *Aggregator

	metrics     map[string]*Metric
	metricsMu   sync.RWMutex

	eventChan   chan *CDCEvent
	stopChan    chan struct{}
	wg          sync.WaitGroup
}

// NewEngine creates a new metrics engine
func NewEngine(store storage.MetricStorage) *Engine {
	e := &Engine{
		storage:   store,
		autoDisc:  NewAutoDiscovery(),
		metrics:   make(map[string]*Metric),
		eventChan: make(chan *CDCEvent, 10000),
		stopChan:  make(chan struct{}),
	}
	e.aggregator = NewAggregator(store)
	return e
}

// Start starts the metrics engine
func (e *Engine) Start(ctx context.Context) error {
	e.wg.Add(1)
	go e.processEvents(ctx)

	e.wg.Add(1)
	go e.runAggregations(ctx)

	return nil
}

// Stop stops the metrics engine
func (e *Engine) Stop() {
	close(e.stopChan)
	e.wg.Wait()
}

// ProcessEvent processes a CDC event and generates metrics
func (e *Engine) ProcessEvent(event *CDCEvent) {
	select {
	case e.eventChan <- event:
	default:
		// Channel full, drop event (should log/metric this)
	}
}

func (e *Engine) processEvents(ctx context.Context) {
	defer e.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-e.stopChan:
			return
		case event := <-e.eventChan:
			e.handleEvent(ctx, event)
		}
	}
}

func (e *Engine) handleEvent(ctx context.Context, event *CDCEvent) {
	// Record event count metrics
	e.recordEventMetrics(ctx, event)

	// Auto-discover and record field-based metrics
	if event.Type != CDCEventDDL {
		e.recordFieldMetrics(ctx, event)
	}
}

func (e *Engine) recordEventMetrics(ctx context.Context, event *CDCEvent) {
	labels := map[string]string{
		"table":  event.Table,
		"schema": event.Schema,
	}

	// Total events counter
	metricName := fmt.Sprintf("%s_events_total", event.Table)
	e.storage.Record(ctx, metricName, 1, labels, event.Timestamp)

	// Events by type
	typeLabels := map[string]string{
		"table":  event.Table,
		"schema": event.Schema,
		"type":   string(event.Type),
	}
	e.storage.Record(ctx, fmt.Sprintf("%s_events_by_type", event.Table), 1, typeLabels, event.Timestamp)

	// Specific type metrics
	switch event.Type {
	case CDCEventInsert:
		e.storage.Record(ctx, fmt.Sprintf("%s_inserts_total", event.Table), 1, labels, event.Timestamp)
	case CDCEventUpdate:
		e.storage.Record(ctx, fmt.Sprintf("%s_updates_total", event.Table), 1, labels, event.Timestamp)
	case CDCEventDelete:
		e.storage.Record(ctx, fmt.Sprintf("%s_deletes_total", event.Table), 1, labels, event.Timestamp)
	}
}

func (e *Engine) recordFieldMetrics(ctx context.Context, event *CDCEvent) {
	data := event.After
	if data == nil {
		data = event.Before
	}
	if data == nil {
		return
	}

	labels := map[string]string{
		"table":  event.Table,
		"schema": event.Schema,
	}

	for field, value := range data {
		fieldType := e.autoDisc.InferFieldType(field, value)

		switch fieldType {
		case FieldTypeNumeric:
			if numVal, ok := toFloat64(value); ok {
				// Record the value for aggregation
				metricName := fmt.Sprintf("%s_%s", event.Table, field)
				e.storage.Record(ctx, metricName, numVal, labels, event.Timestamp)
			}

		case FieldTypeStatus:
			if strVal, ok := value.(string); ok {
				// Record status distribution
				statusLabels := copyLabels(labels)
				statusLabels[field] = strVal
				metricName := fmt.Sprintf("%s_by_%s", event.Table, field)
				e.storage.Record(ctx, metricName, 1, statusLabels, event.Timestamp)
			}
		}
	}
}

func (e *Engine) runAggregations(ctx context.Context) {
	defer e.wg.Done()

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-e.stopChan:
			return
		case <-ticker.C:
			e.aggregator.RunMinuteAggregations(ctx)
		}
	}
}

// RegisterMetric registers a custom metric
func (e *Engine) RegisterMetric(metric *Metric) {
	e.metricsMu.Lock()
	defer e.metricsMu.Unlock()
	e.metrics[metric.Name] = metric
}

// GetMetric returns a metric by name
func (e *Engine) GetMetric(name string) (*Metric, bool) {
	e.metricsMu.RLock()
	defer e.metricsMu.RUnlock()
	m, ok := e.metrics[name]
	return m, ok
}

// ListMetrics returns all registered metrics
func (e *Engine) ListMetrics() []*Metric {
	e.metricsMu.RLock()
	defer e.metricsMu.RUnlock()

	result := make([]*Metric, 0, len(e.metrics))
	for _, m := range e.metrics {
		result = append(result, m)
	}
	return result
}

// Query queries metric data
func (e *Engine) Query(ctx context.Context, metric string, from, to time.Time, aggregation storage.AggregationType) (*storage.QueryResult, error) {
	return e.storage.Query(ctx, metric, from, to, aggregation)
}

// QueryRange queries metric data with time steps
func (e *Engine) QueryRange(ctx context.Context, metric string, from, to time.Time, step time.Duration, aggregation storage.AggregationType) (*storage.QueryResult, error) {
	return e.storage.QueryRange(ctx, metric, from, to, step, aggregation)
}

// Helper functions

func toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int32:
		return float64(val), true
	case int64:
		return float64(val), true
	case uint:
		return float64(val), true
	case uint32:
		return float64(val), true
	case uint64:
		return float64(val), true
	default:
		return 0, false
	}
}

func copyLabels(labels map[string]string) map[string]string {
	copy := make(map[string]string, len(labels))
	for k, v := range labels {
		copy[k] = v
	}
	return copy
}
