package performance

import (
	"sync"
	"testing"
	"unsafe"
)

// ObjectPool tests

func TestNewObjectPool(t *testing.T) {
	pool := NewObjectPool(func() *int {
		n := 0
		return &n
	})

	if pool == nil {
		t.Fatal("NewObjectPool returned nil")
	}
}

func TestObjectPool_GetPut(t *testing.T) {
	pool := NewObjectPool(func() *int {
		n := 0
		return &n
	})

	// Get an object
	obj := pool.Get()
	if obj == nil {
		t.Fatal("Get returned nil")
	}

	// Modify it
	*obj = 42

	// Put it back
	pool.Put(obj)

	// Get again (may or may not be the same object)
	obj2 := pool.Get()
	if obj2 == nil {
		t.Fatal("Second Get returned nil")
	}
}

func TestObjectPool_Concurrent(t *testing.T) {
	pool := NewObjectPool(func() []byte {
		return make([]byte, 1024)
	})

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				buf := pool.Get()
				buf[0] = 'A'
				pool.Put(buf)
			}
		}()
	}

	wg.Wait()
}

// LockFreeRingBuffer tests

func TestNewLockFreeRingBuffer(t *testing.T) {
	rb := NewLockFreeRingBuffer(10)
	if rb == nil {
		t.Fatal("NewLockFreeRingBuffer returned nil")
	}

	// Size should be power of 2
	if len(rb.buffer) != 16 { // next power of 2 after 10
		t.Errorf("buffer size = %d, want 16", len(rb.buffer))
	}
}

func TestLockFreeRingBuffer_PushPop(t *testing.T) {
	rb := NewLockFreeRingBuffer(4)

	data := 42
	ptr := unsafe.Pointer(&data)

	// Push
	if !rb.Push(ptr) {
		t.Error("Push should succeed")
	}

	if rb.Len() != 1 {
		t.Errorf("Len = %d, want 1", rb.Len())
	}

	// Pop
	popped := rb.Pop()
	if popped != ptr {
		t.Error("Pop should return pushed value")
	}

	if rb.Len() != 0 {
		t.Errorf("Len = %d, want 0", rb.Len())
	}
}

func TestLockFreeRingBuffer_PopEmpty(t *testing.T) {
	rb := NewLockFreeRingBuffer(4)

	popped := rb.Pop()
	if popped != nil {
		t.Error("Pop on empty buffer should return nil")
	}
}

func TestLockFreeRingBuffer_PushFull(t *testing.T) {
	rb := NewLockFreeRingBuffer(2) // Will be 2 (power of 2)

	data1 := 1
	data2 := 2
	data3 := 3

	if !rb.Push(unsafe.Pointer(&data1)) {
		t.Error("First Push should succeed")
	}
	if !rb.Push(unsafe.Pointer(&data2)) {
		t.Error("Second Push should succeed")
	}
	if rb.Push(unsafe.Pointer(&data3)) {
		t.Error("Third Push should fail (buffer full)")
	}
}

func TestLockFreeRingBuffer_Concurrent(t *testing.T) {
	rb := NewLockFreeRingBuffer(1024)

	var wg sync.WaitGroup

	// Producers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				data := id*100 + j
				rb.Push(unsafe.Pointer(&data))
			}
		}(i)
	}

	// Consumers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				rb.Pop()
			}
		}()
	}

	wg.Wait()
}

func TestNextPowerOf2(t *testing.T) {
	tests := []struct {
		input    int
		expected int
	}{
		{0, 1},
		{1, 1},
		{2, 2},
		{3, 4},
		{4, 4},
		{5, 8},
		{7, 8},
		{8, 8},
		{9, 16},
		{100, 128},
	}

	for _, tt := range tests {
		result := nextPowerOf2(tt.input)
		if result != tt.expected {
			t.Errorf("nextPowerOf2(%d) = %d, want %d", tt.input, result, tt.expected)
		}
	}
}

// BufferPool tests

func TestNewBufferPool(t *testing.T) {
	bp := NewBufferPool(1024)
	if bp == nil {
		t.Fatal("NewBufferPool returned nil")
	}
}

func TestBufferPool_GetPut(t *testing.T) {
	bp := NewBufferPool(1024)

	buf := bp.Get()
	if buf == nil {
		t.Fatal("Get returned nil")
	}
	if len(buf) != 1024 {
		t.Errorf("buffer len = %d, want 1024", len(buf))
	}

	// Modify and put back
	buf[0] = 'A'
	bp.Put(buf)
}

func TestBufferPool_PutWrongSize(t *testing.T) {
	bp := NewBufferPool(1024)

	// Wrong size buffer should not be pooled
	wrongBuf := make([]byte, 512)
	bp.Put(wrongBuf) // Should not panic

	// Get should still work
	buf := bp.Get()
	if len(buf) != 1024 {
		t.Errorf("buffer len = %d, want 1024", len(buf))
	}
}

