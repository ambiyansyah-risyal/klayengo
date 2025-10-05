package backoff

import (
	"testing"
	"time"
)

func TestCalculator(t *testing.T) {
	// Test with exponential jitter strategy
	calc := NewCalculator(ExponentialJitterStrategy{})

	result := calc.Calculate(1, 100*time.Millisecond, 5*time.Second, 2.0, 0.0)
	expected := 200 * time.Millisecond
	if result != expected {
		t.Errorf("Calculate(1) = %v, want %v", result, expected)
	}

	// Test strategy switching
	calc.SetStrategy(DecorrelatedJitterStrategy{})
	result = calc.Calculate(0, 100*time.Millisecond, 5*time.Second, 2.0, 0.0)
	expected = 100 * time.Millisecond
	if result != expected {
		t.Errorf("After switching strategy, Calculate(0) = %v, want %v", result, expected)
	}

	// Test getter
	strategy := calc.GetStrategy()
	if _, ok := strategy.(DecorrelatedJitterStrategy); !ok {
		t.Errorf("GetStrategy() returned wrong type: %T", strategy)
	}
}

func TestGetExponentialJitterCalculator(t *testing.T) {
	calc := GetExponentialJitterCalculator()
	if calc == nil {
		t.Fatal("GetExponentialJitterCalculator() returned nil")
	}

	// Verify it's using the right strategy
	if _, ok := calc.GetStrategy().(ExponentialJitterStrategy); !ok {
		t.Errorf("GetExponentialJitterCalculator() returned wrong strategy type: %T", calc.GetStrategy())
	}
}

func TestGetDecorrelatedJitterCalculator(t *testing.T) {
	calc := GetDecorrelatedJitterCalculator()
	if calc == nil {
		t.Fatal("GetDecorrelatedJitterCalculator() returned nil")
	}

	// Verify it's using the right strategy
	if _, ok := calc.GetStrategy().(DecorrelatedJitterStrategy); !ok {
		t.Errorf("GetDecorrelatedJitterCalculator() returned wrong strategy type: %T", calc.GetStrategy())
	}
}

func BenchmarkCalculatorExponential(b *testing.B) {
	calc := GetExponentialJitterCalculator()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		calc.Calculate(i%10, 100*time.Millisecond, 5*time.Second, 2.0, 0.1)
	}
}

func BenchmarkCalculatorDecorrelated(b *testing.B) {
	calc := GetDecorrelatedJitterCalculator()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		calc.Calculate(i%10, 100*time.Millisecond, 5*time.Second, 2.0, 0.1)
	}
}