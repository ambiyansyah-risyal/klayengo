package klayengo

import (
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	internalbackoff "github.com/ambiyansyah-risyal/klayengo/internal/backoff"
)

// NewDefaultRetryPolicy creates a retry policy with configurable backoff strategy that only
// retries idempotent methods by default.
func NewDefaultRetryPolicy(maxRetries int, initialBackoff, maxBackoff time.Duration, multiplier, jitter float64) *DefaultRetryPolicy {
	policy := &DefaultRetryPolicy{
		maxRetries:        maxRetries,
		initialBackoff:    initialBackoff,
		maxBackoff:        maxBackoff,
		backoffMultiplier: multiplier,
		jitter:            jitter,
		backoffStrategy:   ExponentialJitter, // Default to current behavior
		isIdempotent:      DefaultIsIdempotent,
	}
	policy.backoffCalculator = internalbackoff.GetExponentialJitterCalculator()
	return policy
}

// NewDefaultRetryPolicyWithStrategy creates a retry policy with a specific backoff strategy.
func NewDefaultRetryPolicyWithStrategy(maxRetries int, initialBackoff, maxBackoff time.Duration, multiplier, jitter float64, strategy BackoffStrategy) *DefaultRetryPolicy {
	policy := &DefaultRetryPolicy{
		maxRetries:        maxRetries,
		initialBackoff:    initialBackoff,
		maxBackoff:        maxBackoff,
		backoffMultiplier: multiplier,
		jitter:            jitter,
		backoffStrategy:   strategy,
		isIdempotent:      DefaultIsIdempotent,
	}

	// Initialize the appropriate calculator based on strategy
	switch strategy {
	case ExponentialJitter:
		policy.backoffCalculator = internalbackoff.GetExponentialJitterCalculator()
	case DecorrelatedJitter:
		policy.backoffCalculator = internalbackoff.GetDecorrelatedJitterCalculator()
	default:
		policy.backoffCalculator = internalbackoff.GetExponentialJitterCalculator()
	}

	return policy
}

// ShouldRetry implements the RetryPolicy interface.
func (p *DefaultRetryPolicy) ShouldRetry(resp *http.Response, err error, attempt int) (time.Duration, bool) {
	if attempt >= p.maxRetries {
		return 0, false
	}

	// Don't retry if the method is not idempotent
	if resp != nil && !p.isIdempotent(resp.Request.Method) {
		return 0, false
	}

	// Check if we should retry based on error or response
	shouldRetry := false
	var delay time.Duration

	if err != nil {
		// Network errors are generally retryable
		shouldRetry = true
	} else if resp != nil {
		// Check for specific status codes
		if resp.StatusCode == 429 || resp.StatusCode >= 500 {
			shouldRetry = true
			// Parse Retry-After header for 429/503 responses
			delay = parseRetryAfter(resp.Header.Get("Retry-After"))
		}
	}

	if !shouldRetry {
		return 0, false
	}

	// If no Retry-After delay was parsed, use exponential backoff
	if delay == 0 {
		delay = p.calculateBackoff(attempt)
	}

	return delay, true
}

// DefaultIsIdempotent returns true for idempotent HTTP methods.
func DefaultIsIdempotent(method string) bool {
	switch method {
	case "GET", "HEAD", "PUT", "DELETE", "OPTIONS":
		return true
	default:
		return false
	}
}

// parseRetryAfter parses the Retry-After header value.
// It supports both delay-seconds format and HTTP-date format.
func parseRetryAfter(value string) time.Duration {
	if value == "" {
		return 0
	}

	// Try parsing as seconds first
	if seconds, err := strconv.Atoi(strings.TrimSpace(value)); err == nil {
		if seconds > 0 {
			delay := time.Duration(seconds) * time.Second
			if delay > time.Hour {
				delay = time.Hour // Cap at 1 hour
			}
			return delay
		}
	}

	// Try parsing as HTTP-date
	if t, err := http.ParseTime(value); err == nil {
		delay := time.Until(t)
		if delay > 0 && delay <= time.Hour { // Cap at 1 hour
			return delay
		}
	}

	return 0
}

