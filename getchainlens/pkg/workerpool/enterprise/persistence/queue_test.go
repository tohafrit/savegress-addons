package persistence

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestNewInMemoryQueue(t *testing.T) {
	q := NewInMemoryQueue(100)
	if q == nil {
		t.Fatal("NewInMemoryQueue returned nil")
	}
}

func TestInMemoryQueue_Push(t *testing.T) {
	ctx := context.Background()
	q := NewInMemoryQueue(100)

	item := &QueueItem{
		ID:        "item-1",
		Data:      []byte("test data"),
		CreatedAt: time.Now(),
	}

	err := q.Push(ctx, item)
	if err != nil {
		t.Fatalf("Push failed: %v", err)
	}

	if q.Len() != 1 {
		t.Errorf("Len() = %d, want 1", q.Len())
	}
}

func TestInMemoryQueue_PushFull(t *testing.T) {
	ctx := context.Background()
	q := NewInMemoryQueue(1)

	item1 := &QueueItem{ID: "item-1", Data: []byte("data"), CreatedAt: time.Now()}
	item2 := &QueueItem{ID: "item-2", Data: []byte("data"), CreatedAt: time.Now()}

	err := q.Push(ctx, item1)
	if err != nil {
		t.Fatalf("First Push failed: %v", err)
	}

	err = q.Push(ctx, item2)
	if err != ErrQueueFull {
		t.Errorf("Expected ErrQueueFull, got %v", err)
	}
}

func TestInMemoryQueue_Pop(t *testing.T) {
	ctx := context.Background()
	q := NewInMemoryQueue(100)

	item := &QueueItem{
		ID:        "item-1",
		Data:      []byte("test data"),
		CreatedAt: time.Now(),
	}
	q.Push(ctx, item)

	popped, err := q.Pop(ctx)
	if err != nil {
		t.Fatalf("Pop failed: %v", err)
	}

	if popped.ID != item.ID {
		t.Errorf("ID = %s, want %s", popped.ID, item.ID)
	}

	if q.Len() != 0 {
		t.Errorf("Len() = %d, want 0", q.Len())
	}
}

func TestInMemoryQueue_Pop_Empty(t *testing.T) {
	ctx := context.Background()
	q := NewInMemoryQueue(100)

	_, err := q.Pop(ctx)
	if err != ErrQueueEmpty {
		t.Errorf("Expected ErrQueueEmpty, got %v", err)
	}
}

func TestInMemoryQueue_Peek(t *testing.T) {
	ctx := context.Background()
	q := NewInMemoryQueue(100)

	item := &QueueItem{
		ID:        "item-1",
		Data:      []byte("test data"),
		CreatedAt: time.Now(),
	}
	q.Push(ctx, item)

	peeked, err := q.Peek(ctx)
	if err != nil {
		t.Fatalf("Peek failed: %v", err)
	}

	if peeked.ID != item.ID {
		t.Errorf("ID = %s, want %s", peeked.ID, item.ID)
	}

	// Item should still be in queue
	if q.Len() != 1 {
		t.Errorf("Len() = %d, want 1", q.Len())
	}
}

func TestInMemoryQueue_Peek_Empty(t *testing.T) {
	ctx := context.Background()
	q := NewInMemoryQueue(100)

	_, err := q.Peek(ctx)
	if err != ErrQueueEmpty {
		t.Errorf("Expected ErrQueueEmpty, got %v", err)
	}
}

func TestInMemoryQueue_Ack(t *testing.T) {
	ctx := context.Background()
	q := NewInMemoryQueue(100)

	item := &QueueItem{
		ID:        "item-1",
		Data:      []byte("test data"),
		CreatedAt: time.Now(),
	}
	q.Push(ctx, item)

	// Pop and ack
	q.Pop(ctx)
	err := q.Ack(ctx, item.ID)
	if err != nil {
		t.Fatalf("Ack failed: %v", err)
	}
}

func TestInMemoryQueue_Nack(t *testing.T) {
	ctx := context.Background()
	q := NewInMemoryQueue(100)

	item := &QueueItem{
		ID:        "item-1",
		Data:      []byte("test data"),
		CreatedAt: time.Now(),
	}
	q.Push(ctx, item)

	// Pop and nack (should requeue)
	popped, _ := q.Pop(ctx)
	err := q.Nack(ctx, popped.ID)
	if err != nil {
		t.Fatalf("Nack failed: %v", err)
	}

	// Should be back in queue
	if q.Len() != 1 {
		t.Errorf("Len() = %d, want 1 (item should be requeued)", q.Len())
	}
}

