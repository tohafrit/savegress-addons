package main

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/yourusername/workerpool"
)

func main() {
	pool, _ := workerpool.NewWorkerPool(workerpool.Config{
		Workers:   2,
		QueueSize: 5, // Small queue for demonstration
	})
	defer pool.Stop()

	submitted := 0
	rejected := 0

	// Try to submit 100 tasks rapidly
	for i := 0; i < 100; i++ {
		taskID := i
		err := pool.TrySubmit(func() error {
			time.Sleep(100 * time.Millisecond)
			fmt.Printf("Task %d executed\n", taskID)
			return nil
		})

		if errors.Is(err, workerpool.ErrQueueFull) {
			rejected++
			fmt.Printf("Task %d rejected (queue full)\n", taskID)
			time.Sleep(10 * time.Millisecond) // Backoff

			// Retry with timeout
			err = pool.SubmitWithTimeout(func() error {
				time.Sleep(100 * time.Millisecond)
				fmt.Printf("Task %d executed (after retry)\n", taskID)
				return nil
			}, 500*time.Millisecond)

			if err != nil {
				log.Printf("Task %d failed even after retry: %v", taskID, err)
			} else {
				submitted++
			}
		} else if err != nil {
			log.Printf("Task %d failed: %v", taskID, err)
		} else {
			submitted++
		}
	}

	pool.Wait()

	stats := pool.Stats()
	fmt.Printf("\n=== Results ===\n")
	fmt.Printf("Submitted: %d\n", submitted)
	fmt.Printf("Rejected: %d\n", rejected)
	fmt.Printf("Completed: %d\n", stats.CompletedTasks)
	fmt.Printf("Total Rejections (stats): %d\n", stats.RejectedTasks)
}