func (p *DefaultRetryPolicy) calculateBackoff(attempt int) time.Duration {
	if p.backoffCalculator == nil {
		// Fallback to direct calculation if calculator is not initialized
		switch p.backoffStrategy {
		case ExponentialJitter:
			return p.calculateExponentialBackoff(attempt)
		case DecorrelatedJitter:
			return p.calculateDecorrelatedBackoff(attempt)
		default:
			return p.calculateExponentialBackoff(attempt)
		}
	}
	return p.backoffCalculator.Calculate(attempt, p.initialBackoff, p.maxBackoff, p.backoffMultiplier, p.jitter)
}

func (p *DefaultRetryPolicy) calculateExponentialBackoff(attempt int) time.Duration {
	if attempt < 0 {
		attempt = 0
	}

	// Prevent overflow by limiting attempt
	if attempt > 30 {
		attempt = 30
	}

	backoff := time.Duration(float64(p.initialBackoff) * internalbackoff.Pow(p.backoffMultiplier, attempt))
	if backoff < 0 || backoff > p.maxBackoff {
		backoff = p.maxBackoff
	}

	jitter := p.jitter
	if jitter < 0 {
		jitter = 0
	}
	if jitter > 1 {
		jitter = 1
	}
	if jitter > 0 {
		jitterAmount := time.Duration(float64(backoff) * jitter * rand.Float64())
		if backoff+jitterAmount > p.maxBackoff {
			backoff = p.maxBackoff
		} else {
			backoff += jitterAmount
		}
	}
	return backoff
}

func (p *DefaultRetryPolicy) calculateDecorrelatedBackoff(attempt int) time.Duration {
	// Decorrelated jitter as per AWS: https://aws.amazon.com/blogs/architecture/exponential-backoff-and-jitter/
	// Formula: random_between(base, min(cap, base * 3))
	// For subsequent attempts: random_between(base, min(cap, previous_delay * 3))

	if attempt <= 0 {
		return p.initialBackoff
	}

	// Prevent overflow by limiting attempt
	if attempt > 10 {
		attempt = 10
	}

	// For decorrelated jitter, we need to track the previous delay
	// Since we don't have state, we'll use a simplified version:
	// random_between(base, min(cap, base * 3^attempt))

	base := float64(p.initialBackoff)
	factor := internalbackoff.Pow(3.0, attempt) // Use 3x multiplier for decorrelated jitter
	upper := base * factor

	// Prevent overflow and respect maxBackoff
	maxBackoffFloat := float64(p.maxBackoff)
	if upper > maxBackoffFloat || upper < 0 {
		upper = maxBackoffFloat
	}

	// Ensure upper is at least base
	if upper < base {
		upper = base
	}

	// Generate random delay between base and upper
	delay := base + rand.Float64()*(upper-base)

	result := time.Duration(delay)
	if result < 0 || result > p.maxBackoff {
		result = p.maxBackoff
	}

	return result
}

// NewRetryBudget creates a new retry budget tracker.
func NewRetryBudget(maxRetries int, perWindow time.Duration) *RetryBudget {
	return &RetryBudget{
		maxRetries:  int64(maxRetries),
		perWindow:   perWindow,
		window:      int64(perWindow),
		current:     0,
		windowStart: time.Now().UnixNano(),
	}
}

// Allow checks if a retry is allowed under the current budget.
func (rb *RetryBudget) Allow() bool {
	now := time.Now().UnixNano()
	windowStart := atomic.LoadInt64(&rb.windowStart)

	// Check if we need to reset the window
	if now-windowStart >= int64(rb.perWindow) {
		// Try to reset the window
		if atomic.CompareAndSwapInt64(&rb.windowStart, windowStart, now) {
			atomic.StoreInt64(&rb.current, 0)
		}
	}

	// Check current retry count
	current := atomic.LoadInt64(&rb.current)
	if current >= rb.maxRetries {
		return false
	}

	// Increment and check again
	newCurrent := atomic.AddInt64(&rb.current, 1)
	return newCurrent <= rb.maxRetries
}

// GetStats returns current retry budget statistics.
func (rb *RetryBudget) GetStats() (current, max int64, windowStart time.Time) {
	return atomic.LoadInt64(&rb.current),
		rb.maxRetries,
		time.Unix(0, atomic.LoadInt64(&rb.windowStart))
}