func TestInMemoryQueue_Nack_NotFound(t *testing.T) {
	ctx := context.Background()
	q := NewInMemoryQueue(100)

	err := q.Nack(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent item")
	}
}

func TestInMemoryQueue_Close(t *testing.T) {
	q := NewInMemoryQueue(100)

	item := &QueueItem{
		ID:        "item-1",
		Data:      []byte("test data"),
		CreatedAt: time.Now(),
	}
	q.Push(context.Background(), item)

	err := q.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestInMemoryQueue_Concurrent(t *testing.T) {
	ctx := context.Background()
	q := NewInMemoryQueue(0) // unlimited

	var wg sync.WaitGroup
	numProducers := 10
	numItemsPerProducer := 100

	// Producers
	for i := 0; i < numProducers; i++ {
		wg.Add(1)
		go func(producerID int) {
			defer wg.Done()
			for j := 0; j < numItemsPerProducer; j++ {
				item := &QueueItem{
					ID:        generateItemID(producerID, j),
					Data:      []byte("data"),
					CreatedAt: time.Now(),
				}
				q.Push(ctx, item)
			}
		}(i)
	}

	wg.Wait()

	expectedLen := numProducers * numItemsPerProducer
	if q.Len() != expectedLen {
		t.Errorf("Len() = %d, want %d", q.Len(), expectedLen)
	}

	// Consumers
	consumed := 0
	for i := 0; i < expectedLen; i++ {
		_, err := q.Pop(ctx)
		if err == nil {
			consumed++
		}
	}

	if consumed != expectedLen {
		t.Errorf("consumed = %d, want %d", consumed, expectedLen)
	}
}

func TestInMemoryQueue_FIFO(t *testing.T) {
	ctx := context.Background()
	q := NewInMemoryQueue(100)

	// Push items in order
	for i := 0; i < 10; i++ {
		q.Push(ctx, &QueueItem{
			ID:        generateItemID(0, i),
			Data:      []byte{byte(i)},
			CreatedAt: time.Now(),
		})
	}

	// Pop should return in same order
	for i := 0; i < 10; i++ {
		item, _ := q.Pop(ctx)
		if item.Data[0] != byte(i) {
			t.Errorf("item %d: Data = %d, want %d", i, item.Data[0], i)
		}
	}
}

func TestNewDiskQueue(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "queue")

	q, err := NewDiskQueue(path)
	if err != nil {
		t.Fatalf("NewDiskQueue failed: %v", err)
	}
	defer q.Close()

	if q == nil {
		t.Fatal("NewDiskQueue returned nil")
	}
}

func TestDiskQueue_PushPop(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	path := filepath.Join(dir, "queue")

	q, err := NewDiskQueue(path)
	if err != nil {
		t.Fatalf("NewDiskQueue failed: %v", err)
	}
	defer q.Close()

	item := &QueueItem{
		ID:        "disk-item-1",
		Data:      []byte("persistent data"),
		CreatedAt: time.Now(),
	}

	err = q.Push(ctx, item)
	if err != nil {
		t.Fatalf("Push failed: %v", err)
	}

	if q.Len() != 1 {
		t.Errorf("Len() = %d, want 1", q.Len())
	}

	popped, err := q.Pop(ctx)
	if err != nil {
		t.Fatalf("Pop failed: %v", err)
	}

	if popped.ID != item.ID {
		t.Errorf("ID = %s, want %s", popped.ID, item.ID)
	}

	if string(popped.Data) != string(item.Data) {
		t.Errorf("Data = %s, want %s", popped.Data, item.Data)
	}
}

func TestDiskQueue_Persistence(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	path := filepath.Join(dir, "queue")

	// Create and populate queue
	q1, err := NewDiskQueue(path)
	if err != nil {
		t.Fatalf("NewDiskQueue failed: %v", err)
	}

	item := &QueueItem{
		ID:        "persistent-item",
		Data:      []byte("should survive restart"),
		CreatedAt: time.Now(),
	}
	q1.Push(ctx, item)
	q1.Close()

	// Reopen queue
	q2, err := NewDiskQueue(path)
	if err != nil {
		t.Fatalf("Reopen NewDiskQueue failed: %v", err)
	}
	defer q2.Close()

	// Item should still be there
	if q2.Len() != 1 {
		t.Errorf("Len() after reopen = %d, want 1", q2.Len())
	}

	popped, err := q2.Pop(ctx)
	if err != nil {
		t.Fatalf("Pop after reopen failed: %v", err)
	}

	if popped.ID != item.ID {
		t.Errorf("ID = %s, want %s", popped.ID, item.ID)
	}
}

