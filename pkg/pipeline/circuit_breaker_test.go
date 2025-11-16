package pipeline

import (
	"errors"
	"testing"
	"time"
)

func TestCircuitBreakerClosed(t *testing.T) {
	cb := NewCircuitBreaker("test", 3, 100*time.Millisecond, 2)

	// Should allow calls when closed
	err := cb.Execute(func() error {
		return nil
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if cb.State() != StateClosed {
		t.Fatalf("Expected closed state, got %v", cb.State())
	}
}

func TestCircuitBreakerOpens(t *testing.T) {
	cb := NewCircuitBreaker("test", 3, 100*time.Millisecond, 2)

	// Generate failures
	for i := 0; i < 3; i++ {
		_ = cb.Execute(func() error {
			return errors.New("failure")
		})
	}

	if cb.State() != StateOpen {
		t.Fatalf("Expected open state after 3 failures, got %v", cb.State())
	}

	// Should reject calls when open
	err := cb.Execute(func() error {
		return nil
	})

	if err == nil || err.Error() != "circuit breaker open" {
		t.Fatalf("Expected 'circuit breaker open' error, got %v", err)
	}
}

func TestCircuitBreakerHalfOpen(t *testing.T) {
	cb := NewCircuitBreaker("test", 2, 50*time.Millisecond, 2)

	// Open the circuit
	for i := 0; i < 2; i++ {
		_ = cb.Execute(func() error { return errors.New("failure") })
	}

	if cb.State() != StateOpen {
		t.Fatalf("Expected open state, got %v", cb.State())
	}

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)

	// Next call should transition to half-open
	err := cb.Execute(func() error {
		return nil
	})

	if err != nil {
		t.Fatalf("Expected success in half-open, got %v", err)
	}

	// One more success should close the circuit
	err = cb.Execute(func() error {
		return nil
	})

	if err != nil {
		t.Fatalf("Expected success, got %v", err)
	}

	if cb.State() != StateClosed {
		t.Fatalf("Expected closed state after successes, got %v", cb.State())
	}
}

func TestCircuitBreakerHalfOpenFailure(t *testing.T) {
	cb := NewCircuitBreaker("test", 2, 50*time.Millisecond, 2)

	// Open the circuit
	for i := 0; i < 2; i++ {
		_ = cb.Execute(func() error { return errors.New("failure") })
	}

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)

	// Failure in half-open should reopen
	_ = cb.Execute(func() error {
		return errors.New("still failing")
	})

	if cb.State() != StateOpen {
		t.Fatalf("Expected open state after half-open failure, got %v", cb.State())
	}
}

func TestCircuitBreakerStats(t *testing.T) {
	cb := NewCircuitBreaker("test", 5, 100*time.Millisecond, 2)

	// Execute some calls
	for i := 0; i < 3; i++ {
		_ = cb.Execute(func() error { return nil })
	}

	for i := 0; i < 2; i++ {
		_ = cb.Execute(func() error { return errors.New("failure") })
	}

	stats := cb.Stats()

	if stats.TotalCalls != 5 {
		t.Errorf("Expected 5 total calls, got %d", stats.TotalCalls)
	}

	if stats.TotalSuccess != 3 {
		t.Errorf("Expected 3 successes, got %d", stats.TotalSuccess)
	}

	if stats.Failures != 2 {
		t.Errorf("Expected 2 failures, got %d", stats.Failures)
	}
}

func TestCircuitBreakerReset(t *testing.T) {
	cb := NewCircuitBreaker("test", 2, 100*time.Millisecond, 2)

	// Open the circuit
	for i := 0; i < 2; i++ {
		_ = cb.Execute(func() error { return errors.New("failure") })
	}

	if cb.State() != StateOpen {
		t.Fatalf("Expected open state, got %v", cb.State())
	}

	// Manual reset
	cb.Reset()

	if cb.State() != StateClosed {
		t.Fatalf("Expected closed state after reset, got %v", cb.State())
	}

	// Should allow calls again
	err := cb.Execute(func() error { return nil })
	if err != nil {
		t.Fatalf("Expected success after reset, got %v", err)
	}
}

func BenchmarkCircuitBreakerSuccess(b *testing.B) {
	cb := NewCircuitBreaker("bench", 5, 100*time.Millisecond, 2)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = cb.Execute(func() error {
				return nil
			})
		}
	})
}
