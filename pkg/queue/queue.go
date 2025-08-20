package queue

import (
	"sync"
	"time"
)

// Item represents a queued payload with enqueue timestamp.
type Item struct {
    Data []byte
    TS   int64
}

// Stats summarizes queue state.
type Stats struct {
    Depth int
    Capacity int
    Dropped uint64
    OldestUnix int64
    NewestUnix int64
}

// Queue interface for different backends.
type Queue interface {
    Enqueue(data []byte, ts time.Time) bool
    Dequeue(max int) []Item
    Stats() Stats
}

// MemQueue simple bounded FIFO.
type MemQueue struct {
    mu sync.Mutex
    items []Item
    capacity int
    dropped uint64
}

func NewMem(capacity int) *MemQueue {
    if capacity <= 0 { capacity = 10000 }
    return &MemQueue{capacity: capacity, items: make([]Item, 0, capacity)}
}

func (q *MemQueue) Enqueue(data []byte, ts time.Time) bool {
    q.mu.Lock(); defer q.mu.Unlock()
    if len(q.items) >= q.capacity {
        // drop oldest
        q.items = q.items[1:]
        q.dropped++
    }
    cp := append([]byte(nil), data...)
    q.items = append(q.items, Item{Data: cp, TS: ts.Unix()})
    return true
}

func (q *MemQueue) Dequeue(max int) []Item {
    q.mu.Lock(); defer q.mu.Unlock()
    if len(q.items) == 0 { return nil }
    n := len(q.items)
    if max > 0 && max < n { n = max }
    out := make([]Item, n)
    copy(out, q.items[:n])
    q.items = q.items[n:]
    return out
}

func (q *MemQueue) Stats() Stats {
    q.mu.Lock(); defer q.mu.Unlock()
    st := Stats{Depth: len(q.items), Capacity: q.capacity, Dropped: q.dropped}
    if len(q.items) > 0 {
        st.OldestUnix = q.items[0].TS
        st.NewestUnix = q.items[len(q.items)-1].TS
    }
    return st
}
