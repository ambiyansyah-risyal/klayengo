package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/ambiyansyah-risyal/klayengo"
)

func main() {
	// Advanced client with all features enabled
	client := klayengo.New(
		klayengo.WithMaxRetries(5),
		klayengo.WithInitialBackoff(100*time.Millisecond),
		klayengo.WithMaxBackoff(5*time.Second),
		klayengo.WithJitter(0.2),                    // 20% jitter
		klayengo.WithRateLimiter(10, 1*time.Second), // 10 req/sec
		klayengo.WithCache(2*time.Minute),           // Cache for 2 minutes
		klayengo.WithTimeout(15*time.Second),
		klayengo.WithCircuitBreaker(klayengo.CircuitBreakerConfig{
			FailureThreshold: 5,
			RecoveryTimeout:  10 * time.Second,
			SuccessThreshold: 2,
		}),
		klayengo.WithMiddleware(
			loggingMiddleware,
			metricsMiddleware,
			authMiddleware,
		),
	)

	fmt.Println("=== Advanced Klayengo Example ===")

	// Test caching
	fmt.Println("\n--- Testing Caching ---")
	testURL := "https://httpbin.org/get"

	// First request - should hit server
	fmt.Println("First request (should hit server):")
	resp1, err := client.Get(context.Background(), testURL)
	if err != nil {
		log.Printf("First request failed: %v", err)
	} else {
		fmt.Printf("Response status: %s\n", resp1.Status)
		resp1.Body.Close()
	}

	// Second request - should use cache
	fmt.Println("Second request (should use cache):")
	resp2, err := client.Get(context.Background(), testURL)
	if err != nil {
		log.Printf("Second request failed: %v", err)
	} else {
		fmt.Printf("Response status: %s\n", resp2.Status)
		resp2.Body.Close()
	}

	fmt.Println("Note: Both requests should succeed, second one from cache")

	// Test per-request cache control
	fmt.Println("\n--- Testing Per-Request Cache Control ---")

	// Request with cache disabled
	fmt.Println("Request with cache disabled:")
	ctx := klayengo.WithContextCacheDisabled(context.Background())
	resp3, err := client.Get(ctx, testURL)
	if err != nil {
		log.Printf("Cache disabled request failed: %v", err)
	} else {
		fmt.Printf("Response status: %s\n", resp3.Status)
		resp3.Body.Close()
	}

	// Request with custom TTL
	fmt.Println("Request with custom TTL:")
	ctx = klayengo.WithContextCacheTTL(context.Background(), 30*time.Minute)
	resp4, err := client.Get(ctx, "https://httpbin.org/uuid")
	if err != nil {
		log.Printf("Custom TTL request failed: %v", err)
	} else {
		fmt.Printf("Response status: %s\n", resp4.Status)
		resp4.Body.Close()
	}

	// Test rate limiting
	fmt.Println("\n--- Testing Rate Limiting ---")
	for i := 0; i < 15; i++ {
		resp, err := client.Get(context.Background(), "https://httpbin.org/get")
		if err != nil {
			if err.Error() == "rate limit exceeded" {
				fmt.Printf("Request %d: Rate limit exceeded\n", i+1)
			} else {
				log.Printf("Request %d failed: %v", i+1, err)
			}
		} else {
			fmt.Printf("Request %d: %s\n", i+1, resp.Status)
			resp.Body.Close()
		}
		time.Sleep(100 * time.Millisecond) // Small delay between requests
	}

	// Test circuit breaker with failing endpoint
	fmt.Println("\n--- Testing Circuit Breaker ---")
	failingURL := "https://httpbin.org/status/500"
	for i := 0; i < 8; i++ {
		resp, err := client.Get(context.Background(), failingURL)
		if err != nil {
			if err.Error() == "circuit breaker is open" {
				fmt.Printf("Request %d: Circuit breaker open\n", i+1)
			} else {
				fmt.Printf("Request %d failed: %v\n", i+1, err)
			}
		} else {
			fmt.Printf("Request %d: %s\n", i+1, resp.Status)
			resp.Body.Close()
		}
	}

	fmt.Println("\n=== Example completed ===")
}

// loggingMiddleware logs request details
func loggingMiddleware(req *http.Request, next klayengo.RoundTripper) (*http.Response, error) {
	start := time.Now()
	fmt.Printf("[LOG] Starting %s %s\n", req.Method, req.URL.Path)

	resp, err := next.RoundTrip(req)

	duration := time.Since(start)
	if err != nil {
		fmt.Printf("[LOG] Request failed after %v: %v\n", duration, err)
	} else {
		fmt.Printf("[LOG] Request completed in %v with status %s\n", duration, resp.Status)
	}

	return resp, err
}

// metricsMiddleware collects basic metrics
func metricsMiddleware(req *http.Request, next klayengo.RoundTripper) (*http.Response, error) {
	resp, err := next.RoundTrip(req)

	if err != nil {
		fmt.Printf("[METRICS] Request failed\n")
	} else {
		fmt.Printf("[METRICS] Response status: %s\n", resp.Status)
	}

	return resp, err
}

// authMiddleware adds authentication headers
func authMiddleware(req *http.Request, next klayengo.RoundTripper) (*http.Response, error) {
	// Example: Add API key header
	req.Header.Set("X-API-Key", "example-key")
	fmt.Printf("[AUTH] Added authentication header\n")

	return next.RoundTrip(req)
}
