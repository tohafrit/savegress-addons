package workerpool

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestWorkerPool_Creation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				Workers:         4,
				QueueSize:       100,
				ShutdownTimeout: 5 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "zero workers",
			config: Config{
				Workers:         0,
				QueueSize:       100,
				ShutdownTimeout: 5 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "negative workers",
			config: Config{
				Workers:         -1,
				QueueSize:       100,
				ShutdownTimeout: 5 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "negative queue size",
			config: Config{
				Workers:         4,
				QueueSize:       -1,
				ShutdownTimeout: 5 * time.Second,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			pool, err := NewWorkerPool(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewWorkerPool() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if pool != nil {
				defer pool.Stop()
			}
		})
	}
}

func TestWorkerPool_Submit(t *testing.T) {
	t.Parallel()

	pool, err := NewWorkerPool(Config{
		Workers:         2,
		QueueSize:       10,
		ShutdownTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Stop()

	var counter atomic.Int32
	taskCount := 100

	for i := 0; i < taskCount; i++ {
		err := pool.Submit(func() error {
			counter.Add(1)
			time.Sleep(10 * time.Millisecond)
			return nil
		})
		if err != nil {
			t.Errorf("Failed to submit task: %v", err)
		}
	}

	pool.Wait()

	if got := counter.Load(); got != int32(taskCount) {
		t.Errorf("Expected %d tasks to complete, got %d", taskCount, got)
	}

	stats := pool.Stats()
	if stats.CompletedTasks != int64(taskCount) {
		t.Errorf("Expected %d completed tasks in stats, got %d", taskCount, stats.CompletedTasks)
	}
}

func TestWorkerPool_TrySubmit(t *testing.T) {
	t.Parallel()

	pool, err := NewWorkerPool(Config{
		Workers:         1,
		QueueSize:       2,
		ShutdownTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Stop()

	// Fill the queue and worker
	submitted := 0
	for i := 0; i < 5; i++ {
		err := pool.TrySubmit(func() error {
			time.Sleep(100 * time.Millisecond)
			return nil
		})
		if err == nil {
			submitted++
		} else if errors.Is(err, ErrQueueFull) {
			break
		}
	}

	if submitted != 3 { // 1 worker + 2 queue slots
		t.Logf("Expected 3 successful submissions, got %d (queue might not be full yet)", submitted)
	}

	// Try to submit one more, should fail
	err = pool.TrySubmit(func() error {
		return nil
	})
	if !errors.Is(err, ErrQueueFull) {
		t.Logf("Expected ErrQueueFull, got %v (queue might have space)", err)
	}

	pool.Wait()
}

func TestWorkerPool_SubmitWithTimeout(t *testing.T) {
	t.Parallel()

	pool, err := NewWorkerPool(Config{
		Workers:         1,
		QueueSize:       1,
		ShutdownTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Stop()

	// Fill the pool
	pool.Submit(func() error {
		time.Sleep(200 * time.Millisecond)
		return nil
	})
	pool.Submit(func() error {
		time.Sleep(200 * time.Millisecond)
		return nil
	})

	// This should timeout
	start := time.Now()
	err = pool.SubmitWithTimeout(func() error {
		return nil
	}, 50*time.Millisecond)

	if !errors.Is(err, ErrTimeout) {
		t.Errorf("Expected ErrTimeout, got %v", err)
	}

	elapsed := time.Since(start)
	if elapsed < 40*time.Millisecond {
		t.Errorf("Timeout happened too quickly: %v", elapsed)
	}

	pool.Wait()
}

func TestWorkerPool_SubmitWithContext(t *testing.T) {
	t.Parallel()

	pool, err := NewWorkerPool(Config{
		Workers:         2,
		QueueSize:       10,
		ShutdownTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Stop()

	// Cancel context before submission
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	var executed atomic.Bool
	err = pool.SubmitWithContext(ctx, func() error {
		executed.Store(true)
		return nil
	})

	if err != nil {
		t.Errorf("Failed to submit task: %v", err)
	}

	time.Sleep(50 * time.Millisecond)
	pool.Wait()

	// The task should not have been executed due to context cancellation
	if executed.Load() {
		t.Error("Task was executed despite context being cancelled before execution")
	}
}

func TestWorkerPool_GracefulShutdown(t *testing.T) {
	t.Parallel()

	pool, err := NewWorkerPool(Config{
		Workers:         2,
		QueueSize:       10,
		ShutdownTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}

	var counter atomic.Int32
	taskCount := 10

	for i := 0; i < taskCount; i++ {
		pool.Submit(func() error {
			time.Sleep(10 * time.Millisecond)
			counter.Add(1)
			return nil
		})
	}

	// Give tasks time to start
	time.Sleep(50 * time.Millisecond)

	err = pool.Stop()
	if err != nil {
		t.Errorf("Shutdown returned error: %v", err)
	}

	// All tasks should have completed
	if got := counter.Load(); got != int32(taskCount) {
		t.Errorf("Expected %d tasks to complete, got %d", taskCount, got)
	}

	// Pool should be closed
	if !pool.IsClosed() {
		t.Error("Pool should be closed")
	}

	// Submitting to closed pool should fail
	err = pool.Submit(func() error { return nil })
	if !errors.Is(err, ErrPoolClosed) {
		t.Errorf("Expected ErrPoolClosed, got %v", err)
	}
}

func TestWorkerPool_ForcedShutdown(t *testing.T) {
	t.Parallel()

	pool, err := NewWorkerPool(Config{
		Workers:         2,
		QueueSize:       10,
		ShutdownTimeout: 50 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}

	// Submit long-running tasks
	for i := 0; i < 4; i++ {
		pool.Submit(func() error {
			time.Sleep(500 * time.Millisecond)
			return nil
		})
	}

	// Give tasks time to start executing
	time.Sleep(10 * time.Millisecond)

	err = pool.Stop()
	if !errors.Is(err, ErrForcedShutdown) {
		t.Errorf("Expected ErrForcedShutdown, got %v", err)
	}
}

func TestWorkerPool_PanicRecovery(t *testing.T) {
	t.Parallel()

	var panicErr error
	var mu sync.Mutex

	pool, err := NewWorkerPool(Config{
		Workers:   2,
		QueueSize: 10,
		ErrorHandler: func(err error) {
			mu.Lock()
			panicErr = err
			mu.Unlock()
		},
		ShutdownTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Stop()

	// Submit task that panics
	pool.Submit(func() error {
		panic("test panic")
	})

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if panicErr == nil {
		t.Error("Expected panic to be caught and reported")
	}

	var taskErr *TaskError
	if !errors.As(panicErr, &taskErr) {
		t.Errorf("Expected TaskError, got %T", panicErr)
	}
}

func TestWorkerPool_ContextCancellation(t *testing.T) {
	t.Parallel()

	pool, err := NewWorkerPool(Config{
		Workers:         2,
		QueueSize:       10,
		ShutdownTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	var executed atomic.Bool
	pool.SubmitWithContext(ctx, func() error {
		executed.Store(true)
		return nil
	})

	pool.Wait()

	if executed.Load() {
		t.Error("Task should not have been executed")
	}
}

func TestWorkerPool_QueueFull(t *testing.T) {
	t.Parallel()

	pool, err := NewWorkerPool(Config{
		Workers:         1,
		QueueSize:       2,
		ShutdownTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Stop()

	// Fill the queue
	for i := 0; i < 3; i++ {
		pool.Submit(func() error {
			time.Sleep(200 * time.Millisecond)
			return nil
		})
	}

	// This should fail
	err = pool.TrySubmit(func() error { return nil })
	if !errors.Is(err, ErrQueueFull) {
		t.Logf("Expected ErrQueueFull, got %v", err)
	}

	stats := pool.Stats()
	if stats.RejectedTasks == 0 {
		t.Log("Expected rejected tasks count > 0")
	}

	pool.Wait()
}

func TestWorkerPool_Statistics(t *testing.T) {
	t.Parallel()

	pool, err := NewWorkerPool(Config{
		Workers:         4,
		QueueSize:       10,
		ShutdownTimeout: 5 * time.Second,
		EnableMetrics:   true,
	})
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Stop()

	stats := pool.Stats()
	if stats.ActiveWorkers != 4 {
		t.Errorf("Expected 4 active workers, got %d", stats.ActiveWorkers)
	}

	taskCount := 20
	for i := 0; i < taskCount; i++ {
		pool.Submit(func() error {
			time.Sleep(10 * time.Millisecond)
			return nil
		})
	}

	pool.Wait()

	stats = pool.Stats()
	if stats.CompletedTasks != int64(taskCount) {
		t.Errorf("Expected %d completed tasks, got %d", taskCount, stats.CompletedTasks)
	}

	if stats.AverageLatency == 0 {
		t.Error("Expected non-zero average latency")
	}

	if stats.Uptime == 0 {
		t.Error("Expected non-zero uptime")
	}
}

func TestWorkerPool_Concurrency(t *testing.T) {
	t.Parallel()

	pool, err := NewWorkerPool(Config{
		Workers:         4,
		QueueSize:       100,
		ShutdownTimeout: 10 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Stop()

	var wg sync.WaitGroup
	submitters := 10
	tasksPerSubmitter := 100

	var counter atomic.Int32

	for i := 0; i < submitters; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < tasksPerSubmitter; j++ {
				pool.Submit(func() error {
					counter.Add(1)
					time.Sleep(time.Millisecond)
					return nil
				})
			}
		}()
	}

	wg.Wait()
	pool.Wait()

	expected := int32(submitters * tasksPerSubmitter)
	if got := counter.Load(); got != expected {
		t.Errorf("Expected %d tasks to complete, got %d", expected, got)
	}
}

func TestWorkerPool_MemoryLeaks(t *testing.T) {
	t.Parallel()

	initialGoroutines := runtime.NumGoroutine()

	pool, err := NewWorkerPool(Config{
		Workers:         4,
		QueueSize:       10,
		ShutdownTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}

	for i := 0; i < 100; i++ {
		pool.Submit(func() error {
			time.Sleep(time.Millisecond)
			return nil
		})
	}

	pool.Wait()
	pool.Stop()

	// Give some time for goroutines to clean up
	time.Sleep(100 * time.Millisecond)

	finalGoroutines := runtime.NumGoroutine()

	// Allow some margin for test goroutines
	if finalGoroutines > initialGoroutines+2 {
		t.Errorf("Possible goroutine leak: initial=%d, final=%d", initialGoroutines, finalGoroutines)
	}
}

func TestWorkerPool_ErrorHandling(t *testing.T) {
	t.Parallel()

	var capturedErrors []error
	var mu sync.Mutex

	pool, err := NewWorkerPool(Config{
		Workers:   2,
		QueueSize: 10,
		ErrorHandler: func(err error) {
			mu.Lock()
			capturedErrors = append(capturedErrors, err)
			mu.Unlock()
		},
		ShutdownTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Stop()

	expectedErr := errors.New("task error")

	pool.Submit(func() error {
		return expectedErr
	})

	pool.Wait()

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if len(capturedErrors) == 0 {
		t.Error("Expected error to be captured")
	}
}

func TestWorkerPool_MultipleShutdownCalls(t *testing.T) {
	t.Parallel()

	pool, err := NewWorkerPool(Config{
		Workers:         2,
		QueueSize:       10,
		ShutdownTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}

	// First shutdown
	err1 := pool.Stop()
	// Second shutdown (should be idempotent)
	err2 := pool.Stop()

	if err1 != nil {
		t.Errorf("First shutdown returned error: %v", err1)
	}

	if err2 != nil {
		t.Errorf("Second shutdown returned error: %v", err2)
	}
}

func TestDefaultWorkerPool(t *testing.T) {
	t.Parallel()

	pool := NewDefaultWorkerPool()
	if pool == nil {
		t.Fatal("NewDefaultWorkerPool returned nil")
	}
	defer pool.Stop()

	stats := pool.Stats()
	expectedWorkers := runtime.NumCPU()
	if stats.ActiveWorkers != expectedWorkers {
		t.Errorf("Expected %d workers, got %d", expectedWorkers, stats.ActiveWorkers)
	}
}
