package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	// ErrQueueEmpty is returned when queue is empty
	ErrQueueEmpty = errors.New("queue is empty")
	// ErrQueueFull is returned when queue is full
	ErrQueueFull = errors.New("queue is full")
)

// QueueItem represents an item in the persistent queue
type QueueItem struct {
	ID        string
	Data      []byte
	Priority  int
	CreatedAt time.Time
	Attempts  int
}

// PersistentQueue is an interface for persistent queues
type PersistentQueue interface {
	Push(ctx context.Context, item *QueueItem) error
	Pop(ctx context.Context) (*QueueItem, error)
	Peek(ctx context.Context) (*QueueItem, error)
	Ack(ctx context.Context, itemID string) error
	Nack(ctx context.Context, itemID string) error
	Len() int
	Close() error
}

// InMemoryQueue is a simple in-memory queue implementation
type InMemoryQueue struct {
	items      []*QueueItem
	processing sync.Map // map[string]*QueueItem
	maxSize    int
	mu         sync.RWMutex
}

// NewInMemoryQueue creates a new in-memory queue
func NewInMemoryQueue(maxSize int) *InMemoryQueue {
	return &InMemoryQueue{
		items:   make([]*QueueItem, 0),
		maxSize: maxSize,
	}
}

// Push adds an item to the queue
func (q *InMemoryQueue) Push(ctx context.Context, item *QueueItem) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.maxSize > 0 && len(q.items) >= q.maxSize {
		return ErrQueueFull
	}

	q.items = append(q.items, item)
	return nil
}

// Pop removes and returns an item from the queue
func (q *InMemoryQueue) Pop(ctx context.Context) (*QueueItem, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.items) == 0 {
		return nil, ErrQueueEmpty
	}

	item := q.items[0]
	q.items = q.items[1:]

	// Track as processing
	q.processing.Store(item.ID, item)

	return item, nil
}

// Peek returns the next item without removing it
func (q *InMemoryQueue) Peek(ctx context.Context) (*QueueItem, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if len(q.items) == 0 {
		return nil, ErrQueueEmpty
	}

	return q.items[0], nil
}

// Ack acknowledges successful processing of an item
func (q *InMemoryQueue) Ack(ctx context.Context, itemID string) error {
	q.processing.Delete(itemID)
	return nil
}

// Nack negatively acknowledges an item (requeue)
func (q *InMemoryQueue) Nack(ctx context.Context, itemID string) error {
	itemI, ok := q.processing.Load(itemID)
	if !ok {
		return errors.New("item not found in processing")
	}

	item := itemI.(*QueueItem)
	q.processing.Delete(itemID)

	// Requeue
	return q.Push(ctx, item)
}

// Len returns the number of items in the queue
func (q *InMemoryQueue) Len() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.items)
}

// Close closes the queue
func (q *InMemoryQueue) Close() error {
	return nil
}

// DiskQueue is a disk-backed queue implementation
type DiskQueue struct {
	path       string
	items      []*QueueItem
	processing sync.Map
	mu         sync.RWMutex
	flushTimer *time.Timer
}

// NewDiskQueue creates a new disk-backed queue
func NewDiskQueue(path string) (*DiskQueue, error) {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, err
	}

	dq := &DiskQueue{
		path:  path,
		items: make([]*QueueItem, 0),
	}

	// Load existing items
	if err := dq.load(); err != nil {
		return nil, err
	}

	return dq, nil
}

// Push adds an item to the queue
func (q *DiskQueue) Push(ctx context.Context, item *QueueItem) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.items = append(q.items, item)

	// Flush to disk
	return q.flush()
}

// Pop removes and returns an item from the queue
func (q *DiskQueue) Pop(ctx context.Context) (*QueueItem, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.items) == 0 {
		return nil, ErrQueueEmpty
	}

	item := q.items[0]
	q.items = q.items[1:]

	// Track as processing
	q.processing.Store(item.ID, item)

	// Flush to disk
	if err := q.flush(); err != nil {
		return nil, err
	}

	return item, nil
}

// Peek returns the next item without removing it
func (q *DiskQueue) Peek(ctx context.Context) (*QueueItem, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if len(q.items) == 0 {
		return nil, ErrQueueEmpty
	}

	return q.items[0], nil
}

// Ack acknowledges successful processing of an item
func (q *DiskQueue) Ack(ctx context.Context, itemID string) error {
	q.processing.Delete(itemID)
	return nil
}

// Nack negatively acknowledges an item (requeue)
func (q *DiskQueue) Nack(ctx context.Context, itemID string) error {
	itemI, ok := q.processing.Load(itemID)
	if !ok {
		return errors.New("item not found in processing")
	}

	item := itemI.(*QueueItem)
	q.processing.Delete(itemID)

	return q.Push(ctx, item)
}

// Len returns the number of items in the queue
func (q *DiskQueue) Len() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.items)
}

// Close closes the queue
func (q *DiskQueue) Close() error {
	return q.flush()
}

// flush writes the queue to disk
func (q *DiskQueue) flush() error {
	data, err := json.Marshal(q.items)
	if err != nil {
		return err
	}

	queueFile := filepath.Join(q.path, "queue.json")
	tempFile := queueFile + ".tmp"

	// Write to temp file
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return err
	}

	// Atomic rename
	return os.Rename(tempFile, queueFile)
}

// load loads the queue from disk
func (q *DiskQueue) load() error {
	queueFile := filepath.Join(q.path, "queue.json")

	data, err := os.ReadFile(queueFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No existing queue
		}
		return err
	}

	return json.Unmarshal(data, &q.items)
}

// RecoverAbandoned moves abandoned items back to the main queue
func (q *DiskQueue) RecoverAbandoned(timeout time.Duration) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	now := time.Now()
	var recovered []*QueueItem

	q.processing.Range(func(key, value interface{}) bool {
		item := value.(*QueueItem)
		if now.Sub(item.CreatedAt) > timeout {
			recovered = append(recovered, item)
			q.processing.Delete(key)
		}
		return true
	})

	// Add recovered items back to queue
	q.items = append(recovered, q.items...)

	if len(recovered) > 0 {
		return q.flush()
	}

	return nil
}
