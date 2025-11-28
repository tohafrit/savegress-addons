package workerpool

import (
	"errors"
	"fmt"
)

// Sentinel errors
var (
	ErrPoolClosed     = errors.New("worker pool is closed")
	ErrQueueFull      = errors.New("task queue is full")
	ErrTimeout        = errors.New("operation timed out")
	ErrInvalidConfig  = errors.New("invalid pool configuration")
	ErrForcedShutdown = errors.New("forced shutdown due to timeout")
)

// TaskError wraps task execution errors
type TaskError struct {
	TaskID string
	Err    error
	Stack  string // Stack trace if panic occurred
}

func (e *TaskError) Error() string {
	if e.Stack != "" {
		return fmt.Sprintf("task %s failed with panic: %v\nStack trace:\n%s", e.TaskID, e.Err, e.Stack)
	}
	return fmt.Sprintf("task %s failed: %v", e.TaskID, e.Err)
}

func (e *TaskError) Unwrap() error {
	return e.Err
}
