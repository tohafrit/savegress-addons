package main

import (
	"errors"
	"fmt"
	"log"
	"time"

	"getchainlens.com/chainlens/pkg/workerpool"
)

func main() {
	// Create pool with error handler
	errorCount := 0
	pool, err := workerpool.NewWorkerPool(workerpool.Config{
		Workers:       4,
		QueueSize:     100,
		EnableMetrics: true,
		ErrorHandler: func(err error) {
			errorCount++
			log.Printf("Task error: %v", err)
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Stop()

	// Submit successful tasks
	for i := 0; i < 50; i++ {
		taskID := i
		pool.Submit(func() error {
			time.Sleep(50 * time.Millisecond)
			fmt.Printf("Task %d completed successfully\n", taskID)
			return nil
		})
	}

	// Submit tasks that fail
	for i := 0; i < 10; i++ {
		taskID := i + 50
		pool.Submit(func() error {
			return errors.New(fmt.Sprintf("task %d failed", taskID))
		})
	}

	// Submit tasks that panic
	for i := 0; i < 5; i++ {
		taskID := i + 60
		pool.Submit(func() error {
			if taskID%2 == 0 {
				panic(fmt.Sprintf("task %d panicked", taskID))
			}
			return nil
		})
	}

	// Monitor statistics in real-time
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for i := 0; i < 10; i++ {
			<-ticker.C
			stats := pool.Stats()
			fmt.Printf("\n[Stats] Active: %d, Queued: %d, Completed: %d, Rejected: %d, Avg Latency: %v\n",
				stats.ActiveWorkers,
				stats.QueuedTasks,
				stats.CompletedTasks,
				stats.RejectedTasks,
				stats.AverageLatency,
			)
		}
	}()

	pool.Wait()

	// Final statistics
	stats := pool.Stats()
	fmt.Printf("\n=== Final Statistics ===\n")
	fmt.Printf("Active Workers: %d\n", stats.ActiveWorkers)
	fmt.Printf("Completed Tasks: %d\n", stats.CompletedTasks)
	fmt.Printf("Rejected Tasks: %d\n", stats.RejectedTasks)
	fmt.Printf("Average Latency: %v\n", stats.AverageLatency)
	fmt.Printf("Uptime: %v\n", stats.Uptime)
	fmt.Printf("Errors Handled: %d\n", errorCount)
}
