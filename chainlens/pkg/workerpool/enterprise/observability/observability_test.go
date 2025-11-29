package observability

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Logger tests

func TestNewLogger(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger(buf, InfoLevel)

	if logger == nil {
		t.Fatal("NewLogger returned nil")
	}
}

func TestNewDefaultLogger(t *testing.T) {
	logger := NewDefaultLogger()
	if logger == nil {
		t.Fatal("NewDefaultLogger returned nil")
	}
}

func TestLogLevel_String(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{DebugLevel, "DEBUG"},
		{InfoLevel, "INFO"},
		{WarnLevel, "WARN"},
		{ErrorLevel, "ERROR"},
		{LogLevel(100), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.level.String(); got != tt.expected {
				t.Errorf("LogLevel.String() = %s, want %s", got, tt.expected)
			}
		})
	}
}

func TestStructuredLogger_Debug(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger(buf, DebugLevel)

	logger.Debug("test message", Field{Key: "key1", Value: "value1"})

	output := buf.String()
	if !strings.Contains(output, "DEBUG") {
		t.Errorf("Expected DEBUG level, got: %s", output)
	}
	if !strings.Contains(output, "test message") {
		t.Errorf("Expected message in output, got: %s", output)
	}
}

func TestStructuredLogger_Info(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger(buf, InfoLevel)

	logger.Info("info message")

	output := buf.String()
	if !strings.Contains(output, "INFO") {
		t.Errorf("Expected INFO level, got: %s", output)
	}
}

func TestStructuredLogger_Warn(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger(buf, WarnLevel)

	logger.Warn("warning message")

	output := buf.String()
	if !strings.Contains(output, "WARN") {
		t.Errorf("Expected WARN level, got: %s", output)
	}
}

func TestStructuredLogger_Error(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger(buf, ErrorLevel)

	logger.Error("error message")

	output := buf.String()
	if !strings.Contains(output, "ERROR") {
		t.Errorf("Expected ERROR level, got: %s", output)
	}
}

func TestStructuredLogger_LevelFiltering(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger(buf, WarnLevel)

	// Debug and Info should be filtered
	logger.Debug("debug message")
	logger.Info("info message")

	if buf.Len() != 0 {
		t.Errorf("Expected no output for filtered levels, got: %s", buf.String())
	}

	// Warn and Error should pass
	logger.Warn("warn message")
	if !strings.Contains(buf.String(), "WARN") {
		t.Errorf("Expected WARN in output")
	}
}

func TestStructuredLogger_SetLevel(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger(buf, ErrorLevel)

	// Debug shouldn't show
	logger.Debug("debug")
	if buf.Len() != 0 {
		t.Error("Debug shouldn't show at ErrorLevel")
	}

	// Change level
	logger.SetLevel(DebugLevel)
	logger.Debug("debug after level change")

	if !strings.Contains(buf.String(), "DEBUG") {
		t.Error("Debug should show after SetLevel to DebugLevel")
	}
}

func TestStructuredLogger_With(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger(buf, InfoLevel)

	childLogger := logger.With(Field{Key: "component", Value: "test"})
	childLogger.Info("child message")

	output := buf.String()
	if !strings.Contains(output, "component") {
		t.Errorf("Expected base field in output, got: %s", output)
	}
}

func TestStructuredLogger_WithSampling(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger(buf, InfoLevel).WithSampling(2, 3)

	// First 2 should pass (initial)
	logger.Info("sampled message")
	logger.Info("sampled message")

	// Then only every 3rd should pass
	buf.Reset()
	for i := 0; i < 6; i++ {
		logger.Info("sampled message")
	}

	// Should have 2 messages (every 3rd of 6 = 2)
	lines := strings.Count(buf.String(), "\n")
	if lines != 2 {
		t.Errorf("Expected 2 sampled messages, got %d", lines)
	}
}

func TestLogSampler_ShouldSample_NoSampler(t *testing.T) {
	var s *LogSampler = nil
	if !s.shouldSample("key") {
		t.Error("nil sampler should always return true")
	}
}

func TestNoOpLogger(t *testing.T) {
	logger := &NoOpLogger{}

	// These should not panic
	logger.Debug("msg")
	logger.Info("msg")
	logger.Warn("msg")
	logger.Error("msg")
	logger.SetLevel(DebugLevel)

	child := logger.With(Field{Key: "k", Value: "v"})
	if child != logger {
		t.Error("NoOpLogger.With should return itself")
	}
}

