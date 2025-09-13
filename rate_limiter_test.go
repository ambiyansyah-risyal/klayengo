package klayengo

import (
	"sync"
	"testing"
	"time"
)

func TestNewRateLimiter(t *testing.T) {
	rl := NewRateLimiter(10, 1*time.Second)

	if rl == nil {
		t.Fatal("NewRateLimiter() returned nil")
	}

	if rl.maxTokens != 10 {
		t.Errorf("Expected maxTokens=10, got %d", rl.maxTokens)
	}

	if rl.tokens != 10 {
		t.Errorf("Expected initial tokens=10, got %d", rl.tokens)
	}

	if rl.refillRate != 1*time.Second {
		t.Errorf("Expected refillRate=1s, got %v", rl.refillRate)
	}
}

func TestRateLimiterAllow(t *testing.T) {
	rl := NewRateLimiter(3, 1*time.Second)

	for i := 0; i < 3; i++ {
		if !rl.Allow() {
			t.Errorf("Expected true for request %d", i+1)
		}
	}

	if rl.Allow() {
		t.Error("Expected false for 4th request")
	}

	if rl.tokens != 0 {
		t.Errorf("Expected tokens=0, got %d", rl.tokens)
	}
}

func TestRateLimiterRefill(t *testing.T) {
	rl := NewRateLimiter(2, 50*time.Millisecond)

	rl.Allow()
	rl.Allow()

	if rl.Allow() {
		t.Error("Expected false when no tokens available")
	}

	time.Sleep(60 * time.Millisecond)

	if !rl.Allow() {
		t.Error("Expected true after refill")
	}

	if rl.tokens != 0 {
		t.Errorf("Expected tokens=0 after refill and consumption, got %d", rl.tokens)
	}
}

func TestRateLimiterPartialRefill(t *testing.T) {
	rl := NewRateLimiter(10, 100*time.Millisecond)

	for i := 0; i < 5; i++ {
		rl.Allow()
	}

	if rl.tokens != 5 {
		t.Errorf("Expected tokens=5, got %d", rl.tokens)
	}

	time.Sleep(110 * time.Millisecond)

	rl.Allow()

	if rl.tokens != 5 {
		t.Errorf("Expected tokens=5 after partial refill, got %d", rl.tokens)
	}
}

func TestRateLimiterMaxTokens(t *testing.T) {
	rl := NewRateLimiter(3, 50*time.Millisecond)

	for i := 0; i < 3; i++ {
		rl.Allow()
	}

	time.Sleep(200 * time.Millisecond)

	if !rl.Allow() {
		t.Error("Expected true after refill")
	}

	if rl.tokens > 3 {
		t.Errorf("Expected tokens <= 3, got %d", rl.tokens)
	}

	if rl.tokens != 2 {
		t.Errorf("Expected tokens=2 after consumption, got %d", rl.tokens)
	}
}

func TestRateLimiterConcurrentAccess(t *testing.T) {
	rl := NewRateLimiter(100, 10*time.Millisecond)

	var wg sync.WaitGroup
	results := make(chan bool, 200)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				results <- rl.Allow()
			}
		}()
	}

	wg.Wait()
	close(results)

	allowed := 0
	denied := 0
	for result := range results {
		if result {
			allowed++
		} else {
			denied++
		}
	}

	if allowed != 100 {
		t.Errorf("Expected 100 allowed requests, got %d", allowed)
	}

	if denied != 100 {
		t.Errorf("Expected 100 denied requests, got %d", denied)
	}
}

func TestRateLimiterZeroTokens(t *testing.T) {
	rl := NewRateLimiter(0, 1*time.Second)

	if rl.Allow() {
		t.Error("Expected false with 0 max tokens")
	}

	if rl.tokens != 0 {
		t.Errorf("Expected tokens=0, got %d", rl.tokens)
	}
}

func TestRateLimiterFastRefill(t *testing.T) {
	rl := NewRateLimiter(5, 1*time.Millisecond)

	for i := 0; i < 5; i++ {
		rl.Allow()
	}

	time.Sleep(10 * time.Millisecond)

	if !rl.Allow() {
		t.Error("Expected true after refill")
	}

	if rl.tokens != 4 {
		t.Errorf("Expected tokens=4 after fast refill and consumption, got %d", rl.tokens)
	}
}

