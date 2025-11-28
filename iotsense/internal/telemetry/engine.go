package telemetry

import (
	"context"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/savegress/iotsense/internal/config"
	"github.com/savegress/iotsense/pkg/models"
)

// Engine handles telemetry ingestion and storage
type Engine struct {
	config      *config.TelemetryConfig
	storage     *TimeSeriesStorage
	buffer      chan *models.TelemetryPoint
	aggregator  *Aggregator
	mu          sync.RWMutex
	running     bool
	stopCh      chan struct{}
	onData      func(*models.TelemetryPoint)
}

// NewEngine creates a new telemetry engine
func NewEngine(cfg *config.TelemetryConfig) *Engine {
	return &Engine{
		config:     cfg,
		storage:    NewTimeSeriesStorage(),
		buffer:     make(chan *models.TelemetryPoint, cfg.BufferSize),
		aggregator: NewAggregator(),
		stopCh:     make(chan struct{}),
	}
}

// Start starts the telemetry engine
func (e *Engine) Start(ctx context.Context) error {
	e.mu.Lock()
	if e.running {
		e.mu.Unlock()
		return nil
	}
	e.running = true
	e.mu.Unlock()

	go e.processLoop(ctx)
	go e.flushLoop(ctx)

	return nil
}

// Stop stops the telemetry engine
func (e *Engine) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.running {
		close(e.stopCh)
		e.running = false
	}
}

// SetDataCallback sets a callback for new data points
func (e *Engine) SetDataCallback(cb func(*models.TelemetryPoint)) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.onData = cb
}

func (e *Engine) processLoop(ctx context.Context) {
	batch := make([]*models.TelemetryPoint, 0, e.config.BatchSize)

	for {
		select {
		case <-ctx.Done():
			return
		case <-e.stopCh:
			return
		case point := <-e.buffer:
			batch = append(batch, point)

			if len(batch) >= e.config.BatchSize {
				e.processBatch(batch)
				batch = batch[:0]
			}
		}
	}
}

func (e *Engine) flushLoop(ctx context.Context) {
	ticker := time.NewTicker(e.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-e.stopCh:
			return
		case <-ticker.C:
			// Force flush any pending data
			e.storage.Flush()
		}
	}
}

func (e *Engine) processBatch(batch []*models.TelemetryPoint) {
	for _, point := range batch {
		// Validate if enabled
		if e.config.ValidationEnabled && !e.validatePoint(point) {
			continue
		}

		// Store the point
		e.storage.Store(point)

		// Update aggregates
		e.aggregator.Add(point)

		// Call callback if set
		e.mu.RLock()
		cb := e.onData
		e.mu.RUnlock()
		if cb != nil {
			cb(point)
		}
	}
}

func (e *Engine) validatePoint(point *models.TelemetryPoint) bool {
	if point.DeviceID == "" {
		return false
	}
	if point.Metric == "" {
		return false
	}
	if point.Timestamp.IsZero() {
		return false
	}
	// Check if data is too old
	if time.Since(point.Timestamp) > e.config.MaxDataAge {
		return false
	}
	return true
}

// Ingest ingests a single telemetry point
func (e *Engine) Ingest(point *models.TelemetryPoint) error {
	if point.Timestamp.IsZero() {
		point.Timestamp = time.Now()
	}
	if point.Quality == "" {
		point.Quality = models.DataQualityGood
	}

	select {
	case e.buffer <- point:
		return nil
	default:
		return ErrBufferFull
	}
}

// IngestBatch ingests a batch of telemetry points
func (e *Engine) IngestBatch(batch *models.TelemetryBatch) error {
	for i := range batch.Points {
		if batch.Points[i].DeviceID == "" {
			batch.Points[i].DeviceID = batch.DeviceID
		}
		if batch.Points[i].Timestamp.IsZero() {
			batch.Points[i].Timestamp = batch.Timestamp
		}
		if err := e.Ingest(&batch.Points[i]); err != nil {
			return err
		}
	}
	return nil
}

// Query queries telemetry data
func (e *Engine) Query(query *models.TimeSeriesQuery) ([]*models.TimeSeriesResult, error) {
	return e.storage.Query(query)
}

// GetLatest gets the latest value for a metric
func (e *Engine) GetLatest(deviceID, metric string) (*models.TelemetryPoint, bool) {
	return e.storage.GetLatest(deviceID, metric)
}

