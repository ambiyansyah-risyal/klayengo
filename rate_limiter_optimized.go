package klayengo

import (
	"sync/atomic"
	"time"
	"unsafe"
)

// OptimizedRateLimiter implements a lock-free token bucket rate limiter
// with atomic token management and efficient refill calculations.
// Designed for high-throughput scenarios with minimal allocation overhead.
type OptimizedRateLimiter struct {
	// Configuration - immutable after creation
	maxTokens     int64
	refillRate    int64 // nanoseconds per token for precise arithmetic
	tokensPerFill int64 // batch refill size for better performance
	
	// Hot path state - cache-line aligned for optimal performance
	tokens     int64 // current available tokens
	lastRefill int64 // last refill timestamp in nanoseconds
	
	// Performance optimization: pre-computed values
	refillThreshold int64 // minimum time between refills
	batchSize       int64 // tokens to add per batch refill
}

// RateLimiterStats provides performance metrics
type RateLimiterStats struct {
	CurrentTokens   int64
	MaxTokens       int64
	RefillRate      time.Duration
	LastRefill      time.Time
	TokensPerSecond float64
	Utilization     float64 // current tokens / max tokens
}

// NewOptimizedRateLimiter creates a high-performance token bucket rate limiter.
// Uses lock-free atomic operations for all token management.
func NewOptimizedRateLimiter(maxTokens int, refillRate time.Duration) *OptimizedRateLimiter {
	if maxTokens <= 0 {
		maxTokens = 1
	}
	if refillRate <= 0 {
		refillRate = time.Second
	}
	
	// Calculate refill rate in nanoseconds per token
	refillRateNano := int64(refillRate) / int64(maxTokens)
	if refillRateNano == 0 {
		refillRateNano = 1 // minimum 1ns per token
	}
	
	// Optimize batch refill size based on rate
	batchSize := int64(1)
	if refillRate >= time.Second {
		batchSize = int64(maxTokens) / 10 // refill 10% at a time for smoother distribution
		if batchSize == 0 {
			batchSize = 1
		}
	}
	
	now := time.Now().UnixNano()
	
	return &OptimizedRateLimiter{
		maxTokens:       int64(maxTokens),
		refillRate:      refillRateNano,
		tokensPerFill:   batchSize,
		tokens:          int64(maxTokens), // start with full bucket
		lastRefill:      now,
		refillThreshold: refillRateNano,
		batchSize:       batchSize,
	}
}

// Allow reports whether a token is available and consumes it atomically.
// This is the hot path optimized for maximum throughput.
//go:noinline
func (rl *OptimizedRateLimiter) Allow() bool {
	// Fast refill check and token consumption in single operation
	rl.fastRefill()
	return rl.consumeTokenFast()
}

// AllowN attempts to consume N tokens atomically.
// Returns true if all N tokens were available and consumed.
func (rl *OptimizedRateLimiter) AllowN(n int64) bool {
	if n <= 0 {
		return true
	}
	if n > rl.maxTokens {
		return false // can never satisfy request larger than bucket
	}
	
	rl.fastRefill()
	
	// Atomic N-token consumption with retry loop
	for {
		current := atomic.LoadInt64(&rl.tokens)
		if current < n {
			return false
		}
		
		if atomic.CompareAndSwapInt64(&rl.tokens, current, current-n) {
			return true
		}
		// CAS failed, retry with updated value
	}
}

// fastRefill optimizes token refill with minimal atomic operations.
// Uses batched refill strategy to reduce contention.
//go:noinline
func (rl *OptimizedRateLimiter) fastRefill() {
	now := time.Now().UnixNano()
	lastRefill := atomic.LoadInt64(&rl.lastRefill)
	
	elapsed := now - lastRefill
	
	// Quick exit if not enough time passed for refill
	if elapsed < rl.refillThreshold {
		return
	}
	
	// Calculate tokens to add based on elapsed time
	tokensToAdd := elapsed / rl.refillRate
	if tokensToAdd == 0 {
		return
	}
	
	// Batch refill optimization: limit maximum tokens added per operation
	maxBatch := rl.batchSize
	if tokensToAdd > maxBatch {
		tokensToAdd = maxBatch
	}
	
	// Try to update last refill timestamp first to prevent races
	newRefillTime := lastRefill + (tokensToAdd * rl.refillRate)
	if !atomic.CompareAndSwapInt64(&rl.lastRefill, lastRefill, newRefillTime) {
		return // Another goroutine updated, skip this refill
	}
	
	// Atomically add tokens without exceeding maximum
	for {
		current := atomic.LoadInt64(&rl.tokens)
		newTokens := current + tokensToAdd
		
		if newTokens > rl.maxTokens {
			newTokens = rl.maxTokens
		}
		
		if newTokens == current {
			break // No change needed
		}
		
		if atomic.CompareAndSwapInt64(&rl.tokens, current, newTokens) {
			break // Successfully updated
		}
		// CAS failed, retry with updated current value
	}
}