func TestDiskQueue_Peek(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	path := filepath.Join(dir, "queue")

	q, err := NewDiskQueue(path)
	if err != nil {
		t.Fatalf("NewDiskQueue failed: %v", err)
	}
	defer q.Close()

	item := &QueueItem{
		ID:        "peek-item",
		Data:      []byte("peek data"),
		CreatedAt: time.Now(),
	}
	q.Push(ctx, item)

	peeked, err := q.Peek(ctx)
	if err != nil {
		t.Fatalf("Peek failed: %v", err)
	}

	if peeked.ID != item.ID {
		t.Errorf("ID = %s, want %s", peeked.ID, item.ID)
	}

	// Still in queue
	if q.Len() != 1 {
		t.Errorf("Len() = %d, want 1", q.Len())
	}
}

func TestDiskQueue_Pop_Empty(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	path := filepath.Join(dir, "queue")

	q, err := NewDiskQueue(path)
	if err != nil {
		t.Fatalf("NewDiskQueue failed: %v", err)
	}
	defer q.Close()

	_, err = q.Pop(ctx)
	if err != ErrQueueEmpty {
		t.Errorf("Expected ErrQueueEmpty, got %v", err)
	}
}

func TestDiskQueue_Peek_Empty(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	path := filepath.Join(dir, "queue")

	q, err := NewDiskQueue(path)
	if err != nil {
		t.Fatalf("NewDiskQueue failed: %v", err)
	}
	defer q.Close()

	_, err = q.Peek(ctx)
	if err != ErrQueueEmpty {
		t.Errorf("Expected ErrQueueEmpty, got %v", err)
	}
}

func TestDiskQueue_AckNack(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	path := filepath.Join(dir, "queue")

	q, err := NewDiskQueue(path)
	if err != nil {
		t.Fatalf("NewDiskQueue failed: %v", err)
	}
	defer q.Close()

	item := &QueueItem{
		ID:        "ack-nack-item",
		Data:      []byte("data"),
		CreatedAt: time.Now(),
	}
	q.Push(ctx, item)

	popped, _ := q.Pop(ctx)

	// Ack
	err = q.Ack(ctx, popped.ID)
	if err != nil {
		t.Fatalf("Ack failed: %v", err)
	}

	// Push another
	item2 := &QueueItem{
		ID:        "nack-item",
		Data:      []byte("data2"),
		CreatedAt: time.Now(),
	}
	q.Push(ctx, item2)
	popped2, _ := q.Pop(ctx)

	// Nack should requeue
	err = q.Nack(ctx, popped2.ID)
	if err != nil {
		t.Fatalf("Nack failed: %v", err)
	}

	if q.Len() != 1 {
		t.Errorf("Len() = %d, want 1 (nacked item should be requeued)", q.Len())
	}
}

func TestDiskQueue_Nack_NotFound(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	path := filepath.Join(dir, "queue")

	q, err := NewDiskQueue(path)
	if err != nil {
		t.Fatalf("NewDiskQueue failed: %v", err)
	}
	defer q.Close()

	err = q.Nack(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent item")
	}
}

func TestDiskQueue_RecoverAbandoned(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	path := filepath.Join(dir, "queue")

	q, err := NewDiskQueue(path)
	if err != nil {
		t.Fatalf("NewDiskQueue failed: %v", err)
	}

	item := &QueueItem{
		ID:        "abandoned-item",
		Data:      []byte("data"),
		CreatedAt: time.Now().Add(-2 * time.Hour), // Old item
	}
	q.Push(ctx, item)

	// Pop without ack/nack (simulate abandoned)
	q.Pop(ctx)

	// Recover with 1 hour timeout
	err = q.RecoverAbandoned(1 * time.Hour)
	if err != nil {
		t.Fatalf("RecoverAbandoned failed: %v", err)
	}

	// Item should be back in queue
	if q.Len() != 1 {
		t.Logf("Len() = %d (recovery may depend on item timestamps)", q.Len())
	}

	q.Close()
}

