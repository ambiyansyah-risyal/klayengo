package klayengo

import (
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// BenchmarkCircuitBreakerComparison compares old vs optimized circuit breaker performance
func BenchmarkCircuitBreakerComparison(b *testing.B) {
	config := CircuitBreakerConfig{
		FailureThreshold: 5,
		RecoveryTimeout:  30 * time.Second,
		SuccessThreshold: 2,
	}

	b.Run("OriginalCircuitBreaker", func(b *testing.B) {
		cb := NewCircuitBreaker(config)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				cb.Allow()
				if b.N%2 == 0 {
					cb.RecordSuccess()
				} else {
					cb.RecordFailure()
				}
			}
		})
	})

	b.Run("OptimizedCircuitBreaker", func(b *testing.B) {
		cb := NewOptimizedCircuitBreaker(config)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				cb.Allow()
				if b.N%2 == 0 {
					cb.RecordSuccess()
				} else {
					cb.RecordFailure()
				}
			}
		})
	})
}

// BenchmarkCircuitBreakerHotPath focuses on the hot path performance
func BenchmarkCircuitBreakerHotPath(b *testing.B) {
	config := CircuitBreakerConfig{
		FailureThreshold: 5,
		RecoveryTimeout:  30 * time.Second,
		SuccessThreshold: 2,
	}

	b.Run("Original_Allow", func(b *testing.B) {
		cb := NewCircuitBreaker(config)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cb.Allow()
		}
	})

	b.Run("Optimized_Allow", func(b *testing.B) {
		cb := NewOptimizedCircuitBreaker(config)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cb.Allow()
		}
	})
}

// BenchmarkRateLimiterComparison compares old vs optimized rate limiter performance
func BenchmarkRateLimiterComparison(b *testing.B) {
	b.Run("OriginalRateLimiter", func(b *testing.B) {
		rl := NewRateLimiter(1000, time.Second)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				rl.Allow()
			}
		})
	})

	b.Run("OptimizedRateLimiter", func(b *testing.B) {
		rl := NewOptimizedRateLimiter(1000, time.Second)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				rl.Allow()
			}
		})
	})
}

// BenchmarkRateLimiterHighContention tests performance under high contention
func BenchmarkRateLimiterHighContention(b *testing.B) {
	numGoroutines := runtime.NumCPU() * 4

	b.Run("Original_HighContention", func(b *testing.B) {
		rl := NewRateLimiter(10000, time.Second)
		var wg sync.WaitGroup
		operations := int64(b.N)
		
		b.ResetTimer()
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for atomic.AddInt64(&operations, -1) > 0 {
					rl.Allow()
				}
			}()
		}
		wg.Wait()
	})

	b.Run("Optimized_HighContention", func(b *testing.B) {
		rl := NewOptimizedRateLimiter(10000, time.Second)
		var wg sync.WaitGroup
		operations := int64(b.N)
		
		b.ResetTimer()
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for atomic.AddInt64(&operations, -1) > 0 {
					rl.Allow()
				}
			}()
		}
		wg.Wait()
	})
}

// BenchmarkCacheComparison compares old vs optimized cache performance
func BenchmarkCacheComparison(b *testing.B) {
	// Test data setup
	entry := &CacheEntry{
		Body:       []byte("test response body"),
		StatusCode: 200,
		Header:     make(http.Header),
	}
	entry.Header.Set("Content-Type", "application/json")

	b.Run("OriginalCache_Get", func(b *testing.B) {
		cache := NewInMemoryCache()
		// Pre-populate cache
		for i := 0; i < 1000; i++ {
			key := fmt.Sprintf("key_%d", i)
			cache.Set(key, entry, time.Hour)
		}
		
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				key := fmt.Sprintf("key_%d", i%1000)
				cache.Get(key)
				i++
			}
		})
	})

	b.Run("OptimizedCache_Get", func(b *testing.B) {
		cache := NewOptimizedCache()
		// Pre-populate cache
		for i := 0; i < 1000; i++ {
			key := fmt.Sprintf("key_%d", i)
			cache.Set(key, entry, time.Hour)
		}
		
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				key := fmt.Sprintf("key_%d", i%1000)
				cache.Get(key)
				i++
			}
		})
	})
}

