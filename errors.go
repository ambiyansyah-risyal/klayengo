package klayengo

import (
	"errors"
	"fmt"
	"time"
)

// Sentinel errors for common failure scenarios
var (
	// ErrCircuitOpen is returned when the circuit breaker is in open state
	ErrCircuitOpen = errors.New("klayengo: circuit open")

	// ErrRateLimited is returned when a request is denied due to rate limiting
	ErrRateLimited = errors.New("klayengo: rate limited")

	// ErrCacheMiss is returned when a cache lookup fails
	ErrCacheMiss = errors.New("klayengo: cache miss")

	// ErrRetryBudgetExceeded is returned when retry budget is exhausted
	ErrRetryBudgetExceeded = errors.New("klayengo: retry budget exceeded")
)

// IsTransient determines if an error represents a transient failure that might succeed on retry.
// Returns true for network errors, timeouts, 5xx server responses, and rate limiting (429).
// Returns false for 4xx client errors (except 429) and configuration errors.
func IsTransient(err error) bool {
	if err == nil {
		return false
	}

	// Check for our sentinel errors
	if errors.Is(err, ErrCircuitOpen) || errors.Is(err, ErrRateLimited) || errors.Is(err, ErrRetryBudgetExceeded) {
		return true
	}

	// Check for ClientError types
	var clientErr *ClientError
	if errors.As(err, &clientErr) {
		switch clientErr.Type {
		case ErrorTypeNetwork, ErrorTypeTimeout, ErrorTypeServer, ErrorTypeRateLimit, ErrorTypeCircuitOpen:
			return true
		case ErrorTypeClient:
			// 429 Too Many Requests is transient
			return clientErr.StatusCode == 429
		default:
			return false
		}
	}

	return false
}

// Error implements error interface.
func (e *ClientError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Cause != nil {
		msg := fmt.Sprintf("%s: %s (%v)", e.Type, e.Message, e.Cause)
		if e.RequestID != "" {
			msg = fmt.Sprintf("[%s] %s", e.RequestID, msg)
		}
		if e.Attempt > 0 {
			msg = fmt.Sprintf("%s (attempt %d/%d)", msg, e.Attempt, e.MaxRetries)
		}
		return msg
	}

	msg := fmt.Sprintf("%s: %s", e.Type, e.Message)
	if e.RequestID != "" {
		msg = fmt.Sprintf("[%s] %s", e.RequestID, msg)
	}
	if e.Attempt > 0 {
		msg = fmt.Sprintf("%s (attempt %d/%d)", msg, e.Attempt, e.MaxRetries)
	}
	return msg
}

// Unwrap returns the underlying cause.
func (e *ClientError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// Is compares error types for errors.Is.
func (e *ClientError) Is(target error) bool {
	if e == nil {
		return false
	}
	if targetErr, ok := target.(*ClientError); ok {
		return e.Type == targetErr.Type
	}
	return false
}

// DebugInfo renders a multi-line string with diagnostic context.
func (e *ClientError) DebugInfo() string {
	if e == nil {
		return "Error: <nil>"
	}
	info := fmt.Sprintf("Error Type: %s\n", e.Type)
	info += fmt.Sprintf("Message: %s\n", e.Message)
	if e.RequestID != "" {
		info += fmt.Sprintf("Request ID: %s\n", e.RequestID)
	}
	if e.Method != "" {
		info += fmt.Sprintf("Method: %s\n", e.Method)
	}
	if e.URL != "" {
		info += fmt.Sprintf("URL: %s\n", e.URL)
	}
	if e.Endpoint != "" {
		info += fmt.Sprintf("Endpoint: %s\n", e.Endpoint)
	}
	if e.StatusCode > 0 {
		info += fmt.Sprintf("Status Code: %d\n", e.StatusCode)
	}
	if e.Attempt > 0 {
		info += fmt.Sprintf("Attempt: %d/%d\n", e.Attempt, e.MaxRetries)
	}
	if !e.Timestamp.IsZero() {
		info += fmt.Sprintf("Timestamp: %s\n", e.Timestamp.Format(time.RFC3339))
	}
	if e.Duration > 0 {
		info += fmt.Sprintf("Duration: %v\n", e.Duration)
	}
	if e.Cause != nil {
		info += fmt.Sprintf("Cause: %v\n", e.Cause)
	}
	return info
}
