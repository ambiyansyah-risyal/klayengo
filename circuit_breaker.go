package klayengo

import "time"

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
		config: config,
		state:  StateClosed,
	}
}

// Allow checks if the request should be allowed through the circuit breaker
func (cb *CircuitBreaker) Allow() bool {
	now := time.Now()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		if now.Sub(cb.lastFailure) >= cb.config.RecoveryTimeout {
			cb.state = StateHalfOpen
			cb.successes = 0
			return true
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
	now := time.Now()

	switch cb.state {
	case StateClosed:
		cb.failures++
		cb.lastFailure = now
		if cb.failures >= cb.config.FailureThreshold {
			cb.state = StateOpen
		}
	case StateOpen:
		// When open, update lastFailure but don't increment failures
		cb.lastFailure = now
	case StateHalfOpen:
		// When half-open, a failure should immediately open the circuit
		cb.failures++
		cb.lastFailure = now
		cb.state = StateOpen
		cb.successes = 0
	}
}

// RecordSuccess records a success in the circuit breaker
func (cb *CircuitBreaker) RecordSuccess() {
	switch cb.state {
	case StateClosed:
		// Success in closed state doesn't change anything
	case StateOpen:
		// Success in open state doesn't change anything
	case StateHalfOpen:
		cb.successes++
		if cb.successes >= cb.config.SuccessThreshold {
			cb.state = StateClosed
			cb.failures = 0
			cb.successes = 0
		}
	}
}
