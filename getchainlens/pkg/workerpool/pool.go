package workerpool

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

// WorkerPool manages a pool of workers
type WorkerPool struct {
	// Enterprise features (initialized based on config)
	enterprise *enterpriseComponents

	config Config
	tasks  chan *Task        // Task queue
	wg     sync.WaitGroup    // Wait for workers
	ctx    context.Context   // Pool context
	cancel context.CancelFunc // Cancel function
	once   sync.Once         // Ensure single shutdown
	closed atomic.Bool       // Pool closed flag

	// Statistics
	stats *statsCollector

	// For Wait() implementation
	waitGroup sync.WaitGroup
}

// NewWorkerPool creates a new worker pool with given configuration.
// Returns error if configuration is invalid.
//
// Example:
//
//	pool, err := workerpool.NewWorkerPool(workerpool.Config{
//	    Workers: 4,
//	    QueueSize: 100,
//	    ShutdownTimeout: 5 * time.Second,
//	})
func NewWorkerPool(config Config) (*WorkerPool, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	pool := &WorkerPool{
		config: config,
		tasks:  make(chan *Task, config.QueueSize),
		ctx:    ctx,
		cancel: cancel,
		stats:  newStatsCollector(),
	}

	pool.startWorkers()

	// Initialize enterprise features if configured
	pool.initializeEnterpriseFeatures()

	return pool, nil
}

// NewDefaultWorkerPool creates a pool with sensible defaults.
// Workers = runtime.NumCPU()
// QueueSize = 1000
// ShutdownTimeout = 30s
func NewDefaultWorkerPool() *WorkerPool {
	pool, _ := NewWorkerPool(DefaultConfig())
	return pool
}

// startWorkers starts the worker goroutines
func (p *WorkerPool) startWorkers() {
	for i := 0; i < p.config.Workers; i++ {
		p.wg.Add(1)
		p.stats.incActiveWorkers()

		go p.worker(i)
	}
}

// worker is the main worker goroutine
func (p *WorkerPool) worker(workerID int) {
	defer func() {
		p.wg.Done()
		p.stats.decActiveWorkers()
	}()

	for {
		select {
		case <-p.ctx.Done():
			return // Pool shutdown
		case task, ok := <-p.tasks:
			if !ok {
				return // Channel closed
			}
			p.executeTask(task)
		}
	}
}

// executeTask executes a single task with panic recovery
func (p *WorkerPool) executeTask(task *Task) {
	defer p.waitGroup.Done()

	start := time.Now()

	defer func() {
		if r := recover(); r != nil {
			err := &TaskError{
				TaskID: task.ID,
				Err:    fmt.Errorf("panic: %v", r),
				Stack:  string(debug.Stack()),
			}
			if p.config.ErrorHandler != nil {
				p.config.ErrorHandler(err)
			}
		}

		// Record completion metrics
		duration := time.Since(start)
		p.stats.recordTaskCompletion(duration)
	}()

	// Check if context is cancelled before execution
	select {
	case <-task.Ctx.Done():
		// Task context cancelled, don't execute
		if p.config.ErrorHandler != nil {
			p.config.ErrorHandler(&TaskError{
				TaskID: task.ID,
				Err:    task.Ctx.Err(),
			})
		}
		return
	default:
	}

	// Execute the task
	if err := task.Fn(); err != nil {
		taskErr := &TaskError{
			TaskID: task.ID,
			Err:    err,
		}
		if p.config.ErrorHandler != nil {
			p.config.ErrorHandler(taskErr)
		}
	}
}

// Submit submits a task to the pool.
// Blocks if queue is full until space is available.
// Returns error if pool is closed.
//
// Example:
//
//	err := pool.Submit(func() error {
//	    // Do work
//	    return nil
//	})
func (p *WorkerPool) Submit(fn func() error) error {
	return p.SubmitWithPriority(fn, PriorityNormal)
}

// SubmitWithContext submits a task with context.
// Task will be cancelled if context is cancelled.
// Returns error if pool is closed.
func (p *WorkerPool) SubmitWithContext(ctx context.Context, fn func() error) error {
	if p.closed.Load() {
		return ErrPoolClosed
	}

	task := newTask(fn, PriorityNormal, ctx)
	p.waitGroup.Add(1)

	select {
	case <-p.ctx.Done():
		p.waitGroup.Done()
		return ErrPoolClosed
	case p.tasks <- task:
		return nil
	}
}

