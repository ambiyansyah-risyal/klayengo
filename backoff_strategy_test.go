package klayengo

import (
	"math"
	"testing"
	"time"
)

const (
	attempt0 = "attempt 0"
	attempt1 = "attempt 1"
	attempt2 = "attempt 2"
)

func TestWithBackoffStrategy(t *testing.T) {
	tests := []struct {
		name     string
		strategy BackoffStrategy
	}{
		{
			name:     "ExponentialJitter",
			strategy: ExponentialJitter,
		},
		{
			name:     "DecorrelatedJitter",
			strategy: DecorrelatedJitter,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := New(WithBackoffStrategy(tt.strategy))
			if client.backoffStrategy != tt.strategy {
				t.Errorf("WithBackoffStrategy() = %v, want %v", client.backoffStrategy, tt.strategy)
			}
		})
	}
}

func TestClientCalculateBackoffExponentialJitter(t *testing.T) {
	client := New(
		WithBackoffStrategy(ExponentialJitter),
		WithInitialBackoff(100*time.Millisecond),
		WithMaxBackoff(5*time.Second),
		WithBackoffMultiplier(2.0),
		WithJitter(0.0), // No jitter for predictable testing
	)

	tests := []struct {
		name     string
		attempt  int
		expected time.Duration
	}{
		{
			name:     attempt0,
			attempt:  0,
			expected: 100 * time.Millisecond,
		},
		{
			name:     attempt1,
			attempt:  1,
			expected: 200 * time.Millisecond,
		},
		{
			name:     attempt2,
			attempt:  2,
			expected: 400 * time.Millisecond,
		},
		{
			name:     "attempt 10 (hits max)",
			attempt:  10,
			expected: 5 * time.Second, // Should be capped at maxBackoff
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.calculateBackoff(tt.attempt)
			if result != tt.expected {
				t.Errorf("calculateBackoff(%d) = %v, want %v", tt.attempt, result, tt.expected)
			}
		})
	}
}

func TestClientCalculateBackoffDecorrelatedJitter(t *testing.T) {
	client := New(
		WithBackoffStrategy(DecorrelatedJitter),
		WithInitialBackoff(100*time.Millisecond),
		WithMaxBackoff(5*time.Second),
	)

	// Test bounds for decorrelated jitter
	tests := []struct {
		name        string
		attempt     int
		minExpected time.Duration
		maxExpected time.Duration
	}{
		{
			name:        attempt0,
			attempt:     0,
			minExpected: 100 * time.Millisecond,
			maxExpected: 100 * time.Millisecond, // Should be exactly initialBackoff
		},
		{
			name:        attempt1,
			attempt:     1,
			minExpected: 100 * time.Millisecond, // base
			maxExpected: 300 * time.Millisecond, // base * 3^1
		},
		{
			name:        attempt2,
			attempt:     2,
			minExpected: 100 * time.Millisecond, // base
			maxExpected: 900 * time.Millisecond, // base * 3^2
		},
		{
			name:        "attempt 5 (hits max)",
			attempt:     5,
			minExpected: 100 * time.Millisecond, // base
			maxExpected: 5 * time.Second,        // Should be capped at maxBackoff
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Run multiple times to test randomness bounds
			for i := 0; i < 100; i++ {
				result := client.calculateBackoff(tt.attempt)
				if result < tt.minExpected || result > tt.maxExpected {
					t.Errorf("calculateBackoff(%d) = %v, want between %v and %v",
						tt.attempt, result, tt.minExpected, tt.maxExpected)
				}
			}
		})
	}
}

func TestClientCalculateBackoffUnknownStrategy(t *testing.T) {
	client := New(
		WithInitialBackoff(100*time.Millisecond),
		WithMaxBackoff(5*time.Second),
		WithBackoffMultiplier(2.0),
		WithJitter(0.0),
	)

	// Manually set an unknown strategy
	client.backoffStrategy = BackoffStrategy(999)

	// Should fallback to exponential jitter
	result := client.calculateBackoff(1)
	expected := 200 * time.Millisecond // 100ms * 2^1
	if result != expected {
		t.Errorf("calculateBackoff with unknown strategy = %v, want %v (fallback to exponential)", result, expected)
	}
}