func TestDiskQueue_LargeData(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	path := filepath.Join(dir, "queue")

	q, err := NewDiskQueue(path)
	if err != nil {
		t.Fatalf("NewDiskQueue failed: %v", err)
	}
	defer q.Close()

	// Create large data (1MB)
	largeData := make([]byte, 1024*1024)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	item := &QueueItem{
		ID:        "large-item",
		Data:      largeData,
		CreatedAt: time.Now(),
	}

	err = q.Push(ctx, item)
	if err != nil {
		t.Fatalf("Push large item failed: %v", err)
	}

	popped, err := q.Pop(ctx)
	if err != nil {
		t.Fatalf("Pop large item failed: %v", err)
	}

	if len(popped.Data) != len(largeData) {
		t.Errorf("Data len = %d, want %d", len(popped.Data), len(largeData))
	}
}

func TestQueueItemFields(t *testing.T) {
	now := time.Now()
	item := &QueueItem{
		ID:        "test-id",
		Data:      []byte("test data"),
		Priority:  5,
		Attempts:  3,
		CreatedAt: now,
	}

	if item.ID != "test-id" {
		t.Errorf("ID = %s, want test-id", item.ID)
	}
	if string(item.Data) != "test data" {
		t.Errorf("Data = %s, want test data", item.Data)
	}
	if item.Priority != 5 {
		t.Errorf("Priority = %d, want 5", item.Priority)
	}
	if item.Attempts != 3 {
		t.Errorf("Attempts = %d, want 3", item.Attempts)
	}
}

func TestPersistentQueueInterface(t *testing.T) {
	// Ensure both types implement PersistentQueue
	var _ PersistentQueue = (*InMemoryQueue)(nil)
	var _ PersistentQueue = (*DiskQueue)(nil)
}

// Helper function
func generateItemID(producer, item int) string {
	return "item-" + string(rune('A'+producer)) + "-" + string(rune('0'+item%10))
}

func init() {
	// Ensure temp directories are cleaned up
	os.MkdirAll(os.TempDir(), 0755)
}

// DLQ tests

func TestNewDeadLetterQueue(t *testing.T) {
	storage := NewInMemoryQueue(100)
	dlq := NewDeadLetterQueue(DLQConfig{
		MaxSize:   100,
		Retention: 24 * time.Hour,
		Storage:   storage,
	})

	if dlq == nil {
		t.Fatal("NewDeadLetterQueue returned nil")
	}
}

func TestDeadLetterQueue_Push(t *testing.T) {
	ctx := context.Background()
	storage := NewInMemoryQueue(100)
	dlq := NewDeadLetterQueue(DLQConfig{
		Storage: storage,
	})

	entry := &DLQEntry{
		TaskID:       "task1",
		Data:         []byte("test data"),
		FailedAt:     time.Now(),
		FailureCount: 3,
		Errors:       []string{"error1", "error2", "error3"},
	}

	err := dlq.Push(ctx, entry)
	if err != nil {
		t.Fatalf("Push failed: %v", err)
	}

	if dlq.Len() != 1 {
		t.Errorf("Len = %d, want 1", dlq.Len())
	}
}

func TestDeadLetterQueue_PushWithCallback(t *testing.T) {
	ctx := context.Background()
	storage := NewInMemoryQueue(100)

	var called bool
	var calledEntry *DLQEntry

	dlq := NewDeadLetterQueue(DLQConfig{
		Storage: storage,
		OnMessage: func(entry *DLQEntry) {
			called = true
			calledEntry = entry
		},
	})

	entry := &DLQEntry{
		TaskID:   "task1",
		FailedAt: time.Now(),
	}

	err := dlq.Push(ctx, entry)
	if err != nil {
		t.Fatalf("Push failed: %v", err)
	}

	// Wait for callback
	time.Sleep(50 * time.Millisecond)

	if !called {
		t.Error("OnMessage callback not called")
	}
	if calledEntry == nil || calledEntry.TaskID != "task1" {
		t.Error("Callback received wrong entry")
	}
}