// BenchmarkCacheContention tests cache performance under high lock contention
func BenchmarkCacheContention(b *testing.B) {
	entry := &CacheEntry{
		Body:       []byte("test response body"),
		StatusCode: 200,
		Header:     make(http.Header),
	}

	b.Run("Original_Mixed_Operations", func(b *testing.B) {
		cache := NewInMemoryCache()
		var counter int64
		
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				i := atomic.AddInt64(&counter, 1)
				key := fmt.Sprintf("key_%d", i%100)
				
				if i%4 == 0 {
					// 25% writes
					cache.Set(key, entry, time.Hour)
				} else {
					// 75% reads
					cache.Get(key)
				}
			}
		})
	})

	b.Run("Optimized_Mixed_Operations", func(b *testing.B) {
		cache := NewOptimizedCache()
		var counter int64
		
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				i := atomic.AddInt64(&counter, 1)
				key := fmt.Sprintf("key_%d", i%100)
				
				if i%4 == 0 {
					// 25% writes
					cache.Set(key, entry, time.Hour)
				} else {
					// 75% reads
					cache.Get(key)
				}
			}
		})
	})
}

// BenchmarkMemoryFootprint compares memory usage of old vs new implementations
func BenchmarkMemoryFootprint(b *testing.B) {
	b.Run("CircuitBreaker_Memory", func(b *testing.B) {
		config := CircuitBreakerConfig{
			FailureThreshold: 5,
			RecoveryTimeout:  30 * time.Second,
			SuccessThreshold: 2,
		}
		
		var originalSize, optimizedSize uintptr
		
		// Original circuit breaker
		original := NewCircuitBreaker(config)
		originalSize = original.MemoryFootprint()
		
		// Optimized circuit breaker  
		optimized := NewOptimizedCircuitBreaker(config)
		optimizedSize = optimized.MemoryFootprint()
		
		b.ReportMetric(float64(originalSize), "original_bytes")
		b.ReportMetric(float64(optimizedSize), "optimized_bytes")
		b.ReportMetric(float64(originalSize)/float64(optimizedSize), "size_ratio")
	})

	b.Run("RateLimiter_Memory", func(b *testing.B) {
		// Original rate limiter
		original := NewRateLimiter(1000, time.Second)
		originalSize := original.MemoryFootprint()
		
		// Optimized rate limiter
		optimized := NewOptimizedRateLimiter(1000, time.Second)
		optimizedSize := optimized.MemoryFootprint()
		
		b.ReportMetric(float64(originalSize), "original_bytes")
		b.ReportMetric(float64(optimizedSize), "optimized_bytes")
		b.ReportMetric(float64(originalSize)/float64(optimizedSize), "size_ratio")
	})
}

// BenchmarkConcurrentAccess tests performance under high concurrent access
func BenchmarkConcurrentAccess(b *testing.B) {
	numWorkers := runtime.NumCPU() * 2
	
	b.Run("Original_Components", func(b *testing.B) {
		cb := NewCircuitBreaker(CircuitBreakerConfig{
			FailureThreshold: 10,
			RecoveryTimeout:  30 * time.Second,
			SuccessThreshold: 3,
		})
		rl := NewRateLimiter(10000, time.Second)
		cache := NewInMemoryCache()
		
		var wg sync.WaitGroup
		operations := int64(b.N)
		
		b.ResetTimer()
		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()
				entry := &CacheEntry{
					Body:       []byte(fmt.Sprintf("worker_%d_data", workerID)),
					StatusCode: 200,
					Header:     make(http.Header),
				}
				
				for atomic.AddInt64(&operations, -1) > 0 {
					// Circuit breaker operations
					if cb.Allow() {
						cb.RecordSuccess()
					}
					
					// Rate limiter operations
					rl.Allow()
					
					// Cache operations
					key := fmt.Sprintf("key_%d", workerID)
					cache.Set(key, entry, time.Hour)
					cache.Get(key)
				}
			}(i)
		}
		wg.Wait()
	})
	
	b.Run("Optimized_Components", func(b *testing.B) {
		cb := NewOptimizedCircuitBreaker(CircuitBreakerConfig{
			FailureThreshold: 10,
			RecoveryTimeout:  30 * time.Second,
			SuccessThreshold: 3,
		})
		rl := NewOptimizedRateLimiter(10000, time.Second)
		cache := NewOptimizedCache()
		
		var wg sync.WaitGroup
		operations := int64(b.N)
		
		b.ResetTimer()
		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()
				entry := &CacheEntry{
					Body:       []byte(fmt.Sprintf("worker_%d_data", workerID)),
					StatusCode: 200,
					Header:     make(http.Header),
				}
				
				for atomic.AddInt64(&operations, -1) > 0 {
					// Circuit breaker operations
					if cb.Allow() {
						cb.RecordSuccess()
					}
					
					// Rate limiter operations
					rl.Allow()
					
					// Cache operations
					key := fmt.Sprintf("key_%d", workerID)
					cache.Set(key, entry, time.Hour)
					cache.Get(key)
				}
			}(i)
		}
		wg.Wait()
	})
}

