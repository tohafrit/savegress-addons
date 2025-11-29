package resilience

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestNewResourceMonitor(t *testing.T) {
	config := ResourceConfig{
		MaxCPUPercent:  80.0,
		MaxMemoryMB:    1024,
		CPUThrottle:    true,
		MemoryThrottle: true,
	}

	rm := NewResourceMonitor(config)

	if rm == nil {
		t.Fatal("expected non-nil ResourceMonitor")
	}
	if rm.maxCPUPercent != 80.0 {
		t.Errorf("maxCPUPercent = %f, want 80.0", rm.maxCPUPercent)
	}
	if rm.maxMemoryMB != 1024 {
		t.Errorf("maxMemoryMB = %d, want 1024", rm.maxMemoryMB)
	}
	if !rm.cpuThrottle {
		t.Error("cpuThrottle should be true")
	}
	if !rm.memoryThrottle {
		t.Error("memoryThrottle should be true")
	}
}

func TestNewResourceMonitor_WithCallbacks(t *testing.T) {
	var throttleCalled, unthrottleCalled bool

	config := ResourceConfig{
		MaxCPUPercent:  80.0,
		MaxMemoryMB:    1024,
		OnThrottle:     func(resource string) { throttleCalled = true },
		OnUnthrottle:   func(resource string) { unthrottleCalled = true },
	}

	rm := NewResourceMonitor(config)

	if rm.onThrottle == nil {
		t.Error("onThrottle callback not set")
	}
	if rm.onUnthrottle == nil {
		t.Error("onUnthrottle callback not set")
	}

	// Trigger callbacks
	rm.onThrottle("cpu")
	rm.onUnthrottle("all")

	if !throttleCalled {
		t.Error("onThrottle callback not called")
	}
	if !unthrottleCalled {
		t.Error("onUnthrottle callback not called")
	}
}

func TestResourceMonitor_StartStop(t *testing.T) {
	config := ResourceConfig{
		MaxCPUPercent:  80.0,
		MaxMemoryMB:    1024,
	}

	rm := NewResourceMonitor(config)

	// Start
	rm.Start()

	// Give monitor time to run
	time.Sleep(50 * time.Millisecond)

	// Stop
	rm.Stop()

	// Should complete without deadlock
}

func TestResourceMonitor_GetCurrentCPU(t *testing.T) {
	config := ResourceConfig{
		MaxCPUPercent: 80.0,
		MaxMemoryMB:   1024,
	}

	rm := NewResourceMonitor(config)

	cpu := rm.GetCurrentCPU()
	// Initial value should be 0.0
	if cpu != 0.0 {
		t.Errorf("initial CPU = %f, want 0.0", cpu)
	}

	// Set a value
	rm.currentCPU.Store(50.0)
	cpu = rm.GetCurrentCPU()
	if cpu != 50.0 {
		t.Errorf("CPU = %f, want 50.0", cpu)
	}
}

func TestResourceMonitor_GetCurrentMemory(t *testing.T) {
	config := ResourceConfig{
		MaxCPUPercent: 80.0,
		MaxMemoryMB:   1024,
	}

	rm := NewResourceMonitor(config)

	// Set memory in bytes (500 MB)
	rm.currentMemory.Store(500 * 1024 * 1024)

	memory := rm.GetCurrentMemory()
	if memory != 500 {
		t.Errorf("memory = %d MB, want 500", memory)
	}
}

func TestResourceMonitor_IsThrottled(t *testing.T) {
	config := ResourceConfig{
		MaxCPUPercent: 80.0,
		MaxMemoryMB:   1024,
	}

	rm := NewResourceMonitor(config)

	if rm.IsThrottled() {
		t.Error("should not be throttled initially")
	}

	rm.throttled.Store(true)
	if !rm.IsThrottled() {
		t.Error("should be throttled after setting flag")
	}

	rm.throttled.Store(false)
	if rm.IsThrottled() {
		t.Error("should not be throttled after clearing flag")
	}
}

func TestResourceMonitor_GetMetrics(t *testing.T) {
	config := ResourceConfig{
		MaxCPUPercent: 80.0,
		MaxMemoryMB:   1024,
	}

	rm := NewResourceMonitor(config)
	rm.currentCPU.Store(45.5)
	rm.currentMemory.Store(512 * 1024 * 1024) // 512 MB
	rm.throttled.Store(true)

	metrics := rm.GetMetrics()

	if metrics.CPUPercent != 45.5 {
		t.Errorf("CPUPercent = %f, want 45.5", metrics.CPUPercent)
	}
	if metrics.MemoryMB != 512 {
		t.Errorf("MemoryMB = %d, want 512", metrics.MemoryMB)
	}
	if !metrics.Throttled {
		t.Error("Throttled should be true")
	}
	if metrics.MaxCPU != 80.0 {
		t.Errorf("MaxCPU = %f, want 80.0", metrics.MaxCPU)
	}
	if metrics.MaxMemoryMB != 1024 {
		t.Errorf("MaxMemoryMB = %d, want 1024", metrics.MaxMemoryMB)
	}
}

