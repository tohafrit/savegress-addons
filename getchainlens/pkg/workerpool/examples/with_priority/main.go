package main

import (
	"fmt"
	"log"
	"time"

	"getchainlens.com/chainlens/pkg/workerpool"
)

func main() {
	pool, err := workerpool.NewWorkerPool(workerpool.Config{
		Workers:   2,
		QueueSize: 10,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Stop()

	// Submit tasks with different priorities
	for i := 0; i < 5; i++ {
		taskID := i
		priority := workerpool.PriorityLow
		if i%2 == 0 {
			priority = workerpool.PriorityHigh
		}

		err := pool.SubmitWithPriority(func() error {
			time.Sleep(100 * time.Millisecond)
			fmt.Printf("Task %d (priority: %d) completed\n", taskID, priority)
			return nil
		}, priority)

		if err != nil {
			log.Printf("Failed to submit task %d: %v", taskID, err)
		}
	}

	pool.Wait()
	fmt.Println("All priority tasks completed")
}
