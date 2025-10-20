package klayengo

import (
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestOptimizedCircuitBreaker(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold: 3,
		RecoveryTimeout:  100 * time.Millisecond,
		SuccessThreshold: 2,
	}

	cb := NewOptimizedCircuitBreaker(config)

	t.Run("InitialState", func(t *testing.T) {
		if !cb.Allow() {
			t.Error("Circuit breaker should initially allow requests")
		}

		stats := cb.GetStats()
		if stats.State != StateClosed {
			t.Errorf("Expected initial state to be Closed, got %v", stats.State)
		}
	})

	t.Run("FailureThreshold", func(t *testing.T) {
		cb.Reset() // Reset to clean state

		// Record failures up to threshold
		for i := 0; i < 3; i++ {
			cb.RecordFailure()
		}

		// Should now be open
		if cb.Allow() {
			t.Error("Circuit breaker should be open after threshold failures")
		}

		stats := cb.GetStats()
		if stats.State != StateOpen {
			t.Errorf("Expected state to be Open, got %v", stats.State)
		}
	})

	t.Run("RecoveryFlow", func(t *testing.T) {
		cb.Reset()

		// Trigger open state
		for i := 0; i < 3; i++ {
			cb.RecordFailure()
		}

		// Wait for recovery timeout
		time.Sleep(150 * time.Millisecond)

		// Should transition to half-open
		if !cb.Allow() {
			t.Error("Circuit breaker should allow request after recovery timeout")
		}

		// Record enough successes to close
		cb.RecordSuccess()
		cb.RecordSuccess()

		stats := cb.GetStats()
		if stats.State != StateClosed {
			t.Errorf("Expected state to be Closed after successes, got %v", stats.State)
		}
	})

	t.Run("ConcurrentAccess", func(t *testing.T) {
		cb.Reset()

		var wg sync.WaitGroup
		numGoroutines := 100

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					cb.Allow()
					if j%2 == 0 {
						cb.RecordSuccess()
					} else {
						cb.RecordFailure()
					}
				}
			}()
		}

		wg.Wait()

		// Should not panic and should have valid state
		stats := cb.GetStats()
		if stats.Failures < 0 || stats.Successes < 0 {
			t.Error("Circuit breaker counters should not be negative after concurrent access")
		}
	})

	t.Run("MemoryFootprint", func(t *testing.T) {
		size := cb.MemoryFootprint()
		if size == 0 {
			t.Error("Memory footprint should be greater than 0")
		}
		t.Logf("OptimizedCircuitBreaker memory footprint: %d bytes", size)
	})
}

func TestOptimizedRateLimiter(t *testing.T) {
	t.Run("BasicOperation", func(t *testing.T) {
		rl := NewOptimizedRateLimiter(10, time.Second)

		// Should initially allow requests
		if !rl.Allow() {
			t.Error("Rate limiter should initially allow requests")
		}

		stats := rl.GetStats()
		if stats.MaxTokens != 10 {
			t.Errorf("Expected max tokens 10, got %d", stats.MaxTokens)
		}
	})

	t.Run("TokenConsumption", func(t *testing.T) {
		rl := NewOptimizedRateLimiter(5, time.Second)

		// Consume all tokens
		for i := 0; i < 5; i++ {
			if !rl.Allow() {
				t.Errorf("Should allow request %d", i+1)
			}
		}

		// Should now deny
		if rl.Allow() {
			t.Error("Should deny request when tokens exhausted")
		}
	})

	t.Run("AllowN", func(t *testing.T) {
		rl := NewOptimizedRateLimiter(10, time.Second)

		// Should allow consuming 5 tokens
		if !rl.AllowN(5) {
			t.Error("Should allow consuming 5 tokens")
		}

		// Should not allow consuming more than available
		if rl.AllowN(6) {
			t.Error("Should not allow consuming more tokens than available")
		}

		// Should allow consuming remaining tokens
		if !rl.AllowN(5) {
			t.Error("Should allow consuming remaining 5 tokens")
		}
	})

	t.Run("Reserve", func(t *testing.T) {
		rl := NewOptimizedRateLimiter(1, 100*time.Millisecond)

		// First request should be available now
		reserveTime := rl.Reserve()
		if time.Since(reserveTime) > time.Millisecond {
			t.Error("First reserve should be available immediately")
		}

		// Second request should be delayed
		reserveTime2 := rl.Reserve()
		if reserveTime2.Before(time.Now()) {
			t.Error("Second reserve should be in the future")
		}
	})

	t.Run("DynamicRateChange", func(t *testing.T) {
		rl := NewOptimizedRateLimiter(10, time.Second)

		originalRate := rl.Rate()

		// Change rate
		rl.SetRate(2 * time.Second)
		newRate := rl.Rate()

		if newRate == originalRate {
			t.Error("Rate should change after SetRate")
		}

		if newRate != 2*time.Second {
			t.Errorf("Expected new rate %v, got %v", 2*time.Second, newRate)
		}
	})

	t.Run("ConcurrentAccess", func(t *testing.T) {
		rl := NewOptimizedRateLimiter(1000, time.Second)

		var wg sync.WaitGroup
		numGoroutines := 50
		allowed := make([]int, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				count := 0
				for j := 0; j < 100; j++ {
					if rl.Allow() {
						count++
					}
				}
				allowed[index] = count
			}(i)
		}

		wg.Wait()

		totalAllowed := 0
		for _, count := range allowed {
			totalAllowed += count
		}

		// Should allow approximately the number of available tokens
		// Allow some margin for concurrent access patterns and refill during test
		if totalAllowed > 1100 { // Allow 10% margin for refill + concurrency
			t.Errorf("Total allowed (%d) significantly exceeds expected capacity (~1000)", totalAllowed)
		}

		// Log the result for analysis
		t.Logf("Concurrent access allowed %d tokens out of 1000 capacity", totalAllowed)
	})

	t.Run("MemoryFootprint", func(t *testing.T) {
		rl := NewOptimizedRateLimiter(100, time.Second)
		size := rl.MemoryFootprint()
		if size == 0 {
			t.Error("Memory footprint should be greater than 0")
		}
		t.Logf("OptimizedRateLimiter memory footprint: %d bytes", size)
	})
}