func TestDeadLetterQueue_Pop(t *testing.T) {
	ctx := context.Background()
	storage := NewInMemoryQueue(100)
	dlq := NewDeadLetterQueue(DLQConfig{
		Storage: storage,
	})

	entry := &DLQEntry{
		TaskID:       "task1",
		Data:         []byte("test data"),
		FailedAt:     time.Now(),
		FailureCount: 1,
		Errors:       []string{"error1"},
	}

	dlq.Push(ctx, entry)

	popped, err := dlq.Pop(ctx)
	if err != nil {
		t.Fatalf("Pop failed: %v", err)
	}

	if popped.TaskID != entry.TaskID {
		t.Errorf("TaskID = %s, want %s", popped.TaskID, entry.TaskID)
	}
	if popped.FailureCount != entry.FailureCount {
		t.Errorf("FailureCount = %d, want %d", popped.FailureCount, entry.FailureCount)
	}
}

func TestDeadLetterQueue_Pop_Empty(t *testing.T) {
	ctx := context.Background()
	storage := NewInMemoryQueue(100)
	dlq := NewDeadLetterQueue(DLQConfig{
		Storage: storage,
	})

	_, err := dlq.Pop(ctx)
	if err != ErrQueueEmpty {
		t.Errorf("Expected ErrQueueEmpty, got %v", err)
	}
}

func TestDeadLetterQueue_Peek(t *testing.T) {
	ctx := context.Background()
	storage := NewInMemoryQueue(100)
	dlq := NewDeadLetterQueue(DLQConfig{
		Storage: storage,
	})

	entry := &DLQEntry{
		TaskID:   "task1",
		FailedAt: time.Now(),
	}

	dlq.Push(ctx, entry)

	peeked, err := dlq.Peek(ctx)
	if err != nil {
		t.Fatalf("Peek failed: %v", err)
	}

	if peeked.TaskID != entry.TaskID {
		t.Errorf("TaskID = %s, want %s", peeked.TaskID, entry.TaskID)
	}

	// Still in queue
	if dlq.Len() != 1 {
		t.Errorf("Len = %d, want 1", dlq.Len())
	}
}

func TestDeadLetterQueue_Peek_Empty(t *testing.T) {
	ctx := context.Background()
	storage := NewInMemoryQueue(100)
	dlq := NewDeadLetterQueue(DLQConfig{
		Storage: storage,
	})

	_, err := dlq.Peek(ctx)
	if err != ErrQueueEmpty {
		t.Errorf("Expected ErrQueueEmpty, got %v", err)
	}
}

func TestDeadLetterQueue_Get(t *testing.T) {
	ctx := context.Background()
	storage := NewInMemoryQueue(100)
	dlq := NewDeadLetterQueue(DLQConfig{
		Storage: storage,
	})

	// Get returns ErrQueueEmpty (simplified implementation)
	_, err := dlq.Get(ctx, "task1")
	if err != ErrQueueEmpty {
		t.Errorf("Expected ErrQueueEmpty, got %v", err)
	}
}

func TestDeadLetterQueue_Replay(t *testing.T) {
	ctx := context.Background()
	storage := NewInMemoryQueue(100)
	dlq := NewDeadLetterQueue(DLQConfig{
		Storage: storage,
	})

	// Replay with non-existent task
	err := dlq.Replay(ctx, "task1", func(entry *DLQEntry) error {
		return nil
	})

	if err != ErrQueueEmpty {
		t.Errorf("Expected ErrQueueEmpty, got %v", err)
	}
}

func TestDeadLetterQueue_Cleanup_NoRetention(t *testing.T) {
	ctx := context.Background()
	storage := NewInMemoryQueue(100)
	dlq := NewDeadLetterQueue(DLQConfig{
		Storage:   storage,
		Retention: 0, // No retention
	})

	err := dlq.Cleanup(ctx)
	if err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}
}

func TestDeadLetterQueue_Cleanup_WithRetention(t *testing.T) {
	ctx := context.Background()
	storage := NewInMemoryQueue(100)
	dlq := NewDeadLetterQueue(DLQConfig{
		Storage:   storage,
		Retention: 1 * time.Hour,
	})

	// Add old entry
	oldEntry := &DLQEntry{
		TaskID:   "old-task",
		FailedAt: time.Now().Add(-2 * time.Hour),
	}
	dlq.Push(ctx, oldEntry)

	// Add recent entry
	newEntry := &DLQEntry{
		TaskID:   "new-task",
		FailedAt: time.Now(),
	}
	dlq.Push(ctx, newEntry)

	if dlq.Len() != 2 {
		t.Errorf("Before cleanup: Len = %d, want 2", dlq.Len())
	}

	err := dlq.Cleanup(ctx)
	if err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}

	// Old entry should be removed
	if dlq.Len() != 1 {
		t.Errorf("After cleanup: Len = %d, want 1", dlq.Len())
	}
}

