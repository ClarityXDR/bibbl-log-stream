package pipeline

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// WorkerPool manages concurrent event processing with backpressure.
type WorkerPool struct {
	workers   int
	batchSize int
	inputCh   chan Event
	batchCh   chan []Event
	stopCh    chan struct{}
	wg        sync.WaitGroup
	processor EventProcessor
	stats     PoolStats
}

type Event struct {
	SourceID string
	Data     []byte
	Metadata map[string]interface{}
}

type EventProcessor interface {
	Process(ctx context.Context, events []Event) error
}

type PoolStats struct {
	Processed atomic.Uint64
	Dropped   atomic.Uint64
	Errors    atomic.Uint64
	BatchTime atomic.Uint64 // microseconds
}

// PoolSnapshot is a point-in-time snapshot of pool statistics.
type PoolSnapshot struct {
	Processed uint64
	Dropped   uint64
	Errors    uint64
	BatchTime uint64 // microseconds
}

// NewWorkerPool creates a high-throughput batch processor.
func NewWorkerPool(workers, batchSize, queueDepth int, proc EventProcessor) *WorkerPool {
	if workers <= 0 {
		workers = 4
	}
	if batchSize <= 0 {
		batchSize = 1000
	}
	if queueDepth <= 0 {
		queueDepth = 10000
	}

	return &WorkerPool{
		workers:   workers,
		batchSize: batchSize,
		inputCh:   make(chan Event, queueDepth),
		batchCh:   make(chan []Event, workers*2),
		stopCh:    make(chan struct{}),
		processor: proc,
	}
}

// Start launches worker goroutines and batch collector.
func (p *WorkerPool) Start(ctx context.Context) {
	// Batch collector aggregates individual events into batches
	p.wg.Add(1)
	go p.batchCollector(ctx)

	// Worker pool processes batches concurrently
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker(ctx, i)
	}
}

// Submit enqueues an event (non-blocking with backpressure).
func (p *WorkerPool) Submit(event Event) bool {
	select {
	case p.inputCh <- event:
		return true
	default:
		p.stats.Dropped.Add(1)
		return false
	}
}

// batchCollector aggregates events into fixed-size batches with time-based flushing.
func (p *WorkerPool) batchCollector(ctx context.Context) {
	defer p.wg.Done()

	batch := make([]Event, 0, p.batchSize)
	ticker := time.NewTicker(100 * time.Millisecond) // Max latency tolerance
	defer ticker.Stop()

	flush := func() {
		if len(batch) == 0 {
			return
		}
		select {
		case p.batchCh <- batch:
			batch = make([]Event, 0, p.batchSize)
		case <-ctx.Done():
			return
		}
	}

	for {
		select {
		case <-ctx.Done():
			flush()
			close(p.batchCh)
			return
		case <-ticker.C:
			flush()
		case event := <-p.inputCh:
			batch = append(batch, event)
			if len(batch) >= p.batchSize {
				flush()
			}
		}
	}
}

// worker processes batches from the batch channel.
func (p *WorkerPool) worker(ctx context.Context, id int) {
	defer p.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case batch, ok := <-p.batchCh:
			if !ok {
				return
			}
			start := time.Now()
			if err := p.processor.Process(ctx, batch); err != nil {
				p.stats.Errors.Add(1)
			} else {
				p.stats.Processed.Add(uint64(len(batch)))
			}
			elapsed := time.Since(start).Microseconds()
			p.stats.BatchTime.Store(uint64(elapsed))
		}
	}
}

// Stop gracefully shuts down the pool.
func (p *WorkerPool) Stop() {
	close(p.inputCh)
	close(p.stopCh)
	p.wg.Wait()
}

// Snapshot returns a point-in-time copy of pool statistics.
func (p *WorkerPool) Snapshot() PoolSnapshot {
	return PoolSnapshot{
		Processed: p.stats.Processed.Load(),
		Dropped:   p.stats.Dropped.Load(),
		Errors:    p.stats.Errors.Load(),
		BatchTime: p.stats.BatchTime.Load(),
	}
}

// QueueDepth returns current input queue utilization.
func (p *WorkerPool) QueueDepth() int {
	return len(p.inputCh)
}