// GetDeviceMetrics gets all metrics for a device
func (e *Engine) GetDeviceMetrics(deviceID string) []string {
	return e.storage.GetDeviceMetrics(deviceID)
}

// GetAggregated gets aggregated metrics
func (e *Engine) GetAggregated(deviceID, metric string, start, end time.Time, interval string) ([]*models.AggregatedMetric, error) {
	return e.aggregator.GetAggregated(deviceID, metric, start, end, interval)
}

// GetStats returns telemetry statistics
func (e *Engine) GetStats() *TelemetryStats {
	return &TelemetryStats{
		TotalPoints:    e.storage.TotalPoints(),
		BufferSize:     len(e.buffer),
		BufferCapacity: cap(e.buffer),
		DeviceCount:    e.storage.DeviceCount(),
		MetricCount:    e.storage.MetricCount(),
	}
}

// TelemetryStats contains telemetry statistics
type TelemetryStats struct {
	TotalPoints    int64 `json:"total_points"`
	BufferSize     int   `json:"buffer_size"`
	BufferCapacity int   `json:"buffer_capacity"`
	DeviceCount    int   `json:"device_count"`
	MetricCount    int   `json:"metric_count"`
}

// TimeSeriesStorage stores time series data in memory
type TimeSeriesStorage struct {
	data   map[string]map[string][]*models.TelemetryPoint // device -> metric -> points
	latest map[string]map[string]*models.TelemetryPoint   // device -> metric -> latest
	mu     sync.RWMutex
	total  int64
}

// NewTimeSeriesStorage creates a new time series storage
func NewTimeSeriesStorage() *TimeSeriesStorage {
	return &TimeSeriesStorage{
		data:   make(map[string]map[string][]*models.TelemetryPoint),
		latest: make(map[string]map[string]*models.TelemetryPoint),
	}
}

// Store stores a telemetry point
func (s *TimeSeriesStorage) Store(point *models.TelemetryPoint) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.data[point.DeviceID] == nil {
		s.data[point.DeviceID] = make(map[string][]*models.TelemetryPoint)
	}
	if s.latest[point.DeviceID] == nil {
		s.latest[point.DeviceID] = make(map[string]*models.TelemetryPoint)
	}

	s.data[point.DeviceID][point.Metric] = append(s.data[point.DeviceID][point.Metric], point)
	s.latest[point.DeviceID][point.Metric] = point
	s.total++
}

// Query queries the storage
func (s *TimeSeriesStorage) Query(query *models.TimeSeriesQuery) ([]*models.TimeSeriesResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []*models.TimeSeriesResult

	deviceIDs := query.DeviceIDs
	if len(deviceIDs) == 0 && query.DeviceID != "" {
		deviceIDs = []string{query.DeviceID}
	}

	metrics := query.Metrics
	if len(metrics) == 0 && query.Metric != "" {
		metrics = []string{query.Metric}
	}

	for _, deviceID := range deviceIDs {
		deviceData, ok := s.data[deviceID]
		if !ok {
			continue
		}

		for _, metric := range metrics {
			points, ok := deviceData[metric]
			if !ok {
				continue
			}

			result := &models.TimeSeriesResult{
				DeviceID: deviceID,
				Metric:   metric,
				Points:   make([]models.TimeSeriesPoint, 0),
			}

			for _, p := range points {
				if !p.Timestamp.Before(query.StartTime) && !p.Timestamp.After(query.EndTime) {
					result.Points = append(result.Points, models.TimeSeriesPoint{
						Timestamp: p.Timestamp,
						Value:     p.Value,
					})
				}
			}

			if len(result.Points) > 0 {
				results = append(results, result)
			}
		}
	}

	return results, nil
}

// GetLatest gets the latest value
func (s *TimeSeriesStorage) GetLatest(deviceID, metric string) (*models.TelemetryPoint, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if deviceData, ok := s.latest[deviceID]; ok {
		if point, ok := deviceData[metric]; ok {
			return point, true
		}
	}
	return nil, false
}

// GetDeviceMetrics gets all metrics for a device
func (s *TimeSeriesStorage) GetDeviceMetrics(deviceID string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var metrics []string
	if deviceData, ok := s.data[deviceID]; ok {
		for metric := range deviceData {
			metrics = append(metrics, metric)
		}
	}
	return metrics
}

// TotalPoints returns total points stored
func (s *TimeSeriesStorage) TotalPoints() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.total
}

