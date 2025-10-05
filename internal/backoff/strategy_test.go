package backoff

import (
	"testing"
	"time"
)

func TestExponentialJitterStrategy(t *testing.T) {
	strategy := ExponentialJitterStrategy{}

	tests := []struct {
		name        string
		attempt     int
		initial     time.Duration
		max         time.Duration
		multiplier  float64
		jitter      float64
		expected    time.Duration
	}{
		{
			name:       "attempt 0",
			attempt:    0,
			initial:    100 * time.Millisecond,
			max:        5 * time.Second,
			multiplier: 2.0,
			jitter:     0.0, // No jitter for predictable testing
			expected:   100 * time.Millisecond,
		},
		{
			name:       "attempt 1",
			attempt:    1,
			initial:    100 * time.Millisecond,
			max:        5 * time.Second,
			multiplier: 2.0,
			jitter:     0.0,
			expected:   200 * time.Millisecond,
		},
		{
			name:       "attempt 2",
			attempt:    2,
			initial:    100 * time.Millisecond,
			max:        5 * time.Second,
			multiplier: 2.0,
			jitter:     0.0,
			expected:   400 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := strategy.Calculate(tt.attempt, tt.initial, tt.max, tt.multiplier, tt.jitter)
			if result != tt.expected {
				t.Errorf("Calculate(%d, %v, %v, %f, %f) = %v, want %v", 
					tt.attempt, tt.initial, tt.max, tt.multiplier, tt.jitter, result, tt.expected)
			}
		})
	}
}

func TestDecorrelatedJitterStrategy(t *testing.T) {
	strategy := DecorrelatedJitterStrategy{}

	tests := []struct {
		name        string
		attempt     int
		initial     time.Duration
		max         time.Duration
		minExpected time.Duration
		maxExpected time.Duration
	}{
		{
			name:        "attempt 0",
			attempt:     0,
			initial:     100 * time.Millisecond,
			max:         5 * time.Second,
			minExpected: 100 * time.Millisecond,
			maxExpected: 100 * time.Millisecond, // Should be exactly initial
		},
		{
			name:        "attempt 1",
			attempt:     1,
			initial:     100 * time.Millisecond,
			max:         5 * time.Second,
			minExpected: 100 * time.Millisecond, // base
			maxExpected: 300 * time.Millisecond, // base * 3^1
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := strategy.Calculate(tt.attempt, tt.initial, tt.max, 2.0, 0.0)
			if result < tt.minExpected || result > tt.maxExpected {
				t.Errorf("Calculate(%d) = %v, want between %v and %v", 
					tt.attempt, result, tt.minExpected, tt.maxExpected)
			}
		})
	}
}

func TestClampJitter(t *testing.T) {
	tests := []struct {
		input    float64
		expected float64
	}{
		{-0.5, 0.0},
		{0.0, 0.0},
		{0.5, 0.5},
		{1.0, 1.0},
		{1.5, 1.0},
	}

	for _, tt := range tests {
		result := clampJitter(tt.input)
		if result != tt.expected {
			t.Errorf("clampJitter(%f) = %f, want %f", tt.input, result, tt.expected)
		}
	}
}

func TestPow(t *testing.T) {
	tests := []struct {
		base     float64
		exponent int
		expected float64
	}{
		{2.0, 0, 1.0},
		{2.0, 1, 2.0},
		{2.0, 3, 8.0},
		{3.0, 2, 9.0},
	}

	for _, tt := range tests {
		result := Pow(tt.base, tt.exponent)
		if result != tt.expected {
			t.Errorf("Pow(%f, %d) = %f, want %f", tt.base, tt.exponent, result, tt.expected)
		}
	}
}

func BenchmarkExponentialJitterStrategy(b *testing.B) {
	strategy := ExponentialJitterStrategy{}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		strategy.Calculate(i%10, 100*time.Millisecond, 5*time.Second, 2.0, 0.1)
	}
}

func BenchmarkDecorrelatedJitterStrategy(b *testing.B) {
	strategy := DecorrelatedJitterStrategy{}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		strategy.Calculate(i%10, 100*time.Millisecond, 5*time.Second, 2.0, 0.1)
	}
}