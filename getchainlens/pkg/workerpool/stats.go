package workerpool

import (
	"sync/atomic"
	"time"
)

// Stats contains pool statistics
type Stats struct {
	ActiveWorkers  int           // Currently active workers
	QueuedTasks    int           // Tasks waiting in queue
	CompletedTasks int64         // Total completed tasks
	RejectedTasks  int64         // Total rejected tasks
	AverageLatency time.Duration // Average task execution time
	LastError      error         // Last error encountered
	Uptime         time.Duration // Pool uptime
}

// statsCollector holds internal statistics
type statsCollector struct {
	activeWorkers  atomic.Int32
	queuedTasks    atomic.Int32
	completedTasks atomic.Int64
	rejectedTasks  atomic.Int64
	totalLatency   atomic.Int64 // in nanoseconds
	startTime      time.Time
}

// newStatsCollector creates a new stats collector
func newStatsCollector() *statsCollector {
	return &statsCollector{
		startTime: time.Now(),
	}
}

// snapshot returns a snapshot of current statistics
func (s *statsCollector) snapshot(queueLen int) Stats {
	completed := s.completedTasks.Load()
	var avgLatency time.Duration
	if completed > 0 {
		avgLatency = time.Duration(s.totalLatency.Load() / completed)
	}

	return Stats{
		ActiveWorkers:  int(s.activeWorkers.Load()),
		QueuedTasks:    queueLen,
		CompletedTasks: completed,
		RejectedTasks:  s.rejectedTasks.Load(),
		AverageLatency: avgLatency,
		Uptime:         time.Since(s.startTime),
	}
}

// recordTaskCompletion records a completed task
func (s *statsCollector) recordTaskCompletion(duration time.Duration) {
	s.completedTasks.Add(1)
	s.totalLatency.Add(int64(duration))
}

// recordTaskRejection records a rejected task
func (s *statsCollector) recordTaskRejection() {
	s.rejectedTasks.Add(1)
}

// incActiveWorkers increments active worker count
func (s *statsCollector) incActiveWorkers() {
	s.activeWorkers.Add(1)
}

// decActiveWorkers decrements active worker count
func (s *statsCollector) decActiveWorkers() {
	s.activeWorkers.Add(-1)
}