// LockFreeStack tests

func TestNewLockFreeStack(t *testing.T) {
	s := NewLockFreeStack()
	if s == nil {
		t.Fatal("NewLockFreeStack returned nil")
	}
}

func TestLockFreeStack_PushPop(t *testing.T) {
	s := NewLockFreeStack()

	s.Push(1)
	s.Push(2)
	s.Push(3)

	// LIFO order
	val, ok := s.Pop()
	if !ok || val != 3 {
		t.Errorf("Pop = %v, %v, want 3, true", val, ok)
	}

	val, ok = s.Pop()
	if !ok || val != 2 {
		t.Errorf("Pop = %v, %v, want 2, true", val, ok)
	}

	val, ok = s.Pop()
	if !ok || val != 1 {
		t.Errorf("Pop = %v, %v, want 1, true", val, ok)
	}
}

func TestLockFreeStack_PopEmpty(t *testing.T) {
	s := NewLockFreeStack()

	val, ok := s.Pop()
	if ok {
		t.Errorf("Pop on empty stack should return false, got %v", val)
	}
}

func TestLockFreeStack_Concurrent(t *testing.T) {
	s := NewLockFreeStack()

	var wg sync.WaitGroup

	// Pushers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				s.Push(id*100 + j)
			}
		}(i)
	}

	// Poppers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				s.Pop()
			}
		}()
	}

	wg.Wait()
}

// BatchProcessor tests

func TestNewBatchProcessor(t *testing.T) {
	bp := NewBatchProcessor(10, func(items []int) error {
		return nil
	})

	if bp == nil {
		t.Fatal("NewBatchProcessor returned nil")
	}
}

func TestBatchProcessor_Add(t *testing.T) {
	var processed [][]int
	var mu sync.Mutex

	bp := NewBatchProcessor(3, func(items []int) error {
		mu.Lock()
		processed = append(processed, append([]int{}, items...))
		mu.Unlock()
		return nil
	})

	// Add items
	for i := 1; i <= 5; i++ {
		err := bp.Add(i)
		if err != nil {
			t.Fatalf("Add failed: %v", err)
		}
	}

	mu.Lock()
	if len(processed) != 1 {
		t.Errorf("Should have processed 1 batch, got %d", len(processed))
	}
	if len(processed) > 0 && len(processed[0]) != 3 {
		t.Errorf("First batch should have 3 items, got %d", len(processed[0]))
	}
	mu.Unlock()
}

func TestBatchProcessor_Flush(t *testing.T) {
	var processed [][]int
	var mu sync.Mutex

	bp := NewBatchProcessor(10, func(items []int) error {
		mu.Lock()
		processed = append(processed, append([]int{}, items...))
		mu.Unlock()
		return nil
	})

	// Add items (less than batch size)
	for i := 1; i <= 5; i++ {
		bp.Add(i)
	}

	// Should not have processed yet
	mu.Lock()
	if len(processed) != 0 {
		t.Error("Should not have processed yet")
	}
	mu.Unlock()

	// Flush
	err := bp.Flush()
	if err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	mu.Lock()
	if len(processed) != 1 {
		t.Errorf("Should have processed 1 batch, got %d", len(processed))
	}
	if len(processed) > 0 && len(processed[0]) != 5 {
		t.Errorf("Flushed batch should have 5 items, got %d", len(processed[0]))
	}
	mu.Unlock()
}

func TestBatchProcessor_FlushEmpty(t *testing.T) {
	var processed int

	bp := NewBatchProcessor(10, func(items []int) error {
		processed++
		return nil
	})

	// Flush on empty should not call process
	err := bp.Flush()
	if err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	if processed != 0 {
		t.Error("Should not have processed empty batch")
	}
}

// Benchmarks

func BenchmarkObjectPool_GetPut(b *testing.B) {
	pool := NewObjectPool(func() []byte {
		return make([]byte, 1024)
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf := pool.Get()
		pool.Put(buf)
	}
}

func BenchmarkLockFreeRingBuffer_PushPop(b *testing.B) {
	rb := NewLockFreeRingBuffer(1024)
	data := 42

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rb.Push(unsafe.Pointer(&data))
		rb.Pop()
	}
}

func BenchmarkLockFreeStack_PushPop(b *testing.B) {
	s := NewLockFreeStack()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Push(i)
		s.Pop()
	}
}

func BenchmarkBufferPool_GetPut(b *testing.B) {
	bp := NewBufferPool(1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf := bp.Get()
		bp.Put(buf)
	}
}

func BenchmarkBatchProcessor_Add(b *testing.B) {
	bp := NewBatchProcessor(100, func(items []int) error {
		return nil
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bp.Add(i)
	}
}
