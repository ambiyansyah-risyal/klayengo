package klayengo

import (
	"testing"
	"time"
)

func TestNewCircuitBreaker(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold: 3,
		RecoveryTimeout:  30 * time.Second,
		SuccessThreshold: 2,
	}

	cb := NewCircuitBreaker(config)

	if cb == nil {
		t.Fatal("NewCircuitBreaker() returned nil")
	}

	if cb.config.FailureThreshold != 3 {
		t.Errorf("Expected FailureThreshold=3, got %d", cb.config.FailureThreshold)
	}

	if cb.config.RecoveryTimeout != 30*time.Second {
		t.Errorf("Expected RecoveryTimeout=30s, got %v", cb.config.RecoveryTimeout)
	}

	if cb.config.SuccessThreshold != 2 {
		t.Errorf("Expected SuccessThreshold=2, got %d", cb.config.SuccessThreshold)
	}

	if CircuitState(cb.state) != StateClosed {
		t.Errorf("Expected initial state=Closed, got %v", CircuitState(cb.state))
	}
}

func TestNewCircuitBreakerDefaults(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{})

	if cb.config.FailureThreshold != 5 {
		t.Errorf("Expected default FailureThreshold=5, got %d", cb.config.FailureThreshold)
	}

	if cb.config.RecoveryTimeout != 60*time.Second {
		t.Errorf("Expected default RecoveryTimeout=60s, got %v", cb.config.RecoveryTimeout)
	}

	if cb.config.SuccessThreshold != 2 {
		t.Errorf("Expected default SuccessThreshold=2, got %d", cb.config.SuccessThreshold)
	}
}

func TestCircuitBreakerAllowClosed(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{})

	// Should allow requests when closed
	if !cb.Allow() {
		t.Error("Expected true when circuit breaker is closed")
	}

	if CircuitState(cb.state) != StateClosed {
		t.Errorf("Expected state=Closed, got %v", CircuitState(cb.state))
	}
}

func TestCircuitBreakerAllowOpen(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 2,
		RecoveryTimeout:  100 * time.Millisecond,
	})

	// Record failures to open circuit
	cb.RecordFailure()
	cb.RecordFailure()

	if CircuitState(cb.state) != StateOpen {
		t.Errorf("Expected state=Open after failures, got %v", CircuitState(cb.state))
	}

	// Should not allow requests when open
	if cb.Allow() {
		t.Error("Expected false when circuit breaker is open")
	}
}

func TestCircuitBreakerAllowHalfOpen(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 2,
		RecoveryTimeout:  50 * time.Millisecond,
		SuccessThreshold: 1,
	})

	// Open the circuit
	cb.RecordFailure()
	cb.RecordFailure()

	// Wait for recovery timeout
	time.Sleep(60 * time.Millisecond)

	// Should allow request and transition to half-open
	if !cb.Allow() {
		t.Error("Expected true when transitioning to half-open")
	}

	if CircuitState(cb.state) != StateHalfOpen {
		t.Errorf("Expected state=HalfOpen, got %v", cb.state)
	}
}

func TestCircuitBreakerRecordFailure(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 3,
	})

	// Record failures
	cb.RecordFailure()
	if cb.failures != 1 {
		t.Errorf("Expected failures=1, got %d", cb.failures)
	}
	if CircuitState(cb.state) != StateClosed {
		t.Errorf("Expected state=Closed after 1 failure, got %v", cb.state)
	}

	cb.RecordFailure()
	if cb.failures != 2 {
		t.Errorf("Expected failures=2, got %d", cb.failures)
	}
	if CircuitState(cb.state) != StateClosed {
		t.Errorf("Expected state=Closed after 2 failures, got %v", cb.state)
	}

	// Third failure should open the circuit
	cb.RecordFailure()
	if cb.failures != 3 {
		t.Errorf("Expected failures=3, got %d", cb.failures)
	}
	if CircuitState(cb.state) != StateOpen {
		t.Errorf("Expected state=Open after 3 failures, got %v", cb.state)
	}

	// Additional failures should not increase failure count when open
	cb.RecordFailure()
	if cb.failures != 3 {
		t.Errorf("Expected failures=3 (unchanged when open), got %d", cb.failures)
	}
}

func TestCircuitBreakerRecordSuccess(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 2,
		RecoveryTimeout:  10 * time.Millisecond, // Short timeout for test
		SuccessThreshold: 2,
	})

	// Open the circuit
	cb.RecordFailure()
	cb.RecordFailure()

	if CircuitState(cb.state) != StateOpen {
		t.Errorf("Expected state=Open, got %v", cb.state)
	}

	// Wait for recovery and transition to half-open
	time.Sleep(15 * time.Millisecond)
	allowed := cb.Allow() // This should transition to half-open

	if !allowed {
		t.Error("Expected true when transitioning to half-open")
	}

	if CircuitState(cb.state) != StateHalfOpen {
		t.Errorf("Expected state=HalfOpen, got %v", cb.state)
	}

	// Record successes
	cb.RecordSuccess()
	if cb.successes != 1 {
		t.Errorf("Expected successes=1, got %d", cb.successes)
	}
	if CircuitState(cb.state) != StateHalfOpen {
		t.Errorf("Expected state=HalfOpen after 1 success, got %v", cb.state)
	}

	// Second success should close the circuit
	cb.RecordSuccess()
	if cb.successes != 0 {
		t.Errorf("Expected successes=0 (reset after closing), got %d", cb.successes)
	}
	if CircuitState(cb.state) != StateClosed {
		t.Errorf("Expected state=Closed after 2 successes, got %v", cb.state)
	}

	// Verify failures are reset
	if cb.failures != 0 {
		t.Errorf("Expected failures=0 after closing, got %d", cb.failures)
	}
}