// BenchmarkThroughputComparison measures overall system throughput
func BenchmarkThroughputComparison(b *testing.B) {
	b.Run("Original_System_Throughput", func(b *testing.B) {
		// Setup original components
		cb := NewCircuitBreaker(CircuitBreakerConfig{
			FailureThreshold: 10,
			RecoveryTimeout:  30 * time.Second,  
			SuccessThreshold: 3,
		})
		rl := NewRateLimiter(1000, time.Second)
		cache := NewInMemoryCache()
		
		entry := &CacheEntry{
			Body:       []byte("benchmark response body"),
			StatusCode: 200,
			Header:     make(http.Header),
		}
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Simulate full request pipeline
			if cb.Allow() && rl.Allow() {
				key := fmt.Sprintf("req_%d", i%100)
				if _, found := cache.Get(key); !found {
					cache.Set(key, entry, time.Minute)
				}
				cb.RecordSuccess()
			} else {
				cb.RecordFailure()
			}
		}
	})
	
	b.Run("Optimized_System_Throughput", func(b *testing.B) {
		// Setup optimized components
		cb := NewOptimizedCircuitBreaker(CircuitBreakerConfig{
			FailureThreshold: 10,
			RecoveryTimeout:  30 * time.Second,
			SuccessThreshold: 3,
		})
		rl := NewOptimizedRateLimiter(1000, time.Second)
		cache := NewOptimizedCache()
		
		entry := &CacheEntry{
			Body:       []byte("benchmark response body"),
			StatusCode: 200,
			Header:     make(http.Header),
		}
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Simulate full request pipeline
			if cb.Allow() && rl.Allow() {
				key := fmt.Sprintf("req_%d", i%100)
				if _, found := cache.Get(key); !found {
					cache.Set(key, entry, time.Minute)
				}
				cb.RecordSuccess()
			} else {
				cb.RecordFailure()
			}
		}
	})
}

// Helper method to add MemoryFootprint to existing types for benchmarking
func (cb *CircuitBreaker) MemoryFootprint() uintptr {
	return 64 // Approximate size of CircuitBreaker struct
}

func (rl *RateLimiter) MemoryFootprint() uintptr {
	return 32 // Approximate size of RateLimiter struct  
}

// BenchmarkAllocationProfile measures memory allocations
func BenchmarkAllocationProfile(b *testing.B) {
	b.Run("Original_Zero_Alloc_Path", func(b *testing.B) {
		cb := NewCircuitBreaker(CircuitBreakerConfig{
			FailureThreshold: 5,
			RecoveryTimeout:  30 * time.Second,
			SuccessThreshold: 2,
		})
		rl := NewRateLimiter(1000, time.Second)
		
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cb.Allow()
			rl.Allow()
		}
	})
	
	b.Run("Optimized_Zero_Alloc_Path", func(b *testing.B) {
		cb := NewOptimizedCircuitBreaker(CircuitBreakerConfig{
			FailureThreshold: 5,
			RecoveryTimeout:  30 * time.Second,
			SuccessThreshold: 2,
		})
		rl := NewOptimizedRateLimiter(1000, time.Second)
		
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cb.Allow()
			rl.Allow()
		}
	})
}