func TestOptimizedCache(t *testing.T) {
	t.Run("BasicOperations", func(t *testing.T) {
		cache := NewOptimizedCache()

		entry := &CacheEntry{
			Body:       []byte("test data"),
			StatusCode: 200,
			Header:     make(http.Header),
		}
		entry.Header.Set("Content-Type", "application/json")

		// Set and get
		cache.Set("key1", entry, time.Hour)

		retrieved, found := cache.Get("key1")
		if !found {
			t.Error("Should find cached entry")
		}

		if string(retrieved.Body) != "test data" {
			t.Errorf("Expected body 'test data', got '%s'", string(retrieved.Body))
		}

		if retrieved.StatusCode != 200 {
			t.Errorf("Expected status code 200, got %d", retrieved.StatusCode)
		}
	})

	t.Run("Expiration", func(t *testing.T) {
		cache := NewOptimizedCache()

		entry := &CacheEntry{
			Body:       []byte("test data"),
			StatusCode: 200,
			Header:     make(http.Header),
		}

		// Set with short TTL
		cache.Set("expiring_key", entry, 50*time.Millisecond)

		// Should be available immediately
		_, found := cache.Get("expiring_key")
		if !found {
			t.Error("Should find entry before expiration")
		}

		// Wait for expiration
		time.Sleep(60 * time.Millisecond)

		// Should be expired
		_, found = cache.Get("expiring_key")
		if found {
			t.Error("Should not find expired entry")
		}
	})

	t.Run("Delete", func(t *testing.T) {
		cache := NewOptimizedCache()

		entry := &CacheEntry{
			Body:       []byte("test data"),
			StatusCode: 200,
			Header:     make(http.Header),
		}

		cache.Set("delete_key", entry, time.Hour)

		// Should exist
		_, found := cache.Get("delete_key")
		if !found {
			t.Error("Should find entry before deletion")
		}

		// Delete
		cache.Delete("delete_key")

		// Should not exist
		_, found = cache.Get("delete_key")
		if found {
			t.Error("Should not find deleted entry")
		}
	})

	t.Run("Clear", func(t *testing.T) {
		cache := NewOptimizedCache()

		entry := &CacheEntry{
			Body:       []byte("test data"),
			StatusCode: 200,
			Header:     make(http.Header),
		}

		// Add multiple entries
		for i := 0; i < 10; i++ {
			key := "key_" + string(rune('0'+i))
			cache.Set(key, entry, time.Hour)
		}

		// Clear all
		cache.Clear()

		// Should not find any
		for i := 0; i < 10; i++ {
			key := "key_" + string(rune('0'+i))
			_, found := cache.Get(key)
			if found {
				t.Errorf("Should not find entry %s after clear", key)
			}
		}
	})

	t.Run("Statistics", func(t *testing.T) {
		cache := NewOptimizedCache()

		entry := &CacheEntry{
			Body:       []byte("test data"),
			StatusCode: 200,
			Header:     make(http.Header),
		}

		// Generate some activity
		cache.Set("stats_key", entry, time.Hour)
		cache.Get("stats_key") // hit
		cache.Get("miss_key")  // miss

		stats := cache.GetStats()

		if stats.TotalHits == 0 {
			t.Error("Should have recorded hits")
		}

		if stats.TotalMisses == 0 {
			t.Error("Should have recorded misses")
		}

		if stats.TotalSets == 0 {
			t.Error("Should have recorded sets")
		}

		if stats.HitRatio <= 0 || stats.HitRatio > 1 {
			t.Errorf("Hit ratio should be between 0 and 1, got %f", stats.HitRatio)
		}

		t.Logf("Cache stats - Hits: %d, Misses: %d, Hit ratio: %.2f",
			stats.TotalHits, stats.TotalMisses, stats.HitRatio)
	})

	t.Run("ConcurrentAccess", func(t *testing.T) {
		cache := NewOptimizedCache()

		var wg sync.WaitGroup
		numGoroutines := 50

		entry := &CacheEntry{
			Body:       []byte("concurrent test data"),
			StatusCode: 200,
			Header:     make(http.Header),
		}

		// Concurrent reads and writes
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()

				key := "concurrent_key"
				for j := 0; j < 100; j++ {
					if j%4 == 0 {
						// 25% writes
						cache.Set(key, entry, time.Hour)
					} else {
						// 75% reads
						cache.Get(key)
					}
				}
			}(i)
		}

		wg.Wait()

		// Should not panic and should have valid statistics
		stats := cache.GetStats()
		if stats.TotalHits < 0 || stats.TotalMisses < 0 || stats.TotalSets < 0 {
			t.Error("Cache statistics should not be negative after concurrent access")
		}
	})
}