// consumeTokenFast optimizes single token consumption with minimal retries.
//go:noinline
func (rl *OptimizedRateLimiter) consumeTokenFast() bool {
	// Optimized CAS loop with early exit
	for attempts := 0; attempts < 3; attempts++ { // limit retry attempts
		current := atomic.LoadInt64(&rl.tokens)
		if current <= 0 {
			return false
		}
		
		if atomic.CompareAndSwapInt64(&rl.tokens, current, current-1) {
			return true
		}
		// Brief pause to reduce contention on high load
		if attempts > 0 {
			time.Sleep(1) // 1 nanosecond pause
		}
	}
	return false // failed after retries
}

// Reserve reserves a token for future use and returns when it will be available.
// This allows for more sophisticated rate limiting patterns.
func (rl *OptimizedRateLimiter) Reserve() time.Time {
	rl.fastRefill()
	
	current := atomic.LoadInt64(&rl.tokens)
	if current > 0 {
		// Token available now
		if atomic.CompareAndSwapInt64(&rl.tokens, current, current-1) {
			return time.Now()
		}
	}
	
	// Calculate wait time for next token
	waitTime := time.Duration(rl.refillRate)
	return time.Now().Add(waitTime)
}

// GetStats returns current rate limiter statistics.
func (rl *OptimizedRateLimiter) GetStats() RateLimiterStats {
	current := atomic.LoadInt64(&rl.tokens)
	lastRefillNano := atomic.LoadInt64(&rl.lastRefill)
	
	var lastRefill time.Time
	if lastRefillNano > 0 {
		lastRefill = time.Unix(0, lastRefillNano)
	}
	
	tokensPerSecond := float64(time.Second) / float64(rl.refillRate)
	utilization := float64(current) / float64(rl.maxTokens)
	
	return RateLimiterStats{
		CurrentTokens:   current,
		MaxTokens:       rl.maxTokens,
		RefillRate:      time.Duration(rl.refillRate * rl.maxTokens),
		LastRefill:      lastRefill,
		TokensPerSecond: tokensPerSecond,
		Utilization:     utilization,
	}
}

// SetRate dynamically updates the refill rate.
// Use carefully as it can cause brief inconsistency.
func (rl *OptimizedRateLimiter) SetRate(newRate time.Duration) {
	if newRate <= 0 {
		return
	}
	
	newRefillRate := int64(newRate) / rl.maxTokens
	if newRefillRate == 0 {
		newRefillRate = 1
	}
	
	atomic.StoreInt64(&rl.refillRate, newRefillRate)
	
	// Update threshold as well
	atomic.StoreInt64(&rl.refillThreshold, newRefillRate)
}

// Reset resets the rate limiter to full capacity.
func (rl *OptimizedRateLimiter) Reset() {
	atomic.StoreInt64(&rl.tokens, rl.maxTokens)
	atomic.StoreInt64(&rl.lastRefill, time.Now().UnixNano())
}

// MemoryFootprint returns the memory usage of this rate limiter instance.
func (rl *OptimizedRateLimiter) MemoryFootprint() uintptr {
	return unsafe.Sizeof(*rl)
}

// Burst returns the maximum burst size (same as max tokens).
func (rl *OptimizedRateLimiter) Burst() int64 {
	return rl.maxTokens
}

// Rate returns the current refill rate.
func (rl *OptimizedRateLimiter) Rate() time.Duration {
	refillRate := atomic.LoadInt64(&rl.refillRate)
	return time.Duration(refillRate * rl.maxTokens)
}