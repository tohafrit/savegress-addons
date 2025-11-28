package performance

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

// ObjectPool is a generic object pool for reducing allocations
type ObjectPool[T any] struct {
	pool sync.Pool
	new  func() T
}

// NewObjectPool creates a new object pool
func NewObjectPool[T any](newFn func() T) *ObjectPool[T] {
	return &ObjectPool[T]{
		new: newFn,
		pool: sync.Pool{
			New: func() interface{} {
				return newFn()
			},
		},
	}
}

// Get gets an object from the pool
func (p *ObjectPool[T]) Get() T {
	return p.pool.Get().(T)
}

// Put returns an object to the pool
func (p *ObjectPool[T]) Put(obj T) {
	p.pool.Put(obj)
}

// LockFreeRingBuffer is a lock-free ring buffer for task queue
type LockFreeRingBuffer struct {
	buffer []unsafe.Pointer
	mask   uint64
	head   atomic.Uint64
	tail   atomic.Uint64
}

// NewLockFreeRingBuffer creates a new lock-free ring buffer
func NewLockFreeRingBuffer(size int) *LockFreeRingBuffer {
	// Size must be power of 2
	actualSize := nextPowerOf2(size)
	return &LockFreeRingBuffer{
		buffer: make([]unsafe.Pointer, actualSize),
		mask:   uint64(actualSize - 1),
	}
}

// Push adds an item to the buffer
func (rb *LockFreeRingBuffer) Push(item unsafe.Pointer) bool {
	for {
		tail := rb.tail.Load()
		head := rb.head.Load()

		if tail-head >= rb.mask+1 {
			return false // Full
		}

		if rb.tail.CompareAndSwap(tail, tail+1) {
			rb.buffer[tail&rb.mask] = item
			return true
		}
	}
}

// Pop removes an item from the buffer
func (rb *LockFreeRingBuffer) Pop() unsafe.Pointer {
	for {
		head := rb.head.Load()
		tail := rb.tail.Load()

		if head >= tail {
			return nil // Empty
		}

		item := rb.buffer[head&rb.mask]
		if rb.head.CompareAndSwap(head, head+1) {
			return item
		}
	}
}

// Len returns the current size
func (rb *LockFreeRingBuffer) Len() int {
	tail := rb.tail.Load()
	head := rb.head.Load()
	return int(tail - head)
}

// nextPowerOf2 returns the next power of 2
func nextPowerOf2(n int) int {
	if n <= 0 {
		return 1
	}
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n++
	return n
}

// BufferPool manages byte buffer pooling
type BufferPool struct {
	pool sync.Pool
	size int
}

// NewBufferPool creates a new buffer pool
func NewBufferPool(size int) *BufferPool {
	return &BufferPool{
		size: size,
		pool: sync.Pool{
			New: func() interface{} {
				return make([]byte, size)
			},
		},
	}
}

// Get gets a buffer from the pool
func (bp *BufferPool) Get() []byte {
	return bp.pool.Get().([]byte)
}

// Put returns a buffer to the pool
func (bp *BufferPool) Put(buf []byte) {
	if len(buf) == bp.size {
		bp.pool.Put(buf)
	}
}

// LockFreeStack is a lock-free stack implementation
type LockFreeStack struct {
	head atomic.Value // *node
}

type node struct {
	value interface{}
	next  *node
}

// NewLockFreeStack creates a new lock-free stack
func NewLockFreeStack() *LockFreeStack {
	s := &LockFreeStack{}
	s.head.Store((*node)(nil))
	return s
}

// Push pushes a value onto the stack
func (s *LockFreeStack) Push(value interface{}) {
	newNode := &node{value: value}
	for {
		oldHead := s.head.Load().(*node)
		newNode.next = oldHead
		if s.head.CompareAndSwap(oldHead, newNode) {
			return
		}
	}
}

// Pop pops a value from the stack
func (s *LockFreeStack) Pop() (interface{}, bool) {
	for {
		oldHead := s.head.Load().(*node)
		if oldHead == nil {
			return nil, false
		}
		if s.head.CompareAndSwap(oldHead, oldHead.next) {
			return oldHead.value, true
		}
	}
}

// Batch processing utilities

// BatchProcessor processes items in batches for efficiency
type BatchProcessor[T any] struct {
	batchSize int
	timeout   atomic.Value // time.Duration
	items     []T
	mu        sync.Mutex
	process   func([]T) error
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor[T any](batchSize int, processFn func([]T) error) *BatchProcessor[T] {
	return &BatchProcessor[T]{
		batchSize: batchSize,
		items:     make([]T, 0, batchSize),
		process:   processFn,
	}
}

// Add adds an item to the batch
func (bp *BatchProcessor[T]) Add(item T) error {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	bp.items = append(bp.items, item)

	if len(bp.items) >= bp.batchSize {
		return bp.flush()
	}

	return nil
}

// Flush processes all pending items
func (bp *BatchProcessor[T]) Flush() error {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	return bp.flush()
}

func (bp *BatchProcessor[T]) flush() error {
	if len(bp.items) == 0 {
		return nil
	}

	err := bp.process(bp.items)
	bp.items = bp.items[:0] // Reset slice, keep capacity
	return err
}