func TestOptimizedCacheProvider(t *testing.T) {
	cache := NewOptimizedCache()
	provider := NewOptimizedCacheProvider(cache, time.Hour, TTLOnly)

	t.Run("ProviderInterface", func(t *testing.T) {
		// Create a response
		resp := &http.Response{
			StatusCode: 200,
			Header:     make(http.Header),
		}
		resp.Header.Set("Content-Type", "application/json")

		// Set through provider
		provider.Set(nil, "provider_key", resp, time.Hour)

		// Get through provider
		retrieved, found := provider.Get(nil, "provider_key")
		if !found {
			t.Error("Should find entry through provider interface")
		}

		if retrieved.StatusCode != 200 {
			t.Errorf("Expected status code 200, got %d", retrieved.StatusCode)
		}

		// Invalidate
		provider.Invalidate(nil, "provider_key")

		_, found = provider.Get(nil, "provider_key")
		if found {
			t.Error("Should not find entry after invalidation")
		}
	})
}

// Integration test combining all optimized components
func TestOptimizedIntegration(t *testing.T) {
	cb := NewOptimizedCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 5,
		RecoveryTimeout:  100 * time.Millisecond,
		SuccessThreshold: 2,
	})

	rl := NewOptimizedRateLimiter(100, time.Second)
	cache := NewOptimizedCache()

	entry := &CacheEntry{
		Body:       []byte("integration test data"),
		StatusCode: 200,
		Header:     make(http.Header),
	}

	t.Run("IntegratedWorkflow", func(t *testing.T) {
		// Simulate request processing workflow
		if !cb.Allow() {
			t.Fatal("Circuit breaker should allow initial request")
		}

		if !rl.Allow() {
			t.Fatal("Rate limiter should allow initial request")
		}

		// Cache miss scenario
		_, found := cache.Get("integration_key")
		if found {
			t.Error("Should not find entry initially")
		}

		// Store result in cache
		cache.Set("integration_key", entry, time.Hour)
		cb.RecordSuccess()

		// Cache hit scenario
		retrieved, found := cache.Get("integration_key")
		if !found {
			t.Error("Should find cached entry")
		}

		if string(retrieved.Body) != "integration test data" {
			t.Error("Cached data should match original")
		}
	})

	t.Run("LoadTest", func(t *testing.T) {
		var wg sync.WaitGroup
		numWorkers := 20
		requestsPerWorker := 100

		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()

				for j := 0; j < requestsPerWorker; j++ {
					// Check circuit breaker
					if cb.Allow() {
						// Check rate limiter
						if rl.Allow() {
							// Check cache
							key := "load_test_key"
							if _, found := cache.Get(key); !found {
								// Simulate cache miss - store data
								cache.Set(key, entry, time.Minute)
							}
							cb.RecordSuccess()
						}
					}
				}
			}(i)
		}

		wg.Wait()

		// Verify all components are still functional
		if !cb.Allow() && cb.GetStats().State == StateClosed {
			t.Error("Circuit breaker should still allow requests if closed")
		}

		cacheStats := cache.GetStats()
		t.Logf("Load test results - Cache hits: %d, misses: %d, hit ratio: %.2f",
			cacheStats.TotalHits, cacheStats.TotalMisses, cacheStats.HitRatio)

		rlStats := rl.GetStats()
		t.Logf("Rate limiter utilization: %.2f", rlStats.Utilization)
	})
}
