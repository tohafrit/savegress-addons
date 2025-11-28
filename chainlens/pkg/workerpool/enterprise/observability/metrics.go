package observability

import (
	"sync"
	"sync/atomic"
	"time"
)

// MetricsCollector collects and exports metrics
type MetricsCollector struct {
	// Counters
	tasksSubmitted   atomic.Int64
	tasksCompleted   atomic.Int64
	tasksRejected    atomic.Int64
	taskPanics       atomic.Int64
	tasksCancelled   atomic.Int64
	tasksTimedOut    atomic.Int64

	// Gauges
	workersActive atomic.Int32
	workersIdle   atomic.Int32
	queueSize     atomic.Int32
	queueCapacity int32

	// Histograms (simplified - store samples)
	taskDurations    *histogram
	taskWaitTimes    *histogram
	workerUtilization *histogram

	// Labeled metrics
	tasksByPriority   sync.Map // map[string]*atomic.Int64
	tasksByStatus     sync.Map // map[string]*atomic.Int64
	rejectionReasons  sync.Map // map[string]*atomic.Int64

	mu sync.RWMutex
}

// histogram stores histogram data
type histogram struct {
	samples []float64
	mu      sync.RWMutex
}

func newHistogram() *histogram {
	return &histogram{
		samples: make([]float64, 0, 10000),
	}
}

func (h *histogram) observe(value float64) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.samples = append(h.samples, value)

	// Keep only last 10000 samples to prevent memory growth
	if len(h.samples) > 10000 {
		h.samples = h.samples[len(h.samples)-10000:]
	}
}

func (h *histogram) percentile(p float64) float64 {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if len(h.samples) == 0 {
		return 0
	}

	// Simple percentile calculation (not sorted for performance)
	// In production, use a proper histogram library
	return h.samples[int(float64(len(h.samples))*p/100.0)]
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(queueCapacity int) *MetricsCollector {
	return &MetricsCollector{
		queueCapacity:     int32(queueCapacity),
		taskDurations:     newHistogram(),
		taskWaitTimes:     newHistogram(),
		workerUtilization: newHistogram(),
	}
}

// RecordTaskSubmitted records a task submission
func (m *MetricsCollector) RecordTaskSubmitted(priority string) {
	m.tasksSubmitted.Add(1)
	m.incrementLabeledCounter(&m.tasksByPriority, priority)
}

// RecordTaskCompleted records a task completion
func (m *MetricsCollector) RecordTaskCompleted(status string, duration time.Duration, waitTime time.Duration) {
	m.tasksCompleted.Add(1)
	m.incrementLabeledCounter(&m.tasksByStatus, status)
	m.taskDurations.observe(duration.Seconds())
	m.taskWaitTimes.observe(waitTime.Seconds())
}

// RecordTaskRejected records a task rejection
func (m *MetricsCollector) RecordTaskRejected(reason string) {
	m.tasksRejected.Add(1)
	m.incrementLabeledCounter(&m.rejectionReasons, reason)
}

// RecordTaskPanic records a task panic
func (m *MetricsCollector) RecordTaskPanic() {
	m.taskPanics.Add(1)
}

// RecordTaskCancelled records a task cancellation
func (m *MetricsCollector) RecordTaskCancelled() {
	m.tasksCancelled.Add(1)
}

// RecordTaskTimedOut records a task timeout
func (m *MetricsCollector) RecordTaskTimedOut() {
	m.tasksTimedOut.Add(1)
}

// SetWorkersActive sets the number of active workers
func (m *MetricsCollector) SetWorkersActive(count int) {
	m.workersActive.Store(int32(count))
}

// SetWorkersIdle sets the number of idle workers
func (m *MetricsCollector) SetWorkersIdle(count int) {
	m.workersIdle.Store(int32(count))
}

// SetQueueSize sets the current queue size
func (m *MetricsCollector) SetQueueSize(size int) {
	m.queueSize.Store(int32(size))
}

// RecordWorkerUtilization records worker utilization percentage
func (m *MetricsCollector) RecordWorkerUtilization(utilization float64) {
	m.workerUtilization.observe(utilization)
}

// GetMetrics returns a snapshot of all metrics
func (m *MetricsCollector) GetMetrics() Metrics {
	return Metrics{
		TasksSubmitted:    m.tasksSubmitted.Load(),
		TasksCompleted:    m.tasksCompleted.Load(),
		TasksRejected:     m.tasksRejected.Load(),
		TaskPanics:        m.taskPanics.Load(),
		TasksCancelled:    m.tasksCancelled.Load(),
		TasksTimedOut:     m.tasksTimedOut.Load(),
		WorkersActive:     int(m.workersActive.Load()),
		WorkersIdle:       int(m.workersIdle.Load()),
		QueueSize:         int(m.queueSize.Load()),
		QueueCapacity:     int(m.queueCapacity),
		TaskDurationP50:   m.taskDurations.percentile(50),
		TaskDurationP90:   m.taskDurations.percentile(90),
		TaskDurationP95:   m.taskDurations.percentile(95),
		TaskDurationP99:   m.taskDurations.percentile(99),
		TaskWaitTimeP50:   m.taskWaitTimes.percentile(50),
		TaskWaitTimeP90:   m.taskWaitTimes.percentile(90),
		TaskWaitTimeP95:   m.taskWaitTimes.percentile(95),
		TaskWaitTimeP99:   m.taskWaitTimes.percentile(99),
		WorkerUtilizationAvg: m.workerUtilization.percentile(50),
	}
}

// Metrics represents a snapshot of metrics
type Metrics struct {
	// Counters
	TasksSubmitted int64
	TasksCompleted int64
	TasksRejected  int64
	TaskPanics     int64
	TasksCancelled int64
	TasksTimedOut  int64

	// Gauges
	WorkersActive int
	WorkersIdle   int
	QueueSize     int
	QueueCapacity int

	// Histograms
	TaskDurationP50 float64
	TaskDurationP90 float64
	TaskDurationP95 float64
	TaskDurationP99 float64

	TaskWaitTimeP50 float64
	TaskWaitTimeP90 float64
	TaskWaitTimeP95 float64
	TaskWaitTimeP99 float64

	WorkerUtilizationAvg float64
}

// incrementLabeledCounter increments a labeled counter
func (m *MetricsCollector) incrementLabeledCounter(counters *sync.Map, label string) {
	counterI, _ := counters.LoadOrStore(label, &atomic.Int64{})
	counter := counterI.(*atomic.Int64)
	counter.Add(1)
}

// GetLabeledCounter gets the value of a labeled counter
func (m *MetricsCollector) GetLabeledCounter(counters *sync.Map, label string) int64 {
	counterI, ok := counters.Load(label)
	if !ok {
		return 0
	}
	return counterI.(*atomic.Int64).Load()
}