func TestRateLimiterRefillTiming(t *testing.T) {
	rl := NewRateLimiter(2, 100*time.Millisecond)

	startTime := time.Now()

	rl.Allow()
	rl.Allow()

	if rl.Allow() {
		t.Error("Expected false when no tokens")
	}

	time.Sleep(100 * time.Millisecond)

	if !rl.Allow() {
		t.Error("Expected true after one refill period")
	}

	elapsed := time.Since(startTime)
	if elapsed < 100*time.Millisecond {
		t.Errorf("Test completed too quickly: %v", elapsed)
	}
}

func TestRateLimiterLargeMaxTokens(t *testing.T) {
	rl := NewRateLimiter(1000, 1*time.Second)

	for i := 0; i < 1000; i++ {
		if !rl.Allow() {
			t.Errorf("Expected true for request %d", i+1)
		}
	}

	if rl.Allow() {
		t.Error("Expected false after using all tokens")
	}
}

func TestRateLimiterRefillRateZero(t *testing.T) {
	rl := NewRateLimiter(5, 0)

	for i := 0; i < 5; i++ {
		rl.Allow()
	}

	time.Sleep(100 * time.Millisecond)

	if rl.Allow() {
		t.Error("Expected false with zero refill rate")
	}

	if rl.tokens != 0 {
		t.Errorf("Expected tokens=0 with zero refill rate, got %d", rl.tokens)
	}
}

func TestRateLimiterNegativeRefill(t *testing.T) {
	rl := NewRateLimiter(5, -1*time.Second)

	for i := 0; i < 5; i++ {
		rl.Allow()
	}

	time.Sleep(100 * time.Millisecond)

	if rl.tokens != 0 {
		t.Errorf("Expected tokens=0 with negative refill rate, got %d", rl.tokens)
	}
}

func TestRateLimiterInitialization(t *testing.T) {
	rl := NewRateLimiter(10, 500*time.Millisecond)

	if rl.maxTokens != 10 {
		t.Errorf("Expected maxTokens=10, got %d", rl.maxTokens)
	}

	if rl.tokens != 10 {
		t.Errorf("Expected tokens=10, got %d", rl.tokens)
	}

	if rl.refillRate != 500*time.Millisecond {
		t.Errorf("Expected refillRate=500ms, got %v", rl.refillRate)
	}

	if time.Since(time.Unix(0, rl.lastRefill)) > 10*time.Millisecond {
		t.Error("Last refill time not properly initialized")
	}
}

func TestRateLimiterTokenConsumption(t *testing.T) {
	rl := NewRateLimiter(3, 1*time.Second)

	initialTokens := rl.tokens

	for i := 0; i < 3; i++ {
		if !rl.Allow() {
			t.Errorf("Expected true for consumption %d", i+1)
		}

		expectedTokens := initialTokens - int64(i) - 1
		if rl.tokens != expectedTokens {
			t.Errorf("Expected tokens=%d after consumption %d, got %d",
				expectedTokens, i+1, rl.tokens)
		}
	}

	if rl.Allow() {
		t.Error("Expected false after consuming all tokens")
	}
}

func TestRateLimiterRefillCalculation(t *testing.T) {
	rl := NewRateLimiter(10, 100*time.Millisecond)

	for i := 0; i < 10; i++ {
		rl.Allow()
	}

	rl.lastRefill = time.Now().Add(-250 * time.Millisecond).UnixNano()

	if !rl.Allow() {
		t.Error("Expected true after refill calculation")
	}

	expectedTokens := int64(1)
	if rl.tokens != expectedTokens {
		t.Errorf("Expected tokens=%d after refill and consumption, got %d", expectedTokens, rl.tokens)
	}
}

func TestRateLimiterBoundaryConditions(t *testing.T) {
	rl := NewRateLimiter(1, 1*time.Millisecond)

	if !rl.Allow() {
		t.Error("Expected true for single token")
	}

	if rl.Allow() {
		t.Error("Expected false after using single token")
	}

	rl2 := NewRateLimiter(1000000, 1*time.Nanosecond)

	if !rl2.Allow() {
		t.Error("Expected true for large token count")
	}
}
