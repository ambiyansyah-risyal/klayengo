// Package klayengo provides a resilient HTTP client with composable reliability primitives:
//
//   - Retries with exponential backoff + jitter
//   - Rate limiting (token bucket)
//   - In‑memory response caching with per‑request overrides
//   - Circuit breaker (open / half‑open / closed states)
//   - Request de‑duplication (merges concurrent identical in‑flight requests)
//   - Middleware chain for cross‑cutting concerns (auth, logging, tracing, etc.)
//   - Prometheus metrics and lightweight structured debug logging
//
// Design goals:
//   - Small surface area – functional options configure everything
//   - Zero allocations on hot paths where practical
//   - Safe concurrent use of a single *Client instance
//   - Extensibility via user supplied middleware & pluggable cache / metrics
//
// Typical usage:
//
//	client := klayengo.New(
//	    klayengo.WithMaxRetries(3),
//	    klayengo.WithRateLimiter(10, time.Second),
//	    klayengo.WithCache(5*time.Minute),
//	    klayengo.WithCircuitBreaker(klayengo.CircuitBreakerConfig{}),
//	    klayengo.WithDeduplication(),
//	)
//	resp, err := client.Get(ctx, "https://api.example.com/data")
//
// Only advanced / non 2xx responses trigger retries by default; override with WithRetryCondition.
// The library avoids opinionated logging: provide a Logger (e.g. via WithSimpleLogger) + enable
// debug flags selectively (WithDebug / WithDebugConfig) for insight without noise.
package klayengo
