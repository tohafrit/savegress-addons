package observability

import (
	"encoding/json"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// HealthStatus represents the health status of a component
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusDegraded  HealthStatus = "degraded"
)

// HealthChecker provides Kubernetes-style health checks
type HealthChecker struct {
	isAlive       atomic.Bool
	isReady       atomic.Bool
	isStarted     atomic.Bool
	panicRate     atomic.Int64
	totalTasks    atomic.Int64
	queueSize     atomic.Int32
	queueCapacity int32
	activeWorkers atomic.Int32
	startTime     time.Time

	mu sync.RWMutex
}

// HealthCheck represents a health check result
type HealthCheck struct {
	Status    HealthStatus       `json:"status"`
	Timestamp time.Time          `json:"timestamp"`
	Uptime    string             `json:"uptime"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(queueCapacity int) *HealthChecker {
	hc := &HealthChecker{
		queueCapacity: int32(queueCapacity),
		startTime:     time.Now(),
	}
	hc.isAlive.Store(true)
	hc.isReady.Store(false)
	hc.isStarted.Store(false)
	return hc
}

// MarkStarted marks the pool as started
func (h *HealthChecker) MarkStarted() {
	h.isStarted.Store(true)
	h.isReady.Store(true)
}

// MarkStopped marks the pool as stopped
func (h *HealthChecker) MarkStopped() {
	h.isReady.Store(false)
	h.isAlive.Store(false)
}

// RecordPanic records a task panic
func (h *HealthChecker) RecordPanic() {
	h.panicRate.Add(1)
	h.totalTasks.Add(1)
}

// RecordTaskCompletion records a successful task
func (h *HealthChecker) RecordTaskCompletion() {
	h.totalTasks.Add(1)
}

// UpdateMetrics updates health check metrics
func (h *HealthChecker) UpdateMetrics(queueSize, activeWorkers int) {
	h.queueSize.Store(int32(queueSize))
	h.activeWorkers.Store(int32(activeWorkers))
}

// Liveness checks if the pool is alive
// Returns false if:
// - Pool is explicitly stopped
// - Panic rate is too high (>50%)
func (h *HealthChecker) Liveness() bool {
	if !h.isAlive.Load() {
		return false
	}

	// Check panic rate
	total := h.totalTasks.Load()
	if total > 100 { // Only check if we have enough samples
		panics := h.panicRate.Load()
		panicRate := float64(panics) / float64(total)
		if panicRate > 0.5 {
			return false
		}
	}

	return true
}

// Readiness checks if the pool is ready to accept tasks
// Returns false if:
// - Pool is not started
// - Pool is closed
// - Queue is nearly full (>90%)
// - No active workers
func (h *HealthChecker) Readiness() bool {
	if !h.isReady.Load() {
		return false
	}

	// Check queue capacity
	queueSize := h.queueSize.Load()
	if float64(queueSize) > float64(h.queueCapacity)*0.9 {
		return false
	}

	// Check active workers
	if h.activeWorkers.Load() == 0 {
		return false
	}

	return true
}

// Startup checks if the pool has started successfully
func (h *HealthChecker) Startup() bool {
	return h.isStarted.Load()
}

// GetStatus returns detailed health status
func (h *HealthChecker) GetStatus() HealthCheck {
	status := HealthStatusHealthy

	if !h.Liveness() {
		status = HealthStatusUnhealthy
	} else if !h.Readiness() {
		status = HealthStatusDegraded
	}

	total := h.totalTasks.Load()
	panics := h.panicRate.Load()
	panicRate := 0.0
	if total > 0 {
		panicRate = float64(panics) / float64(total)
	}

	return HealthCheck{
		Status:    status,
		Timestamp: time.Now(),
		Uptime:    time.Since(h.startTime).String(),
		Details: map[string]interface{}{
			"alive":          h.isAlive.Load(),
			"ready":          h.isReady.Load(),
			"started":        h.isStarted.Load(),
			"queue_size":     h.queueSize.Load(),
			"queue_capacity": h.queueCapacity,
			"active_workers": h.activeWorkers.Load(),
			"panic_rate":     panicRate,
			"total_tasks":    total,
		},
	}
}

// LivenessHandler returns an HTTP handler for liveness probe
func (h *HealthChecker) LivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if h.Liveness() {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "alive"})
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{"status": "not alive"})
		}
	}
}

// ReadinessHandler returns an HTTP handler for readiness probe
func (h *HealthChecker) ReadinessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if h.Readiness() {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{"status": "not ready"})
		}
	}
}

// StartupHandler returns an HTTP handler for startup probe
func (h *HealthChecker) StartupHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if h.Startup() {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "started"})
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{"status": "not started"})
		}
	}
}

// HealthzHandler returns a comprehensive health status handler
func (h *HealthChecker) HealthzHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := h.GetStatus()

		w.Header().Set("Content-Type", "application/json")
		if status.Status == HealthStatusHealthy {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}

		json.NewEncoder(w).Encode(status)
	}
}
