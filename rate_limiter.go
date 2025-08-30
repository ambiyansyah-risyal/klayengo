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
	rl.refillTokens()
	return rl.consumeToken()
}

// refillTokens refills tokens based on elapsed time since last refill
func (rl *RateLimiter) refillTokens() {
	now := time.Now().UnixNano()

	for {
		currentTokens := atomic.LoadInt64(&rl.tokens)
		lastRefill := atomic.LoadInt64(&rl.lastRefill)

		// Calculate tokens to add
		elapsed := now - lastRefill
		tokensToAdd := int64(0)
		if rl.refillRate > 0 {
			tokensToAdd = elapsed / int64(rl.refillRate)
		}

		if tokensToAdd == 0 {
			// No tokens to add, exit
			break
		}

		newTokens := currentTokens + tokensToAdd
		if newTokens > rl.maxTokens {
			newTokens = rl.maxTokens
		}

		// Update last refill time
		newLastRefill := lastRefill + (tokensToAdd * int64(rl.refillRate))

		// Try to update lastRefill first
		if !atomic.CompareAndSwapInt64(&rl.lastRefill, lastRefill, newLastRefill) {
			// If lastRefill changed, retry
			continue
		}

		// Now update tokens
		atomic.StoreInt64(&rl.tokens, newTokens)
		break
	}
}

// consumeToken attempts to consume one token
func (rl *RateLimiter) consumeToken() bool {
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
