package workerpool

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"
)

// Priority levels
type Priority int

const (
	PriorityLow Priority = iota
	PriorityNormal
	PriorityHigh
)

// Task represents a unit of work
type Task struct {
	ID       string          // Unique task identifier
	Fn       func() error    // Task function
	Priority Priority        // Task priority level
	Ctx      context.Context // Task context for cancellation
	Created  time.Time       // Task creation timestamp
}

var taskCounter atomic.Uint64

// generateTaskID generates a unique task ID
func generateTaskID() string {
	id := taskCounter.Add(1)
	return fmt.Sprintf("task-%d", id)
}

// newTask creates a new task with the given function and priority
func newTask(fn func() error, priority Priority, ctx context.Context) *Task {
	if ctx == nil {
		ctx = context.Background()
	}
	return &Task{
		ID:       generateTaskID(),
		Fn:       fn,
		Priority: priority,
		Ctx:      ctx,
		Created:  time.Now(),
	}
}
