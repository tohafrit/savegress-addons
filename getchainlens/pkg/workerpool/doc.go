// Package workerpool provides a production-ready worker pool implementation
// for concurrent task execution in Go.
//
// Features:
//   - Fixed number of worker goroutines
//   - Buffered task queue with backpressure
//   - Graceful shutdown with timeout
//   - Context-aware task cancellation
//   - Panic recovery
//   - Comprehensive statistics
//   - Zero dependencies (stdlib only)
//
// # Basic Usage
//
//	pool, err := workerpool.NewWorkerPool(workerpool.Config{
//	    Workers: 4,
//	    QueueSize: 100,
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer pool.Stop()
//
//	// Submit tasks
//	err = pool.Submit(func() error {
//	    // Do work
//	    return nil
//	})
//
// # Advanced Usage
//
// With context:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//	defer cancel()
//
//	err = pool.SubmitWithContext(ctx, func() error {
//	    // Work that respects context cancellation
//	    return nil
//	})
//
// Non-blocking submission:
//
//	err = pool.TrySubmit(func() error {
//	    return nil
//	})
//	if errors.Is(err, workerpool.ErrQueueFull) {
//	    // Handle full queue
//	}
//
// # Statistics
//
//	stats := pool.Stats()
//	fmt.Printf("Active: %d, Completed: %d\n",
//	    stats.ActiveWorkers, stats.CompletedTasks)
package workerpool