// DeviceCount returns number of devices
func (s *TimeSeriesStorage) DeviceCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.data)
}

// MetricCount returns total unique metrics
func (s *TimeSeriesStorage) MetricCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	metrics := make(map[string]bool)
	for _, deviceData := range s.data {
		for metric := range deviceData {
			metrics[metric] = true
		}
	}
	return len(metrics)
}

// Flush flushes any pending operations
func (s *TimeSeriesStorage) Flush() {
	// In memory storage doesn't need flushing
	// Would persist to disk/database in production
}

// Aggregator aggregates telemetry data
type Aggregator struct {
	buckets map[string]map[string]*AggregationBucket // device:metric:interval -> bucket
	mu      sync.RWMutex
}

// AggregationBucket contains aggregated values
type AggregationBucket struct {
	Count  int
	Sum    float64
	Min    float64
	Max    float64
	Values []float64
}

// NewAggregator creates a new aggregator
func NewAggregator() *Aggregator {
	return &Aggregator{
		buckets: make(map[string]map[string]*AggregationBucket),
	}
}

// Add adds a point to aggregation
func (a *Aggregator) Add(point *models.TelemetryPoint) {
	a.mu.Lock()
	defer a.mu.Unlock()

	key := point.DeviceID + ":" + point.Metric

	// Get numeric value
	var value float64
	switch v := point.Value.(type) {
	case float64:
		value = v
	case float32:
		value = float64(v)
	case int:
		value = float64(v)
	case int64:
		value = float64(v)
	default:
		return // Non-numeric values not aggregated
	}

	// Create hourly bucket
	hourKey := point.Timestamp.Truncate(time.Hour).Format(time.RFC3339)

	if a.buckets[key] == nil {
		a.buckets[key] = make(map[string]*AggregationBucket)
	}

	bucket := a.buckets[key][hourKey]
	if bucket == nil {
		bucket = &AggregationBucket{
			Min: value,
			Max: value,
		}
		a.buckets[key][hourKey] = bucket
	}

	bucket.Count++
	bucket.Sum += value
	bucket.Values = append(bucket.Values, value)
	if value < bucket.Min {
		bucket.Min = value
	}
	if value > bucket.Max {
		bucket.Max = value
	}
}

// GetAggregated returns aggregated metrics
func (a *Aggregator) GetAggregated(deviceID, metric string, start, end time.Time, interval string) ([]*models.AggregatedMetric, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	key := deviceID + ":" + metric
	buckets := a.buckets[key]
	if buckets == nil {
		return nil, nil
	}

	var results []*models.AggregatedMetric

	for hourKey, bucket := range buckets {
		t, _ := time.Parse(time.RFC3339, hourKey)
		if t.Before(start) || t.After(end) {
			continue
		}

		avg := bucket.Sum / float64(bucket.Count)

		// Calculate std dev
		var variance float64
		for _, v := range bucket.Values {
			variance += (v - avg) * (v - avg)
		}
		stdDev := math.Sqrt(variance / float64(bucket.Count))

		// Calculate percentiles
		sorted := make([]float64, len(bucket.Values))
		copy(sorted, bucket.Values)
		sort.Float64s(sorted)

		var p95, p99 float64
		if len(sorted) > 0 {
			p95Index := int(float64(len(sorted)) * 0.95)
			p99Index := int(float64(len(sorted)) * 0.99)
			if p95Index >= len(sorted) {
				p95Index = len(sorted) - 1
			}
			if p99Index >= len(sorted) {
				p99Index = len(sorted) - 1
			}
			p95 = sorted[p95Index]
			p99 = sorted[p99Index]
		}

		results = append(results, &models.AggregatedMetric{
			DeviceID:  deviceID,
			Metric:    metric,
			StartTime: t,
			EndTime:   t.Add(time.Hour),
			Count:     bucket.Count,
			Min:       bucket.Min,
			Max:       bucket.Max,
			Sum:       bucket.Sum,
			Avg:       avg,
			StdDev:    stdDev,
			P95:       p95,
			P99:       p99,
		})
	}

	// Sort by time
	sort.Slice(results, func(i, j int) bool {
		return results[i].StartTime.Before(results[j].StartTime)
	})

	return results, nil
}

// Errors
var (
	ErrBufferFull = &Error{Code: "BUFFER_FULL", Message: "Telemetry buffer is full"}
)

// Error represents a telemetry error
type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string {
	return e.Message
}
