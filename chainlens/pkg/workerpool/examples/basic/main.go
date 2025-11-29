package main

import (
	"fmt"
	"log"
	"time"

	"github.com/chainlens/chainlens/pkg/workerpool"
)

func main() {
	// Create pool with 4 workers
	pool, err := workerpool.NewWorkerPool(workerpool.Config{
		Workers:   4,
		QueueSize: 100,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Stop()

	// Submit 100 tasks
	for i := 0; i < 100; i++ {
		taskID := i
		err := pool.Submit(func() error {
			time.Sleep(100 * time.Millisecond)
			fmt.Printf("Task %d completed\n", taskID)
			return nil
		})
		if err != nil {
			log.Printf("Failed to submit task %d: %v", taskID, err)
		}
	}

	// Wait for completion
	pool.Wait()

	// Print statistics
	stats := pool.Stats()
	fmt.Printf("\nStatistics:\n")
	fmt.Printf("Completed: %d\n", stats.CompletedTasks)
	fmt.Printf("Rejected: %d\n", stats.RejectedTasks)
	fmt.Printf("Average Latency: %v\n", stats.AverageLatency)
	fmt.Printf("Uptime: %v\n", stats.Uptime)
}