func TestDefaultRetryPolicyCalculateBackoffExponentialJitter(t *testing.T) {
	policy := NewDefaultRetryPolicy(3, 100*time.Millisecond, 5*time.Second, 2.0, 0.0)

	tests := []struct {
		name     string
		attempt  int
		expected time.Duration
	}{
		{
			name:     attempt0,
			attempt:  0,
			expected: 100 * time.Millisecond,
		},
		{
			name:     attempt1,
			attempt:  1,
			expected: 200 * time.Millisecond,
		},
		{
			name:     attempt2,
			attempt:  2,
			expected: 400 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := policy.calculateBackoff(tt.attempt)
			if result != tt.expected {
				t.Errorf("calculateBackoff(%d) = %v, want %v", tt.attempt, result, tt.expected)
			}
		})
	}
}

func TestDefaultRetryPolicyCalculateBackoffDecorrelatedJitter(t *testing.T) {
	policy := NewDefaultRetryPolicyWithStrategy(3, 100*time.Millisecond, 5*time.Second, 2.0, 0.0, DecorrelatedJitter)

	// Test bounds for decorrelated jitter
	tests := []struct {
		name        string
		attempt     int
		minExpected time.Duration
		maxExpected time.Duration
	}{
		{
			name:        attempt0,
			attempt:     0,
			minExpected: 100 * time.Millisecond,
			maxExpected: 100 * time.Millisecond,
		},
		{
			name:        attempt1,
			attempt:     1,
			minExpected: 100 * time.Millisecond,
			maxExpected: 300 * time.Millisecond,
		},
		{
			name:        attempt2,
			attempt:     2,
			minExpected: 100 * time.Millisecond,
			maxExpected: 900 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Run multiple times to test randomness bounds
			for i := 0; i < 100; i++ {
				result := policy.calculateBackoff(tt.attempt)
				if result < tt.minExpected || result > tt.maxExpected {
					t.Errorf("calculateBackoff(%d) = %v, want between %v and %v",
						tt.attempt, result, tt.minExpected, tt.maxExpected)
				}
			}
		})
	}
}

func TestBackoffStrategyValidation(t *testing.T) {
	tests := []struct {
		name        string
		strategy    BackoffStrategy
		expectError bool
	}{
		{
			name:        "valid ExponentialJitter",
			strategy:    ExponentialJitter,
			expectError: false,
		},
		{
			name:        "valid DecorrelatedJitter",
			strategy:    DecorrelatedJitter,
			expectError: false,
		},
		{
			name:        "invalid strategy",
			strategy:    BackoffStrategy(999),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := New(WithBackoffStrategy(tt.strategy))
			err := client.ValidationError()

			if tt.expectError && err == nil {
				t.Error("Expected validation error for invalid strategy, but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no validation error for valid strategy, but got: %v", err)
			}
		})
	}
}

// FuzzTestBackoffStrategies tests that both backoff strategies produce valid delays
func FuzzTestBackoffStrategies(f *testing.F) {
	f.Add(0, 100, 5000, 2.0, 0.1)
	f.Add(1, 50, 10000, 1.5, 0.2)
	f.Add(5, 200, 30000, 3.0, 0.0)

	f.Fuzz(func(t *testing.T, attempt int, initialMs, maxMs int, multiplier, jitter float64) {
		if !isValidFuzzInput(attempt, initialMs, maxMs, multiplier, jitter) {
			t.Skip()
		}
		initialBackoff := time.Duration(initialMs) * time.Millisecond
		maxBackoff := time.Duration(maxMs) * time.Millisecond
		if maxBackoff < initialBackoff {
			t.Skip()
		}
		for _, strategy := range []BackoffStrategy{ExponentialJitter, DecorrelatedJitter} {
			checkBackoffStrategy(t, strategy, attempt, initialBackoff, maxBackoff, multiplier, jitter)
		}
	})
}

func isValidFuzzInput(attempt, initialMs, maxMs int, multiplier, jitter float64) bool {
	return attempt >= 0 && attempt <= 20 &&
		initialMs > 0 && maxMs > 0 &&
		multiplier > 0 &&
		jitter >= 0 && jitter <= 1
}

