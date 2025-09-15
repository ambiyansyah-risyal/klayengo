// Example demonstrating klayengo's resilient HTTP client capabilities:
// 1. Basic client with retry, rate limiting, caching, circuit breaking, and deduplication
// 2. Advanced client with custom retry conditions, middleware, and metrics
// 3. New retry policy system with Retry-After header support and configurable backoff
// 4. Retry budget functionality to prevent retry storms and resource exhaustion
//
// This example showcases the features implemented in GitHub issue #10:
// - Retry-After header parsing and respect
// - Global retry budget with time windows
// - Idempotent method detection
// - Enhanced retry policy interface
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/ambiyansyah-risyal/klayengo"
)

const httpbinJSON = "https://httpbin.org/json"

func main() {
	// --- Basic client (batteries‑included defaults) ---
	basic := klayengo.New(
		klayengo.WithMaxRetries(3),
		klayengo.WithInitialBackoff(100*time.Millisecond),
		klayengo.WithMaxBackoff(5*time.Second),
		klayengo.WithRateLimiter(10, time.Second),
		klayengo.WithCache(2*time.Minute),
		klayengo.WithCircuitBreaker(klayengo.CircuitBreakerConfig{}),
		klayengo.WithDeduplication(),
		klayengo.WithSimpleLogger(),
		klayengo.WithDebug(),
	)
	if !basic.IsValid() {
		log.Fatalf("invalid basic client config: %v", basic.ValidationError())
	}
	ctx := context.Background()
	resp, err := basic.Get(ctx, httpbinJSON)
	if err != nil {
		log.Fatalf("basic GET failed: %v", err)
	}
	_ = resp.Body.Close()
	fmt.Println("basic GET status", resp.StatusCode)

	// --- Advanced snippet: custom retry condition + middleware + metrics ---
	advanced := klayengo.New(
		klayengo.WithRetryCondition(func(r *http.Response, e error) bool { return e != nil || (r != nil && r.StatusCode >= 500) }),
		klayengo.WithMiddleware(func(req *http.Request, next klayengo.RoundTripper) (*http.Response, error) {
			req.Header.Set("User-Agent", "klayengo-example")
			start := time.Now()
			res, err := next.RoundTrip(req)
			fmt.Printf("request %s took %v\n", req.URL.Host, time.Since(start))
			return res, err
		}),
		klayengo.WithMetrics(),
		klayengo.WithCircuitBreaker(klayengo.CircuitBreakerConfig{FailureThreshold: 3, RecoveryTimeout: 5 * time.Second, SuccessThreshold: 1}),
		klayengo.WithMaxRetries(2),
	)
	r2, err := advanced.Get(ctx, httpbinJSON)
	if err != nil {
		log.Fatalf("advanced GET failed: %v", err)
	}
	_ = r2.Body.Close()
	fmt.Println("advanced GET status", r2.StatusCode)

	// --- Retry Policy with Retry-After header and budget ---
	retryPolicy := klayengo.NewDefaultRetryPolicy(
		3,                    // maxRetries
		100*time.Millisecond, // initialBackoff
		5*time.Second,        // maxBackoff
		2.0,                  // multiplier
		0.1,                  // jitter (10% randomness)
	)

	retryClient := klayengo.New(
		klayengo.WithRetryPolicy(retryPolicy),
		klayengo.WithRetryBudget(5, time.Minute), // 5 retries per minute budget
		klayengo.WithSimpleLogger(),
		klayengo.WithDebug(),
		// Note: not using WithMetrics() to avoid duplicate registration in this example
	)

	if !retryClient.IsValid() {
		log.Fatalf("invalid retry client config: %v", retryClient.ValidationError())
	}

	fmt.Println("\n--- Testing Retry Policy with Retry-After Support ---")
	r3, err := retryClient.Get(ctx, httpbinJSON)
	if err != nil {
		// This is expected to work since httpbin.org should be available
		log.Printf("retry policy GET failed (this might be expected): %v", err)
	} else {
		_ = r3.Body.Close()
		fmt.Printf("retry policy GET successful, status: %d\n", r3.StatusCode)
	}

	// --- Demonstrate retry budget exhaustion with a failing endpoint ---
	fmt.Println("\n--- Testing Retry Budget Exhaustion ---")

	// Create a client with a very small retry budget
	budgetClient := klayengo.New(
		klayengo.WithRetryBudget(2, 10*time.Second), // Only 2 retries in 10 seconds
		klayengo.WithMaxRetries(5),                  // Would normally retry 5 times
		klayengo.WithInitialBackoff(10*time.Millisecond),
		klayengo.WithSimpleLogger(),
	)

	// Try to hit a non-existent endpoint that will trigger retries
	_, err1 := budgetClient.Get(ctx, "https://httpbin.org/status/500")
	if err1 != nil {
		fmt.Printf("First request failed as expected: %v\n", err1)
	}

	// This should fail immediately due to retry budget exhaustion
	_, err2 := budgetClient.Get(ctx, "https://httpbin.org/status/500")
	if err2 != nil {
		fmt.Printf("Second request failed due to retry budget: %v\n", err2)
	}
}