func TestLogEntry_JSON(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger(buf, InfoLevel)
	logger.Info("test", Field{Key: "number", Value: 42})

	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes()[:len(buf.Bytes())-1], &entry); err != nil {
		t.Fatalf("Failed to unmarshal log entry: %v", err)
	}

	if entry.Level != "INFO" {
		t.Errorf("Level = %s, want INFO", entry.Level)
	}
	if entry.Message != "test" {
		t.Errorf("Message = %s, want test", entry.Message)
	}
	if entry.Fields["number"] != float64(42) { // JSON numbers are float64
		t.Errorf("Field number = %v, want 42", entry.Fields["number"])
	}
}

// Metrics tests

func TestNewMetricsCollector(t *testing.T) {
	mc := NewMetricsCollector(100)
	if mc == nil {
		t.Fatal("NewMetricsCollector returned nil")
	}
}

func TestMetricsCollector_RecordTaskSubmitted(t *testing.T) {
	mc := NewMetricsCollector(100)

	mc.RecordTaskSubmitted("high")
	mc.RecordTaskSubmitted("high")
	mc.RecordTaskSubmitted("low")

	metrics := mc.GetMetrics()
	if metrics.TasksSubmitted != 3 {
		t.Errorf("TasksSubmitted = %d, want 3", metrics.TasksSubmitted)
	}
}

func TestMetricsCollector_RecordTaskCompleted(t *testing.T) {
	mc := NewMetricsCollector(100)

	mc.RecordTaskCompleted("success", 100*time.Millisecond, 10*time.Millisecond)
	mc.RecordTaskCompleted("error", 50*time.Millisecond, 5*time.Millisecond)

	metrics := mc.GetMetrics()
	if metrics.TasksCompleted != 2 {
		t.Errorf("TasksCompleted = %d, want 2", metrics.TasksCompleted)
	}
}

func TestMetricsCollector_RecordTaskRejected(t *testing.T) {
	mc := NewMetricsCollector(100)

	mc.RecordTaskRejected("queue_full")
	mc.RecordTaskRejected("queue_full")
	mc.RecordTaskRejected("timeout")

	metrics := mc.GetMetrics()
	if metrics.TasksRejected != 3 {
		t.Errorf("TasksRejected = %d, want 3", metrics.TasksRejected)
	}
}

func TestMetricsCollector_RecordTaskPanic(t *testing.T) {
	mc := NewMetricsCollector(100)

	mc.RecordTaskPanic()

	metrics := mc.GetMetrics()
	if metrics.TaskPanics != 1 {
		t.Errorf("TaskPanics = %d, want 1", metrics.TaskPanics)
	}
}

func TestMetricsCollector_RecordTaskCancelled(t *testing.T) {
	mc := NewMetricsCollector(100)

	mc.RecordTaskCancelled()

	metrics := mc.GetMetrics()
	if metrics.TasksCancelled != 1 {
		t.Errorf("TasksCancelled = %d, want 1", metrics.TasksCancelled)
	}
}

func TestMetricsCollector_RecordTaskTimedOut(t *testing.T) {
	mc := NewMetricsCollector(100)

	mc.RecordTaskTimedOut()

	metrics := mc.GetMetrics()
	if metrics.TasksTimedOut != 1 {
		t.Errorf("TasksTimedOut = %d, want 1", metrics.TasksTimedOut)
	}
}

func TestMetricsCollector_SetWorkersActive(t *testing.T) {
	mc := NewMetricsCollector(100)

	mc.SetWorkersActive(5)

	metrics := mc.GetMetrics()
	if metrics.WorkersActive != 5 {
		t.Errorf("WorkersActive = %d, want 5", metrics.WorkersActive)
	}
}

func TestMetricsCollector_SetWorkersIdle(t *testing.T) {
	mc := NewMetricsCollector(100)

	mc.SetWorkersIdle(3)

	metrics := mc.GetMetrics()
	if metrics.WorkersIdle != 3 {
		t.Errorf("WorkersIdle = %d, want 3", metrics.WorkersIdle)
	}
}

func TestMetricsCollector_SetQueueSize(t *testing.T) {
	mc := NewMetricsCollector(100)

	mc.SetQueueSize(50)

	metrics := mc.GetMetrics()
	if metrics.QueueSize != 50 {
		t.Errorf("QueueSize = %d, want 50", metrics.QueueSize)
	}
}

func TestMetricsCollector_RecordWorkerUtilization(t *testing.T) {
	mc := NewMetricsCollector(100)

	for i := 0; i < 100; i++ {
		mc.RecordWorkerUtilization(0.75)
	}

	metrics := mc.GetMetrics()
	if metrics.WorkerUtilizationAvg == 0 {
		t.Error("WorkerUtilizationAvg should not be 0")
	}
}

