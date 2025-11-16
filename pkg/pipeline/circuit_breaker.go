package pipeline

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

// CircuitState represents the current state of a circuit breaker.
type CircuitState int32

const (
	StateClosed   CircuitState = iota // Normal operation
	StateOpen                         // Failing, reject requests
	StateHalfOpen                     // Testing if recovery possible
)

// CircuitBreaker implements the circuit breaker pattern for resilient output handling.
type CircuitBreaker struct {
	name string

	// Configuration
	maxFailures  uint32        // Failures before opening
	timeout      time.Duration // Time to wait before half-open
	successesReq uint32        // Successes needed in half-open to close

	// State
	state        atomic.Int32  // CircuitState
	failures     atomic.Uint32 // Current failure count
	successes    atomic.Uint32 // Successes in half-open state
	lastFailTime atomic.Int64  // Unix nano of last failure

	// Metrics
	totalCalls   atomic.Uint64
	totalSuccess atomic.Uint64
	totalReject  atomic.Uint64

	mu sync.RWMutex
}

// NewCircuitBreaker creates a circuit breaker with the given parameters.
func NewCircuitBreaker(name string, maxFailures uint32, timeout time.Duration, successesReq uint32) *CircuitBreaker {
	if maxFailures == 0 {
		maxFailures = 5
	}
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	if successesReq == 0 {
		successesReq = 2
	}

	cb := &CircuitBreaker{
		name:         name,
		maxFailures:  maxFailures,
		timeout:      timeout,
		successesReq: successesReq,
	}
	cb.state.Store(int32(StateClosed))
	return cb
}

// Execute runs the given function through the circuit breaker.
func (cb *CircuitBreaker) Execute(fn func() error) error {
	cb.totalCalls.Add(1)

	state := CircuitState(cb.state.Load())
	if state == StateOpen {
		// Check if timeout has elapsed
		lastFail := time.Unix(0, cb.lastFailTime.Load())
		if time.Since(lastFail) > cb.timeout {
			// Transition to half-open
			if cb.state.CompareAndSwap(int32(StateOpen), int32(StateHalfOpen)) {
				cb.successes.Store(0)
			}
			state = StateHalfOpen
		} else {
			cb.totalReject.Add(1)
			return errors.New("circuit breaker open")
		}
	}

	// Execute function
	err := fn()

	if err != nil {
		cb.onFailure()
		return err
	}

	cb.onSuccess()
	return nil
}

// onFailure handles a failed execution.
func (cb *CircuitBreaker) onFailure() {
	cb.failures.Add(1)
	cb.lastFailTime.Store(time.Now().UnixNano())

	state := CircuitState(cb.state.Load())

	switch state {
	case StateClosed:
		if cb.failures.Load() >= cb.maxFailures {
			cb.state.Store(int32(StateOpen))
		}
	case StateHalfOpen:
		// Single failure in half-open -> back to open
		cb.state.Store(int32(StateOpen))
		cb.failures.Store(0)
		cb.successes.Store(0)
	}
}

// onSuccess handles a successful execution.
func (cb *CircuitBreaker) onSuccess() {
	cb.totalSuccess.Add(1)

	state := CircuitState(cb.state.Load())

	switch state {
	case StateClosed:
		// Reset failure counter on success
		cb.failures.Store(0)

	case StateHalfOpen:
		successes := cb.successes.Add(1)
		if successes >= cb.successesReq {
			// Enough successes -> close circuit
			cb.state.Store(int32(StateClosed))
			cb.failures.Store(0)
			cb.successes.Store(0)
		}
	}
}

// State returns the current circuit state.
func (cb *CircuitBreaker) State() CircuitState {
	return CircuitState(cb.state.Load())
}

// Stats returns current circuit breaker statistics.
type CircuitStats struct {
	Name         string
	State        string
	Failures     uint32
	TotalCalls   uint64
	TotalSuccess uint64
	TotalReject  uint64
}

func (cb *CircuitBreaker) Stats() CircuitStats {
	state := cb.State()
	stateName := "closed"
	switch state {
	case StateOpen:
		stateName = "open"
	case StateHalfOpen:
		stateName = "half-open"
	}

	return CircuitStats{
		Name:         cb.name,
		State:        stateName,
		Failures:     cb.failures.Load(),
		TotalCalls:   cb.totalCalls.Load(),
		TotalSuccess: cb.totalSuccess.Load(),
		TotalReject:  cb.totalReject.Load(),
	}
}

// Reset manually resets the circuit breaker to closed state.
func (cb *CircuitBreaker) Reset() {
	cb.state.Store(int32(StateClosed))
	cb.failures.Store(0)
	cb.successes.Store(0)
}