// TrySubmit attempts to submit a task without blocking.
// Returns ErrQueueFull if queue is full.
// Returns ErrPoolClosed if pool is closed.
//
// Example:
//
//	err := pool.TrySubmit(func() error {
//	    // Do work
//	    return nil
//	})
//	if errors.Is(err, workerpool.ErrQueueFull) {
//	    // Handle full queue
//	}
func (p *WorkerPool) TrySubmit(fn func() error) error {
	if p.closed.Load() {
		return ErrPoolClosed
	}

	task := newTask(fn, PriorityNormal, context.Background())

	select {
	case <-p.ctx.Done():
		return ErrPoolClosed
	case p.tasks <- task:
		p.waitGroup.Add(1)
		return nil
	default:
		p.stats.recordTaskRejection()
		return ErrQueueFull
	}
}

// SubmitWithTimeout submits a task with timeout.
// Waits up to timeout duration for queue space.
// Returns ErrTimeout if timeout exceeded.
func (p *WorkerPool) SubmitWithTimeout(fn func() error, timeout time.Duration) error {
	if p.closed.Load() {
		return ErrPoolClosed
	}

	task := newTask(fn, PriorityNormal, context.Background())
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-p.ctx.Done():
		return ErrPoolClosed
	case p.tasks <- task:
		p.waitGroup.Add(1)
		return nil
	case <-timer.C:
		p.stats.recordTaskRejection()
		return ErrTimeout
	}
}

// SubmitWithPriority submits a task with priority.
// High priority tasks are executed before lower priority tasks.
// Note: Priority queue not fully implemented in this version.
func (p *WorkerPool) SubmitWithPriority(fn func() error, priority Priority) error {
	if p.closed.Load() {
		return ErrPoolClosed
	}

	task := newTask(fn, priority, context.Background())
	p.waitGroup.Add(1)

	select {
	case <-p.ctx.Done():
		p.waitGroup.Done()
		return ErrPoolClosed
	case p.tasks <- task:
		return nil
	}
}

// Stop gracefully shuts down the worker pool.
// Stops accepting new tasks and waits for in-flight tasks to complete.
// Respects ShutdownTimeout from config.
// Returns error if forced shutdown occurred (timeout exceeded).
//
// Example:
//
//	if err := pool.Stop(); err != nil {
//	    log.Printf("Forced shutdown: %v", err)
//	}
func (p *WorkerPool) Stop() error {
	var shutdownErr error

	// Stop enterprise components
	p.stopEnterpriseComponents()
	p.once.Do(func() {
		// Mark as closed
		p.closed.Store(true)

		// Cancel context to signal workers
		p.cancel()

		// Close task channel (no more tasks accepted)
		close(p.tasks)

		// Wait for workers with timeout
		done := make(chan struct{})
		go func() {
			p.wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			// Graceful shutdown completed
		case <-time.After(p.config.ShutdownTimeout):
			// Timeout exceeded
			shutdownErr = ErrForcedShutdown
		}
	})

	return shutdownErr
}

// StopWithContext stops the pool respecting context cancellation.
// Returns immediately if context is cancelled.
func (p *WorkerPool) StopWithContext(ctx context.Context) error {
	var shutdownErr error

	p.once.Do(func() {
		// Mark as closed
		p.closed.Store(true)

		// Cancel context to signal workers
		p.cancel()

		// Close task channel (no more tasks accepted)
		close(p.tasks)

		// Wait for workers with timeout or context cancellation
		done := make(chan struct{})
		go func() {
			p.wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			// Graceful shutdown completed
		case <-ctx.Done():
			// Context cancelled
			shutdownErr = fmt.Errorf("shutdown cancelled: %w", ctx.Err())
		case <-time.After(p.config.ShutdownTimeout):
			// Timeout exceeded
			shutdownErr = ErrForcedShutdown
		}
	})

	return shutdownErr
}

// IsClosed returns true if pool is closed.
func (p *WorkerPool) IsClosed() bool {
	return p.closed.Load()
}

// Stats returns current pool statistics.
// Safe for concurrent access.
//
// Example:
//
//	stats := pool.Stats()
//	fmt.Printf("Active: %d, Queued: %d, Completed: %d\n",
//	    stats.ActiveWorkers, stats.QueuedTasks, stats.CompletedTasks)
func (p *WorkerPool) Stats() Stats {
	return p.stats.snapshot(len(p.tasks))
}

// Wait blocks until all queued tasks are completed.
// Does not prevent new task submission.
// Use Stop() for graceful shutdown.
func (p *WorkerPool) Wait() {
	p.waitGroup.Wait()
}