func TestMetricsCollector_GetLabeledCounter(t *testing.T) {
	mc := NewMetricsCollector(100)

	mc.RecordTaskSubmitted("high")
	mc.RecordTaskSubmitted("high")
	mc.RecordTaskSubmitted("low")

	highCount := mc.GetLabeledCounter(&mc.tasksByPriority, "high")
	if highCount != 2 {
		t.Errorf("high priority count = %d, want 2", highCount)
	}

	lowCount := mc.GetLabeledCounter(&mc.tasksByPriority, "low")
	if lowCount != 1 {
		t.Errorf("low priority count = %d, want 1", lowCount)
	}

	// Non-existent label
	noneCount := mc.GetLabeledCounter(&mc.tasksByPriority, "none")
	if noneCount != 0 {
		t.Errorf("none priority count = %d, want 0", noneCount)
	}
}

func TestHistogram(t *testing.T) {
	h := newHistogram()

	// Empty histogram
	if h.percentile(50) != 0 {
		t.Error("Empty histogram should return 0")
	}

	// Add samples
	for i := 0; i < 100; i++ {
		h.observe(float64(i))
	}

	// Check percentile
	p50 := h.percentile(50)
	if p50 < 40 || p50 > 60 {
		t.Errorf("P50 = %f, expected around 50", p50)
	}
}

func TestHistogram_MaxSize(t *testing.T) {
	h := newHistogram()

	// Add more than 10000 samples
	for i := 0; i < 15000; i++ {
		h.observe(float64(i))
	}

	h.mu.RLock()
	sampleCount := len(h.samples)
	h.mu.RUnlock()

	if sampleCount > 10000 {
		t.Errorf("Histogram should cap at 10000 samples, got %d", sampleCount)
	}
}

// Health checker tests

func TestNewHealthChecker(t *testing.T) {
	hc := NewHealthChecker(100)
	if hc == nil {
		t.Fatal("NewHealthChecker returned nil")
	}

	// Initial state
	if !hc.isAlive.Load() {
		t.Error("isAlive should be true initially")
	}
	if hc.isReady.Load() {
		t.Error("isReady should be false initially")
	}
	if hc.isStarted.Load() {
		t.Error("isStarted should be false initially")
	}
}

func TestHealthChecker_MarkStarted(t *testing.T) {
	hc := NewHealthChecker(100)
	hc.MarkStarted()

	if !hc.isStarted.Load() {
		t.Error("isStarted should be true after MarkStarted")
	}
	if !hc.isReady.Load() {
		t.Error("isReady should be true after MarkStarted")
	}
}

func TestHealthChecker_MarkStopped(t *testing.T) {
	hc := NewHealthChecker(100)
	hc.MarkStarted()
	hc.MarkStopped()

	if hc.isReady.Load() {
		t.Error("isReady should be false after MarkStopped")
	}
	if hc.isAlive.Load() {
		t.Error("isAlive should be false after MarkStopped")
	}
}

func TestHealthChecker_Liveness(t *testing.T) {
	hc := NewHealthChecker(100)

	// Initially alive
	if !hc.Liveness() {
		t.Error("Should be alive initially")
	}

	// Stop
	hc.MarkStopped()
	if hc.Liveness() {
		t.Error("Should not be alive after stop")
	}
}

func TestHealthChecker_Liveness_HighPanicRate(t *testing.T) {
	hc := NewHealthChecker(100)

	// Record many panics (>50%)
	for i := 0; i < 110; i++ {
		hc.RecordPanic()
	}

	if hc.Liveness() {
		t.Error("Should not be alive with high panic rate")
	}
}

func TestHealthChecker_Readiness(t *testing.T) {
	hc := NewHealthChecker(100)

	// Not ready before started
	if hc.Readiness() {
		t.Error("Should not be ready before started")
	}

	// Start but no workers
	hc.MarkStarted()
	if hc.Readiness() {
		t.Error("Should not be ready without workers")
	}

	// Add workers
	hc.UpdateMetrics(10, 5)
	if !hc.Readiness() {
		t.Error("Should be ready with workers")
	}
}

func TestHealthChecker_Readiness_QueueFull(t *testing.T) {
	hc := NewHealthChecker(100)
	hc.MarkStarted()
	hc.UpdateMetrics(95, 5) // Queue at 95%

	if hc.Readiness() {
		t.Error("Should not be ready when queue is >90% full")
	}
}

func TestHealthChecker_Startup(t *testing.T) {
	hc := NewHealthChecker(100)

	if hc.Startup() {
		t.Error("Startup should be false initially")
	}

	hc.MarkStarted()
	if !hc.Startup() {
		t.Error("Startup should be true after MarkStarted")
	}
}

func TestHealthChecker_RecordPanic(t *testing.T) {
	hc := NewHealthChecker(100)

	hc.RecordPanic()
	hc.RecordPanic()

	if hc.panicRate.Load() != 2 {
		t.Errorf("panicRate = %d, want 2", hc.panicRate.Load())
	}
	if hc.totalTasks.Load() != 2 {
		t.Errorf("totalTasks = %d, want 2", hc.totalTasks.Load())
	}
}

