package resilience

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// ResourceMonitor monitors CPU and memory usage
type ResourceMonitor struct {
	maxCPUPercent  float64
	maxMemoryMB    int64
	cpuThrottle    bool
	memoryThrottle bool

	currentCPU    atomic.Value // float64
	currentMemory atomic.Int64 // bytes

	throttled    atomic.Bool
	onThrottle   func(resource string)
	onUnthrottle func(resource string)

	stopChan chan struct{}
	wg       sync.WaitGroup
}

// ResourceConfig configures resource monitoring
type ResourceConfig struct {
	MaxCPUPercent  float64
	MaxMemoryMB    int64
	CPUThrottle    bool
	MemoryThrottle bool
	OnThrottle     func(resource string)
	OnUnthrottle   func(resource string)
}

// NewResourceMonitor creates a new resource monitor
func NewResourceMonitor(config ResourceConfig) *ResourceMonitor {
	rm := &ResourceMonitor{
		maxCPUPercent:  config.MaxCPUPercent,
		maxMemoryMB:    config.MaxMemoryMB,
		cpuThrottle:    config.CPUThrottle,
		memoryThrottle: config.MemoryThrottle,
		onThrottle:     config.OnThrottle,
		onUnthrottle:   config.OnUnthrottle,
		stopChan:       make(chan struct{}),
	}

	rm.currentCPU.Store(0.0)

	return rm
}

// Start starts the resource monitor
func (rm *ResourceMonitor) Start() {
	rm.wg.Add(1)
	go rm.monitorLoop()
}

// Stop stops the resource monitor
func (rm *ResourceMonitor) Stop() {
	close(rm.stopChan)
	rm.wg.Wait()
}

// monitorLoop is the main monitoring loop
func (rm *ResourceMonitor) monitorLoop() {
	defer rm.wg.Done()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	var lastCPUTime time.Duration
	lastCheck := time.Now()

	for {
		select {
		case <-rm.stopChan:
			return
		case <-ticker.C:
			now := time.Now()
			elapsed := now.Sub(lastCheck)

			// Get CPU usage
			cpuPercent := rm.getCPUUsage(lastCPUTime, elapsed)
			rm.currentCPU.Store(cpuPercent)
			lastCPUTime = rm.getTotalCPUTime()
			lastCheck = now

			// Get memory usage
			memoryMB := rm.getMemoryUsage()
			rm.currentMemory.Store(memoryMB * 1024 * 1024) // Convert to bytes

			// Check thresholds
			shouldThrottle := false

			if cpuPercent > rm.maxCPUPercent && rm.cpuThrottle {
				shouldThrottle = true
				if rm.onThrottle != nil && !rm.throttled.Load() {
					go rm.onThrottle("cpu")
				}
			}

			if memoryMB > rm.maxMemoryMB && rm.memoryThrottle {
				shouldThrottle = true
				if rm.onThrottle != nil && !rm.throttled.Load() {
					go rm.onThrottle("memory")
				}
			}

			if shouldThrottle {
				rm.throttled.Store(true)
			} else if rm.throttled.Load() {
				rm.throttled.Store(false)
				if rm.onUnthrottle != nil {
					go rm.onUnthrottle("all")
				}
			}
		}
	}
}

// getCPUUsage calculates CPU usage percentage
func (rm *ResourceMonitor) getCPUUsage(lastCPUTime time.Duration, elapsed time.Duration) float64 {
	currentCPUTime := rm.getTotalCPUTime()
	cpuDelta := currentCPUTime - lastCPUTime

	if elapsed == 0 {
		return 0
	}

	// CPU percentage = (CPU time used / wall clock time) * 100 * num CPUs
	numCPU := float64(runtime.NumCPU())
	cpuPercent := (float64(cpuDelta) / float64(elapsed)) * 100.0 * numCPU

	return cpuPercent
}

// getTotalCPUTime returns total CPU time used by the process
func (rm *ResourceMonitor) getTotalCPUTime() time.Duration {
	// Simplified: In production, use OS-specific APIs
	// For now, approximate using runtime stats
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)

	// This is a placeholder - in production, use proper CPU time measurement
	return time.Duration(stats.PauseTotalNs)
}

// getMemoryUsage returns memory usage in MB
func (rm *ResourceMonitor) getMemoryUsage() int64 {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)

	// Return allocated memory in MB
	return int64(stats.Alloc / 1024 / 1024)
}

// GetCurrentCPU returns current CPU usage percentage
func (rm *ResourceMonitor) GetCurrentCPU() float64 {
	return rm.currentCPU.Load().(float64)
}

// GetCurrentMemory returns current memory usage in MB
func (rm *ResourceMonitor) GetCurrentMemory() int64 {
	return rm.currentMemory.Load() / 1024 / 1024
}

// IsThrottled returns true if resources are currently throttled
func (rm *ResourceMonitor) IsThrottled() bool {
	return rm.throttled.Load()
}

// GetMetrics returns current resource metrics
func (rm *ResourceMonitor) GetMetrics() ResourceMetrics {
	return ResourceMetrics{
		CPUPercent:  rm.GetCurrentCPU(),
		MemoryMB:    rm.GetCurrentMemory(),
		Throttled:   rm.IsThrottled(),
		MaxCPU:      rm.maxCPUPercent,
		MaxMemoryMB: rm.maxMemoryMB,
	}
}

// ResourceMetrics contains resource usage metrics
type ResourceMetrics struct {
	CPUPercent  float64
	MemoryMB    int64
	Throttled   bool
	MaxCPU      float64
	MaxMemoryMB int64
}

// SetLimits updates resource limits
func (rm *ResourceMonitor) SetLimits(maxCPU float64, maxMemoryMB int64) {
	rm.maxCPUPercent = maxCPU
	rm.maxMemoryMB = maxMemoryMB
}
