package syslog

import (
	"sync"
	"time"
)

// BatchHandler is a handler interface that receives batches of messages.
type BatchHandler interface {
	HandleBatch(messages []string)
}

// BatchCollector buffers incoming messages and flushes them in batches.
type BatchCollector struct {
	handler   BatchHandler
	batchSize int
	flushTime time.Duration

	mu      sync.Mutex
	batch   []string
	stopCh  chan struct{}
	flushCh chan struct{}
	doneCh  chan struct{}
}

// NewBatchCollector creates a collector that batches messages.
// batchSize: max events before auto-flush
// flushTime: max time to wait before flushing partial batch
func NewBatchCollector(handler BatchHandler, batchSize int, flushTime time.Duration) *BatchCollector {
	if batchSize <= 0 {
		batchSize = 1000
	}
	if flushTime <= 0 {
		flushTime = 100 * time.Millisecond
	}

	bc := &BatchCollector{
		handler:   handler,
		batchSize: batchSize,
		flushTime: flushTime,
		batch:     make([]string, 0, batchSize),
		stopCh:    make(chan struct{}),
		flushCh:   make(chan struct{}, 1),
		doneCh:    make(chan struct{}),
	}

	go bc.flusher()
	return bc
}

// Handle implements Handler interface for single-message compatibility.
func (bc *BatchCollector) Handle(message string) {
	bc.mu.Lock()
	bc.batch = append(bc.batch, message)
	needsFlush := len(bc.batch) >= bc.batchSize
	bc.mu.Unlock()

	if needsFlush {
		select {
		case bc.flushCh <- struct{}{}:
		default:
		}
	}
}

// flusher runs in a goroutine, flushing on timer or demand.
func (bc *BatchCollector) flusher() {
	defer close(bc.doneCh)
	ticker := time.NewTicker(bc.flushTime)
	defer ticker.Stop()

	for {
		select {
		case <-bc.stopCh:
			bc.flush()
			return
		case <-ticker.C:
			bc.flush()
		case <-bc.flushCh:
			bc.flush()
		}
	}
}

// flush sends the current batch to the handler.
func (bc *BatchCollector) flush() {
	bc.mu.Lock()
	if len(bc.batch) == 0 {
		bc.mu.Unlock()
		return
	}

	toSend := bc.batch
	bc.batch = make([]string, 0, bc.batchSize)
	bc.mu.Unlock()

	if bc.handler != nil {
		bc.handler.HandleBatch(toSend)
	}
}

// Stop gracefully shuts down the collector and flushes pending messages.
func (bc *BatchCollector) Stop() {
	close(bc.stopCh)
	<-bc.doneCh
}