func TestCircuitBreakerRecoveryTimeout(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 2,
		RecoveryTimeout:  100 * time.Millisecond,
	})

	// Open the circuit
	cb.RecordFailure()
	cb.RecordFailure()

	// Should not allow immediately
	if cb.Allow() {
		t.Error("Expected false when circuit is open")
	}

	// Wait for recovery timeout
	time.Sleep(110 * time.Millisecond)

	// Should allow and transition to half-open
	if !cb.Allow() {
		t.Error("Expected true after recovery timeout")
	}

	if CircuitState(cb.state) != StateHalfOpen {
		t.Errorf("Expected state=HalfOpen, got %v", cb.state)
	}
}

func TestCircuitBreakerStateTransitions(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 2,
		RecoveryTimeout:  50 * time.Millisecond,
		SuccessThreshold: 1,
	})

	// Start closed
	if CircuitState(cb.state) != StateClosed {
		t.Errorf("Expected initial state=Closed, got %v", cb.state)
	}

	// Transition to open
	cb.RecordFailure()
	cb.RecordFailure()
	if CircuitState(cb.state) != StateOpen {
		t.Errorf("Expected state=Open after failures, got %v", cb.state)
	}

	// Wait and transition to half-open
	time.Sleep(60 * time.Millisecond)
	cb.Allow()
	if CircuitState(cb.state) != StateHalfOpen {
		t.Errorf("Expected state=HalfOpen, got %v", cb.state)
	}

	// Transition back to closed
	cb.RecordSuccess()
	if CircuitState(cb.state) != StateClosed {
		t.Errorf("Expected state=Closed after success, got %v", cb.state)
	}
}

func TestCircuitBreakerHalfOpenFailure(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 2,
		RecoveryTimeout:  50 * time.Millisecond,
		SuccessThreshold: 2,
	})

	// Open the circuit
	cb.RecordFailure()
	cb.RecordFailure()

	// Wait and transition to half-open
	time.Sleep(60 * time.Millisecond)
	cb.Allow()

	if CircuitState(cb.state) != StateHalfOpen {
		t.Errorf("Expected state=HalfOpen, got %v", cb.state)
	}

	// Record failure in half-open state
	cb.RecordFailure()

	// Should transition back to open
	if CircuitState(cb.state) != StateOpen {
		t.Errorf("Expected state=Open after failure in half-open, got %v", cb.state)
	}

	// Success count should be reset
	if cb.successes != 0 {
		t.Errorf("Expected successes=0 after failure, got %d", cb.successes)
	}
}

func TestCircuitBreakerConcurrentAccess(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 5,
		RecoveryTimeout:  10 * time.Millisecond,
		SuccessThreshold: 2,
	})

	// Test concurrent access
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				cb.Allow()
				if j%2 == 0 {
					cb.RecordSuccess()
				} else {
					cb.RecordFailure()
				}
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Circuit breaker should still be in a valid state
	if CircuitState(cb.state) != StateClosed && CircuitState(cb.state) != StateOpen && CircuitState(cb.state) != StateHalfOpen {
		t.Errorf("Invalid circuit breaker state after concurrent access: %v", cb.state)
	}
}

func TestCircuitBreakerWithZeroConfig(t *testing.T) {
	// Test with zero values (should use defaults)
	cb := NewCircuitBreaker(CircuitBreakerConfig{})

	// Should work with default values
	if !cb.Allow() {
		t.Error("Expected true with default config")
	}

	cb.RecordFailure()
	cb.RecordSuccess()

	if CircuitState(cb.state) != StateClosed {
		t.Errorf("Expected state=Closed with defaults, got %v", cb.state)
	}
}

func TestCircuitBreakerStateString(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 2, // Set threshold to 2 for this test
	})

	// Test state values
	if StateClosed != 0 {
		t.Errorf("Expected StateClosed=0, got %d", StateClosed)
	}

	if StateOpen != 1 {
		t.Errorf("Expected StateOpen=1, got %d", StateOpen)
	}

	if StateHalfOpen != 2 {
		t.Errorf("Expected StateHalfOpen=2, got %d", StateHalfOpen)
	}

	// Test state transitions maintain correct values
	cb.RecordFailure()
	cb.RecordFailure() // Should open

	if CircuitState(cb.state) != StateOpen {
		t.Errorf("Expected state=1 (Open), got %d", cb.state)
	}
}
