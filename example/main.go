// Minimal example for klayengo demonstrating a basic resilient GET plus
// a slightly more advanced client showing custom retry logic, middleware,
// metrics and circuit breaking. The full original verbose scenarios were
// removed intentionally to keep the example approachable. See README for
// extended patterns.
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
}
