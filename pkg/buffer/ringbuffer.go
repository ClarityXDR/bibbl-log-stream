package buffer

import (
	"sync"
	"sync/atomic"
)

// LockFreeRing is a high-performance circular buffer using atomic operations.
// Optimized for single-writer, multiple-reader scenarios (SSE broadcasts).
type LockFreeRing struct {
	data     []string
	mask     uint64
	writeIdx atomic.Uint64
	size     uint64
}

// NewLockFreeRing creates a ring buffer with power-of-2 capacity for efficient masking.
func NewLockFreeRing(capacity int) *LockFreeRing {
	// Round up to next power of 2
	cap := uint64(1)
	for cap < uint64(capacity) {
		cap <<= 1
	}
	return &LockFreeRing{
		data: make([]string, cap),
		mask: cap - 1,
		size: cap,
	}
}

// Add appends a message without locks (single writer assumed).
func (r *LockFreeRing) Add(msg string) {
	idx := r.writeIdx.Add(1) - 1
	r.data[idx&r.mask] = msg
}

// Tail returns the last n messages (thread-safe for readers).
func (r *LockFreeRing) Tail(n int) []string {
	if n <= 0 {
		return nil
	}
	writePos := r.writeIdx.Load()
	if writePos == 0 {
		return nil
	}

	available := writePos
	if available > r.size {
		available = r.size
	}
	if uint64(n) > available {
		n = int(available)
	}

	out := make([]string, n)
	start := writePos - uint64(n)
	for i := 0; i < n; i++ {
		out[i] = r.data[(start+uint64(i))&r.mask]
	}
	return out
}

// BatchRing for high-throughput batch processing with minimal contention.
type BatchRing struct {
	mu       sync.RWMutex
	batches  [][]string
	head     int
	tail     int
	count    int
	capacity int
	dropped  atomic.Uint64
}

// NewBatchRing creates a ring of batches for lock-free reads under contention.
func NewBatchRing(numBatches, batchSize int) *BatchRing {
	batches := make([][]string, numBatches)
	for i := range batches {
		batches[i] = make([]string, 0, batchSize)
	}
	return &BatchRing{
		batches:  batches,
		capacity: numBatches,
	}
}

// AddBatch atomically appends a batch (caller holds write lock or single writer).
func (r *BatchRing) AddBatch(msgs []string) {
	if len(msgs) == 0 {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.count >= r.capacity {
		// Drop oldest batch
		r.head = (r.head + 1) % r.capacity
		r.count--
		r.dropped.Add(uint64(len(r.batches[r.head])))
	}

	r.batches[r.tail] = append(r.batches[r.tail][:0], msgs...)
	r.tail = (r.tail + 1) % r.capacity
	r.count++
}

// TailBatches returns recent batches for streaming (copy-on-read).
func (r *BatchRing) TailBatches(n int) [][]string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if n > r.count {
		n = r.count
	}
	if n == 0 {
		return nil
	}

	out := make([][]string, n)
	idx := (r.tail - n + r.capacity) % r.capacity
	for i := 0; i < n; i++ {
		batch := r.batches[idx]
		out[i] = make([]string, len(batch))
		copy(out[i], batch)
		idx = (idx + 1) % r.capacity
	}
	return out
}

// ObjectPool for reusable byte slices and string builders.
type BytePool struct {
	pool sync.Pool
}

func NewBytePool(size int) *BytePool {
	return &BytePool{
		pool: sync.Pool{
			New: func() interface{} {
				b := make([]byte, 0, size)
				return &b
			},
		},
	}
}

func (p *BytePool) Get() *[]byte {
	return p.pool.Get().(*[]byte)
}

func (p *BytePool) Put(b *[]byte) {
	*b = (*b)[:0]
	p.pool.Put(b)
}
