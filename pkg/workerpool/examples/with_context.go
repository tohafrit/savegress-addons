package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/yourusername/workerpool"
)

func main() {
	pool := workerpool.NewDefaultWorkerPool()
	defer pool.Stop()

	// Task with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := pool.SubmitWithContext(ctx, func() error {
		// Long-running task that respects context
		select {
		case <-time.After(10 * time.Second):
			fmt.Println("Task completed")
			return nil
		case <-ctx.Done():
			fmt.Println("Task cancelled due to context")
			return ctx.Err()
		}
	})

	if err != nil {
		log.Printf("Task submission error: %v", err)
	}

	// Wait a bit to see the cancellation
	time.Sleep(6 * time.Second)

	pool.Wait()
	fmt.Println("All tasks completed or cancelled")
}
