package klayengo

import (
	"sync/atomic"
	"time"
)

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	if config.FailureThreshold == 0 {
		config.FailureThreshold = 5
	}
	if config.RecoveryTimeout == 0 {
		config.RecoveryTimeout = 60 * time.Second
	}
	if config.SuccessThreshold == 0 {
		config.SuccessThreshold = 2
	}

	return &CircuitBreaker{
		config:      config,
		state:       int64(StateClosed),
		failures:    0,
		lastFailure: 0,
		successes:   0,
	}
}

// Allow checks if the request should be allowed through the circuit breaker
func (cb *CircuitBreaker) Allow() bool {
	now := time.Now().UnixNano()
	state := CircuitState(atomic.LoadInt64(&cb.state))

	switch state {
	case StateClosed:
		return true
	case StateOpen:
		lastFailure := atomic.LoadInt64(&cb.lastFailure)
		if now-lastFailure >= int64(cb.config.RecoveryTimeout) {
			// Try to transition to half-open
			if atomic.CompareAndSwapInt64(&cb.state, int64(StateOpen), int64(StateHalfOpen)) {
				atomic.StoreInt64(&cb.successes, 0)
				return true
			}
		}
		return false
	case StateHalfOpen:
		return true
	default:
		return false
	}
}

// RecordFailure records a failure in the circuit breaker
func (cb *CircuitBreaker) RecordFailure() {
	now := time.Now().UnixNano()
	atomic.StoreInt64(&cb.lastFailure, now)

	state := CircuitState(atomic.LoadInt64(&cb.state))

	switch state {
	case StateClosed:
		failures := atomic.AddInt64(&cb.failures, 1)
		if failures >= int64(cb.config.FailureThreshold) {
			atomic.StoreInt64(&cb.state, int64(StateOpen))
		}
	case StateOpen:
		// When open, just update lastFailure
	case StateHalfOpen:
		// When half-open, a failure should immediately open the circuit
		atomic.AddInt64(&cb.failures, 1)
		atomic.StoreInt64(&cb.state, int64(StateOpen))
		atomic.StoreInt64(&cb.successes, 0)
	}
}

// RecordSuccess records a success in the circuit breaker
func (cb *CircuitBreaker) RecordSuccess() {
	state := CircuitState(atomic.LoadInt64(&cb.state))

	switch state {
	case StateClosed:
		// Success in closed state doesn't change anything
	case StateOpen:
		// Success in open state doesn't change anything
	case StateHalfOpen:
		successes := atomic.AddInt64(&cb.successes, 1)
		if successes >= int64(cb.config.SuccessThreshold) {
			atomic.StoreInt64(&cb.state, int64(StateClosed))
			atomic.StoreInt64(&cb.failures, 0)
			atomic.StoreInt64(&cb.successes, 0)
		}
	}
}
