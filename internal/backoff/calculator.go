package backoff

import (
	"time"
)

// Calculator provides backoff calculation using configurable strategies.
// This centralizes backoff logic that was previously duplicated across Client and DefaultRetryPolicy.
type Calculator struct {
	strategy Strategy
}

// NewCalculator creates a new backoff calculator with the specified strategy.
func NewCalculator(strategy Strategy) *Calculator {
	return &Calculator{
		strategy: strategy,
	}
}

// Calculate computes the backoff duration for the given attempt and parameters.
// It delegates to the configured strategy for the actual calculation.
func (c *Calculator) Calculate(attempt int, initialBackoff, maxBackoff time.Duration, multiplier, jitter float64) time.Duration {
	return c.strategy.Calculate(attempt, initialBackoff, maxBackoff, multiplier, jitter)
}

// SetStrategy updates the backoff strategy used by this calculator.
// This allows for runtime strategy changes if needed.
func (c *Calculator) SetStrategy(strategy Strategy) {
	c.strategy = strategy
}

// GetStrategy returns the current strategy being used by this calculator.
func (c *Calculator) GetStrategy() Strategy {
	return c.strategy
}

// GetExponentialJitterCalculator returns a calculator configured with exponential jitter strategy.
// This is a convenience function for the most common use case.
func GetExponentialJitterCalculator() *Calculator {
	return NewCalculator(ExponentialJitterStrategy{})
}

// GetDecorrelatedJitterCalculator returns a calculator configured with decorrelated jitter strategy.
// This is a convenience function for AWS-style decorrelated jitter.
func GetDecorrelatedJitterCalculator() *Calculator {
	return NewCalculator(DecorrelatedJitterStrategy{})
}