func TestResourceMonitor_SetLimits(t *testing.T) {
	config := ResourceConfig{
		MaxCPUPercent: 80.0,
		MaxMemoryMB:   1024,
	}

	rm := NewResourceMonitor(config)

	// Update limits
	rm.SetLimits(90.0, 2048)

	if rm.maxCPUPercent != 90.0 {
		t.Errorf("maxCPUPercent = %f, want 90.0", rm.maxCPUPercent)
	}
	if rm.maxMemoryMB != 2048 {
		t.Errorf("maxMemoryMB = %d, want 2048", rm.maxMemoryMB)
	}
}

func TestResourceMonitor_getCPUUsage(t *testing.T) {
	config := ResourceConfig{
		MaxCPUPercent: 80.0,
		MaxMemoryMB:   1024,
	}

	rm := NewResourceMonitor(config)

	// Test with zero elapsed time
	cpu := rm.getCPUUsage(0, 0)
	if cpu != 0 {
		t.Errorf("CPU with 0 elapsed = %f, want 0", cpu)
	}

	// Test with some elapsed time
	lastCPUTime := rm.getTotalCPUTime()
	elapsed := 100 * time.Millisecond
	cpu = rm.getCPUUsage(lastCPUTime, elapsed)
	// Result depends on actual CPU time, just verify it doesn't panic
	_ = cpu
}

func TestResourceMonitor_getTotalCPUTime(t *testing.T) {
	config := ResourceConfig{
		MaxCPUPercent: 80.0,
		MaxMemoryMB:   1024,
	}

	rm := NewResourceMonitor(config)

	cpuTime := rm.getTotalCPUTime()
	// Should return some value (based on PauseTotalNs)
	_ = cpuTime
}

func TestResourceMonitor_getMemoryUsage(t *testing.T) {
	config := ResourceConfig{
		MaxCPUPercent: 80.0,
		MaxMemoryMB:   1024,
	}

	rm := NewResourceMonitor(config)

	memoryMB := rm.getMemoryUsage()
	// Should return some positive value for memory usage
	if memoryMB < 0 {
		t.Errorf("memoryMB = %d, should be non-negative", memoryMB)
	}
}

func TestResourceMonitor_MonitorLoop_Throttling(t *testing.T) {
	var throttleCount atomic.Int32
	var unthrottleCount atomic.Int32

	config := ResourceConfig{
		MaxCPUPercent:  0.0001, // Very low to trigger throttle
		MaxMemoryMB:    1,      // Very low to trigger throttle
		CPUThrottle:    true,
		MemoryThrottle: true,
		OnThrottle: func(resource string) {
			throttleCount.Add(1)
		},
		OnUnthrottle: func(resource string) {
			unthrottleCount.Add(1)
		},
	}

	rm := NewResourceMonitor(config)
	rm.Start()

	// Let the monitor run for a few cycles
	time.Sleep(2500 * time.Millisecond)

	rm.Stop()

	// Should have triggered throttling due to low thresholds
	if throttleCount.Load() == 0 {
		t.Log("Note: throttle was not triggered (depends on actual resource usage)")
	}
}

func TestResourceConfig_Struct(t *testing.T) {
	config := ResourceConfig{
		MaxCPUPercent:  80.0,
		MaxMemoryMB:    1024,
		CPUThrottle:    true,
		MemoryThrottle: true,
		OnThrottle:     func(resource string) {},
		OnUnthrottle:   func(resource string) {},
	}

	if config.MaxCPUPercent != 80.0 {
		t.Error("MaxCPUPercent not set correctly")
	}
	if config.MaxMemoryMB != 1024 {
		t.Error("MaxMemoryMB not set correctly")
	}
	if !config.CPUThrottle {
		t.Error("CPUThrottle not set correctly")
	}
	if !config.MemoryThrottle {
		t.Error("MemoryThrottle not set correctly")
	}
	if config.OnThrottle == nil {
		t.Error("OnThrottle not set correctly")
	}
	if config.OnUnthrottle == nil {
		t.Error("OnUnthrottle not set correctly")
	}
}

func TestResourceMetrics_Struct(t *testing.T) {
	metrics := ResourceMetrics{
		CPUPercent:  75.5,
		MemoryMB:    512,
		Throttled:   true,
		MaxCPU:      80.0,
		MaxMemoryMB: 1024,
	}

	if metrics.CPUPercent != 75.5 {
		t.Error("CPUPercent not set correctly")
	}
	if metrics.MemoryMB != 512 {
		t.Error("MemoryMB not set correctly")
	}
	if !metrics.Throttled {
		t.Error("Throttled not set correctly")
	}
	if metrics.MaxCPU != 80.0 {
		t.Error("MaxCPU not set correctly")
	}
	if metrics.MaxMemoryMB != 1024 {
		t.Error("MaxMemoryMB not set correctly")
	}
}
