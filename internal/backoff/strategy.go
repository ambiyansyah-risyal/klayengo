package backoff

import (
	"math/rand"
	"time"
)

// Strategy defines the interface for backoff calculation algorithms.
// This allows for extensible backoff strategies while maintaining consistent behavior.
type Strategy interface {
	// Calculate returns the backoff duration for the given attempt number and parameters.
	Calculate(attempt int, initialBackoff, maxBackoff time.Duration, multiplier, jitter float64) time.Duration
}

// ExponentialJitterStrategy implements exponential backoff with uniform jitter.
// This is the current default behavior maintained for backward compatibility.
type ExponentialJitterStrategy struct{}

// Calculate implements the Strategy interface for exponential backoff with jitter.
func (s ExponentialJitterStrategy) Calculate(attempt int, initialBackoff, maxBackoff time.Duration, multiplier, jitter float64) time.Duration {
	if attempt < 0 {
		attempt = 0
	}

	// Prevent overflow by limiting attempt
	if attempt > 30 {
		attempt = 30
	}

	backoff := time.Duration(float64(initialBackoff) * pow(multiplier, attempt))
	if backoff < 0 || backoff > maxBackoff {
		backoff = maxBackoff
	}

	// Apply jitter
	jitter = clampJitter(jitter)
	if jitter > 0 {
		jitterAmount := time.Duration(float64(backoff) * jitter * rand.Float64())
		if backoff+jitterAmount > maxBackoff {
			backoff = maxBackoff
		} else {
			backoff += jitterAmount
		}
	}
	return backoff
}

// DecorrelatedJitterStrategy implements decorrelated jitter as per AWS paper.
// This provides smoother tail latencies compared to exponential jitter.
type DecorrelatedJitterStrategy struct{}

// Calculate implements the Strategy interface for decorrelated jitter.
func (s DecorrelatedJitterStrategy) Calculate(attempt int, initialBackoff, maxBackoff time.Duration, multiplier, jitter float64) time.Duration {
	// Decorrelated jitter as per AWS: https://aws.amazon.com/blogs/architecture/exponential-backoff-and-jitter/
	// Formula: random_between(base, min(cap, base * 3))
	// For subsequent attempts: random_between(base, min(cap, previous_delay * 3))

	if attempt <= 0 {
		return initialBackoff
	}

	// Prevent overflow by limiting attempt
	if attempt > 10 {
		attempt = 10
	}

	// For decorrelated jitter, we need to track the previous delay
	// Since we don't have state, we'll use a simplified version:
	// random_between(base, min(cap, base * 3^attempt))

	base := float64(initialBackoff)
	factor := pow(3.0, attempt) // Use 3x multiplier for decorrelated jitter
	upper := base * factor

	// Prevent overflow and respect maxBackoff
	maxBackoffFloat := float64(maxBackoff)
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
	if result < 0 || result > maxBackoff {
		result = maxBackoff
	}

	return result
}

// clampJitter ensures jitter is within valid bounds [0, 1].
func clampJitter(jitter float64) float64 {
	if jitter < 0 {
		return 0
	}
	if jitter > 1 {
		return 1
	}
	return jitter
}

// pow calculates base^exponent using integer exponentiation.
// This is a helper function maintained for consistency with existing behavior.
func pow(base float64, exponent int) float64 {
	result := 1.0
	for i := 0; i < exponent; i++ {
		result *= base
	}
	return result
}

// Pow is a public version of pow for backward compatibility.
func Pow(base float64, exponent int) float64 {
	return pow(base, exponent)
}