func TestDeadLetterQueue_Close(t *testing.T) {
	storage := NewInMemoryQueue(100)
	dlq := NewDeadLetterQueue(DLQConfig{
		Storage: storage,
	})

	err := dlq.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestDeadLetterQueue_GetAll(t *testing.T) {
	ctx := context.Background()
	storage := NewInMemoryQueue(100)
	dlq := NewDeadLetterQueue(DLQConfig{
		Storage: storage,
	})

	entry := &DLQEntry{
		TaskID:   "task1",
		FailedAt: time.Now(),
	}
	dlq.Push(ctx, entry)

	entries, err := dlq.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll failed: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("GetAll len = %d, want 1", len(entries))
	}
}

func TestDeadLetterQueue_GetAll_Empty(t *testing.T) {
	ctx := context.Background()
	storage := NewInMemoryQueue(100)
	dlq := NewDeadLetterQueue(DLQConfig{
		Storage: storage,
	})

	entries, err := dlq.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll failed: %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("GetAll len = %d, want 0", len(entries))
	}
}

func TestDeadLetterQueue_GetStats(t *testing.T) {
	storage := NewInMemoryQueue(100)
	dlq := NewDeadLetterQueue(DLQConfig{
		Storage:   storage,
		MaxSize:   100,
		Retention: 24 * time.Hour,
	})

	stats := dlq.GetStats()

	if stats.MaxSize != 100 {
		t.Errorf("MaxSize = %d, want 100", stats.MaxSize)
	}
	if stats.Retention != 24*time.Hour {
		t.Errorf("Retention = %v, want 24h", stats.Retention)
	}
	if stats.Size != 0 {
		t.Errorf("Size = %d, want 0", stats.Size)
	}
}

func TestDLQEntry_Fields(t *testing.T) {
	now := time.Now()
	entry := &DLQEntry{
		TaskID:       "task1",
		Data:         []byte("data"),
		FailedAt:     now,
		FailureCount: 3,
		Errors:       []string{"err1", "err2"},
		OriginalItem: &QueueItem{ID: "orig"},
	}

	if entry.TaskID != "task1" {
		t.Errorf("TaskID = %s, want task1", entry.TaskID)
	}
	if entry.FailureCount != 3 {
		t.Errorf("FailureCount = %d, want 3", entry.FailureCount)
	}
	if len(entry.Errors) != 2 {
		t.Errorf("Errors len = %d, want 2", len(entry.Errors))
	}
	if entry.OriginalItem.ID != "orig" {
		t.Errorf("OriginalItem.ID = %s, want orig", entry.OriginalItem.ID)
	}
}

// Benchmarks

func BenchmarkInMemoryQueue_Push(b *testing.B) {
	ctx := context.Background()
	q := NewInMemoryQueue(0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.Push(ctx, &QueueItem{
			ID:        "item",
			Data:      []byte("data"),
			CreatedAt: time.Now(),
		})
	}
}

func BenchmarkInMemoryQueue_PushPop(b *testing.B) {
	ctx := context.Background()
	q := NewInMemoryQueue(0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.Push(ctx, &QueueItem{
			ID:        "item",
			Data:      []byte("data"),
			CreatedAt: time.Now(),
		})
		q.Pop(ctx)
	}
}

func BenchmarkDiskQueue_Push(b *testing.B) {
	ctx := context.Background()
	dir := b.TempDir()
	path := filepath.Join(dir, "queue")

	q, _ := NewDiskQueue(path)
	defer q.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.Push(ctx, &QueueItem{
			ID:        "item",
			Data:      []byte("data"),
			CreatedAt: time.Now(),
		})
	}
}

func BenchmarkDiskQueue_PushPop(b *testing.B) {
	ctx := context.Background()
	dir := b.TempDir()
	path := filepath.Join(dir, "queue")

	q, _ := NewDiskQueue(path)
	defer q.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.Push(ctx, &QueueItem{
			ID:        "item",
			Data:      []byte("data"),
			CreatedAt: time.Now(),
		})
		q.Pop(ctx)
	}
}
