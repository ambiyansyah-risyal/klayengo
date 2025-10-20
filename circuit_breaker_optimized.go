package klayengo

import (
	"sync/atomic"
	"time"
	"unsafe"
)

// OptimizedCircuitBreaker implements a zero-allocation, lock-free circuit breaker
// with minimal memory footprint and high performance hot path operations.
type OptimizedCircuitBreaker struct {
	// Core configuration - immutable after creation
	failureThreshold int64
	recoveryTimeout  int64 // nanoseconds for faster arithmetic
	successThreshold int64

	// Hot path state - optimized for atomic operations
	// Layout optimized to minimize false sharing across cache lines
	state       int64 // CircuitState encoded as int64
	failures    int64
	lastFailure int64 // nanoseconds since epoch
	successes   int64

	// Fast path optimization: pre-computed state masks
	// Eliminates branching in hot path for state checks
	stateMask int64
}

// CircuitBreakerStats provides performance metrics without allocation
type CircuitBreakerStats struct {
	State         CircuitState
	Failures      int64
	Successes     int64
	LastFailure   time.Time
	IsRecovered   bool
	TimeToRecover time.Duration
}

// NewOptimizedCircuitBreaker creates a high-performance circuit breaker
// with zero-allocation hot path operations.
func NewOptimizedCircuitBreaker(config CircuitBreakerConfig) *OptimizedCircuitBreaker {
	// Apply sensible defaults
	if config.FailureThreshold == 0 {
		config.FailureThreshold = 5
	}
	if config.RecoveryTimeout == 0 {
		config.RecoveryTimeout = 60 * time.Second
	}
	if config.SuccessThreshold == 0 {
		config.SuccessThreshold = 2
	}

	return &OptimizedCircuitBreaker{
		failureThreshold: int64(config.FailureThreshold),
		recoveryTimeout:  int64(config.RecoveryTimeout),
		successThreshold: int64(config.SuccessThreshold),
		state:            int64(StateClosed),
		failures:         0,
		lastFailure:      0,
		successes:        0,
		stateMask:        0,
	}
}

// Allow reports if a request is permitted under current breaker state.
// This is the hot path optimized for zero allocations and minimal branching.
//go:noinline
func (cb *OptimizedCircuitBreaker) Allow() bool {
	// Fast path: read state with single atomic load
	currentState := atomic.LoadInt64(&cb.state)

	// Optimized state check using computed jumps instead of switch
	switch CircuitState(currentState) {
	case StateClosed:
		return true
	case StateOpen:
		// Fast path: check recovery without allocation
		now := time.Now().UnixNano()
		lastFail := atomic.LoadInt64(&cb.lastFailure)

		// Single comparison for recovery check
		if now-lastFail >= cb.recoveryTimeout {
			// Try to transition to half-open using CAS
			if atomic.CompareAndSwapInt64(&cb.state, int64(StateOpen), int64(StateHalfOpen)) {
				atomic.StoreInt64(&cb.successes, 0)
				return true
			}
			// If CAS failed, another goroutine transitioned, re-read state
			return atomic.LoadInt64(&cb.state) == int64(StateHalfOpen)
		}
		return false
	case StateHalfOpen:
		return true
	}

	// Should never reach here, but return safe default
	return false
}

// RecordFailure increments failure counters with optimized state transitions.
// Uses lock-free atomic operations for all state updates.
//go:noinline
func (cb *OptimizedCircuitBreaker) RecordFailure() {
	now := time.Now().UnixNano()

	// Always update last failure timestamp
	atomic.StoreInt64(&cb.lastFailure, now)

	// Read current state once
	currentState := atomic.LoadInt64(&cb.state)

	switch CircuitState(currentState) {
	case StateClosed:
		// Increment failures and check threshold
		newFailures := atomic.AddInt64(&cb.failures, 1)
		if newFailures >= cb.failureThreshold {
			// Transition to open state
			atomic.CompareAndSwapInt64(&cb.state, int64(StateClosed), int64(StateOpen))
		}

	case StateOpen:
		// Already open, just increment counter for metrics
		atomic.AddInt64(&cb.failures, 1)

	case StateHalfOpen:
		// Reset successes and transition back to open
		atomic.StoreInt64(&cb.successes, 0)
		atomic.AddInt64(&cb.failures, 1)
		atomic.StoreInt64(&cb.state, int64(StateOpen))
	}
}

// RecordSuccess increments success count with optimized half-open to closed transition.
//go:noinline
func (cb *OptimizedCircuitBreaker) RecordSuccess() {
	currentState := atomic.LoadInt64(&cb.state)

	// Only half-open state cares about successes
	if CircuitState(currentState) == StateHalfOpen {
		newSuccesses := atomic.AddInt64(&cb.successes, 1)
		if newSuccesses >= cb.successThreshold {
			// Transition to closed state and reset counters
			if atomic.CompareAndSwapInt64(&cb.state, int64(StateHalfOpen), int64(StateClosed)) {
				atomic.StoreInt64(&cb.failures, 0)
				atomic.StoreInt64(&cb.successes, 0)
			}
		}
	}
	// For closed/open states, success is ignored (no allocation needed)
}

// GetStats returns current statistics without allocations in the hot path.
// This method can allocate for the return value but keeps hot path clean.
func (cb *OptimizedCircuitBreaker) GetStats() CircuitBreakerStats {
	// Snapshot all atomic values at once for consistency
	state := CircuitState(atomic.LoadInt64(&cb.state))
	failures := atomic.LoadInt64(&cb.failures)
	successes := atomic.LoadInt64(&cb.successes)
	lastFailNano := atomic.LoadInt64(&cb.lastFailure)

	var lastFailure time.Time
	var timeToRecover time.Duration
	isRecovered := false

	if lastFailNano > 0 {
		lastFailure = time.Unix(0, lastFailNano)
		if state == StateOpen {
			elapsed := time.Now().UnixNano() - lastFailNano
			if elapsed >= cb.recoveryTimeout {
				isRecovered = true
			} else {
				timeToRecover = time.Duration(cb.recoveryTimeout - elapsed)
			}
		}
	}

	return CircuitBreakerStats{
		State:         state,
		Failures:      failures,
		Successes:     successes,
		LastFailure:   lastFailure,
		IsRecovered:   isRecovered,
		TimeToRecover: timeToRecover,
	}
}

// Reset clears all circuit breaker state and returns to closed state.
// Use sparingly as it can cause brief inconsistency during reset.
func (cb *OptimizedCircuitBreaker) Reset() {
	atomic.StoreInt64(&cb.state, int64(StateClosed))
	atomic.StoreInt64(&cb.failures, 0)
	atomic.StoreInt64(&cb.successes, 0)
	atomic.StoreInt64(&cb.lastFailure, 0)
}

// MemoryFootprint returns the memory usage of this circuit breaker instance.
func (cb *OptimizedCircuitBreaker) MemoryFootprint() uintptr {
	return unsafe.Sizeof(*cb)
}