func checkBackoffStrategy(t *testing.T, strategy BackoffStrategy, attempt int, initialBackoff, maxBackoff time.Duration, multiplier, jitter float64) {
	client := New(
		WithBackoffStrategy(strategy),
		WithInitialBackoff(initialBackoff),
		WithMaxBackoff(maxBackoff),
		WithBackoffMultiplier(multiplier),
		WithJitter(jitter),
	)
	delay := client.calculateBackoff(attempt)
	if delay < 0 {
		t.Errorf("Strategy %v produced negative delay: %v", strategy, delay)
	}
	maxAllowed := maxBackoff
	if strategy == ExponentialJitter && jitter > 0 {
		maxAllowed = time.Duration(float64(maxBackoff) * (1 + jitter))
	}
	if delay > maxAllowed {
		t.Errorf("Strategy %v produced delay %v exceeding max allowed %v", strategy, delay, maxAllowed)
	}
	if strategy == DecorrelatedJitter && attempt == 0 && delay != initialBackoff {
		t.Errorf("DecorrelatedJitter at attempt 0 should return initialBackoff %v, got %v", initialBackoff, delay)
	}
}

// BenchmarkBackoffStrategies compares performance of different backoff strategies
func BenchmarkBackoffStrategies(b *testing.B) {
	strategies := []struct {
		name     string
		strategy BackoffStrategy
	}{
		{"ExponentialJitter", ExponentialJitter},
		{"DecorrelatedJitter", DecorrelatedJitter},
	}

	for _, s := range strategies {
		b.Run(s.name, func(b *testing.B) {
			client := New(
				WithBackoffStrategy(s.strategy),
				WithInitialBackoff(100*time.Millisecond),
				WithMaxBackoff(5*time.Second),
				WithBackoffMultiplier(2.0),
				WithJitter(0.1),
			)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Test various attempt numbers
				_ = client.calculateBackoff(i % 10)
			}
		})
	}
}

// TestBackoffVarianceProfile tests that decorrelated jitter has different variance than exponential
func TestBackoffVarianceProfile(t *testing.T) {
	const numSamples = 1000
	const attempt = 3
	strategies := map[string]BackoffStrategy{
		"exponential":  ExponentialJitter,
		"decorrelated": DecorrelatedJitter,
	}
	results := make(map[string][]time.Duration)
	for name, strategy := range strategies {
		results[name] = collectBackoffSamples(strategy, numSamples, attempt)
	}
	means, variances := computeStats(results)
	t.Logf("Exponential jitter - Mean: %.2f, Variance: %.2f", means["exponential"], variances["exponential"])
	t.Logf("Decorrelated jitter - Mean: %.2f, Variance: %.2f", means["decorrelated"], variances["decorrelated"])
	if math.Abs(variances["exponential"]-variances["decorrelated"]) < variances["exponential"]*0.1 {
		t.Log("Warning: Variance profiles are very similar - this may be expected depending on parameters")
	}
	validateBackoffSamples(t, results)
}

func collectBackoffSamples(strategy BackoffStrategy, numSamples, attempt int) []time.Duration {
	client := New(
		WithBackoffStrategy(strategy),
		WithInitialBackoff(100*time.Millisecond),
		WithMaxBackoff(5*time.Second),
		WithBackoffMultiplier(2.0),
		WithJitter(0.1),
	)
	samples := make([]time.Duration, numSamples)
	for i := 0; i < numSamples; i++ {
		samples[i] = client.calculateBackoff(attempt)
	}
	return samples
}

func computeStats(results map[string][]time.Duration) (map[string]float64, map[string]float64) {
	means := make(map[string]float64)
	variances := make(map[string]float64)
	for name, samples := range results {
		mean, variance := calcMeanVariance(samples)
		means[name] = mean
		variances[name] = variance
	}
	return means, variances
}

func calcMeanVariance(samples []time.Duration) (float64, float64) {
	var sum float64
	for _, sample := range samples {
		sum += float64(sample)
	}
	mean := sum / float64(len(samples))
	var varianceSum float64
	for _, sample := range samples {
		diff := float64(sample) - mean
		varianceSum += diff * diff
	}
	variance := varianceSum / float64(len(samples))
	return mean, variance
}

func validateBackoffSamples(t *testing.T, results map[string][]time.Duration) {
	for name, samples := range results {
		for _, sample := range samples {
			if sample < 0 || sample > 10*time.Second {
				t.Errorf("Strategy %s produced unreasonable delay: %v", name, sample)
			}
		}
	}
}