func TestHealthChecker_RecordTaskCompletion(t *testing.T) {
	hc := NewHealthChecker(100)

	hc.RecordTaskCompletion()

	if hc.totalTasks.Load() != 1 {
		t.Errorf("totalTasks = %d, want 1", hc.totalTasks.Load())
	}
}

func TestHealthChecker_UpdateMetrics(t *testing.T) {
	hc := NewHealthChecker(100)

	hc.UpdateMetrics(50, 10)

	if hc.queueSize.Load() != 50 {
		t.Errorf("queueSize = %d, want 50", hc.queueSize.Load())
	}
	if hc.activeWorkers.Load() != 10 {
		t.Errorf("activeWorkers = %d, want 10", hc.activeWorkers.Load())
	}
}

func TestHealthChecker_GetStatus_Healthy(t *testing.T) {
	hc := NewHealthChecker(100)
	hc.MarkStarted()
	hc.UpdateMetrics(10, 5)

	status := hc.GetStatus()

	if status.Status != HealthStatusHealthy {
		t.Errorf("Status = %s, want healthy", status.Status)
	}
	if status.Uptime == "" {
		t.Error("Uptime should not be empty")
	}
}

func TestHealthChecker_GetStatus_Degraded(t *testing.T) {
	hc := NewHealthChecker(100)
	hc.MarkStarted()
	hc.UpdateMetrics(95, 5) // Queue nearly full

	status := hc.GetStatus()

	if status.Status != HealthStatusDegraded {
		t.Errorf("Status = %s, want degraded", status.Status)
	}
}

func TestHealthChecker_GetStatus_Unhealthy(t *testing.T) {
	hc := NewHealthChecker(100)
	hc.MarkStopped()

	status := hc.GetStatus()

	if status.Status != HealthStatusUnhealthy {
		t.Errorf("Status = %s, want unhealthy", status.Status)
	}
}

func TestHealthChecker_LivenessHandler(t *testing.T) {
	hc := NewHealthChecker(100)

	// Healthy case
	req := httptest.NewRequest("GET", "/livez", nil)
	w := httptest.NewRecorder()
	hc.LivenessHandler()(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	// Unhealthy case
	hc.MarkStopped()
	w = httptest.NewRecorder()
	hc.LivenessHandler()(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

func TestHealthChecker_ReadinessHandler(t *testing.T) {
	hc := NewHealthChecker(100)
	hc.MarkStarted()
	hc.UpdateMetrics(10, 5)

	// Ready case
	req := httptest.NewRequest("GET", "/readyz", nil)
	w := httptest.NewRecorder()
	hc.ReadinessHandler()(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	// Not ready case
	hc.UpdateMetrics(95, 5)
	w = httptest.NewRecorder()
	hc.ReadinessHandler()(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

func TestHealthChecker_StartupHandler(t *testing.T) {
	hc := NewHealthChecker(100)

	// Not started
	req := httptest.NewRequest("GET", "/startupz", nil)
	w := httptest.NewRecorder()
	hc.StartupHandler()(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}

	// Started
	hc.MarkStarted()
	w = httptest.NewRecorder()
	hc.StartupHandler()(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHealthChecker_HealthzHandler(t *testing.T) {
	hc := NewHealthChecker(100)
	hc.MarkStarted()
	hc.UpdateMetrics(10, 5)

	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	hc.HealthzHandler()(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %s, want application/json", w.Header().Get("Content-Type"))
	}

	var status HealthCheck
	if err := json.NewDecoder(w.Body).Decode(&status); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if status.Status != HealthStatusHealthy {
		t.Errorf("Status = %s, want healthy", status.Status)
	}
}

// Benchmarks

func BenchmarkLogger_Info(b *testing.B) {
	buf := &bytes.Buffer{}
	logger := NewLogger(buf, InfoLevel)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message", Field{Key: "iteration", Value: i})
	}
}

func BenchmarkLogger_WithSampling(b *testing.B) {
	buf := &bytes.Buffer{}
	logger := NewLogger(buf, InfoLevel).WithSampling(10, 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message", Field{Key: "iteration", Value: i})
	}
}

func BenchmarkMetricsCollector_RecordTask(b *testing.B) {
	mc := NewMetricsCollector(1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mc.RecordTaskSubmitted("high")
		mc.RecordTaskCompleted("success", 100*time.Millisecond, 10*time.Millisecond)
	}
}

func BenchmarkHealthChecker_Liveness(b *testing.B) {
	hc := NewHealthChecker(100)
	hc.MarkStarted()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hc.Liveness()
	}
}
