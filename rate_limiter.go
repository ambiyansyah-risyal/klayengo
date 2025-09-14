package klayengo

import (
	"sync/atomic"
	"time"
)

// NewRateLimiter returns a token bucket rate limiter.
func NewRateLimiter(maxTokens int, refillRate time.Duration) *RateLimiter {
	return &RateLimiter{
		maxTokens:  int64(maxTokens),
		tokens:     int64(maxTokens),
		refillRate: refillRate,
		lastRefill: time.Now().UnixNano(),
	}
}

// Allow reports whether a token is available (and consumes it if so).
func (rl *RateLimiter) Allow() bool {
	rl.refillTokens()
	return rl.consumeToken()
}

func (rl *RateLimiter) refillTokens() {
	now := time.Now().UnixNano()

	for {
		currentTokens := atomic.LoadInt64(&rl.tokens)
		lastRefill := atomic.LoadInt64(&rl.lastRefill)

		elapsed := now - lastRefill
		tokensToAdd := int64(0)
		if rl.refillRate > 0 {
			tokensToAdd = elapsed / int64(rl.refillRate)
		}

		if tokensToAdd == 0 {
			break
		}

		newTokens := currentTokens + tokensToAdd
		if newTokens > rl.maxTokens {
			newTokens = rl.maxTokens
		}

		newLastRefill := lastRefill + (tokensToAdd * int64(rl.refillRate))

		if !atomic.CompareAndSwapInt64(&rl.lastRefill, lastRefill, newLastRefill) {
			continue
		}

		atomic.StoreInt64(&rl.tokens, newTokens)
		break
	}
}

func (rl *RateLimiter) consumeToken() bool {
	for {
		currentTokens := atomic.LoadInt64(&rl.tokens)
		if currentTokens <= 0 {
			return false
		}

		if atomic.CompareAndSwapInt64(&rl.tokens, currentTokens, currentTokens-1) {
			return true
		}
	}
}
