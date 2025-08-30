package klayengo

import (
	"sync/atomic"
	"time"
)

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(maxTokens int, refillRate time.Duration) *RateLimiter {
	return &RateLimiter{
		maxTokens:  int64(maxTokens),
		tokens:     int64(maxTokens),
		refillRate: refillRate,
		lastRefill: time.Now().UnixNano(),
	}
}

// Allow checks if a request is allowed by the rate limiter
func (rl *RateLimiter) Allow() bool {
	now := time.Now().UnixNano()

	// Try to refill tokens atomically
	for {
		currentTokens := atomic.LoadInt64(&rl.tokens)
		lastRefill := atomic.LoadInt64(&rl.lastRefill)

		// Calculate tokens to add
		elapsed := now - lastRefill
		tokensToAdd := int64(0)
		if rl.refillRate > 0 {
			tokensToAdd = elapsed / int64(rl.refillRate)
		}

		newTokens := currentTokens + tokensToAdd
		if newTokens > int64(rl.maxTokens) {
			newTokens = int64(rl.maxTokens)
		}

		// Update last refill time
		newLastRefill := lastRefill
		if rl.refillRate > 0 {
			newLastRefill = lastRefill + (tokensToAdd * int64(rl.refillRate))
		}

		// Try to update atomically
		if atomic.CompareAndSwapInt64(&rl.lastRefill, lastRefill, newLastRefill) {
			atomic.StoreInt64(&rl.tokens, newTokens)
			break
		}
		// If CAS failed, retry
	}

	// Try to consume a token
	for {
		currentTokens := atomic.LoadInt64(&rl.tokens)
		if currentTokens <= 0 {
			return false
		}

		if atomic.CompareAndSwapInt64(&rl.tokens, currentTokens, currentTokens-1) {
			return true
		}
		// If CAS failed, retry
	}
}
