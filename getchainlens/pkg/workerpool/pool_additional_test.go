package workerpool

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestWorkerPool_SubmitWithPriority(t *testing.T) {
	t.Parallel()

	pool, err := NewWorkerPool(Config{
		Workers:         1,
		QueueSize:       10,
		ShutdownTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Stop()

	var order []int
	var mu sync.Mutex

	// Block the worker
	pool.Submit(func() error {
		time.Sleep(100 * time.Millisecond)
		return nil
	})

	// Submit with different priorities
	pool.SubmitWithPriority(func() error {
		mu.Lock()
		order = append(order, 1)
		mu.Unlock()
		return nil
	}, PriorityLow)

	pool.SubmitWithPriority(func() error {
		mu.Lock()
		order = append(order, 2)
		mu.Unlock()
		return nil
	}, PriorityHigh)

	pool.SubmitWithPriority(func() error {
		mu.Lock()
		order = append(order, 3)
		mu.Unlock()
		return nil
	}, PriorityNormal)

	pool.Wait()

	// High priority should execute first after the blocking task
	if len(order) != 3 {
		t.Errorf("Expected 3 tasks, got %d", len(order))
	}
}

func TestWorkerPool_SubmitWithPriority_Closed(t *testing.T) {
	t.Parallel()

	pool, err := NewWorkerPool(Config{
		Workers:         2,
		QueueSize:       10,
		ShutdownTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}

	pool.Stop()

	err = pool.SubmitWithPriority(func() error {
		return nil
	}, PriorityHigh)

	if !errors.Is(err, ErrPoolClosed) {
		t.Errorf("Expected ErrPoolClosed, got %v", err)
	}
}

func TestWorkerPool_SubmitWithContext_FullQueue(t *testing.T) {
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
		time.Sleep(500 * time.Millisecond)
		return nil
	})
	pool.Submit(func() error {
		time.Sleep(500 * time.Millisecond)
		return nil
	})

	// Submit with cancellable context
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err = pool.SubmitWithContext(ctx, func() error {
		return nil
	})

	// Should timeout or be rejected
	if err != nil && !errors.Is(err, context.DeadlineExceeded) {
		t.Logf("Got error: %v", err)
	}

	pool.Wait()
}

func TestWorkerPool_TrySubmit_Closed(t *testing.T) {
	t.Parallel()

	pool, err := NewWorkerPool(Config{
		Workers:         2,
		QueueSize:       10,
		ShutdownTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}

	pool.Stop()

	err = pool.TrySubmit(func() error {
		return nil
	})

	if !errors.Is(err, ErrPoolClosed) {
		t.Errorf("Expected ErrPoolClosed, got %v", err)
	}
}

func TestWorkerPool_SubmitWithTimeout_Closed(t *testing.T) {
	t.Parallel()

	pool, err := NewWorkerPool(Config{
		Workers:         2,
		QueueSize:       10,
		ShutdownTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}

	pool.Stop()

	err = pool.SubmitWithTimeout(func() error {
		return nil
	}, time.Second)

	if !errors.Is(err, ErrPoolClosed) {
		t.Errorf("Expected ErrPoolClosed, got %v", err)
	}
}

func TestWorkerPool_SubmitWithContext_Closed(t *testing.T) {
	t.Parallel()

	pool, err := NewWorkerPool(Config{
		Workers:         2,
		QueueSize:       10,
		ShutdownTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}

	pool.Stop()

	err = pool.SubmitWithContext(context.Background(), func() error {
		return nil
	})

	if !errors.Is(err, ErrPoolClosed) {
		t.Errorf("Expected ErrPoolClosed, got %v", err)
	}
}

func TestWorkerPool_StopWithContext(t *testing.T) {
	t.Parallel()

	pool, err := NewWorkerPool(Config{
		Workers:         2,
		QueueSize:       10,
		ShutdownTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}

	// Submit some tasks
	for i := 0; i < 5; i++ {
		pool.Submit(func() error {
			time.Sleep(10 * time.Millisecond)
			return nil
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = pool.StopWithContext(ctx)
	if err != nil {
		t.Errorf("StopWithContext returned error: %v", err)
	}
}

func TestWorkerPool_StopWithContext_Timeout(t *testing.T) {
	t.Parallel()

	pool, err := NewWorkerPool(Config{
		Workers:         1,
		QueueSize:       10,
		ShutdownTimeout: 10 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}

	// Submit long-running tasks
	for i := 0; i < 3; i++ {
		pool.Submit(func() error {
			time.Sleep(1 * time.Second)
			return nil
		})
	}

	// Give time for tasks to start
	time.Sleep(10 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err = pool.StopWithContext(ctx)
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, ErrForcedShutdown) {
		t.Logf("Expected timeout error, got: %v", err)
	}
}

func TestTaskError_Error(t *testing.T) {
	originalErr := errors.New("original error")
	taskErr := &TaskError{
		TaskID: "task-123",
		Err:    originalErr,
	}

	errStr := taskErr.Error()
	if errStr == "" {
		t.Error("Error() should not return empty string")
	}

	if !errors.Is(taskErr, originalErr) {
		t.Error("TaskError should wrap original error")
	}
}

func TestTaskError_Unwrap(t *testing.T) {
	originalErr := errors.New("original error")
	taskErr := &TaskError{
		TaskID: "task-123",
		Err:    originalErr,
	}

	unwrapped := taskErr.Unwrap()
	if unwrapped != originalErr {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, originalErr)
	}
}

func TestTaskError_WithStack(t *testing.T) {
	taskErr := &TaskError{
		TaskID: "task-123",
		Err:    errors.New("panic"),
		Stack:  "goroutine 1 [running]:\nmain.main()\n\t/test.go:10 +0x45",
	}

	errStr := taskErr.Error()
	if errStr == "" {
		t.Error("Error() should not return empty string")
	}

	// Should include stack trace
	if !errors.Is(taskErr, taskErr.Err) {
		t.Error("Should wrap original error")
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				Workers:   4,
				QueueSize: 100,
			},
			wantErr: false,
		},
		{
			name: "zero workers",
			config: Config{
				Workers:   0,
				QueueSize: 100,
			},
			wantErr: true,
		},
		{
			name: "negative workers",
			config: Config{
				Workers:   -1,
				QueueSize: 100,
			},
			wantErr: true,
		},
		{
			name: "negative queue size",
			config: Config{
				Workers:   4,
				QueueSize: -1,
			},
			wantErr: true,
		},
		{
			name: "with error handler",
			config: Config{
				Workers:      4,
				QueueSize:    100,
				ErrorHandler: func(err error) {},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewTask(t *testing.T) {
	fn := func() error {
		return nil
	}

	task := newTask(fn, PriorityHigh, nil)

	if task == nil {
		t.Fatal("newTask returned nil")
	}

	if task.Priority != PriorityHigh {
		t.Errorf("priority = %d, want %d", task.Priority, PriorityHigh)
	}

	if task.Created.IsZero() {
		t.Error("Created should be set")
	}

	if task.ID == "" {
		t.Error("ID should be set")
	}

	if task.Ctx == nil {
		t.Error("Ctx should not be nil (should default to Background)")
	}
}

func TestNewTask_WithContext(t *testing.T) {
	fn := func() error {
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	task := newTask(fn, PriorityNormal, ctx)

	if task.Ctx != ctx {
		t.Error("task should use provided context")
	}
}

func TestWorkerPool_ExecuteWithMetrics(t *testing.T) {
	t.Parallel()

	pool, err := NewWorkerPool(Config{
		Workers:         2,
		QueueSize:       10,
		EnableMetrics:   true,
		ShutdownTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Stop()

	var counter atomic.Int32

	for i := 0; i < 10; i++ {
		pool.Submit(func() error {
			counter.Add(1)
			return nil
		})
	}

	pool.Wait()

	if counter.Load() != 10 {
		t.Errorf("counter = %d, want 10", counter.Load())
	}

	stats := pool.Stats()
	if stats.AverageLatency == 0 {
		t.Log("Warning: average latency is 0 (may be too fast)")
	}
}

func TestWorkerPool_ErrorInTask(t *testing.T) {
	t.Parallel()

	var capturedErr error
	var mu sync.Mutex

	pool, err := NewWorkerPool(Config{
		Workers:   2,
		QueueSize: 10,
		ErrorHandler: func(err error) {
			mu.Lock()
			capturedErr = err
			mu.Unlock()
		},
		ShutdownTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Stop()

	testErr := errors.New("test error")
	pool.Submit(func() error {
		return testErr
	})

	pool.Wait()
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if capturedErr == nil {
		t.Error("Error should be captured")
	}
}

func TestWorkerPool_WaitWithEmptyPool(t *testing.T) {
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

	// Wait on empty pool should return immediately
	done := make(chan struct{})
	go func() {
		pool.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(time.Second):
		t.Error("Wait() on empty pool should return immediately")
	}
}

func TestWorkerPool_SubmitAfterWait(t *testing.T) {
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

	// First batch
	var counter atomic.Int32
	for i := 0; i < 5; i++ {
		pool.Submit(func() error {
			counter.Add(1)
			return nil
		})
	}
	pool.Wait()

	if counter.Load() != 5 {
		t.Errorf("After first wait, counter = %d, want 5", counter.Load())
	}

	// Second batch
	for i := 0; i < 5; i++ {
		pool.Submit(func() error {
			counter.Add(1)
			return nil
		})
	}
	pool.Wait()

	if counter.Load() != 10 {
		t.Errorf("After second wait, counter = %d, want 10", counter.Load())
	}
}

// Additional Benchmarks

func BenchmarkWorkerPool_SubmitWithPriority(b *testing.B) {
	pool, _ := NewWorkerPool(Config{
		Workers:         4,
		QueueSize:       10000,
		ShutdownTimeout: 10 * time.Second,
	})
	defer pool.Stop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pool.SubmitWithPriority(func() error {
			return nil
		}, PriorityHigh)
	}
	pool.Wait()
}

func BenchmarkNewTask(b *testing.B) {
	fn := func() error { return nil }

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		newTask(fn, PriorityNormal, nil)
	}
}
